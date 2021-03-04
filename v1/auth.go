package zplorama

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"
)

// JWT is stupidly heavyweight for this. The cookie is:
// $username|$expiration_epoch|hmac('$username|$expiration_epoch', secret)

const partSplit = "|"

func validateLoginCookieString(validationString string) (*mail.Address, error) {
	parts := strings.Split(validationString, partSplit)

	if len(parts) != 3 {
		return nil, errors.New("Validation string is wrong length")
	}

	email := parts[0]
	expiration := parts[1]
	checksum := parts[2]

	if !hmac.Equal([]byte(checksum), []byte(makeHmacString(fmt.Sprintf("%s%s%s", email, partSplit, expiration)))) {
		return nil, errors.New("Login string did not pass hash check")
	}

	expepoch, err := strconv.ParseInt(expiration, 10, 64)

	if err != nil {
		return nil, nil
	}

	if time.Now().UTC().Unix() > int64(expepoch) {
		return nil, errors.New("This token is expired")
	}

	unm, err := base64.StdEncoding.DecodeString(email)

	if err != nil {
		return nil, err
	}

	address, err := mail.ParseAddress(string(unm))

	if err != nil {
		return nil, err
	}

	return address, err
}

func makeHmacString(stringer string) string {
	h := hmac.New(sha256.New, []byte(Config.AuthSecret))
	h.Write([]byte(stringer))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func makeLoginCookieString(username string) (string, error) {
	lifetimeDuration, err := time.ParseDuration(Config.AuthtokenLifetime)

	if err != nil {
		return "", err
	}

	// Validate user string looks like an email address
	address, err := mail.ParseAddress(username)
	if err != nil {
		return "", err
	}

	emailFoundInAllowedList := false

emailCheck:
	for _, email := range Config.AllowedLogins {
		if address.Address == email {
			emailFoundInAllowedList = true
			break emailCheck
		} else if email[0] == '@' {
			parts := strings.SplitN(address.Address, "@", 1)
			if "@"+parts[1] == email {
				emailFoundInAllowedList = true
				break emailCheck
			}
		}
	}

	if !emailFoundInAllowedList {
		return "", errors.New("User did not match allowed list")
	}

	expiration := (time.Now().UTC().Add(lifetimeDuration)).Unix()

	userValidationString := fmt.Sprintf("%s%s%v", base64.StdEncoding.Strict().EncodeToString([]byte(username)), partSplit, expiration)

	hmacString := makeHmacString(userValidationString)

	return fmt.Sprintf("%s%s%s", userValidationString, partSplit, hmacString), nil
}

func getLoginInfo(r *http.Request) (*mail.Address, error) {
	cookie, err := r.Cookie("login")

	if err != nil {
		return nil, err
	}

	if cookie == nil {
		err = errors.New("Login cookie is empty")
	} else if cookie.Value == "" {
		err = errors.New("Login cookie value is empty")
	} else {
		return validateLoginCookieString(cookie.Value)
	}

	return nil, err
}
