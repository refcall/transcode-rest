package main

import "C"
import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/buckket/go-blurhash"
	"github.com/davidbyttow/govips/v2/vips"
)

var (
	// GitHash is set at compilation if available
	GitHash = "unknown"

	// Version is set at compilation if tag
	GitBranch = "unknown"

	// BuildTime is set at compilation
	BuildTime = "unknown"
)

type PdfInfo struct {
	URL    string
	Pages  int
	Height int
	Width  int
}

type Blur struct {
	Code   string
	Height int
	Width  int
}

var tempFiles = map[string]int64{}

func main() {
	fmt.Println("transcode-rest")
	fmt.Println("git hash: " + GitHash)
	fmt.Println("git branch: " + GitBranch)
	fmt.Println("build time: " + BuildTime)
	fmt.Println()

	tmp := getEnv("STORAGE_DIRECTORY", os.TempDir())
	vips.LoggingSettings(func(messageDomain string, messageLevel vips.LogLevel, message string) {
		log.Printf("%v: %v", messageDomain, message)
	}, vips.LogLevelWarning)
	vips.Startup(nil)

	go func() {
		keepFilesDuration, err := time.ParseDuration(getEnv("STORAGE_DURATION", "10m"))
		if err != nil {
			log.Fatal("duration from STORAGE_DURATION is invalid")
		}
		log.Printf("auto delete files older than %s", keepFilesDuration.String())

		for {
			log.Println("cleanup files")
			now := time.Now()
			for k, v := range tempFiles {
				if time.Unix(v, 0).Add(keepFilesDuration).Before(now) {
					log.Println("remove", k)
					if err := os.Remove(k); err != nil {
						log.Fatalf("cannot remove file %s", k)
					}
					delete(tempFiles, k)
				}
			}
			log.Println("cleanup done")
			time.Sleep(1 * time.Minute)
		}
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		str, _ := json.Marshal(map[string]string{
			"gitBranch": GitBranch,
			"gitHash":   GitHash,
			"buildTime": BuildTime,
		})
		w.Write(str)
	})

	http.HandleFunc("/transcode", func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.Query().Get("url")
		if url == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "url param needed")
			return
		}

		h := hash(url)
		name := h + ".mp4"
		file := tmp + "/" + name

		log.Println("transcode", url, "to", file)

		if stat, err := os.Stat(file); err == nil {
			log.Println("  already exist")
			f, err := os.Open(file)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "cannot open file: %s", err.Error())
				return
			}
			defer f.Close()
			http.ServeContent(w, r, name, stat.ModTime(), f)
			tempFiles[file] = time.Now().Unix()
			return
		}

		p, err := probe(url)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "cannot probe: %s", err.Error())
			return
		}

		bitrate := 0
		for _, s := range p.Streams {
			if s.Tags == nil || s.Tags.VariantBitrate == nil || (*s.Tags.VariantBitrate) == "" {
				continue
			}
			b, err := strconv.Atoi(*s.Tags.VariantBitrate)
			if err != nil {
				continue
			}
			if b > bitrate {
				bitrate = b
			}
		}

		log.Println("  start transcode", url, "with bitrate", bitrate, "to", file)

		args := []string{
			"-i", url,
		}
		if bitrate != 0 {
			args = append(args, "-map", "m:variant_bitrate:"+strconv.Itoa(bitrate))
		}
		args = append(args,
			"-c:v", "copy",
			"-c:a", "copy",
			file,
		)
		ffmpeg := exec.Command(
			getEnv("FFMPEG_PATH", "ffmpeg"),
			args...,
		)

		if err := ffmpeg.Start(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "cannot start ffmpeg: %s", err.Error())
			return
		}

		if err := ffmpeg.Wait(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "cannot open file: %s", err.Error())
			return
		}

		log.Println("  serve file", file)
		f, err := os.Open(file)
		if err != nil {
			http.Error(w, "cannot open file: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer f.Close()

		tempFiles[file] = time.Now().Unix()
		http.ServeContent(w, r, name, time.Now(), f)
	})

	http.HandleFunc("/video/thumbnail", func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.Query().Get("url")
		if url == "" {
			http.Error(w, "url param needed", http.StatusBadRequest)
			return
		}

		h := hash(url)
		name := h + ".jpg"
		file := tmp + "/" + name

		log.Println("video thumbnail", url, "to", file)

		if stat, err := os.Stat(file); err == nil {
			log.Println("  already exist")
			f, err := os.Open(file)
			if err != nil {
				http.Error(w, "cannot open file: "+err.Error(), http.StatusInternalServerError)
				return
			}
			defer f.Close()
			http.ServeContent(w, r, name, stat.ModTime(), f)
			tempFiles[file] = time.Now().Unix()
			return
		}

		args := []string{
			"-y",
			"-i", url,
			"-frames:v", "1",
			file,
		}

		ffmpeg := exec.Command(
			getEnv("FFMPEG_PATH", "ffmpeg"),
			args...,
		)

		if err := ffmpeg.Start(); err != nil {
			http.Error(w, "cannot start ffmpeg: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := ffmpeg.Wait(); err != nil {
			http.Error(w, "cannot open file: "+err.Error(), http.StatusInternalServerError)
			return
		}

		log.Println(" . serve file", file)
		f, err := os.Open(file)
		if err != nil {
			http.Error(w, "cannot open file: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer f.Close()

		tempFiles[file] = time.Now().Unix()
		http.ServeContent(w, r, name, time.Now(), f)
	})

	// this leaks for sure
	http.HandleFunc("/blur", func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.Query().Get("url")
		if url == "" {
			http.Error(w, "url param needed: ", http.StatusBadRequest)
			return
		}

		log.Println("blur", url)

		res, err := http.Get(url)
		if err != nil {
			http.Error(w, "cannot get url: "+err.Error(), http.StatusBadRequest)
		}
		defer res.Body.Close()

		loadedImage, _, err := image.Decode(res.Body)
		if err != nil {
			http.Error(w, "cannot decode image: "+err.Error(), http.StatusUnprocessableEntity)
			return
		}

		log.Println("  encode image of", loadedImage.Bounds().Dy(), loadedImage.Bounds().Dx())
		str, _ := blurhash.Encode(4, 3, loadedImage)
		blur := Blur{
			Code:   str,
			Height: loadedImage.Bounds().Dy(),
			Width:  loadedImage.Bounds().Dx(),
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(blur)
		if err != nil {
			http.Error(w, "Failed to encode JSON: "+err.Error(), http.StatusInternalServerError)
			return
		}

	})

	http.HandleFunc("/pdf/info", func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.Query().Get("url")
		if url == "" {
			w.WriteHeader(http.StatusBadRequest)
			http.Error(w, "url param needed: ", http.StatusBadRequest)
			return
		}

		log.Println("pdf info", url)

		res, err := http.Get(url)
		if err != nil {
			http.Error(w, "cannot get url: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer res.Body.Close()

		inputImage, err := vips.NewImageFromReader(res.Body)
		if err != nil {
			http.Error(w, "cannot create a bimg image from reader: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer inputImage.Close()

		pdfInfo := PdfInfo{
			URL:    url,
			Pages:  inputImage.Pages(),
			Height: inputImage.Height(),
			Width:  inputImage.Width(),
		}
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(pdfInfo)
		if err != nil {
			http.Error(w, "Failed to encode JSON: "+err.Error(), http.StatusInternalServerError)
			return
		}
	})

	http.HandleFunc("/pdf/thumbnail", func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.Query().Get("url")
		if url == "" {
			http.Error(w, "url param needed", http.StatusBadRequest)
			return
		}
		page := r.URL.Query().Get("page")
		if page == "" {
			http.Error(w, "page param needed", http.StatusBadRequest)
			return
		}
		intPage, _ := strconv.Atoi(page)

		log.Println("pdf thumbnail", url, "page", intPage)

		res, err := http.Get(url)
		if err != nil {
			http.Error(w, "cannot get url: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer res.Body.Close()
		log.Println("  res status", res.Status)

		bytesRes, err := io.ReadAll(res.Body)
		if err != nil {
			http.Error(w, "cannot extract bytes from url body: "+err.Error(), http.StatusInternalServerError)
			return
		}

		ip := vips.NewImportParams()
		ip.Page.Set(intPage)
		ip.Density.Set(120)

		log.Println("  loading image from buffer")
		imageFile, err := vips.LoadImageFromBuffer(bytesRes, ip)
		if err != nil {
			http.Error(w, "cannot load image file from url bytes and page: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer imageFile.Close()

		ep := vips.NewJpegExportParams()
		ep.Quality = 80

		log.Println("  exporting jpeg")
		out, _, err := imageFile.ExportJpeg(ep)
		if err != nil {
			http.Error(w, "cannot export file to jpeg: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/jpeg")
		_, err = w.Write(out)
		if err != nil {
			http.Error(w, "failed to convert pdf to jpeg: "+err.Error(), http.StatusInternalServerError)
			return
		}
	})

	log.Fatal(http.ListenAndServe(getEnv("LISTEN_PORT", ":8080"), nil))
}

func getEnv(key string, def string) string {
	r := os.Getenv(key)
	if r == "" {
		return def
	}
	return r
}

func probe(url string) (*Probe, error) {
	cmd := exec.Command(getEnv("FFPROBE_PATH", "ffprobe"),
		"-i", url,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		"-show_chapters",
	)
	res, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var probe *Probe
	if err := json.Unmarshal(res, &probe); err != nil {
		return nil, err
	}
	return probe, nil
}

func hash(s string) string {
	h := fnv.New128a()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum([]byte{}))
}
