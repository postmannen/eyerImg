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
	"image"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/nfnt/resize"
	"golang.org/x/oauth2"
)

//shrinkImage will shrink an image to the width specified in size, and keep
// the verical aspect ratio for that width.
func shrinkImage(inReader io.Reader, outWriter io.Writer, size uint) error {
	decIm, _, err := image.Decode(inReader)
	if err != nil {
		return err
	}

	rezIm := resize.Resize(size, 0, decIm, resize.Lanczos3)

	err = jpeg.Encode(outWriter, rezIm, nil)
	if err != nil {
		return err
	}

	return nil
}

const (
	mainImageSize    = 700
	thumbnailSize    = 200
	serverListenPort = "0.0.0.0:8080"
)

func (d *data) uploadImage(w http.ResponseWriter, r *http.Request) {
	// ********* SESSION ************
	session, _ := store.Get(r, "cookie-name")

	// Check if user is authenticated
	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	// ******************************

	if err := d.templ.ExecuteTemplate(w, "upload", d); err != nil {
		log.Println("error: failed executing template for upload: ", err)
	}

	var err error

	//Takes max size of form to parse, so here we can limit the size of the image to upload.
	err = r.ParseMultipartForm(32 << 20)
	if err != nil {
		log.Println("error: ParseMultipartForm: ", err)
		//NB: NB: We have to return here if it fails, since the first show of the
		// page no files are specified, and the page will panic on the code below
		// if it continues and try to get the FormFile.
		return
	}

	//Get a handler for the file found in the web form.
	//Returns the first file for the provided key.
	inFile, inFileHeader, err := r.FormFile("myFile")
	if err != nil {
		log.Println("error: failed to return web file: ", err)
		return
	}
	defer inFile.Close()
	fmt.Printf("File uploaded : %v\n", inFileHeader.Filename)
	fmt.Printf("File size : %v\n", inFileHeader.Size)
	fmt.Printf("File MIME header : %v\n", inFileHeader.Header)

	// ------------------------- Creating main image ----------------------------------

	mainOutFile, err := ioutil.TempFile("./", "tmp100-*.jpg")
	if err != nil {
		log.Println("error: creating TempFile: ", err)
		return
	}

	if err := shrinkImage(inFile, mainOutFile, thumbnailSize); err != nil {
		log.Println("error: shrink image failed: ", err)
	}

	mainOutFile.Close()
	// ------------------------- Creating thumbnail ----------------------------------
	thumbOutFile, err := ioutil.TempFile("./", "tmp400-*.jpg")
	if err != nil {
		log.Println("error: creating TempFile: ", err)
	}

	//Seek back to the beginning of the file, so we can read it once more.
	_, err = inFile.Seek(0, 0)
	if err != nil {
		log.Println("error: Failed seek to the start of read file: ", err)
	}

	if err := shrinkImage(inFile, thumbOutFile, mainImageSize); err != nil {
		log.Println("error: shrink image failed: ", err)
	}

	thumbOutFile.Close()
}

// -----------------------------------------------------------------------
// -------------------------------- Main HTTP ----------------------------

type data struct {
	templ             *template.Template
	googleOauthConfig *oauth2.Config
	oauthStateString  string
	UploadURL         string
}

//newData will return a *data, which holds all the templates parsed.
func newData() *data {
	t, err := template.ParseFiles("./static/index.html", "./static/upload.html")
	if err != nil {
		log.Println("error: failed parsing template: ", err)
	}

	return &data{templ: t}
}

func (d *data) mainPage(w http.ResponseWriter, r *http.Request) {
	err := d.templ.ExecuteTemplate(w, "mainHTML", nil)
	if err != nil {
		log.Println("error: executing template: ", err)
	}
}

func main() {
	uploadURL := flag.String("uploadURL", "http://localhost:8080/upload", "The complete URL to the upload handler")
	flag.Parse()

	http.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("./static"))))
	d := newData()
	d.googleOauthConfig = newOauthConfig()
	//TODO: Replace with random value for each session.
	// Move this inside the /login, and create a map for
	// each user containing the State string for each
	// authentication request, and eventually other
	// variables tied to the individual user.
	d.oauthStateString = "pseudo-random2"
	d.UploadURL = *uploadURL

	http.HandleFunc("/", d.mainPage)
	http.HandleFunc("/login", d.login)
	http.HandleFunc("/logout", d.logout)
	http.HandleFunc("/callback", d.handleGoogleCallback)
	http.HandleFunc("/upload", d.uploadImage)

	fmt.Println("Web server started, listening at port ", serverListenPort)
	err := http.ListenAndServe(serverListenPort, nil)
	if err != nil {
		log.Println("error: ListenAndServer failed: ", err)
	}
}
