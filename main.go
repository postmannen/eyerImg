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

	_ "net/http/pprof"

	"github.com/nfnt/resize"
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

func (d *server) uploadImage(w http.ResponseWriter, r *http.Request) {
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

type server struct {
	templ     *template.Template
	UploadURL string //the whole url for upload, ex. http://fqdn/upload
}

//newServer will return a *server, and will hold all the
// server specific variables.
func newServer(uploadURL string) *server {
	t, err := template.ParseFiles("./static/index.html", "./static/upload.html")
	if err != nil {
		log.Println("error: failed parsing template: ", err)
	}

	return &server{
		templ:     t,
		UploadURL: uploadURL,
	}
}

//mainPage is the main web page.
func (d *server) mainPage(w http.ResponseWriter, r *http.Request) {
	err := d.templ.ExecuteTemplate(w, "mainHTML", nil)
	if err != nil {
		log.Println("error: executing template: ", err)
	}
}

//handlers contains all the handlers used for this service.
func handlers(d *server, a *auth) {
	http.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("/", d.mainPage)
	http.HandleFunc("/login", a.login)
	http.HandleFunc("/logout", a.logout)
	http.HandleFunc("/callback", a.handleGoogleCallback)
	http.HandleFunc("/upload", isAuthenticated(d.uploadImage))
}

func main() {
	uploadURL := flag.String("uploadURL", "http://localhost:8080/upload", "The complete URL to the upload handler")
	flag.Parse()

	d := newServer(*uploadURL)
	a := newAuth()

	handlers(d, a)

	fmt.Println("Web server started, listening at port ", serverListenPort)
	err := http.ListenAndServe(serverListenPort, nil)
	if err != nil {
		log.Println("error: ListenAndServer failed: ", err)
	}
}
