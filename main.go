package main

import (
	"errors"
	"image"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	// if you don't need to use jpeg.Encode, import like so:
	// _ "image/jpeg"
)

var imagesPath, cachePath string

var minHeight, maxHeight, minWidth, maxWidth int

var categories []string

func isCached(category, fn string) bool {
	f, err := os.Open(path.Join(cachePath, category, fn))

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

func pickFileName(category string) (fn string, ok bool) {
	f, _ := ioutil.ReadDir(path.Join(imagesPath, category))

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
	return strings.Join([]string{strconv.Itoa(width), strconv.Itoa(height), fn}, "_")
}

func createCached(category, fn string, img *image.NRGBA) {
	cacheFile, err := os.Create(path.Join(cachePath, category, fn))

	if err == nil {
		imaging.Encode(cacheFile, img, imaging.JPEG)
		defer cacheFile.Close()
	} else {
		log.Println(err)
	}

}

func getCroppedImg(file image.Image, width, height int, c chan *image.NRGBA) {
	var size int

	if width > height {
		size = width
	} else {
		size = height
	}

	resized := bigFit(file, size, imaging.Lanczos)
	cropped := imaging.CropCenter(resized, width, height)

	c <- cropped
}

func isValidCategory(category string) bool {
	if category == "" {
		return false
	}

	for _, c := range categories {
		if c == category {
			return true
		}
	}

	return false
}

func writeNoCacheHeader(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
}

func serveFile(w http.ResponseWriter, r *http.Request, category string, width, height int) {
	fn, ok := pickFileName(category)

	if !ok {
		http.Error(w, errors.New("No file found").Error(), 404)
		return
	}

	cachedName := getCachedName(fn, width, height)

	if isCached(category, cachedName) {
		writeNoCacheHeader(w)
		http.ServeFile(w, r, path.Join(cachePath, category, cachedName))
		log.Println("serve cached", category, cachedName, getRemoteAddr(r))
		return
	}

	file, err := imaging.Open(path.Join(imagesPath, category, fn))

	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}

	var cropped *image.NRGBA
	channel := make(chan *image.NRGBA)

	go getCroppedImg(file, width, height, channel)

	cropped = <-channel

	writeNoCacheHeader(w)
	imaging.Encode(w, cropped, imaging.JPEG)
	log.Println("serve", category, fn, getRemoteAddr(r))

	go createCached(category, cachedName, cropped)
}

func isValidSize(width, height int) bool {
	return width <= maxWidth &&
		height <= maxHeight &&
		width >= minWidth &&
		height >= minHeight
}

func getWidthHeight(filePath []string) (width, height int, er error) {
	var widthS, heightS string

	if len(filePath) == 4 {
		widthS = filePath[2]
		heightS = filePath[3]
	} else if len(filePath) == 3 {
		widthS = filePath[1]
		heightS = filePath[2]
	} else {
		return 0, 0, errors.New("Bad URL")
	}

	widthN, err := strconv.Atoi(widthS)

	if err != nil {
		return 0, 0, err
	}

	heightN, err := strconv.Atoi(heightS)

	if err != nil {
		return 0, 0, err
	}

	if !isValidSize(widthN, heightN) {
		return 0, 0, errors.New("Size not allowed")
	}

	return widthN, heightN, nil
}

func pickCategory() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	return categories[r.Intn(len(categories))]
}

func getCategory(filePath []string) string {
	filePathLen := len(filePath)

	var output string

	if filePathLen == 4 {
		output = filePath[1]
	} else if filePathLen == 3 {
		output = pickCategory()
	}

	return output
}

func imgHandler(w http.ResponseWriter, r *http.Request) {
	filePath := strings.Split(r.URL.Path, "/")

	category := getCategory(filePath)

	if !isValidCategory(category) {
		http.Error(w, errors.New("Invalid category").Error(), 400)
		return
	}

	width, height, err := getWidthHeight(filePath)

	if err != nil {
		http.Error(w, err.Error(), 400)
	} else {
		serveFile(w, r, category, width, height)
	}
}

func getRemoteAddr(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")

	if forwarded != "" {
		return forwarded
	}

	return r.RemoteAddr
}

func init() {
	imagesPath = "imgs"
	cachePath = "cache"

	categories = []string{"kitten"}

	minHeight = 10
	maxHeight = 500
	minWidth = 10
	maxWidth = 500
}

func main() {
	http.HandleFunc("/", imgHandler)

	log.Fatal(http.ListenAndServe(":8001", nil))
}
