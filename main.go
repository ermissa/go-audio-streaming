package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
)

func main() {
	listenAddr := os.Getenv("LISTEN_ADDR")
	addr := listenAddr + `:` + os.Getenv("PORT")

	http.HandleFunc("/watch", stream)
	log.Printf("starting server at %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func stream(w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query().Get("v")
	if v == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "use format /watch?v=...")
		return
	}
	err := downloadVideoAndExtractAudio(v, w)
	if err != nil {
		log.Printf("stream error: %v", err)
		fmt.Fprintf(w, "use format /watch?v=...")
	}
}

// Creates io.Pipe
// Downloads video via youtube-dl,
// writes output of youtube-dl to pipe's writer (w)
// ffmpeg stdin reads from same pipe's reader (r)
// ffmpeg writes to http.ResponseWriter (named out in downloadVideoAndExtractAudio)
func downloadVideoAndExtractAudio(id string, out io.Writer) error {
	url := fmt.Sprintf("https://youtube.com/watch?v=" + id)

	// pipes
	// ytdl > (w)-----(r) > ffmpeg > (out)
	r, w := io.Pipe()
	defer r.Close()

	ytdl := exec.Command("youtube-dl", url, "-o-")
	ytdl.Stdout = w         // PIPE INPUT
	ytdl.Stderr = os.Stderr // show progress

	// ffmpeg reads input from /dev/stdin namely standart input
	ffmpeg := exec.Command("ffmpeg", "-i", "/dev/stdin", "-f", "mp3", "-ab", "96000", "-vn", "-")

	ffmpeg.Stdin = r    // PIPE OUTPUT
	ffmpeg.Stdout = out // PIPE2 INPUT
	ffmpeg.Stderr = os.Stderr

	// ytdl runs on parallel goroutine and ffmpeg reads the output of ytdl on main goroutine.
	go func() {
		if err := ytdl.Run(); err != nil {
			log.Printf("WARN: ytdl error: %v", err)
		}
	}()
	err := ffmpeg.Run()
	log.Printf("stream finished")
	return err
}
