package main

import (
	"errors"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	// _ "net/http/pprof"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	// if you don't need to use jpeg.Encode, import like so:
	// _ "image/jpeg"
)

var imagesPath, cachePath string

var minHeight, maxHeight, minWidth, maxWidth int

var categories []string

func isCached(category, fn string) bool {
	if _, err := os.Stat(path.Join(cachePath, category, fn)); os.IsNotExist(err) {
		return false
	}

	return true
}

func pickFileName(category string) (fn string, ok bool) {
	f, _ := ioutil.ReadDir(path.Join(imagesPath, category))

	if len(f) < 1 {
		return "", false
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	return f[r.Intn(len(f))].Name(), true
}

func getCachedName(fn string, width, height int) string {
	return strings.Join([]string{strconv.Itoa(width), strconv.Itoa(height), fn}, "_")
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

type DeferredFileServe struct {
	w *http.ResponseWriter
	r *http.Request
}

func writeNoCacheHeader(w *http.ResponseWriter) {
	writer := *w
	writer.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
}

func serveFile(w http.ResponseWriter, r *http.Request, category string, width, height int) {
	fn, ok := pickFileName(category)

	log.Println("request to", fn)

	if !ok {
		http.Error(w, errors.New("No file found").Error(), 404)
		return
	}

	cachedName := getCachedName(fn, width, height)

	cachedPath := getCachedPath(category, cachedName)

	if isCached(category, cachedName) {
		// writeNoCacheHeader(&w)
		log.Println("serve cached", category, cachedName, getRemoteAddr(r))
		http.ServeFile(w, r, cachedPath)
		return
	}

	filePath := path.Join(imagesPath, category, fn)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, err.Error(), 404)
		return
	}

	// log.Println("defer", category, cachedName, getRemoteAddr(r))

	c := createCropped(&ImgForCrop{filePath, cachedName, category, width, height})

	<-c

	writeNoCacheHeader(&w)
	http.ServeFile(w, r, cachedPath)
	// log.Println("serve cached - after create", category, cachedName, getRemoteAddr(r))
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

func createDir(dirPath string) {
	err := os.MkdirAll(dirPath, 0755)

	if err != nil {
		log.Fatal(err)
	}
}

func getCategories() []string {
	f, _ := ioutil.ReadDir(imagesPath)

	output := make([]string, 1)

	for _, folder := range f {
		output = append(output, folder.Name())
	}

	return output
}

func init() {
	imagesPath = "imgs"
	cachePath = "cache"

	categories = getCategories()

	minHeight = 10
	maxHeight = 500
	minWidth = 10
	maxWidth = 500

	chanslice = make(map[string][]chan bool)

	for _, category := range categories {
		dirPath := path.Join(cachePath, category)
		createDir(dirPath)
	}
}

func main() {
	http.HandleFunc("/", imgHandler)

	go scheduleFileDelete()

	var port = os.Getenv("PORT")

	if port == "" {
		port = "8001"
	}

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
