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
	"os"

	"github.com/nfnt/resize"
	"github.com/postmannen/authsession"
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
	mainImageSize = 700
	thumbnailSize = 200
)

//uploadImage will open a form for the user for uploading images.
// 2 images will be produced in the function and saved to disk.
// One main image, and one thumbnail.
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
func newServer(proto string, host string, port string) *server {
	t, err := template.ParseFiles("./static/index.html", "./static/upload.html")
	if err != nil {
		log.Println("error: failed parsing template: ", err)
	}

	return &server{
		templ:     t,
		UploadURL: proto + "://" + host + port + "/upload",
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
func handlers(d *server, a *authsession.Auth) {
	http.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("/", d.mainPage)
	http.HandleFunc("/upload", a.IsAuthenticated(d.uploadImage))
}

func main() {
	//Check flags
	host := flag.String("host", "localhost", "The FQDN for the web server. Used for the client to know where to upload to.")
	port := flag.String("port", ":8080", "The port, like :8080")
	proto := flag.String("proto", "http", "http or https")
	hostListen := flag.String("hostListen", "localhost", "The ip of the interface where the web server will listen. Typically 0.0.0.0 for an internet facing server")
	flag.Parse()

	d := newServer(*proto, *host, *port)

	cookieStoreKey := os.Getenv("cookiestorekey")
	clientIDKey := os.Getenv("clientidkey")
	clientSecret := os.Getenv("clientsecret")
	a := authsession.NewAuth(*proto, *host, *port, cookieStoreKey, clientIDKey, clientSecret)
	a.Run()

	handlers(d, a)

	fmt.Println("Web server started, listening at port ", *host+*port)
	err := http.ListenAndServe(*hostListen+*port, nil)
	if err != nil {
		log.Println("error: ListenAndServer failed: ", err)
	}
}
