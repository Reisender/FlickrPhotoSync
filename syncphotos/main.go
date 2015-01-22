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
	opt := getOptions()

	// ensure the config file exists
	if _, err := os.Stat(opt.ConfigPath); os.IsNotExist(err) {
		fmt.Printf("config file not found: %s", opt.ConfigPath)
		return
	}

	config := photosync.PhotosyncConfig{}

	if err := photosync.LoadConfig(&opt.ConfigPath,&config); err != nil {
		log.Fatalf("Error reading configuration, %v", err)
	}

	fl := photosync.NewFlickrAPI(&config)

	user, errr := fl.GetLogin()
	if errr != nil {
		log.Fatal(errr)
	}

	var err error
	var photos, videos *photosync.PhotosMap
	var albums *photosync.AlbumsMap

	photos, err = fl.GetPhotos(user)
	if err != nil { log.Fatal(err) }
	videos, err = fl.GetVideos(user)
	if err != nil { log.Fatal(err) }
	albums, err = fl.GetAlbums(user)
	if err != nil { log.Fatal(err) }

	fmt.Println(len(*photos),"Flickr photos found")
	fmt.Println(len(*videos),"Flickr videos found")
	fmt.Println(len(*albums),"Flickr albums found")

	if opt.Dryrun { fmt.Println("--+ Dry Run +--") }

	// now walk the directory
	rencnt, excnt, newcnt, errCnt, err := photosync.Sync(fl,photos,videos,albums,opt)
	if err != nil {
		log.Fatal(errCnt,err)
	}

	if opt.Dryrun { fmt.Println("--+ Dry Run +--") }

	fmt.Println(rencnt, " renamed")
	fmt.Println(excnt, " existing")
	fmt.Println(newcnt, " uploaded")
	fmt.Println(errCnt, " failed")
}

func getOptions() *photosync.Options {
	u, _ := user.Current()
	defaultConfPath := u.HomeDir + "/.syncphotos.conf.json"

	configPath := flag.String("config", defaultConfPath, "Path to configuration file containing the application's credentials.")
	dry_run := flag.Bool("dry-run", false, "dry run means don't actually upload or rename files")
	dryrun := flag.Bool("dryrun", false, "dry run means don't actually upload or rename files")

	no_upload := flag.Bool("no-upload", false, "no-upload means don't actually upload files")
	noupload := flag.Bool("noupload", false, "no-upload means don't actually upload files")

	daemon := flag.Bool("daemon", false, "run as a daemon that watches the dirs in the config for newly created files")

	retroTags := flag.Bool("retro-tags", false, "retroactively set the tags for images found in a folder with tags in the config")
	retroAlbums := flag.Bool("retro-albums", false, "retroactively set the albums for images found in a folder with albums in the config")

	flag.Parse()

	// consolidate options
	*dryrun = *dryrun || *dry_run
	*noupload = *noupload || *no_upload

	return &photosync.Options{ *configPath, *dryrun, *noupload, *daemon, *retroTags, *retroAlbums }
}

