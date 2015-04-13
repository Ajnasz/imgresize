package main

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"strconv"
	"time"
)

func keepResolution(width, height int) bool {
	return width%10 == 0 && height%10 == 0 && width < 300 && height < 300
}

func isDeletable(fn string, modTime time.Time) bool {
	dateLimit := time.Now().Add(-time.Hour * 24)

	if modTime.After(dateLimit) {
		return false
	}

	re := regexp.MustCompile("^(\\d+)+_(\\d+)")

	match := re.FindAllStringSubmatch(fn, -1)

	width, _ := strconv.Atoi(match[0][1])
	height, _ := strconv.Atoi(match[0][2])

	return !keepResolution(width, height)
}

func getCachedFiles() []string {
	dirs, er := ioutil.ReadDir(cachePath)

	output := make([]string, 0)

	if er != nil {
		log.Fatal(er)
	}

	for _, folder := range dirs {
		files, _ := ioutil.ReadDir(path.Join(cachePath, folder.Name()))

		for _, file := range files {
			fn := file.Name()
			if isDeletable(fn, file.ModTime()) {
				output = append(output, path.Join(cachePath, folder.Name(), fn))
			}
		}
	}

	return output
}

func deleteCachedFiles(files []string) {
	if len(files) < 1 {
		return
	}
	for _, file := range files {
		err := os.Remove(file)

		if err != nil {
			log.Println(err, file)
		} else {
			log.Println("File deleted", file)
		}
	}
}

func scheduleFileDelete() {
	log.Println("Schedule file delete")
	deleteCachedFiles(getCachedFiles())
	for {
		select {
		case <-time.After(time.Hour):
			deleteCachedFiles(getCachedFiles())
		}
	}
}
