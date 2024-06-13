package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"
)

var (
	// GitHash is set at compilation if available
	GitHash = "unknown"

	// Version is set at compilation if tag
	GitBranch = "unknown"

	// BuildTime is set at compilation
	BuildTime = "unknown"
)

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
