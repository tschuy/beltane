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
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go"
)

const (
	s3host     = "http://127.0.0.1:9000"
	bucketName = "beltane"
)

var minioClient *minio.Client

type UploadResponse struct {
	Sha string `json:"sha"`
	Url string `json:"access_url"`
}

type Metadata struct {
	Sha       string    `json:"sha"`
	MachineId string    `json:"machine_id"`
	Time      time.Time `json:"created_at"`
}

type OutputMetadata struct {
	Sha       string    `json:"sha"`
	MachineId string    `json:"machine_id"`
	Time      time.Time `json:"created_at"`
	Url       string    `json:"download_url"`
}

type FailureResponse struct {
	Error string `json:"error"`
}

func StreamToByte(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}

func httperror(w http.ResponseWriter, errstr string, errcode int) {
	res, _ := json.Marshal(&FailureResponse{
		Error: errstr,
	})
	http.Error(w, string(res), errcode)
	return
}

func upload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	r.ParseMultipartForm(32 << 20)
	file, _, err := r.FormFile("targz")

	// verify file was passed in
	if err != nil {
		fmt.Println(err)
		httperror(w, "missing required targz field", 422)
		return
	}
	defer file.Close()

	// TODO don't zipbomb
	// verify file is a gzip file
	gzf, err := gzip.NewReader(file)
	if err != nil {
		httperror(w, "invalid gz file", 400)
		return
	}

	// verify gzip file contains tar file
	tarReader := tar.NewReader(gzf)
	_, err = tarReader.Next()
	if err != nil {
		httperror(w, "invalid tar file", 400)
		return
	}

	// save and return
	h := sha1.New()
	file.Seek(0, 0)
	_, _ = io.Copy(h, file)
	sha := hex.EncodeToString(h.Sum(nil))
	log.Printf("uploaded: %s", sha)

	res, err := json.Marshal(&UploadResponse{
		Sha: sha,
		Url: "http://localhost:8080/?token=" + sha,
	})

	fmt.Fprintf(w, string(res))
	n := time.Now()

	//  => YYYY/MM/YYYYMMDDHHminmil
	prefix := n.Format("2006") + "/" + n.Format("01") + "/" + n.Format("02") + "/" + n.Format("20060102150405")
	machine_id := r.FormValue("machine_id")
	if machine_id == "" {
		machine_id = "none"
	}
	machine_id = strings.TrimSpace(machine_id)

	file.Seek(0, 0)
	// upload to s3

	name := prefix + "-" + machine_id + "-" + sha + ".tar.gz"
	ln, _ := minioClient.PutObject(bucketName, name, file, "application/gzip")
	log.Printf("Wrote %s to %s (%d bytes)", name, bucketName, ln)
}

func index(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("index.gtpl")
	t.Execute(w, nil)
}

func dumps(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	params := r.URL.Query()
	var err error

	n := time.Now()

	if len(params["date"]) != 0 {
		n, err = time.Parse("2006/01/02", params["date"][0])
		if err != nil {
			httperror(w, "could not parse time parameter", 400)
			return
		}
	}

	log.Print(n)

	objects := getByDate(n)

	var output []OutputMetadata

	for _, obj := range objects {
		key := obj.Key
		// [2016/09/08/20160908132709 2cf61eb789e243b59adf1a850fc51a44 5dfca0aa0cd01bcdfca1a8cf7e6f955aaf23af9c.tar.gz]
		parts := strings.Split(key, "-")

		duration := time.Duration(1) * time.Hour
		getUrl, _ := minioClient.PresignedGetObject(bucketName, obj.Key, duration, nil)
		t, _ := time.Parse("20060102150405", parts[0][11:])
		m := OutputMetadata{
			Time:      t,
			MachineId: parts[1],
			Sha:       strings.Split(parts[2], ".")[0],
			Url:       getUrl.String(),
		}

		output = append([]OutputMetadata{m}, output...)
	}
	j, _ := json.Marshal(output)
	fmt.Fprintf(w, string(j))
}

func chanToSlice(channel <-chan minio.ObjectInfo) []minio.ObjectInfo {
	var slice []minio.ObjectInfo
	for element := range channel {
		slice = append(slice, element)
	}
	return slice
}

func prefix(year string, month string, day string) string {
	return year + "/" + month + "/" + day + "/"
}

func pop(p string) string {
	sl := strings.SplitAfter(p, "/")
	sl = sl[:len(sl)-2]
	return strings.Join(sl, "")
}

func getByDate(n time.Time) []minio.ObjectInfo {
	p := prefix(n.Format("2006"), n.Format("01"), n.Format("02"))
	objects := chanToSlice(minioClient.ListObjectsV2(bucketName, p, false, nil))
	if len(objects) > 0 {
		return objects
	}

	// TODO recursive popping -- be able to go from 2016/01/01 to 2015/11/30
	p = pop(p)
	objects = chanToSlice(minioClient.ListObjectsV2(bucketName, p, false, nil))
	log.Print(len(objects))
	log.Print(objects[len(objects)-1].Key)
	objects = chanToSlice(minioClient.ListObjectsV2(bucketName, objects[len(objects)-1].Key, false, nil))
	return objects
}

func main() {
	endpoint := "127.0.0.1:9000"
	accessKeyID := "CXULQKAQHP7IV3U9UXAC"
	secretAccessKey := "w6UB2TZSvDqNLC/mzazp8X5AnWD8BTw3f8JFoxXk"
	useSSL := false

	var err error

	// Initialize minio client object.
	minioClient, err = minio.New(endpoint, accessKeyID, secretAccessKey, useSSL)
	if err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/v1/upload", upload).Methods("POST")
	r.HandleFunc("/v1/", index).Methods("GET")
	r.HandleFunc("/v1/dumps", dumps).Methods("GET")

	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	http.Handle("/", r)

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
