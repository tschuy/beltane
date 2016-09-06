package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type UploadResponse struct {
	Sha string `json:"sha"`
	Url string `json:"access_url"`
}

type FailureResponse struct {
	Error string `json:"error"`
}

func StreamToByte(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}

func upload(w http.ResponseWriter, r *http.Request) {
	// method check
	if r.Method != "POST" {
		res, _ := json.Marshal(&FailureResponse{
			Error: "Method Not Allowed (use POST)",
		})
		http.Error(w, string(res), 405)
		return
	}
	r.ParseMultipartForm(32 << 20)
	file, _, err := r.FormFile("targz")

	// verify file was passed in
	if err != nil {
		fmt.Println(err)

		res, _ := json.Marshal(&FailureResponse{
			Error: "missing required targz field",
		})
		http.Error(w, string(res), 422)
		return
	}
	defer file.Close()

	// verify file is a gzip file
	gzf, err := gzip.NewReader(file)
	if err != nil {
		res, _ := json.Marshal(&FailureResponse{
			Error: "invalid gz file",
		})
		http.Error(w, string(res), 400)
		return
	}

	// verify gzip file contains tar file
	tarReader := tar.NewReader(gzf)
	_, err = tarReader.Next()
	if err != nil {
		res, _ := json.Marshal(&FailureResponse{
			Error: "invalid tar file",
		})
		http.Error(w, string(res), 400)
		return
	}

	// save and return
	h := sha1.New()
	io.Copy(h, file)
	sha := hex.EncodeToString(h.Sum(nil))
	log.Printf("%s", sha)

	res, err := json.Marshal(&UploadResponse{
		Sha: sha,
		Url: "http://localhost:8080/dump/" + sha,
	})

	fmt.Fprintf(w, string(res))
	f, err := os.OpenFile("./test/"+sha+".tar.gz", os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	io.Copy(f, file)
}

func main() {
	http.HandleFunc("/upload", upload)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
