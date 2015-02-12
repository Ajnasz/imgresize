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

	"github.com/nfnt/resize"
	// if you don't need to use jpeg.Encode, import like so:
	// _ "image/jpeg"
)

func serveFile(w http.ResponseWriter, width, height uint) {
	file, err := ioutil.ReadFile("kep.jpg")

	if err != nil {
		log.Fatal(err)
	}

	img, _, err := image.Decode(bytes.NewReader(file))

	newImage := resize.Resize(width, height, img, resize.Lanczos3)

	err = jpeg.Encode(w, newImage, nil)
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
