package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"crypto/rand"

	"github.com/gorilla/sessions"
)

//createRandomKey will create a random []byte with the size taken as input.
func createRandomKey(size int) ([]byte, error) {
	b := make([]byte, size)

	//rand.Read() will read random values from the crypto package,
	// and read them into the []byte b.
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

var (
	// key must be 16, 24 or 32 bytes long (AES-128, AES-192 or AES-256)
	// Note: Don't store your key in your source code. Pass it via an
	// environmental variable, or flag (or both), and don't accidentally commit it
	// alongside your code. Ensure your key is sufficiently random - i.e. use Go's
	// crypto/rand or securecookie.GenerateRandomKey(32) and persist the result.
	//
	key = []byte(os.Getenv("cookiestorekey"))
)

func (a *auth) login(w http.ResponseWriter, r *http.Request) {

	// Authentication goes here
	// ...
	url := a.googleOauthConfig.AuthCodeURL(a.oauthStateString)
	//??? Will redirect to / if authentication fails
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

//logout will logout the user, and revoke the session cookie.
func (a *auth) logout(w http.ResponseWriter, r *http.Request) {
	var err error
	session, err := a.store.Get(r, "cookie-name")
	if err != nil {
		log.Println("error: store.Get in /logout: ", err)
	}

	// Revoke users authentication
	session.Values["authenticated"] = false
	err = session.Save(r, w)
	if err != nil {
		log.Println("error: session.Save on /logout: ", err)
		return
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// -------------------------- OAUTH ---------------------------------------
// ------------------------------------------------------------------------

func newOauthConfig() *oauth2.Config {
	return &oauth2.Config{
		RedirectURL:  "http://localhost:8080/callback",
		ClientID:     os.Getenv("googlekey"),
		ClientSecret: os.Getenv("googlesecret"),
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		Endpoint:     google.Endpoint,
	}
}

//handleGoogleCallback is the handler used when google wants to tell if
// the authentication of the user was ok or not.
// If the authentication is ok, the token.Valid() is set to true, and
// we can then create a cookie with the value "authenticated" for the user.
// We can then check later if that value is present in the cookie to grant
// access to handlers.
func (a *auth) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	//content, err := getUserInfo(r.FormValue("state"), r.FormValue("code"))
	//if err != nil {
	//	fmt.Println(err.Error())
	//	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	//	return
	//}
	//
	//fmt.Fprintf(w, "Content: %s\n", content)

	// ----------------------

	state := r.FormValue("state")
	code := r.FormValue("code")

	fmt.Println(" *** Entering handleGoogleCallback function")

	token, err := a.googleOauthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Println("code exchange failed: ", err.Error())
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}

	fmt.Println("--- state : ", state)
	fmt.Println("--- code : ", code)
	//fmt.Println("token = ", token)

	if !token.Valid() {
		log.Println("error: token not valid in callback function. Token value = ", token.Valid())
		return
	}

	//Get information about user logged in.
	rawUserInfo, err := a.getUserInfo(state, token)
	if err != nil {
		log.Println("error: getUserInfo failed: ", err)
	}

	//"{\n  \"id\": \"109373192721308265542\",\n  \"email\": \"postmannen@gmail.com\",\n  \"verified_email\": true,\n  \"picture\": \"https://lh5.googleusercontent.com/-yKcekqyo_ng/AAAAAAAAAAI/AAAAAAAAAoY/v1RtpnQT494/photo.jpg\"\n}\n"

	userInfo := struct {
		Id            string `json:"id"`
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"verified_email"`
		Picture       string `json:"picture"`
	}{}

	if err := json.Unmarshal(rawUserInfo, &userInfo); err != nil {
		log.Println("error: marshall of the userInfo failed: ", err)
	}
	fmt.Printf("%v\n", userInfo)

	//If all  checks above were ok, we know the the authentication went ok,
	// and we can create a session cookie to use from here.
	session, err := a.store.Get(r, "cookie-name")
	if err != nil {
		log.Println("error: store.Get in /login failed: ", err)
	}

	session.Values["authenticated"] = true
	//set token expire to 1 whole day
	//session.Options = &sessions.Options{MaxAge: 60 * 60 * 24}
	err = session.Save(r, w)
	if err != nil {
		log.Println("error: session.Save on /login: ", err)
		return
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)

}

func (a *auth) getUserInfo(state string, token *oauth2.Token) ([]byte, error) {
	if state != a.oauthStateString {
		return nil, fmt.Errorf("invalid oauth state")
	}

	fmt.Println("Token expire, ", token.Expiry)

	response, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed getting user info: %s", err.Error())
	}

	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading response body: %s", err.Error())
	}

	return contents, nil
}

// -------------------------Auth-------------------------------

type auth struct {
	googleOauthConfig *oauth2.Config
	oauthStateString  string
	store             *sessions.CookieStore
}

func newAuth() *auth {
	return &auth{
		googleOauthConfig: newOauthConfig(),
		//TODO: Replace with random value for each session.
		// Move this inside the /login, and create a map for
		// each user containing the State string for each
		// authentication request, and eventually other
		// variables tied to the individual user.
		oauthStateString: "pseudo-random",
		store:            sessions.NewCookieStore(key),
	}
}

func (a *auth) isAuthenticated(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// ********* SESSION ************
		session, _ := a.store.Get(r, "cookie-name")

		// Check if user is authenticated
		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		// ******************************

		h(w, r)
	}
}
