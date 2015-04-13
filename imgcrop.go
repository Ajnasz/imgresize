package main

import (
	"github.com/disintegration/imaging"
	"image"
	"log"
	"os"
	"path"
)

type ImgForCrop struct {
	File       string
	CachedName string
	Category   string
	Width      int
	Height     int
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

func getCachedPath(category, fn string) string {
	return path.Join(cachePath, category, fn)
}

func createCached(file string, img *image.NRGBA) {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		cacheFile, err := os.Create(file)

		if err == nil {
			defer cacheFile.Close()
			imaging.Encode(cacheFile, img, imaging.JPEG)

			log.Println("Cached file written", file)
		} else {
			log.Println(err)
		}
	}

}

var chanslice map[string][]chan bool

func createCropped(img *ImgForCrop) chan bool {
	fn := getCachedPath(img.Category, img.CachedName)

	if len(chanslice[fn]) == 0 {
		go func() {
			if _, err := os.Stat(fn); os.IsNotExist(err) {
				log.Println("Create cropped", fn)

				file, _ := imaging.Open(img.File)

				cropped := getCroppedImg(file, img.Width, img.Height)

				createCached(fn, cropped)

				for _, listener := range chanslice[fn] {
					listener <- true
				}

				chanslice[fn] = []chan bool{}
			} else {
				log.Println("File aready exists", fn)
				for _, listener := range chanslice[fn] {
					listener <- true
				}

				chanslice[fn] = []chan bool{}
			}
		}()
	}

	// writeNoCacheHeader(w)
	// imaging.Encode(w, cropped, imaging.JPEG)
	// log.Println("serve", category, fn, getRemoteAddr(r))

	c := make(chan bool)

	chanslice[fn] = append(chanslice[fn], c)

	return c
}
