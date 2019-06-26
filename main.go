/*
	Test file upload on web page with MultiPart web page.
	Will read the whole file into memory,
	and write it all back to a temporary file.
*/

package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/gorilla/sessions"

	"github.com/mholt/certmagic"
	"github.com/postmannen/authsession"
)

// -----------------------------------------------------------------------
// -------------------------------- Main HTTP ----------------------------

type server struct {
	templ     *template.Template
	UploadURL string //the whole url for upload, ex. http://fqdn/upload
	store     *sessions.CookieStore
}

//newServer will return a *server, and will hold all the
// server specific variables.
func newServer(proto string, host string, port string, store *sessions.CookieStore) *server {
	t, err := template.ParseFiles("./static/index.html", "./static/upload.html")
	if err != nil {
		log.Println("error: failed parsing template: ", err)
	}

	return &server{
		templ:     t,
		UploadURL: proto + "://" + host + ":" + port + "/upload",
		store:     store,
	}
}

type TokenData struct {
	Authenticated bool
	ID            string
	Fullame       string
	Email         string
}

//mainPage is the main web page.
func (d *server) mainPage(w http.ResponseWriter, r *http.Request) {
	var err error
	session, err := d.store.Get(r, "cookie-name")
	if err != nil {
		log.Printf("--- error: d.store.get failed: %v\n", err)
	}

	tplData := TokenData{}
	//Since the session values are a map we have to check if there is actual
	// values in the map before we try to convert below, or else it will panic.
	if session.Values["authenticated"] != nil {
		tplData = TokenData{
			Email:         session.Values["email"].(string),
			Authenticated: session.Values["authenticated"].(bool),
		}
		fmt.Println("--- email : ", session.Values["email"].(string))
		fmt.Println("--- auth : ", session.Values["authenticated"].(bool))

	}

	err = d.templ.ExecuteTemplate(w, "mainHTML", tplData)
	if err != nil {
		log.Println("error: executing template: ", err)
	}

}

//handlers contains all the handlers used for this service.
func handlers(d *server, a *authsession.Auth) {
	http.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("/", d.mainPage)
	http.HandleFunc("/upload", a.IsAuthenticated(d.uploadImage))
}

func main() {
	//Check flags
	host := flag.String("host", "localhost", "The FQDN for the web server. Used for the client to know where to upload to.")
	port := flag.String("port", "8080", "The port, like 8080")
	proto := flag.String("proto", "http", "http or https")
	hostListen := flag.String("hostListen", "localhost", "The ip of the interface where the web server will listen. Typically 0.0.0.0 for an internet facing server")
	flag.Parse()

	//Get secret values for authenticating to the google cloud app
	// from environment variables. Then create a new 'auth', and start it.
	cookieStoreKey := os.Getenv("cookiestorekey")
	clientIDKey := os.Getenv("clientidkey")
	clientSecret := os.Getenv("clientsecret")
	a, store := authsession.NewAuth(*proto, *host, *port, cookieStoreKey, clientIDKey, clientSecret)
	a.Run()

	//Greate a new server type that will hold all handlers, and web variable data.
	d := newServer(*proto, *host, *port, store)

	//Initialize the handlers for this program.
	handlers(d, a)

	//if the -proto flag is given 'http', we start a https session
	// with a certificate from letsencrypt.
	if *proto == "https" {
		// read and agree to your CA's legal documents
		certmagic.Default.Agreed = true
		// provide an email address
		certmagic.Default.Email = "you@yours.com"
		// use the staging endpoint while we're developing
		certmagic.Default.CA = certmagic.LetsEncryptStagingCA

		err := certmagic.HTTPS([]string{"eyer.io"}, nil)
		if err != nil {
			log.Println("--- error: cermagic.HTTPS failed: ", err)
			return
		}

	}

	//If no -proto flag was given it will default to serving the page
	// over http.
	log.Println("Web server started, listening at port ", *host+*port)
	err := http.ListenAndServe(*hostListen+":"+*port, nil)
	if err != nil {
		log.Println("error: ListenAndServer failed: ", err)
		return
	}
}
