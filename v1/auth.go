package zplorama

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo"
	"github.com/yosuke-furukawa/json5/encoding/json5"
)

// JWT is stupidly heavyweight for this. The cookie is:
// $username|$expiration_epoch|hmac('$username|$expiration_epoch', secret)

const partSplit = "|"
const cookieKey = "applogin"
const cookiePictureKey = "applogin.picture"

var openIDAuthEndpoint string
var openIDTokenEndpoint string

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
			if strings.Index(address.Address, "@") != -1 {
				parts := strings.SplitN(address.Address, "@", 2)
				if "@"+parts[1] == email {
					emailFoundInAllowedList = true
					break emailCheck
				}
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

func getLoginInfo(c echo.Context) (*mail.Address, string, error) {
	cookie, err := c.Cookie(cookieKey)

	if err != nil {
		return nil, "", err
	}

	if cookie == nil {
		err = errors.New("Login cookie is empty")
	} else if cookie.Value == "" {
		err = errors.New("Login cookie value is empty")
	} else {
		pictureCookie, err := c.Cookie(cookiePictureKey)
		picture := ""

		if err == nil && pictureCookie != nil {
			picture = pictureCookie.Value
		}

		mail, err := validateLoginCookieString(cookie.Value)
		return mail, picture, err
	}

	return nil, "", err
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

func setPicture(c echo.Context, URI string) {
	var lifetime time.Duration

	lifetime, err := time.ParseDuration(Config.AuthtokenLifetime)

	if err != nil {
		lifetime = time.Hour * 4320
	}

	cookie := http.Cookie{
		Name:    cookiePictureKey,
		Value:   URI,
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
		Name    string `json:"name"`
		Email   string `json:"email"`
		Picture string `json:"picture"`
	}

	decoder := json5.NewDecoder(response.Body)
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
	setPicture(c, identityResponse.Picture)

	return nil
}

func deleteIDToken(c echo.Context) {
	cookie := http.Cookie{
		Name:    cookieKey,
		Value:   "",
		Expires: time.Now(),
		MaxAge:  -1,
	}
	c.SetCookie(&cookie)
}

func createOpenIDConnectToken() string {
	startString := uuid.NewString()

	h := hmac.New(sha256.New, []byte(Config.AuthSecret))
	h.Write([]byte(startString))
	stringSig := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return fmt.Sprintf("%v.%v", startString, stringSig)
}

func validateOpenIDConnectToken(instring string) bool {
	parts := strings.SplitN(instring, ".", 2)

	h := hmac.New(sha256.New, []byte(Config.AuthSecret))
	h.Write([]byte(parts[0]))
	stringSig := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return stringSig == parts[1]
}

func getOpenIDAuthorizationEndpoint() string {
	// Cache
	if openIDAuthEndpoint != "" {
		return openIDAuthEndpoint
	}

	var authEndpoint struct {
		AuthorizationEndpoint string `json:"authorization_endpoint"`
		TokenEndpoint         string `json:"token_endpoint"`
	}

	response, err := http.Get("https://accounts.google.com/.well-known/openid-configuration")

	if err != nil {
		panic(err)
	}

	dec := json5.NewDecoder(response.Body)
	dec.Decode(&authEndpoint)

	openIDAuthEndpoint = authEndpoint.AuthorizationEndpoint

	return authEndpoint.AuthorizationEndpoint
}

func generateOpenIDAuthURL() string {
	authURL := getOpenIDAuthorizationEndpoint()

	urlParts := []string{
		// client_id, which you obtain from the API Console Credentials page .
		fmt.Sprintf("client_id=%v", url.QueryEscape(Config.GoogleSite)),
		// response_type, which in a basic authorization code flow request should be code. (Read more at response_type.)
		"response_type=code",
		//scope, which in a basic request should be openid email. (Read more at scope.)
		"scope=openid%20email%20profile",
		//redirect_uri should be the HTTP endpoint on your server that will receive the response from Google. The value must exactly match one of the authorized redirect URIs for the OAuth 2.0 client, which you configured in the API Console Credentials page. If this value doesn't match an authorized URI, the request will fail with a redirect_uri_mismatch error.
		fmt.Sprintf("redirect_uri=%v", url.QueryEscape(Config.AuthCallback)),
		//state should include the value of the anti-forgery unique session token, as well as any other information needed to recover the context when the user returns to your application, e.g., the starting URL. (Read more at state.)
		fmt.Sprintf("state=%v", url.QueryEscape(createOpenIDConnectToken())),
		// nonce is a random value generated by your app that enables replay protection when present.
		fmt.Sprintf("nonce=%v", url.QueryEscape(createOpenIDConnectToken())),
	}

	queryString := strings.Join(urlParts, "&")

	return fmt.Sprintf("%v?%v", authURL, queryString)
}

func loginMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, picture, _ := getLoginInfo(c)
		c.Set("login", user)

		if user != nil {
			if user.Name == "" {
				c.Set("user_name", user.Address)
			} else {
				c.Set("user_name", user.Name)
			}
			c.Set("picture", picture)
			c.Set("logged_in", true)
		} else {
			c.Set("user_name", "")
			c.Set("picture", "")
			c.Set("logged_in", false)
		}

		return next(c)
	}
}
