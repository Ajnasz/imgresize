package main

import (
	"errors"
	"image"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	// if you don't need to use jpeg.Encode, import like so:
	// _ "image/jpeg"
)

func pickFile() (fn string, ok bool) {
	f, _ := ioutil.ReadDir("imgs")

	if len(f) < 1 {
		return "", false
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	return f[r.Intn(len(f))].Name(), true
}

func bigFit(img image.Image, size int, filter imaging.ResampleFilter) *image.NRGBA {

	srcBounds := img.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	srcAspectRatio := float64(srcW) / float64(srcH)

	var newW, newH int

	if srcW < srcH {
		newW = size
		newH = int(float64(newW) / srcAspectRatio)
	} else {
		newH = size
		newW = int(float64(newH) * srcAspectRatio)
	}

	return imaging.Resize(img, newW, newH, filter)
}

func serveFile(w http.ResponseWriter, width, height int) {

	fn, ok := pickFile()

	if !ok {
		serveErr(w, errors.New("No file found"), 404)
		return
	}

	file, err := imaging.Open("imgs/" + fn)

	if err != nil {
		serveErr(w, err, 404)
		return
	}

	var size int

	if width > height {
		size = width
	} else {
		size = height
	}

	resized := bigFit(file, size, imaging.Lanczos)
	cropped := imaging.CropCenter(resized, width, height)

	imaging.Encode(w, cropped, imaging.JPEG)
}

func serveErr(w http.ResponseWriter, err error, status int) {
	http.Error(w, err.Error(), status)
}

func getWidthHeight(path []string) (width, height int, er error) {
	if len(path) != 4 {
		return 0, 0, errors.New("Bad URL")
	}

	widthN, err := strconv.Atoi(path[2])

	if err != nil {
		return 0, 0, err
	}

	heightN, err := strconv.Atoi(path[3])

	if err != nil {
		return 0, 0, err
	}

	return widthN, heightN, nil
}

type ServeFastCGI struct{}

func (s ServeFastCGI) ImgHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.Split(r.URL.Path, "/")

	width, height, err := getWidthHeight(path)

	if err != nil {
		serveErr(w, err, 500)
	} else {
		serveFile(w, width, height)
	}
}

func main() {
	serveHandler := new(ServeFastCGI)

	http.HandleFunc("/kitten/", serveHandler.ImgHandler)

	log.Fatal(http.ListenAndServe(":8001", nil))
}
