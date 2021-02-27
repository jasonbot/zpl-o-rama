package zplorama

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
)

// Represents the state of the print job
type pictureStatus string

const (
	pending    pictureStatus = "PENDING"
	processing               = "PROCESSING"
	succeeded                = "SUCCEEDED"
	failed                   = "FAILED"
	missing                  = "MISSING"
)

const emptyPNG string = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII="

type printJobRequest struct {
	Jobid string `json:"jobid,omitempty"`
	ZPL   string `json:"ZPL"`
}

type printJobStatus struct {
	Jobid    string        `json:"jobid"`
	Status   pictureStatus `json:"status"`
	ZPL      string        `json:"ZPL"`
	ImageB64 string        `json:"image"`
}

func startJob(db *bolt.DB, jobID string) {
	db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(jobTimeTable))
		bucket.Put([]byte(time.Now().Format(time.RFC3339)), []byte(jobID))

		return nil
	})
}

func updateJob(db *bolt.DB, job *printJobStatus) {

	db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(printjobTable))

		jobBytes, err := json.Marshal(job)

		if err != nil {
			err = bucket.Put([]byte(job.Jobid), jobBytes)
		}

		return err
	})
}

func sendZPL(dial, zpl string) error {
	conn, err := net.Dial("tcp", dial)

	if err != nil {
		return err
	}

	_, err = conn.Write([]byte(zpl))

	return err
}

func takePicture() ([]byte, error) {
	out, err := exec.Command("raspistill", "-t", "0", "-f", "png", "-o", "-").Output()

	return out, err
}

func handleJobs(jobCache chan *printJobRequest, db *bolt.DB, printerAddress string) error {
	for {
		select {
		case jobToDo := <-jobCache:

			startJob(db, jobToDo.Jobid)

			status := printJobStatus{
				Jobid:    jobToDo.Jobid,
				ZPL:      jobToDo.ZPL,
				Status:   processing,
				ImageB64: emptyPNG,
			}

			updateJob(db, &status)

			err := sendZPL(printerAddress, status.ZPL)

			if err != nil {
				status.Status = failed
			} else {
				time.Sleep(5 * time.Second)
				imageBytes, err := takePicture()

				if err == nil {
					status.ImageB64 = base64.StdEncoding.EncodeToString(imageBytes)
					status.Status = succeeded
				} else {
					status.Status = failed
				}
			}

			updateJob(db, &status)
		}
	}
}

func printJobWatcher(router *mux.Router, db *bolt.DB) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		vars := mux.Vars(request)

		db.View(func(tx *bolt.Tx) error {
			var err error

			bucket := tx.Bucket([]byte(printjobTable))
			statusBytes := bucket.Get([]byte(vars["id"]))

			if len(statusBytes) == 0 {
				writer.WriteHeader(http.StatusNotFound)
				err = errors.New("Empty record")
			}

			if err == nil {
				var jobStatus printJobStatus
				err = json.Unmarshal(statusBytes, &jobStatus)

				if err != nil {
					writer.WriteHeader(http.StatusInternalServerError)
				}
			}

			writer.Header().Add("cache-control", "no-store")
			writer.Header().Add("content-type", "application/json")

			if err != nil {
				statusBytes, err = json.Marshal(struct {
					Errmsg string `json:"error"`
				}{Errmsg: err.Error()})
			}

			writer.Write(statusBytes)

			return err
		})
	}
}

func printJobRequestor(router *mux.Router, requestor chan *printJobRequest, db *bolt.DB) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, httpRequest *http.Request) {
		var printRequest printJobRequest
		var val []byte
		var err error

		decoder := json.NewDecoder(httpRequest.Body)
		fmt.Printf("%v\n", httpRequest.Body)
		err = decoder.Decode(&printRequest)

		response := printJobStatus{
			Jobid:    printRequest.Jobid,
			Status:   pending,
			ZPL:      printRequest.ZPL,
			ImageB64: "",
		}

		if printRequest.Jobid == "" || printRequest.ZPL == "" {
			err = errors.New("Empty params")
		} else {
			requestor <- &printRequest

			val, err = json.Marshal(&response)
		}

		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			writer.Header().Add("content-type", "application/json")

			val, err = json.Marshal(struct {
				Errmsg string `json:"error"`
			}{Errmsg: err.Error()})
		} else {
			writer.WriteHeader(http.StatusOK)
			writer.Header().Add("content-type", "application/json")
		}

		writer.Write(val)
	}
}

// RunPrintServer executes the HTTP server
func RunPrintServer(port int, printerDialAddress string) {
	database := createDB()
	defer database.Close()

	fmt.Println("ALIVE")

	jobRunner := make(chan *printJobRequest, 20)
	go handleJobs(jobRunner, database, printerDialAddress)

	serverMux := mux.NewRouter()
	serverMux.HandleFunc("/job/{id}", printJobWatcher(serverMux, database))
	serverMux.HandleFunc("/print", printJobRequestor(serverMux, jobRunner, database))

	server := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%v", port),
		Handler: serverMux,
	}
	server.ListenAndServe()
}
