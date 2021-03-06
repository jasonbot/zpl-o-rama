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
	var err error
	templates, err = template.ParseFS(templatesFS, "template/*.tpl")

	if err != nil {
		panic(err)
	}
}

type loginRequestStruct struct {
	IDToken string `json:"id_token"`
}

func doLogin(c echo.Context) error {
	loginRequest := new(loginRequestStruct)
	c.Bind(&loginRequest)

	err := verifyIDToken(loginRequest.IDToken, c)

	if err == nil {
		var buffer bytes.Buffer
		userName := c.Get("user_name").(string)

		templates.ExecuteTemplate(&buffer, "loginbar", struct{ User string }{User: userName})
		return c.JSON(http.StatusOK, &hotwireResponse{
			Message: "Logged in",
			DivID:   "loginbar",
			HTML:    string(buffer.Bytes()),
		})
	} else {
		return c.JSON(http.StatusUnauthorized, &errJSON{Errmsg: fmt.Sprintf("Cannot log in: %v", err)})
	}
}

func doLoginForce(c echo.Context) error {
	cookieLogin, _ := makeLoginCookieString("Jason Scheirer <jason.scheirer@gmail.com>")
	setLoginInfo(c, cookieLogin)

	var buffer bytes.Buffer
	userName := c.Get("user_name").(string)

	templates.ExecuteTemplate(
		&buffer,
		"loginbar",
		struct {
			User string
		}{
			User: userName,
		})

	return c.JSON(http.StatusOK, &hotwireResponse{
		Message: "Logged in",
		DivID:   "loginbar",
		HTML:    string(buffer.Bytes()),
	})
}

func doLogout(c echo.Context) error {
	deleteIDToken(c)

	var buffer bytes.Buffer
	templates.ExecuteTemplate(
		&buffer,
		"loginbar",
		struct {
			User string
		}{})

	return c.JSON(http.StatusOK, &hotwireResponse{
		Message: "Logged in",
		DivID:   "loginbar",
		HTML:    string(buffer.Bytes()),
	})
}

func homePage(c echo.Context) error {
	sessionUsername := c.Get("user_name")

	userName := ""

	if sessionUsername != nil {
		userName = sessionUsername.(string)
	}

	return c.Render(http.StatusOK, "main", struct {
		Title string
		User  string
		Body  string
	}{
		Title: "ZPL-O-Rama: Home",
		User:  userName,
		Body:  "Hi there.",
	})
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
	e.GET("/login", doLoginForce, loginMiddleware)
	e.POST("/login", doLogin, loginMiddleware)
	e.POST("/logout", doLogout, loginMiddleware)

	// Webapp paths
	e.GET("/home", homePage, loginMiddleware)

	// Serve up static files
	e.GET("/static/*", echo.WrapHandler(http.FileServer(http.FS(staticContent))))

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%v", port)))
}
