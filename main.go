package main

import (
	"errors"
	"image"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	// if you don't need to use jpeg.Encode, import like so:
	// _ "image/jpeg"
)

func isCached(fn string) bool {
	f, err := os.Open("cache/" + fn)

	if err != nil {
		return false
	}

	defer f.Close()

	fi, err := f.Stat()

	if err != nil {
		return false
	}

	mode := fi.Mode()
	return mode.IsRegular()
}

func pickFileName() (fn string, ok bool) {
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

func getCachedName(fn string, width, height int) string {
	return strconv.Itoa(width) + "_" + strconv.Itoa(height) + "_" + fn
}

func createCached(fn string, img *image.NRGBA) {
	cacheFile, err := os.Create("cache/" + fn)

	if err == nil {
		imaging.Encode(cacheFile, img, imaging.JPEG)
	} else {
		log.Println(err)
	}
}

func getCroppedImg(file image.Image, width, height int) *image.NRGBA {
	var size int

	if width > height {
		size = width
	} else {
		size = height
	}

	resized := bigFit(file, size, imaging.Lanczos)
	cropped := imaging.CropCenter(resized, width, height)

	return cropped
}

func serveFile(w http.ResponseWriter, r *http.Request, width, height int) {

	fn, ok := pickFileName()

	if !ok {
		serveErr(w, errors.New("No file found"), 404)
		return
	}

	cachedName := getCachedName(fn, width, height)

	if isCached(cachedName) {
		log.Println("serve cached", "cache/"+cachedName)
		http.ServeFile(w, r, "cache/"+cachedName)
		return
	}

	file, err := imaging.Open("imgs/" + fn)

	if err != nil {
		serveErr(w, err, 404)
		return
	}

	cropped := getCroppedImg(file, width, height)

	imaging.Encode(w, cropped, imaging.JPEG)
	createCached(cachedName, cropped)
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
		serveFile(w, r, width, height)
	}
}

func main() {
	serveHandler := new(ServeFastCGI)

	http.HandleFunc("/kitten/", serveHandler.ImgHandler)

	log.Fatal(http.ListenAndServe(":8001", nil))
}
