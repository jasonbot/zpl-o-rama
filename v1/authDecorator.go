package zplorama

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
)

// AuthLevelRequired marks if an auth is required or not
type AuthLevelRequired int

// AuthErrorFormat is how the error should go to the user
type AuthErrorFormat int

const (
	// LoginOptional CAN be logged in
	LoginOptional AuthLevelRequired = iota
	// LoginMandatory MUST be logged in
	LoginMandatory
)

const (
	// ErrorFormatJSON Return error page as JSON
	ErrorFormatJSON AuthErrorFormat = iota
	// ErrorFormatText Return error page as plain text
	ErrorFormatText
)

type authContextKey int

const (
	authKeyUserInfo authContextKey = iota
)

func renderError(w http.ResponseWriter, err error, returnFormat AuthErrorFormat) {
	if returnFormat == ErrorFormatJSON {
		w.WriteHeader(http.StatusForbidden)
		if returnFormat == ErrorFormatJSON {
			w.Header().Add("content-type", "application.json")

			returnBytes, _ := json.Marshal(struct {
				Message string `json:"message"`
			}{Message: err.Error()})

			w.Write(returnBytes)

		} else {
			w.Header().Add("content-type", "text/plain")
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
			renderError(w, errors.New("A login is required"), returnFormat)
			return
		}

		// Decorate request with user's login state context
		nextContext := context.WithValue(r.Context(), authKeyUserInfo, loginAddress)
		next.ServeHTTP(w, r.WithContext(nextContext))
	})
}

// GetUserLogin gets the (possibly nil) email address of the user
func GetUserLogin(r *http.Request) *mail.Address {
	val := r.Context().Value(authKeyUserInfo)

	if val != nil {
		return val.(*mail.Address)
	}

	return nil
}
