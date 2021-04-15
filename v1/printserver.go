package zplorama

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/boltdb/bolt"
	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/hashicorp/mdns"
	"github.com/labstack/echo"
)

const resetCommand string = `^XA
^FWN
^LL1218
^PW812
^PON
^LH0,0
^LT0
^CI28
^XZ
`

func startJob(db *bolt.DB, jobID string) {
	jobRecord := jobTimestamp{
		Timestamp: time.Now().Format(time.RFC3339),
		Jobid:     jobID,
	}
	PutRecord(db, &jobRecord)
}

func updateJob(db *bolt.DB, job *printJobStatus) {
	if job.Log == nil {
		job.Log = make([]string, 0)
	}

	if job.Message != "" && (len(job.Log) == 0 || job.Log[len(job.Log)-1] != job.Message) {
		job.Log = append(job.Log, job.Message)
	}

	job.Updated = time.Now().Format(time.RFC3339)
	PutRecord(db, job)
}

func sendZPL(dial, zpl string) error {
	conn, err := net.DialTimeout("tcp", dial, 1*time.Second)

	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(zpl))
	return err
}

func takePicture() ([]byte, error) {
	out, err := exec.Command("raspistill", "-t", "3000", "-e", "png", "-o", "-").Output()

	return out, err
}

func handleJobs(jobCache chan *printJobRequest, db *bolt.DB, printerAddress string) error {
	for jobToDo := range jobCache {
		startJob(db, jobToDo.jobid)

		status := printJobStatus{
			Jobid:         jobToDo.jobid,
			Status:        processing,
			ZPL:           jobToDo.ZPL,
			ImageB64:      emptyPNG,
			ImageB64Small: emptyPNG,
			Created:       time.Now().Format(time.RFC3339),
			Updated:       time.Now().Format(time.RFC3339),
			Author:        jobToDo.Author,
			Message:       "Job started, enqueueing",
			Log:           make([]string, 0),
			Done:          false,
		}

		updateJob(db, &status)

		var err error

		if status.ZPL != "" {
			err = sendZPL(printerAddress, resetCommand)
			if err != nil {
				return err
			}

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
			status.ImageB64 = sadFace
			status.ImageB64Small = sadFace
		} else {

			imageBytes, err := takePicture()
			var b64string, b64smallstring string
			if err == nil {
				b64string = base64.StdEncoding.EncodeToString(imageBytes)
				b64smallstring, err = shrinkImage(b64string)
			}

			if err == nil {
				status.Status = succeeded
				status.Message = "Successfully processed request"
				status.ImageB64 = b64string
				status.ImageB64Small = b64smallstring
			} else {
				status.Message = err.Error()
				status.Status = failed
				status.ImageB64 = sadFace
				status.ImageB64Small = sadFace
			}
		}
		status.Done = true

		updateJob(db, &status)
	}

	return nil
}

func getJob(database *bolt.DB) func(echo.Context) error {
	return func(c echo.Context) error {
		jobStatus := new(printJobStatus)
		jobStatus.Jobid = c.Param("id")

		err := GetRecord(database, jobStatus)

		if err != nil {
			return c.JSON(http.StatusNotFound, errJSON{Errmsg: "Job not found"})
		}

		return c.JSON(http.StatusOK, jobStatus)
	}
}

func printJob(database *bolt.DB, requestor chan *printJobRequest) func(echo.Context) error {
	return func(c echo.Context) error {
		var err error

		printRequest := new(printJobRequest)
		c.Bind(&printRequest)

		jobid := uuid.NewString()
		printRequest.jobid = jobid

		response := printJobStatus{
			Jobid:    jobid,
			Status:   pending,
			ZPL:      printRequest.ZPL,
			ImageB64: emptyPNG,
			Created:  time.Now().Format(time.RFC3339),
			Updated:  time.Now().Format(time.RFC3339),
			Author:   printRequest.Author,
			Message:  "Job created",
			Done:     false,
		}

		updateJob(database, &response)

		select {
		case requestor <- printRequest:
			break
		case <-time.After(5 * time.Second):
			err = errors.New("Failed to queue job in time")
			response.Status = failed
			response.ImageB64 = sadFace
			response.Message = "Timed out waiting to job to worker; is the system overloaded?"
			response.Done = true
			updateJob(database, &response)
		}

		if err != nil {
			return c.JSON(http.StatusBadRequest, struct {
				Errmsg string `json:"error"`
			}{Errmsg: err.Error()})
		}

		return c.Redirect(http.StatusFound, fmt.Sprintf("/job/%s", jobid))
	}
}

func shrinkImage(imageB64 string) (string, error) {
	data, _ := base64.StdEncoding.DecodeString(imageB64)

	image, err := imaging.Decode(bytes.NewBuffer(data))

	if err != nil {
		return "", err
	}

	resizedImage := imaging.Resize(image, 800, 0, imaging.Box)

	var smallImage bytes.Buffer
	imaging.Encode(&smallImage, resizedImage, imaging.PNG)

	return base64.StdEncoding.EncodeToString(smallImage.Bytes()), nil
}

// RunPrintServer executes the HTTP server
func RunPrintServer(serviceHost string, port int, printerDialAddress string) {
	database := createDB(Config.BackendDatabase)
	defer database.Close()

	requestChain := make(chan *printJobRequest, 20)
	go handleJobs(requestChain, database, printerDialAddress)

	// Announce on network it exists
	host, _ := os.Hostname()
	info := []string{"ZPL Printer REST service"}
	service, _ := mdns.NewMDNSService(host, "_zplrest._tcp", "", "", port, nil, info)
	mserver, _ := mdns.NewServer(&mdns.Config{Zone: service})
	defer mserver.Shutdown()

	e := echo.New()
	e.HideBanner = true
	e.Debug = true

	e.GET("/job/:id", getJob(database))
	e.POST("/print", printJob(database, requestChain))
	e.Logger.Fatal(e.Start(fmt.Sprintf(":%v", port)))
}
