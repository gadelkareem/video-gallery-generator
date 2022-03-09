package main

import (
	"encoding/json"
	"flag"
	"fmt"
	h "github.com/gadelkareem/go-helpers"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	port := flag.String("p", "8282", "port to serve on")
	d := flag.String("d", ".", "static file folder")
	generate := flag.Bool("g", false, "generate thumbnails")
	flag.Parse()

	fs := listFiles(*d)
	if *generate {
		generateThumbs(*d, fs)
	}
	writeVars(*d, fs)
	http.Handle("/", http.FileServer(http.Dir(*d)))

	log.Printf("Serving %s on HTTP port: http://0.0.0.0:%s\n", *d, *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}

func listFiles(d string) []string {
	files, err := ioutil.ReadDir(d)
	if err != nil {
		log.Fatal(err)
	}
	var fs []string
	for _, f := range files {
		r, err := regexp.MatchString("(?i)\\.(mp4|mov|mpg|mpeg|avi)$", f.Name())
		if err == nil && r {
			fs = append(fs, path.Join(d, f.Name()))
		}
	}
	//log.Printf("%v", fs)

	return fs
}

func generateThumbs(d string, fs []string) {
	thumbsDir := path.Join(d, "thumbs")
	os.MkdirAll(thumbsDir, os.ModePerm)
	for _, f := range fs {
		log.Printf("Generating thumbnail for %s\n", f)
		thumb := path.Join(thumbsDir, path.Base(f)+".png")
		if h.FileExists(thumb) {
			log.Printf("Thumbnail already exists for %s\n", f)
			continue
		}
		cmd := fmt.Sprintf("ffprobe -v error -select_streams v:0 -show_entries stream=duration -of default=noprint_wrappers=1:nokey=1 %s", f)
		//log.Printf("%s\n", cmd)
		b, err := exec.Command("/bin/bash", "-c", cmd).Output()
		if err != nil {
			log.Printf("Error getting video duration for %s, Error: %v\n", f, err)
		}
		s := strings.TrimSuffix(string(b), "\n")
		//log.Printf("Video has %s seconds\n", s)
		numSec, _ := strconv.ParseFloat(s, 64)
		log.Printf("Video has %f seconds\n", numSec)
		cmd = fmt.Sprintf("ffmpeg -y -i '%s' -vf scale=220:-1 -vframes 1 -ss %f '%s'", f, numSec/2, thumb)
		//log.Printf("%s\n", cmd)
		err = exec.Command("/bin/bash", "-c", cmd).Run()
		if err != nil {
			log.Printf("Error generating thumbnail for %s, Error: %v\n", f, err)
		}
	}
}

func writeVars(d string, fs []string) {
	log.Printf("Generating Video List")
	lj, err := json.Marshal(fs)
	if err != nil {
		log.Fatal(err)
	}
	js := fmt.Sprintf("var files = %s", lj)
	err = ioutil.WriteFile(path.Join(d, "vars.js"), []byte(js), os.ModePerm)
	if err != nil {
		log.Fatalf("Error writing list to json file %+v\n", err)
	}
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current path %+v\n", err)
	}

	c, err := h.ReadFile(path.Join(pwd, "gallery.html"))
	if err != nil {
		log.Fatalf("Error reading gallery.html file %+v\n", err)
	}
	err = h.WriteFile(path.Join(d, "gallery.html"), c)
	if err != nil {
		log.Fatalf("Error writing gallery.html file %+v\n", err)
	}
}
