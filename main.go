package main

import (
	"archive/zip"
	_ "embed"
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
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

//go:embed gallery.html
var html string

var maxConcurrency = flag.Int64("c", 2, "Maximum number of generators to run concurrently")

func main() {
	port := flag.String("p", "8282", "port to serve on")
	d := flag.String("d", ".", "static file folder")
	generate := flag.Bool("g", false, "generate thumbnails")
	rename := flag.Bool("r", false, "rename files and add _180x180_3dh suffix")
	spatialMedia := flag.Bool("s", false, "add spatial media metadata")
	flag.Parse()

	fs := listFiles(*d)
	if *rename {
		renameFiles(fs)
		fs = listFiles(*d)
	}
	if *spatialMedia {
		addSpatialMedias(*d, fs)
		fs = listFiles(*d)
	}
	writeVars(*d, fs)
	if *generate {
		go generateThumbs(*d, fs)
	}
	http.Handle("/", http.FileServer(http.Dir(*d)))

	log.Printf("Serving %s on HTTP port: http://0.0.0.0:%s\n", *d, *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}

func listFiles(d string) []string {
	var fs []string
	e := filepath.Walk(d, func(p string, f os.FileInfo, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		r, err := regexp.MatchString("(?i)\\.(mp4|mov|mpg|mpeg|avi)$", f.Name())
		if err == nil && r {
			fs = append(fs, p)
		}
		return err
	})
	if e != nil {
		log.Fatal(e)
	}
	//log.Printf("%v", fs)

	return fs
}

func renameFiles(fs []string) {
	wg := h.NewWgExec(*maxConcurrency)
	for _, f := range fs {
		wg.Run(rename, f)
	}
	wg.Wait()
	log.Printf("Renamed %d files\n", len(fs))
}

func rename(ps ...interface{}) {
	of := ps[0].(string)
	f := newName(of)
	if f == "" {
		return
	}
	err := os.Rename(of, f)
	if err != nil {
		log.Printf("Error renaming %s to %s: %v", of, f, err)
	}
	log.Printf("Renamed %s to %s", of, f)
}

func newName(f string) string {
	ext := path.Ext(f)
	format := "_180x180_3dh"
	if strings.HasSuffix(f, format+ext) {
		return ""
	}
	return fmt.Sprintf("%s%s%s", strings.TrimSuffix(f, ext), format, ext)
}

func addSpatialMedias(d string, fs []string) {
	tmpdir := "tmp"
	smPath := path.Join(tmpdir, "spatial-media")
	if !h.FileExists(smPath) {
		err := os.MkdirAll(tmpdir, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
		err = downloadFile("https://github.com/google/spatial-media/archive/refs/tags/v2.1.zip", "spatial-media.zip")
		if err != nil {
			log.Fatal(err)
		}
		err = unzip("spatial-media.zip", "tmp", os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
		err = os.Remove("spatial-media.zip")
		if err != nil {
			log.Fatal(err)
		}
		err = os.Rename(path.Join(tmpdir, "spatial-media-2.1"), smPath)
		if err != nil {
			log.Fatal(err)
		}
	}
	wg := h.NewWgExec(*maxConcurrency)
	for _, f := range fs {
		wg.Run(addSpatialMedia, d, f, smPath)
	}
	wg.Wait()
	log.Printf("Spatial media added to %v files", len(fs))
}

func addSpatialMedia(ps ...interface{}) {
	of := ps[1].(string)
	smPath := ps[2].(string)
	f := newName(of)
	if f == "" {
		return
	}
	//log.Printf("Adding spatial media metadata to %s\n", of)
	cmd := fmt.Sprintf("python2.7 %s/spatialmedia -i -s left-right '%s' '%s'", smPath, of, f)
	//log.Printf("%s\n", cmd)
	b, err := exec.Command("/bin/bash", "-c", cmd).Output()
	if err != nil {
		log.Printf("Error writing spatial media metadata %s, Error: %v Output %s\n", f, err, b)
	} else {
		e := os.Remove(of)
		if e != nil {
			log.Fatal(e)
		}
	}
	log.Printf("Updated spatial media for %s\n", f)
}

func downloadFile(u, dest string) error {
	b, err := h.GetUrl(u)
	if err != nil {
		return err
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = out.Write(b)
	if err != nil {
		return err
	}
	return nil
}

func unzip(source, dest string, prem os.FileMode) error {
	read, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer read.Close()
	for _, file := range read.File {
		if file.Mode().IsDir() {
			continue
		}
		open, err := file.Open()
		if err != nil {
			return err
		}
		name := path.Join(dest, file.Name)
		err = os.MkdirAll(path.Dir(name), prem)
		if err != nil {
			return err
		}
		create, err := os.Create(name)
		if err != nil {
			return err
		}
		defer create.Close()
		_, err = create.ReadFrom(open)
		if err != nil {
			return err
		}
	}
	return nil
}

func generateThumbs(d string, fs []string) {
	thumbsDir := path.Join(d, "thumbs")
	err := os.MkdirAll(thumbsDir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	wg := h.NewWgExec(*maxConcurrency)
	for _, f := range fs {
		wg.Run(createThumb, thumbsDir, f)
	}
	wg.Wait()
	thumbs, err := os.ReadDir(thumbsDir)
	if err != nil {
		log.Fatal(err)
	}

thumbs:
	for _, t := range thumbs {
		for _, f := range fs {
			if path.Base(f) == strings.TrimSuffix(t.Name(), ".png") {
				continue thumbs
			}
		}
		err = os.Remove(path.Join(thumbsDir, t.Name()))
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Printf("Thumbnail generated for %v files", len(fs))
}

func createThumb(ps ...interface{}) {
	thumbsDir := ps[0].(string)
	f := ps[1].(string)
	thumb := path.Join(thumbsDir, path.Base(f)+".png")
	if h.FileExists(thumb) {
		//log.Printf("Thumbnail already exists for %s\n", f)
		return
	}
	log.Printf("Generating thumbnail for %s\n", f)
	cmd := fmt.Sprintf("ffprobe -v error -select_streams v:0 -show_entries stream=duration -of default=noprint_wrappers=1:nokey=1 '%s'", f)
	//log.Printf("%s\n", cmd)
	b, err := exec.Command("/bin/bash", "-c", cmd).Output()
	if err != nil {
		log.Printf("Error getting video duration for %s, Error: %v\n", f, err)
	}
	s := strings.TrimSuffix(string(b), "\n")
	//log.Printf("Video has %s seconds\n", s)
	numSec, _ := strconv.ParseFloat(s, 64)
	//log.Printf("%f seconds in '%s'\n", numSec, f)
	cmd = fmt.Sprintf("ffmpeg -y -i '%s' -vf scale=220:-1 -vframes 1 -ss %f '%s'", f, numSec/2, thumb)
	//log.Printf("%s\n", cmd)
	err = exec.Command("/bin/bash", "-c", cmd).Run()
	if err != nil {
		log.Printf("Error generating thumbnail for %s, Error: %v\n", f, err)
	}
}

func writeVars(d string, fs []string) {
	//log.Printf("Generating Video List")
	var fss []string
	d = strings.TrimSuffix(d, "/") + "/"
	for i, _ := range fs {
		fss = append(fss, strings.Replace(fs[i], d, "", 1))
	}
	lj, err := json.Marshal(fss)
	if err != nil {
		log.Fatal(err)
	}
	js := fmt.Sprintf("var files = %s", lj)
	err = ioutil.WriteFile(path.Join(d, "vars.js"), []byte(js), os.ModePerm)
	if err != nil {
		log.Fatalf("Error writing list to json file %+v\n", err)
	}

	err = h.WriteFile(path.Join(d, "gallery.html"), html)
	if err != nil {
		log.Fatalf("Error writing gallery.html file %+v\n", err)
	}
}
