package zplorama

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo"
)

// JWT is stupidly heavyweight for this. The cookie is:
// $username|$expiration_epoch|hmac('$username|$expiration_epoch', secret)

const partSplit = "|"
const cookieKey = "applogin"

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

func getLoginInfo(c echo.Context) (*mail.Address, error) {
	cookie, err := c.Cookie(cookieKey)

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

func setLoginInfo(c echo.Context, cookieString string) {
	var lifetime time.Duration

	lifetime, err := time.ParseDuration(Config.AuthtokenLifetime)

	if err != nil {
		lifetime = time.Hour * 4320
	}

	cookie := http.Cookie{
		Name:    cookieKey,
		Value:   cookieString,
		Expires: time.Now().Add(lifetime),
	}
	c.SetCookie(&cookie)
}

func verifyIDToken(idToken string, c echo.Context) error {
	qToken := url.QueryEscape(idToken)
	fetchURL := fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", qToken)
	response, err := http.Get(fetchURL)

	if err != nil {
		return nil
	}

	if response.StatusCode != http.StatusOK {
		return errors.New("Did not get back a valid response from login server")
	}

	var identityResponse struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&identityResponse)

	if err != nil {
		return nil
	}

	if identityResponse.Name == "" {
		identityResponse.Name = identityResponse.Email
	}

	email := mail.Address{
		Name:    identityResponse.Name,
		Address: identityResponse.Email,
	}

	cookieLogin, err := makeLoginCookieString(email.String())

	if err != nil {
		return err
	}

	setLoginInfo(c, cookieLogin)

	return c.Redirect(http.StatusFound, "/application")
}

func deleteIDToken(c echo.Context) {
	cookie := http.Cookie{
		Name:    cookieKey,
		Value:   "",
		Expires: time.Now(),
	}
	c.SetCookie(&cookie)
}

func loginMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, _ := getLoginInfo(c)

		c.Set("login", user)

		return next(c)
	}
}
