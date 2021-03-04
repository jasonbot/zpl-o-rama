package zplorama

import (
	"context"
	"errors"
	"net/http"
)

type AuthLevelRequired int
type AuthErrorFormat int

const (
	LoginOptional AuthLevelRequired = iota
	LoginMandatory
)

const (
	ErrorFormatJSON AuthErrorFormat = iota
	ErrorFormatText
)

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
