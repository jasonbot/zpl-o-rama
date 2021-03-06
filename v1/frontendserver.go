package zplorama

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"net/http"
	"text/template"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
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
	c.Bind(&loginRequest)

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
	return c.String(http.StatusGone, "oof")
}

func displayJob(c echo.Context) error {
	return c.String(http.StatusGone, "oof")
}

func displayJobPartial(c echo.Context) error {
	return c.String(http.StatusGone, "oof")
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
	e.GET("/print", printMedia, loginMiddleware)
	e.GET("/job/:id", displayJob, loginMiddleware)
	e.GET("/job/:id/partial", displayJobPartial)

	// Serve up static files
	e.GET("/static/*", echo.WrapHandler(http.FileServer(http.FS(staticContent))))

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%v", port)))
}
