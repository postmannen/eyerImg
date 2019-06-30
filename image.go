package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"

	"github.com/nfnt/resize"
)

const (
	mainImageSize = 700
	thumbnailSize = 200
)

const picturePath = "./pictures/"

//uploadImage will open a form for the user for uploading images.
// 2 images will be produced in the function and saved to disk.
// One main image, and one thumbnail.
func (s *server) uploadImage(w http.ResponseWriter, r *http.Request) {
	var err error

	//prepare data to use in template.
	tplData := s.prepTemplateData(r)

	fmt.Println("------------------BEFORE VIEWALL")
	picMap, err := dbViewAll(s.db, bucketName)
	if err != nil {
		log.Println("error: failed failed to get key/values from db: ", err)
	}

	fmt.Println("------------------AFTER VIEWALL")
	//We store the picMap in the TemplateData struct, so we can use it's
	// values inside the template.
	tplData.PictureMap = picMap

	//---TESTEND

	if err := s.templ.ExecuteTemplate(w, "upload", tplData); err != nil {
		log.Println("error: failed executing template for upload: ", err)
	}

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

	mainOutFile, err := ioutil.TempFile(picturePath, "*.jpg")
	if err != nil {
		log.Println("error: creating TempFile: ", err)
		return
	}
	fileNameMain := mainOutFile.Name()

	if err := shrinkImage(inFile, mainOutFile, thumbnailSize); err != nil {
		log.Println("error: shrink image failed: ", err)
	}

	mainOutFile.Close()
	// ------------------------- Creating thumbnail ----------------------------------
	splitName := strings.Split(fileNameMain, ".")

	thumbOutFile, err := os.Create(splitName[0] + "tmb" + ".jpg")
	if err != nil {
		log.Println("error: creating TempFile: ", err)
	}

	fileNameThumb := thumbOutFile.Name()

	//Seek back to the beginning of the file, so we can read it once more.
	_, err = inFile.Seek(0, 0)
	if err != nil {
		log.Println("error: Failed seek to the start of read file: ", err)
	}

	if err := shrinkImage(inFile, thumbOutFile, mainImageSize); err != nil {
		log.Println("error: shrink image failed: ", err)
	}

	fmt.Println("----------- preparing to update db")
	err = dbUpdate(s.db, bucketName, fileNameThumb, fileNameMain)
	if err != nil {
		log.Println("error: failed to update db with new key/value: ", err)
	}
	fmt.Println("---------- done storing to db --------------------")

	thumbOutFile.Close()

	//parse the request to check if the upload button was pressed.
	// If the button was pressed call a template who will redirect
	// to a new /upload page to update shown images on page.
	// TODO: The picture update should be replaced with a websocket.
	r.ParseForm()
	if r.FormValue("fileSubmit") == "upload" {
		s.templ.ExecuteTemplate(w, "redirectToUpload", nil)
	}
}

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
