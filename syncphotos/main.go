package main

import (
	"fmt"
	"log"
	"flag"
	"os"
	"os/user"
	"github.com/Reisender/photosync"
)

func main() {
	u, _ := user.Current()
	defaultConfPath := u.HomeDir + "/.syncphotos.conf.json"

	configPath := flag.String("config", defaultConfPath, "Path to configuration file containing the application's credentials.")
	dry_run := flag.Bool("dry-run", false, "dry run means don't actually upload files")
	dryrun := flag.Bool("dryrun", false, "dry run means don't actually upload files")

	flag.Parse()

	// consolidate options
	*dryrun = *dryrun || *dry_run

	// ensure the config file exists
	if _, err := os.Stat(*configPath); os.IsNotExist(err) {
		fmt.Printf("config file not found: %s", configPath)
		return
	}

	config := photosync.PhotosyncConfig{}

	if err := photosync.LoadConfig(configPath,&config); err != nil {
		log.Fatalf("Error reading configuration, %v", err)
	}

	fl := photosync.NewFlickrAPI(&config)

	user, errr := fl.GetLogin()
	if errr != nil {
		log.Fatal(errr)
	}

	var err error
	var photos, videos *photosync.PhotosMap
	videos, err = fl.GetVideos(user)
	if err != nil { log.Fatal(err) }
	photos, err = fl.GetPhotos(user)
	if err != nil { log.Fatal(err) }

	fmt.Println(len(*photos),"Flickr photos found")
	fmt.Println(len(*videos),"Flickr videos found")

	if *dryrun { fmt.Println("--+ Dry Run +--") }

	// now walk the directory
	excnt, newcnt, err := photosync.Sync(fl,photos,videos,*dryrun)
	if err != nil {
		log.Fatal(err)
	}

	if *dryrun { fmt.Println("--+ Dry Run +--") }

	fmt.Println(excnt, " existing")
	fmt.Println(newcnt, " uploaded")
}
