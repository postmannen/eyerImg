/*
	Test file upload on web page with MultiPart web page.
	Will read the whole file into memory,
	and write it all back to a temporary file.
*/

package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/boltdb/bolt"

	"github.com/gorilla/sessions"

	"github.com/mholt/certmagic"
	"github.com/postmannen/authsession"
)

// -----------------------------------------------------------------------
// -------------------------------- Main HTTP ----------------------------

type server struct {
	templ         *template.Template
	UploadURL     string //the whole url for upload, ex. http://fqdn/upload
	store         *sessions.CookieStore
	Email         string
	Authenticated bool
	db            *bolt.DB
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

//TemplateData is the type describing the information gathered from the token.
type TemplateData struct {
	Authenticated bool
	ID            string
	Fullame       string
	Email         string
	UploadURL     string //the whole url for upload, ex. http://fqdn/upload
	PictureMap    map[string]string
}

//authorized will check if the user is authenticated and authorized for page.
func (s *server) authorized(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//Map to work as authorization scheme.
		allowedUser := make(map[string]bool)
		allowedUser["postmannen@gmail.com"] = true
		allowedUser["hanslad@gmail.com"] = true
		allowedUser["oeystbe2@gmail.com"] = true

		//Check for cookie, and if found put the result in 'session'.
		var err error
		session, err := s.store.Get(r, "cookie-name")
		if err != nil {
			log.Printf("--- error: d.store.get failed: %v\n", err)
		}

		//Check if user is authenticated.
		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			http.Error(w, "Not authenticated", http.StatusForbidden)
			log.Println("info: user not authenticated")
			return
		}

		//Check if user if authorized for access to page.
		if eMail, ok := session.Values["email"].(string); !ok || eMail != "" {
			_, ok := allowedUser[eMail]
			if !ok {
				http.Error(w, "Not authorized..", http.StatusForbidden)
				log.Println("info: not authorized: ", eMail)
				return
			}
		}

		s.Email = session.Values["email"].(string)
		s.Authenticated = session.Values["authenticated"].(bool)

		//We need to execute the HandlerFunc.
		h(w, r)

	}

}

//prepTemplateData will will gather user information and other data from
// the *server type to use inside the template.
// The idea with this function is that we don't pass the whole *server
// struct into the template, only the data we need, and for one specific
// user.
func (s *server) prepTemplateData(r *http.Request) TemplateData {
	var err error
	session, err := s.store.Get(r, "cookie-name")
	if err != nil {
		log.Printf("--- error: d.store.get failed: %v\n", err)
	}

	tplData := TemplateData{}
	//Since the session values are a map we have to check if there is actual
	// values in the map before we try to convert below, or else it will panic.
	if session.Values["authenticated"] != nil {
		tplData = TemplateData{
			Email:         session.Values["email"].(string),
			Authenticated: session.Values["authenticated"].(bool),
			UploadURL:     s.UploadURL,
		}
		return tplData
	}

	return tplData
}

//mainPage is the main web page.
func (s *server) mainPage(w http.ResponseWriter, r *http.Request) {
	var err error

	//get information about user from token, to use with template.
	tplData := s.prepTemplateData(r)

	err = s.templ.ExecuteTemplate(w, "mainHTML", tplData)
	if err != nil {
		log.Println("error: executing template: ", err)
	}

}

//handlers contains all the handlers used for this service.
func handlers(s *server, a *authsession.Auth) {
	http.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("./static"))))
	http.Handle("/pictures/", http.StripPrefix("/pictures", http.FileServer(http.Dir("./pictures"))))
	http.HandleFunc("/", s.mainPage)
	http.HandleFunc("/upload", s.authorized(s.uploadImage))
}

const dbName = "./db.bolt"
const bucketName = "pictures"

func main() {
	var err error

	//Check flags
	host := flag.String("host", "localhost", "The FQDN for the web server. Used for the client to know where to upload to.")
	port := flag.String("port", "8080", "The port, like 8080")
	proto := flag.String("proto", "http", "http or https")
	hostListen := flag.String("hostListen", "localhost", "The ip of the interface where the web server will listen. Typically 0.0.0.0 for an internet facing server")
	flag.Parse()

	//Get secret values for authenticating to the google cloud app
	// from environment variables.
	cookieStoreKey := os.Getenv("cookiestorekey")
	clientIDKey := os.Getenv("clientidkey")
	clientSecret := os.Getenv("clientsecret")

	//Prepare and start the authentication functionality.
	a, store := authsession.NewAuth(*proto, *host, *port, cookieStoreKey, clientIDKey, clientSecret)
	a.Run()

	//Greate a new server type that will hold all handlers, and web variable data.
	srv := newServer(*proto, *host, *port, store)

	//Initialize the handlers for this program.
	handlers(srv, a)

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

	//open takes the file name, permissions for that file, and database options.
	//srv.db, err = bolt.Open(dbName, 0600, nil)
	//if err != nil {
	//	//If we cannot open a db we close the program and print the error
	//	log.Fatalln("error: bolt.Open: ", err)
	//}

	srv.db, err = bolt.Open(dbName, 0600, &bolt.Options{
		Timeout:  1 * time.Second,
		ReadOnly: false,
	})

	if err != nil {
		log.Printf("error: failed to open database in main : %v\n", err)
		return
	}

	err = srv.db.Update(func(tx *bolt.Tx) error {
		bu, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if bu == nil {
			log.Printf("error: bucket = nil in main, and err = %v\n", err)
		}

		return err
	})

	defer srv.db.Close()

	//If no -proto flag was given it will default to serving the page
	// over http.
	log.Println("Web server started, listening at port ", *host+*port)
	err = http.ListenAndServe(*hostListen+":"+*port, nil)
	if err != nil {
		log.Println("error: ListenAndServer failed: ", err)
		return
	}
}
