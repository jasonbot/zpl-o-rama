package zplorama

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/boltdb/bolt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/mdns"
	"github.com/yosuke-furukawa/json5/encoding/json5"
)

func startJob(db *bolt.DB, jobID string) {
	db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(jobTimeTable))
		return bucket.Put([]byte(time.Now().Format(time.RFC3339)), []byte(jobID))
	})
}

func updateJob(db *bolt.DB, job *printJobStatus) {
	db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(printjobTable))

		job.Updated = time.Now().Format(time.RFC3339)

		jobBytes, err := json5.Marshal(job)

		if err == nil {
			err = bucket.Put([]byte(job.Jobid), jobBytes)

		}

		return err
	})
}

func sendZPL(dial, zpl string) error {
	conn, err := net.DialTimeout("tcp", dial, 5*time.Second)

	if err != nil {
		return err
	}
	defer conn.Close()

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
				Status:   processing,
				ZPL:      jobToDo.ZPL,
				ImageB64: emptyPNG,
				Created:  time.Now().Format(time.RFC3339),
				Updated:  time.Now().Format(time.RFC3339),
				Author:   jobToDo.Author,
				Message:  "",
			}

			updateJob(db, &status)

			var err error

			if status.ZPL != "" {
				err = sendZPL(printerAddress, status.ZPL)

				if err == nil {
					td, err := time.ParseDuration(Config.PrintTime)

					if err != nil {
						td = 5 * time.Second
						err = nil
					}
					time.Sleep(td)
				}
			}

			if err != nil {
				status.Status = failed
				status.Message = err.Error()
			} else {
				imageBytes, err := takePicture()
				status.ImageB64 = base64.StdEncoding.EncodeToString(imageBytes)

				if err == nil {
					status.Status = succeeded
				} else {
					status.Message = err.Error()
					status.Status = failed
				}
			}

			updateJob(db, &status)
		default: // Channel closed, out of work.
			return nil
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
				err = json5.Unmarshal(statusBytes, &jobStatus)

				if err != nil {
					writer.WriteHeader(http.StatusInternalServerError)
				}
			}

			writer.Header().Add("cache-control", "no-store")
			writer.Header().Add("content-type", "application/json")

			if err != nil {
				statusBytes, err = json5.Marshal(struct {
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

		decoder := json5.NewDecoder(httpRequest.Body)
		err = decoder.Decode(&printRequest)

		printRequest.Jobid = uuid.NewString()

		response := printJobStatus{
			Jobid:    printRequest.Jobid,
			Status:   pending,
			ZPL:      printRequest.ZPL,
			ImageB64: emptyPNG,
			Created:  time.Now().Format(time.RFC3339),
			Updated:  time.Now().Format(time.RFC3339),
			Author:   printRequest.Author,
			Message:  "I'm new here",
		}

		if printRequest.Jobid == "" || printRequest.ZPL == "" {
			err = errors.New("Empty params")
		} else {
			updateJob(db, &response)
			requestor <- &printRequest

			val, err = json5.Marshal(&response)
		}

		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			writer.Header().Add("content-type", "application/json")

			val, err = json5.Marshal(struct {
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
func RunPrintServer(serviceHost string, port int, printerDialAddress string) {
	database := createDB()
	defer database.Close()

	// Announce on network it exists
	host, _ := os.Hostname()
	info := []string{"ZPL Printer REST service"}
	service, _ := mdns.NewMDNSService(host, "_zplrest._tcp", "", "", port, nil, info)
	mserver, _ := mdns.NewServer(&mdns.Config{Zone: service})
	defer mserver.Shutdown()

	// Spin up the worker goroutine that takes pictures
	jobRunner := make(chan *printJobRequest, 20)
	go handleJobs(jobRunner, database, printerDialAddress)
	defer close(jobRunner)

	// Spin up HTTP API
	serverMux := mux.NewRouter()
	serverMux.HandleFunc("/job/{id}", printJobWatcher(serverMux, database))
	serverMux.HandleFunc("/print", printJobRequestor(serverMux, jobRunner, database))

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%v", serviceHost, port),
		Handler: serverMux,
	}
	server.ListenAndServe()
}
