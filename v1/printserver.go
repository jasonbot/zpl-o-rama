package zplorama

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/boltdb/bolt"
	"github.com/google/uuid"
	"github.com/hashicorp/mdns"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/yosuke-furukawa/json5/encoding/json5"
)

type errJSON struct {
	Errmsg string `json:"error"`
}

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
	conn, err := net.DialTimeout("tcp", dial, 1*time.Second)

	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(zpl))
	return err
}

func takePicture() ([]byte, error) {
	out, err := exec.Command("raspistill", "-t", "1000", "-e", "png", "-o", "-").Output()

	return out, err
}

func handleJobs(jobCache chan *printJobRequest, db *bolt.DB, printerAddress string) error {
	for jobToDo := range jobCache {
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
	}

	return nil
}

func getJob(database *bolt.DB) func(echo.Context) error {
	return func(c echo.Context) error {
		database.View(func(tx *bolt.Tx) error {
			var err error

			bucket := tx.Bucket([]byte(printjobTable))
			statusBytes := bucket.Get([]byte(c.Param("id")))

			if statusBytes == nil || len(statusBytes) == 0 {
				return c.JSON(http.StatusNotFound, errJSON{Errmsg: "Job not found"})
			}

			var jobStatus printJobStatus
			err = json5.Unmarshal(statusBytes, &jobStatus)

			if err != nil {
				return c.JSON(http.StatusBadRequest, errJSON{Errmsg: err.Error()})
			}

			c.Response().Header().Add("Cache-Control", "no-store")

			if err != nil {
				c.JSON(http.StatusBadRequest, errJSON{Errmsg: err.Error()})
			}

			return c.JSON(http.StatusOK, jobStatus)
		})

		return nil
	}
}

func printJob(database *bolt.DB, requestor chan *printJobRequest) func(echo.Context) error {
	return func(c echo.Context) error {
		var err error

		printRequest := new(printJobRequest)
		c.Bind(&printRequest)

		printRequest.Jobid = uuid.NewString()

		response := printJobStatus{
			Jobid:    printRequest.Jobid,
			Status:   pending,
			ZPL:      printRequest.ZPL,
			ImageB64: emptyPNG,
			Created:  time.Now().Format(time.RFC3339),
			Updated:  time.Now().Format(time.RFC3339),
			Author:   printRequest.Author,
			Message:  "Job created",
		}

		updateJob(database, &response)

		select {
		case requestor <- printRequest:
			break
		case <-time.After(5 * time.Second):
			response.Status = failed
			response.Message = "Timed out waiting to job to worker; is the system overloaded?"
		}

		if err != nil {
			return c.JSON(http.StatusBadRequest, struct {
				Errmsg string `json:"error"`
			}{Errmsg: err.Error()})
		}

		return c.Redirect(http.StatusFound, fmt.Sprintf("/job/%s", printRequest.Jobid))
	}
}

// RunPrintServer executes the HTTP server
func RunPrintServer(serviceHost string, port int, printerDialAddress string) {
	database := createDB()
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

	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: 5,
	}))

	e.GET("/job/:id", getJob(database))
	e.POST("/print", printJob(database, requestChain))
	e.Logger.Fatal(e.Start(fmt.Sprintf(":%v", port)))
}
