package zplorama

import (
	"embed"
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

//go:embed static/*
var staticContent embed.FS

type loginRequestStruct struct {
	IDToken string `json:"id_token"`
}

func doLogin(c echo.Context) error {
	loginRequest := new(loginRequestStruct)
	c.Bind(&loginRequest)

	return verifyIDToken(loginRequest.IDToken, c)
}

func doLogout(c echo.Context) error {
	deleteIDToken(c)

	return c.String(http.StatusNoContent, "")
}

// RunFrontendServer runs the server
func RunFrontendServer(port int, apiendpoint string) {
	e := echo.New()
	e.HideBanner = true
	e.Debug = true

	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{Level: 5}))

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	e.POST("/login", doLogin)
	e.POST("/logout", doLogout)
	e.GET("/static/*", echo.WrapHandler(http.FileServer(http.FS(staticContent))))

	e.Use(loginMiddleware)
	e.Logger.Fatal(e.Start(fmt.Sprintf(":%v", port)))

}
