package zplorama

import (
	"bytes"
	"embed"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
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
		picture := c.Get("picture").(string)

		html := renderTemplateString("loginbar",
			struct {
				User    string
				Picture string
			}{
				User:    userName,
				Picture: picture,
			})
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

func doSignIn(c echo.Context) error {
	if c.Get("logged_in").(bool) {
		return c.Redirect(http.StatusFound, "/")
	}

	return c.Redirect(http.StatusSeeOther, generateOpenIDAuthURL())
}

func doSignInCallback(c echo.Context) error {
	code := c.QueryParam("code")
	state := c.QueryParam("state")

	if !validateOpenIDConnectToken(state) {
		return c.JSON(http.StatusExpectationFailed, errJSON{Errmsg: "Potential MitM attack?"})
	}

	params := []string{
		fmt.Sprintf("code=%v", url.QueryEscape(code)),
		fmt.Sprintf("client_id=%v", url.QueryEscape(Config.GoogleSite)),
		fmt.Sprintf("client_secret=%v", url.QueryEscape(Config.AppSecret)),
		fmt.Sprintf("redirect_uri=%v", url.QueryEscape(Config.AuthCallback)),
		"grant_type=authorization_code",
	}

	bodyBytes := []byte(strings.Join(params, "&"))

	response, err := http.Post(openIDTokenEndpoint, "application/x-www-form-urlencoded", bytes.NewBuffer(bodyBytes))

	if err != nil {
		return c.JSON(http.StatusExpectationFailed, errJSON{Errmsg: "Post to OpenID: " + openIDTokenEndpoint + " -- " + err.Error()})
	}

	var idToken openIDResponseToken
	decoder := json5.NewDecoder(response.Body)
	err = decoder.Decode(&idToken)

	if err != nil {
		return c.JSON(http.StatusExpectationFailed, errJSON{Errmsg: "Decoding response: " + err.Error()})
	}

	tokenParts := strings.SplitN(idToken.IDtoken, ".", 3)

	jsonBlob, err := base64.StdEncoding.DecodeString(tokenParts[1])

	if err != nil {
		return c.JSON(http.StatusExpectationFailed, errJSON{Errmsg: "Unbase64ing idtoken: " + err.Error()})
	}

	var token openIDResponseIDToken

	err = json5.Unmarshal(jsonBlob, &token)

	if err != nil {
		return c.JSON(http.StatusExpectationFailed, errJSON{Errmsg: "Decoding idtoken: " + err.Error()})
	}

	user := mail.Address{Name: token.Name, Address: token.Email}
	if user.Name == "" {
		user.Name = user.Address
	}

	tokenString, err := makeLoginCookieString(user.String())

	if err != nil {
		return c.JSON(http.StatusForbidden, errJSON{Errmsg: "Can't log you in: " + user.String() + err.Error()})
	}

	setLoginInfo(c, tokenString)
	setPicture(c, token.Picture)

	return c.Redirect(http.StatusFound, "/home")
}

func homePage(c echo.Context) error {
	var body string
	var userName string
	var picture string

	if c.Get("logged_in").(bool) == true {
		userName = c.Get("user_name").(string)
		picture = c.Get("picture").(string)
		body = renderTemplateString("input-zpl-form", nil)
	} else {
		body = renderTemplateString("please-log-in", nil)
	}

	return c.Render(http.StatusOK, "main", struct {
		Title   string
		User    string
		Body    string
		Picture string
	}{
		Title:   "ZPL-O-Rama: Home",
		User:    userName,
		Body:    body,
		Picture: picture,
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

	if err != nil {
		return c.JSON(http.StatusExpectationFailed, errJSON{Errmsg: err.Error()})
	}

	var userName, picture string

	if c.Get("logged_in").(bool) == true {
		userName = c.Get("user_name").(string)
		picture = c.Get("picture").(string)
	}

	body := renderTemplateString("job-status", job)

	if job.Done {
		c.Response().Header().Set("Cache-Control", "max-age=31536000")
	} else {
		c.Response().Header().Set("Cache-Control", "max-age=0")
	}

	return c.Render(http.StatusOK, "main", struct {
		Title   string
		User    string
		Body    string
		Picture string
	}{
		Title:   "ZPL-O-Rama: Print Job",
		User:    userName,
		Body:    body,
		Picture: picture,
	})
}

func displaySmallJobImage(c echo.Context) error {
	job, err := fetchJobCall(c.Param("id"))

	if job.Done {
		c.Response().Header().Set("Cache-Control", "max-age=31536000")
	} else {
		c.Response().Header().Set("Cache-Control", "max-age=10")
	}

	if err != nil {
		return c.JSON(http.StatusExpectationFailed, errJSON{Errmsg: err.Error()})
	}

	if job.Done {
		c.Response().Header().Set("Cache-Control", "max-age=31536000")
	} else {
		c.Response().Header().Set("Cache-Control", "max-age=0")
	}

	if job.ImageB64Small == "" {
		job.ImageB64Small, err = shrinkImage(job.ImageB64)
	}

	if err != nil {
		return c.JSON(http.StatusInternalServerError, errJSON{Errmsg: err.Error()})
	}

	data, _ := base64.StdEncoding.DecodeString(job.ImageB64Small)

	return c.Blob(http.StatusOK, "image/png", data)
}

func displayJobImage(c echo.Context) error {
	job, err := fetchJobCall(c.Param("id"))

	if err != nil {
		return c.JSON(http.StatusExpectationFailed, errJSON{Errmsg: err.Error()})
	}

	if job.Done {
		c.Response().Header().Set("Cache-Control", "max-age=31536000")
	} else {
		c.Response().Header().Set("Cache-Control", "max-age=0")
	}

	c.Response().Header().Set(
		"Content-Disposition",
		fmt.Sprintf(
			"attachment; filename=\"%v-original.png\"",
			job.Jobid,
		))

	data, _ := base64.StdEncoding.DecodeString(job.ImageB64)

	return c.Blob(http.StatusOK, "image/png", data)
}

func displayJobPartial(c echo.Context) error {
	job, err := fetchJobCall(c.Param("id"))

	if err != nil {
		return c.JSON(http.StatusExpectationFailed, errJSON{Errmsg: err.Error()})
	}

	if job.Done {
		c.Response().Header().Set("Cache-Control", "max-age=31536000")
	} else {
		c.Response().Header().Set("Cache-Control", "max-age=0")
	}

	body := renderTemplateString("job-status-part", job)

	return c.JSON(
		http.StatusOK,
		hotwireResponse{
			Message: string(job.Status),
			DivID:   "jobstatus",
			HTML:    body,
		})
}

// RunFrontendServer runs the server
func RunFrontendServer(port int, apiendpoint string) {
	e := echo.New()

	// Turn on/off logging knobs
	e.HideBanner = true
	e.Debug = true

	e.Renderer = &feTemplate{templates: templates}

	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/home")
	})

	// Login-adjacents
	e.POST("/login", doLogin, loginMiddleware)
	e.POST("/logout", doLogout, loginMiddleware)

	// Login-adjacents
	e.GET("/signin", doSignIn, loginMiddleware)
	e.GET("/callback", doSignInCallback, loginMiddleware)
	e.POST("/signout", doLogout, loginMiddleware)

	// Webapp paths
	e.GET("/home", homePage, loginMiddleware)
	e.POST("/print", printMedia, loginMiddleware)
	e.GET("/job/:id", displayJob, loginMiddleware, middleware.Gzip())
	e.GET("/job/:id/image.png", displaySmallJobImage, middleware.Gzip())
	e.GET("/job/:id/original.png", displayJobImage, middleware.Gzip())
	e.GET("/job/:id/partial", displayJobPartial, middleware.Gzip())

	// Serve up static files
	e.GET("/static/*", echo.WrapHandler(http.FileServer(http.FS(staticContent))))

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%v", port)))
}
