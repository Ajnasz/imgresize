package main

import (
	"bytes"
	"image"
	"image/jpeg"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"os"

	"github.com/nfnt/resize"
	// if you don't need to use jpeg.Encode, import like so:
	// _ "image/jpeg"
)

func getImageSize(r io.Reader) [2]int {
	im, _, err := image.DecodeConfig(file)
}

func resizeImage(file []byte, width, height uint) (image.Image, error) {

	img, _, err := image.Decode(bytes.NewReader(file))

	if err != nil {
		return nil, err
	}

	newImage := resize.Resize(width, height, img, resize.Lanczos3)

	return newImage, err
}

func serveFile(w http.ResponseWriter, width, height uint) {
	reader, err := os.Open("kep.jpg"); err != nil {
		defer reader.Close()

		getImageSize(reader)
	}
	file, err := ioutil.ReadFile("kep.jpg")

	if err != nil {
		serveErr(w, err)
	} else {

		image, err := resizeImage(file, width, height)

		if err != nil {
			serveErr(w, err)
		} else {
			jpeg.Encode(w, image, nil)
		}
	}
}

func serveErr(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), 500)
}

func main() {
	http.HandleFunc("/kitten/", func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.URL.Path)
		path := strings.Split(r.URL.Path, "/")

		width, err := strconv.ParseUint(path[2], 0, 64)
		if err != nil {
			serveErr(w, err)
		}
		height, err := strconv.ParseUint(path[3], 0, 64)

		if err != nil {
			serveErr(w, err)
		} else {
			serveFile(w, uint(width), uint(height))
		}
	})

	log.Fatal(http.ListenAndServe(":8001", nil))
}
