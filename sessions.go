package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/gorilla/sessions"
)

var (
	// key must be 16, 24 or 32 bytes long (AES-128, AES-192 or AES-256)
	key   = []byte("super-secret-key")
	store = sessions.NewCookieStore(key)
)

func (d *data) login(w http.ResponseWriter, r *http.Request) {
	//var err error

	//session, err := store.Get(r, "cookie-name")
	//if err != nil {
	//	log.Println("error: store.Get in /login failed: ", err)
	//}

	// Authentication goes here
	// ...
	url := googleOauthConfig.AuthCodeURL(oauthStateString)
	//??? Will redirect to / if authentication fails
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)

	// Set user as authenticated
	//session.Values["authenticated"] = true
	//err = session.Save(r, w)
	//if err != nil {
	//	log.Println("error: session.Save on /login: ", err)
	//	return
	//}
}

func (d *data) logout(w http.ResponseWriter, r *http.Request) {
	var err error
	session, err := store.Get(r, "cookie-name")
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
}

// -------------------------- OAUTH ---------------------------------------
// ------------------------------------------------------------------------

var (
	// TODO: randomize it
	oauthStateString = "pseudo-random2"
)

var (
	googleOauthConfig *oauth2.Config
)

func init() {
	googleOauthConfig = &oauth2.Config{
		RedirectURL:  "http://localhost:8080/callback",
		ClientID:     os.Getenv("googlekey"),
		ClientSecret: os.Getenv("googlesecret"),
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		Endpoint:     google.Endpoint,
	}
}

func (d *data) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	//content, err := getUserInfo(r.FormValue("state"), r.FormValue("code"))
	//if err != nil {
	//	fmt.Println(err.Error())
	//	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	//	return
	//}
	//
	//fmt.Fprintf(w, "Content: %s\n", content)

	// ----------------------

	fmt.Println(" *** Entering handleGoogleCallback function")

	token, err := googleOauthConfig.Exchange(oauth2.NoContext, r.FormValue("code"))
	if err != nil {
		log.Println("code exchange failed: ", err.Error())
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}

	if !token.Valid() {
		log.Println("error: token not valid in callback function. Token value = ", token.Valid())

	}

	session, err := store.Get(r, "cookie-name")
	if err != nil {
		log.Println("error: store.Get in /login failed: ", err)
	}

	session.Values["authenticated"] = true
	err = session.Save(r, w)
	if err != nil {
		log.Println("error: session.Save on /login: ", err)
		return
	}

}

func (d *data) getUserInfo(state string, code string) ([]byte, error) {
	if state != oauthStateString {
		return nil, fmt.Errorf("invalid oauth state")
	}

	token, err := googleOauthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, fmt.Errorf("code exchange failed: %s", err.Error())
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
