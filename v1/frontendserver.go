package zplorama

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"text/template"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/yosuke-furukawa/json5/encoding/json5"
)

//go:embed static/*
var staticContent embed.FS

type feTemplate struct {
	templates *template.Template
}

func (t *feTemplate) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

//go:embed template/*
var templatesFS embed.FS
var templates *template.Template

func init() {
	fm := make(template.FuncMap)
	fm["GoogleSite"] = func() string { return Config.GoogleSite }

	var err error
	templates, err = template.New("webapp").Funcs(fm).ParseFS(templatesFS, "template/*.tpl")

	if err != nil {
		panic(err)
	}
}

func renderTemplateString(name string, data interface{}) string {
	var buffer bytes.Buffer
	templates.ExecuteTemplate(&buffer, name, data)

	return string(buffer.Bytes())
}

type loginRequestStruct struct {
	IDToken string `json:"id_token"`
}

func doLogin(c echo.Context) error {
	loginRequest := new(loginRequestStruct)
	c.Bind(loginRequest)

	err := verifyIDToken(loginRequest.IDToken, c)

	if err == nil {
		userName := c.Get("user_name").(string)

		html := renderTemplateString("loginbar", struct{ User string }{User: userName})
		return c.JSON(http.StatusOK, &hotwireResponse{
			Message: "Logged in",
			DivID:   "loginbar",
			HTML:    html,
		})
	}

	return c.JSON(http.StatusUnauthorized, &errJSON{Errmsg: fmt.Sprintf("Cannot log in: %v", err)})
}

func doLogout(c echo.Context) error {
	deleteIDToken(c)

	return c.JSON(http.StatusOK, &hotwireResponse{
		Message: "Logged out",
		DivID:   "loginbar",
		HTML:    "Logged out.",
	})
}

func homePage(c echo.Context) error {
	var body string
	var userName string

	if c.Get("logged_in").(bool) == true {
		userName = c.Get("user_name").(string)
		body = renderTemplateString("input-zpl-form", nil)
	} else {
		body = renderTemplateString("please-log-in", nil)
	}

	return c.Render(http.StatusOK, "main", struct {
		Title string
		User  string
		Body  string
	}{
		Title: "ZPL-O-Rama: Home",
		User:  userName,
		Body:  body,
	})
}

func printMedia(c echo.Context) error {
	if !(c.Get("logged_in").(bool)) {
		return c.JSON(http.StatusUnauthorized, errJSON{Errmsg: "You're not logged in."})
	}

	printRequest := new(printJobRequest)
	c.Bind(printRequest)

	printRequest.Author = c.Get("login").(*mail.Address).Address

	printHost := fmt.Sprintf("http://%v:%v/print", Config.PrintserviceHost, Config.PrintservicePort)

	body, _ := json5.Marshal(&printRequest)
	buf := bytes.NewBuffer(body)

	response, err := http.Post(printHost, "application/json", buf)

	if err != nil {
		return c.JSON(http.StatusBadRequest, errJSON{Errmsg: err.Error()})
	}

	if response.StatusCode != http.StatusOK {
		var errMsg errJSON

		dec := json5.NewDecoder(response.Body)
		dec.Decode(&errMsg)

		return c.JSON(http.StatusBadRequest, errMsg)
	} else {
		var status printJobStatus

		dec := json5.NewDecoder(response.Body)
		dec.Decode(&status)
		if status.Jobid != "" {
			return c.Redirect(http.StatusFound, fmt.Sprintf("/job/%v", status.Jobid))
		}
	}

	return c.JSON(http.StatusInternalServerError, errJSON{Errmsg: "No idea what happened here."})
}

func fetchJobCall(jobID string) (printJobStatus, error) {
	jobURL := fmt.Sprintf("http://%v:%v/job/%v", Config.PrintserviceHost, Config.PrintservicePort, jobID)

	response, err := http.Get(jobURL)

	if err != nil {
		return printJobStatus{}, err
	}

	if response.StatusCode != http.StatusOK {
		var errMsg errJSON

		dec := json5.NewDecoder(response.Body)
		dec.Decode(&errMsg)

		return printJobStatus{}, errors.New(errMsg.Errmsg)
	} else {
		var status printJobStatus

		dec := json5.NewDecoder(response.Body)
		dec.Decode(&status)
		if status.Jobid != "" {
			return status, nil
		}
	}

	return printJobStatus{}, errors.New("Unknown failure")
}

func displayJob(c echo.Context) error {
	job, err := fetchJobCall(c.Param("id"))

	var jobString string

	if err == nil {
		var jobBytes []byte

		jobBytes, err = json5.MarshalIndent(job, "", " ")
		jobString = string(jobBytes)
	}

	if err == nil {
		var userName string

		if c.Get("logged_in").(bool) == true {
			userName = c.Get("user_name").(string)
		}

		body := renderTemplateString(
			"job-status",
			struct {
				Code string
			}{
				Code: jobString,
			},
		)

		return c.Render(http.StatusOK, "main", struct {
			Title string
			User  string
			Body  string
		}{
			Title: "ZPL-O-Rama: Print Job",
			User:  userName,
			Body:  body,
		})
	}

	return c.JSON(http.StatusExpectationFailed, errJSON{Errmsg: err.Error()})
}

func displayJobPartial(c echo.Context) error {
	job, err := fetchJobCall(c.Param("id"))

	if job.Created != "" {
		return c.JSON(http.StatusOK, job)
	} else {
		return c.JSON(http.StatusExpectationFailed, errJSON{Errmsg: err.Error()})
	}
}

// RunFrontendServer runs the server
func RunFrontendServer(port int, apiendpoint string) {
	e := echo.New()

	// Turn on/off logging knobs
	e.HideBanner = true
	e.Debug = true

	e.Renderer = &feTemplate{templates: templates}

	e.Use(middleware.Gzip())

	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/home")
	})

	// Login-adjacents
	e.POST("/login", doLogin, loginMiddleware)
	e.POST("/logout", doLogout, loginMiddleware)

	// Webapp paths
	e.GET("/home", homePage, loginMiddleware)
	e.POST("/print", printMedia, loginMiddleware)
	e.GET("/job/:id", displayJob, loginMiddleware)
	e.GET("/job/:id/partial", displayJobPartial)

	// Serve up static files
	e.GET("/static/*", echo.WrapHandler(http.FileServer(http.FS(staticContent))))

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%v", port)))
}
