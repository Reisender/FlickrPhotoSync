package main

import (
	"os"
	"fmt"
	"log"
	"flag"
	"path/filepath"
	"github.com/Reisender/photosync"
)

func main() {
	configPath := flag.String("config", "config.json", "Path to configuration file containing the application's credentials.")
	dryrun := flag.Bool("dryrun", false, "dry run means don't actually upload files")

	flag.Parse()

	config := photosync.PhotosyncConfig{}

	if err := photosync.LoadConfig(configPath,&config); err != nil {
		log.Fatalf("Error reading configuration, %v", err)
	}

	if !(*dryrun) {
		// check for valid exif
		for _, dir := range config.WatchDir {
			err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
				newpath, done, errr := photosync.FixExif(path, f, err)
				fmt.Println(newpath)
				done()
				return errr
			})

			if err != nil { log.Fatal(err) }
		}
	}

	fmt.Println("done")
}
