package main

import "C"
import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"image/jpeg"
	"io"
	"log"
	"net/http"
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

func main() {
	fmt.Println("transcode-rest")
	fmt.Println("git hash: " + GitHash)
	fmt.Println("git branch: " + GitBranch)
	fmt.Println("build time: " + BuildTime)
	fmt.Println()

	tmp := getEnv("STORAGE_DIRECTORY", os.TempDir())

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

		h := hash(url)
		name := h + ".mp4"
		file := tmp + "/" + name

		if _, err := os.Stat(file); err == nil {
			log.Println("already exist")
			f, err := os.Open(file)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "cannot open file: %s", err.Error())
				return
			}
			defer f.Close()
			http.ServeContent(w, r, name, time.Now(), f)
			return
		}

		log.Println("transcode", url, "with bitrate", bitrate, "to", file)
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

		log.Println("serve file")
		f, err := os.Open(file)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "cannot open file: %s", err.Error())
			return
		}
		defer f.Close()
		http.ServeContent(w, r, name, time.Now(), f)
	})

	http.HandleFunc("/video/thumbnail", func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.Query().Get("url")
		if url == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "url param needed")
			return
		}

		h := hash(url)
		name := h + ".jpg"
		file := tmp + "/" + name

		if _, err := os.Stat(file); err == nil {
			log.Println("already exist")
			f, err := os.Open(file)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "cannot open file: %s", err.Error())
				return
			}
			defer f.Close()
			http.ServeContent(w, r, name, time.Now(), f)
			return
		}

		log.Println("transcode", url, "to", file)
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
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "cannot start ffmpeg: %s", err.Error())
			return
		}

		if err := ffmpeg.Wait(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "cannot open file: %s", err.Error())
			return
		}

		log.Println("serve file")
		f, err := os.Open(file)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "cannot open file: %s", err.Error())
			return
		}
		defer f.Close()
		http.ServeContent(w, r, name, time.Now(), f)
	})

	http.HandleFunc("/blur", func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.Query().Get("url")
		if url == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "url param needed")
			return
		}

		res, err := http.Get(url)
		if err != nil {
			log.Fatal(err)
		}
		defer res.Body.Close()

		loadedImage, err := jpeg.Decode(res.Body)
		if err != nil {
			fmt.Fprintf(w, "cannot decode image : %s", err)
			return
		}

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
			fmt.Fprintf(w, "url param needed")
			return
		}

		res, err := http.Get(url)
		if err != nil {
			log.Fatal(err)
		}
		defer res.Body.Close()

		inputImage, err := vips.NewImageFromReader(res.Body)
		if err != nil {
			log.Fatal(err)
		}

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
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "url param needed")
			return
		}
		page := r.URL.Query().Get("page")
		if page == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "page param needed")
			return
		}
		intPage, _ := strconv.Atoi(page)

		res, err := http.Get(url)
		if err != nil {
			fmt.Fprintf(w, "cannot get url body: %s", err.Error())
			return
		}
		defer res.Body.Close()

		bytesRes, err := io.ReadAll(res.Body)
		if err != nil {
			fmt.Fprintf(w, "cannot extract bytes from url body: %s", err.Error())
			return
		}

		ip := vips.NewImportParams()
		ip.Page.Set(intPage)
		ip.Density.Set(120)

		imageFile, err := vips.LoadImageFromBuffer(bytesRes, ip)
		if err != nil {
			fmt.Fprintf(w, "cannot load image file from url bytes and page: %s", err.Error())
			return
		}

		ep := vips.NewJpegExportParams()
		ep.Quality = 100

		out, _, err := imageFile.ExportJpeg(ep)
		if err != nil {
			fmt.Fprintf(w, "cannot export file to jpeg: %s", err.Error())
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(out)
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
