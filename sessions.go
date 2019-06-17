package main

import (
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

const storeKeySize = 16

var (
	// key must be 16, 24 or 32 bytes long (AES-128, AES-192 or AES-256)
	// Note: Don't store your key in your source code. Pass it via an
	// environmental variable, or flag (or both), and don't accidentally commit it
	// alongside your code. Ensure your key is sufficiently random - i.e. use Go's
	// crypto/rand or securecookie.GenerateRandomKey(32) and persist the result.
	//
	//TODO: Make Storage of key persistant.
	//NB: Right now it will create a new key everytime the app i restarted.
	key, _ = createRandomKey(storeKeySize)
	store  = sessions.NewCookieStore(key)
)

func (d *data) login(w http.ResponseWriter, r *http.Request) {

	// Authentication goes here
	// ...
	url := d.googleOauthConfig.AuthCodeURL(d.oauthStateString)
	//??? Will redirect to / if authentication fails
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

//logout will logout the user, and revoke the session cookie.
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

	token, err := d.googleOauthConfig.Exchange(oauth2.NoContext, r.FormValue("code"))
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

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)

}

func (d *data) getUserInfo(state string, code string) ([]byte, error) {
	if state != d.oauthStateString {
		return nil, fmt.Errorf("invalid oauth state")
	}

	token, err := d.googleOauthConfig.Exchange(oauth2.NoContext, code)
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
