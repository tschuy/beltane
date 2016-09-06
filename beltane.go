package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type UploadResponse struct {
	Sha string `json:"sha"`
	Url string `json:"access_url"`
}

type Metadata struct {
	Sha       string    `json:"sha"`
	MachineId string    `json:"machine_id"`
	Time      time.Time `json:"created_at"`
}

type Directory struct {
	Files []string
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

	// TODO don't zipbomb
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
	// TODO only return the above if file successfully saves
	f, err := os.OpenFile("./test/"+sha+".tar.gz", os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	metadata, _ := json.Marshal(&Metadata{
		Sha:       sha,
		MachineId: r.FormValue("machine_id"),
		Time:      time.Now(),
	})

	ioutil.WriteFile("./test/"+sha+".json", metadata, 0666)
	io.Copy(f, file)
}

func dump(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	dat, _ := ioutil.ReadFile("./test/" + params["name"] + ".json")
	log.Print(dat)
	fmt.Fprintf(w, string(dat))
}

func listing(w http.ResponseWriter, r *http.Request) {
	dir, _ := ioutil.ReadDir("./test")
	var names []string
	for _, v := range dir {
		if strings.HasSuffix(v.Name(), ".json") {
			names = append(names, strings.TrimSuffix(v.Name(), ".json"))
		}
	}

	t, _ := template.ParseFiles("index.gtpl")
	log.Print(names)
	t.Execute(w, names)
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/upload", upload).Methods("POST")
	r.HandleFunc("/", listing).Methods("GET")
	r.HandleFunc("/dump/{name:[a-z0-9]{40}}", dump).Methods("GET")

	http.Handle("/", r)
	http.Handle("/raw/", http.StripPrefix("/raw/", http.FileServer(http.Dir("/home/tschuy/projects/beltane/test"))))

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
