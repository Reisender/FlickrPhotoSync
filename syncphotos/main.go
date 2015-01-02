package main

import (
	"fmt"
	"log"
	"flag"
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

	fl := photosync.NewFlickrAPI(&config)

	user, err := fl.GetLogin()
	if err != nil {
		log.Fatal(err)
	}

	photos, err := fl.GetPhotos(user)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(len(*photos),"Flickr photos found")

	if *dryrun { fmt.Println("--+ Dry Run +--") }

	// now walk the directory
	excnt, newcnt, err := photosync.Sync(fl,photos,*dryrun)
	if err != nil {
		log.Fatal(err)
	}

	if *dryrun { fmt.Println("--+ Dry Run +--") }

	fmt.Println(excnt, " existing")
	fmt.Println(newcnt, " uploaded")
}
