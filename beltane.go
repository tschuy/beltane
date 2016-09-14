package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
)

const (
	s3host          = "127.0.0.1:9000"
	accessKeyID     = "CXULQKAQHP7IV3U9UXAC"
	secretAccessKey = "w6UB2TZSvDqNLC/mzazp8X5AnWD8BTw3f8JFoxXk"
	useSSL          = false
	port            = ":8080"
)

var bucketName = "beltane"
var s3Client *s3.S3

type UploadResponse struct {
	Sha    string `json:"sha"`
	Access string `json:"access_key"`
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

	// save and return
	h := sha1.New()
	file.Seek(0, 0)
	_, _ = io.Copy(h, file)
	sha := hex.EncodeToString(h.Sum(nil))
	log.Printf("uploaded: %s", sha)

	timestamp := math.MaxInt64 - time.Now().Unix()
	timestr := strconv.FormatInt(timestamp, 16)

	machine_id := r.FormValue("machine_id")
	if machine_id == "" {
		machine_id = "none"
	}
	machine_id = strings.TrimSpace(machine_id)

	res, err := json.Marshal(&UploadResponse{
		Sha:    sha,
		Access: timestr + "-" + machine_id + sha,
	})

	fmt.Fprintf(w, string(res))

	file.Seek(0, 0)
	// upload to s3

	name := timestr + "-" + machine_id + "-" + sha + ".tar.gz"
	_, err = s3Client.PutObject(&s3.PutObjectInput{
		Body:   file,
		Bucket: &bucketName,
		Key:    &name,
	})
	if err != nil {
		log.Printf("Failed to upload data to %s/%s, %s\n", bucketName, name, file)
		return
	}

	log.Printf("Wrote %s to %s", name, bucketName)
}

func getDumpsByDate(n time.Time, maxKeys int) ([]OutputMetadata, error) {

	n = n.AddDate(0, 0, 1) // add one day to get all things made *on* that day, not before
	marker := strconv.FormatInt(math.MaxInt64-n.Unix(), 16)
	log.Printf("listing from %s (%s)", marker, n)

	maxKeys64 := int64(maxKeys)
	objects, err := s3Client.ListObjects(&s3.ListObjectsInput{
		Bucket:  &bucketName,
		MaxKeys: &maxKeys64,
		// according to S3 documentation, marker needs to be an existing key
		// manual testing shows any string works just fine
		Marker: &marker,
	})

	if err != nil {
		return []OutputMetadata{}, err
	}

	return processObjects(objects)
}

func getDumpsByToken(token string) ([]OutputMetadata, error) {
	var maxKeys = int64(1)

	objects, err := s3Client.ListObjects(&s3.ListObjectsInput{
		Bucket:  &bucketName,
		MaxKeys: &maxKeys,
		Marker:  &token,
	})

	if err != nil {
		return []OutputMetadata{}, err
	}

	return processObjects(objects)
}

func processObjects(objects *s3.ListObjectsOutput) ([]OutputMetadata, error) {
	var output []OutputMetadata

	for _, obj := range objects.Contents {
		key := obj.Key
		// [7fffffffa828c65f 2cf61eb789e243b59adf1a850fc51a44 5dfca0aa0cd01bcdfca1a8cf7e6f955aaf23af9c.tar.gz]
		parts := strings.Split(*key, "-")

		getUrl := "#"
		timestamp, _ := strconv.ParseInt(parts[0], 16, 64)

		t := time.Unix(math.MaxInt64-timestamp, 0).UTC()
		m := OutputMetadata{
			Time:      t,
			MachineId: parts[1],
			Sha:       strings.Split(parts[2], ".")[0],
			// TODO downloading
			Url: getUrl,
		}

		output = append(output, m)
	}
	return output, nil
}

func dumps(w http.ResponseWriter, r *http.Request) {
	// possible get params: "date", "num"
	w.Header().Set("Content-Type", "application/json")

	params := r.URL.Query()
	var err error
	var output []OutputMetadata

	n := time.Now() // default time to show before
	num := 20       // default number of items to show

	if len(params["token"]) != 0 {
		output, err = getDumpsByToken(params["token"][0])
	} else {
		if len(params["date"]) != 0 {
			n, err = time.Parse("2006/01/02", params["date"][0])
			if err != nil {
				log.Print(err)
				httperror(w, "could not parse time parameter", 400)
				return
			}
		}

		if len(params["num"]) != 0 {
			num, err = strconv.Atoi(params["num"][0])
			if err != nil {
				log.Print(err)
				httperror(w, "could not parse num parameter", 400)
				return
			}
		}
		output, err = getDumpsByDate(n, num)
	}

	if err != nil {
		log.Print(err)
		httperror(w, "error processing request", 500)
	}

	j, _ := json.Marshal(output)
	fmt.Fprintf(w, string(j))
}

func main() {
	var err error
	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
		Endpoint:         aws.String(s3host),
		Region:           aws.String("us-east-1"),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(true),
	}
	newSession := session.New(s3Config)
	s3Client = s3.New(newSession)

	r := mux.NewRouter()
	r.HandleFunc("/v1/upload", upload).Methods("POST")
	r.HandleFunc("/v1/dumps", dumps).Methods("GET")
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	http.Handle("/", r)

	err = http.ListenAndServe(port, nil)
	if err != nil {
		panic(err)
	}
}
