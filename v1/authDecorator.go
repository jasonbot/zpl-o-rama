package zplorama

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

// AuthLevelRequired marks if an auth is required or not
type AuthLevelRequired int
// AuthErrorFormat is how the error should go to the user
type AuthErrorFormat int

const (
	// LoginOptional CAN be logged in
	LoginOptional AuthLevelRequired = iota
	// LoginMandatory
	LoginMandatory MUST be logged in
)

const (
	// ErrorFormatJSON Return error page as JSON
	ErrorFormatJSON AuthErrorFormat = iota
	// ErrorFormatText Return error page as plain text
	ErrorFormatText
)


func renderError(w http.ResponseWriter, err error, returnFormat AuthErrorFormat) {
	if returnFormat == ErrorFormatJSON {
		w.WriteHeader(http.StatusForbidden)
		if (returnFormat == ErrorFormatJSON) {
		w.WriteHeader("content-type", "application.json")

		errorStruct := struct{
Message string `json:"message"`
		}{Message: err.Error()}
		returnBytes, _ := json.Marshal(errorStruct)

		w.Write(returnBytes)

		} else {
			w.WriteHeader("content-type", "text/plain")
			w.Write([]byte(err.Error()))
		}
	}
}

// AuthDecorator makes sure a request enforces the expected login levels
func AuthDecorator(next http.Handler, level AuthLevelRequired, returnFormat AuthErrorFormat) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loginAddress, err := getLoginInfo(r)

		if err != nil && level == LoginMandatory {
			renderError(w, err, returnFormat)
			return
		}

		if loginAddress == nil && level == LoginMandatory {
			err = errors.New("A login is required")
		}

		// Decorate request with user's login state context
		nextContext := context.WithValue(r.Context(), "login", loginAddress)
		next.ServeHTTP(w, r.WithContext(nextContext))
	})
}
