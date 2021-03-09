package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const filesDir = "files"
const chunksDir = "output"
const port = 8888

type RequestStruct struct {
	Url string
}

func downloadFile(dir string, url string) (string, error) {
	// Create the file
	fileId := randToken(12)
	filePath := filepath.Join(dir, fileId+".mp4")
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return "", err
	}
	out, err := os.Create(filePath)
	if err != nil {
		return "", err
	}

	// Get the data
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != 200 {
		return "", err
	}

	// Write the body to the file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	defer out.Close()

	err = splitToChunks(fileId, filePath)
	if err != nil {
		return "", err
	}
	return fileId, err
}

func splitToChunks(fileName string, filePath string) error {
	_ = os.MkdirAll(chunksDir, 0700)
	chunksPath := filepath.Join(chunksDir, fileName)
	s := fmt.Sprintf("ffmpeg -i %s -c:v libx264 "+
		"-crf 21 -preset veryfast -c:a aac -b:a 128k -ac 2 -f hls "+
		"-hls_time 8 -hls_playlist_type event %s.m3u8", filePath, chunksPath)
	cmd := strings.Split(s, " ")
	_, err := exec.Command(cmd[0], cmd[1:]...).Output()
	return err
}

func randToken(len int) string {
	b := make([]byte, len)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// enable CORS
func addHeaders(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		handler.ServeHTTP(w, request)
	}
}

func uploadFileHandler(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var t RequestStruct
	err := decoder.Decode(&t)
	if err != nil {
		panic(err)
	}
	fileId, err := downloadFile(filesDir, t.Url)
	if err != nil {
		panic(err)
	}
	_, _ = fmt.Fprintf(w, fileId)
}

func main() {
	// serve mp4 files
	http.Handle("/", addHeaders(http.FileServer(http.Dir(chunksDir))))
	http.HandleFunc("/upload", uploadFileHandler)
	fmt.Printf("Server started on %v\n", port)

	// serve and log errors
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port), nil))
}
