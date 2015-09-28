package main

import (
	"fmt"
	"github.com/Reisender/photosync"
	"github.com/codegangsta/cli"
	"log"
	"os"
	"os/user"
)

const syncphotos_version_string = "0.1.0"

func run(opt *photosync.Options) {

	// ensure the config file exists
	if _, err := os.Stat(opt.ConfigPath); os.IsNotExist(err) {
		fmt.Printf("config file not found: %s", opt.ConfigPath)
		return
	}

	config := photosync.PhotosyncConfig{}

	if err := photosync.LoadConfig(&opt.ConfigPath, &config); err != nil {
		log.Fatalf("Error reading configuration, %v", err)
	}

	fl := photosync.NewFlickrAPI(&config)

	user, errr := fl.GetLogin()
	if errr != nil {
		log.Fatal(errr)
	}

	var err error
	var photos = &photosync.PhotosMap{}
	var videos = &photosync.PhotosMap{}
	var albums = &photosync.AlbumsMap{}

	if !opt.NoUpload {
		photos, err = fl.GetPhotos(user)
		if err != nil {
			log.Fatal(err)
		}
		videos, err = fl.GetVideos(user)
		if err != nil {
			log.Fatal(err)
		}
		albums, err = fl.GetAlbums(user)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(len(*photos), "Flickr photos found")
		fmt.Println(len(*videos), "Flickr videos found")
		fmt.Println(len(*albums), "Flickr albums found")
	}

	if opt.Dryrun {
		fmt.Println("--+ Dry Run +--")
	}

	// now walk the directory
	rencnt, excnt, newcnt, errCnt, err := photosync.Sync(fl, photos, videos, albums, opt)
	if err != nil {
		log.Fatal(errCnt, err)
	}

	if opt.Dryrun {
		fmt.Println("--+ Dry Run +--")
	}

	fmt.Println(rencnt, " renamed")
	fmt.Println(excnt, " existing")
	if !opt.NoUpload {
		fmt.Println(newcnt, " uploaded")
	}
	fmt.Println(errCnt, " failed")
}

func main() {
	u, _ := user.Current()

	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = "photosync"
	app.Usage = "manage photos and sync with flickr"
	//app.Action = run

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "config",
			Value:  fmt.Sprintf("%s/.syncphotos.conf.json", u.HomeDir),
			Usage:  "path to config json file",
			EnvVar: "PHOTOSYNC_CONFIG",
		},
		cli.BoolFlag{
			Name:   "dry-run, dryrun",
			Usage:  "don't actually make any changes or upload anything",
			EnvVar: "PHOTOSYNC_DRYRUN",
		},
		cli.BoolFlag{
			Name:   "deamon",
			Usage:  "run as a daemon that watches the dirs in the config for newly created files",
			EnvVar: "PHOTOSYNC_DAEMON",
		},
	}

	renameFlags := append(app.Flags, []cli.Flag{}...)

	syncFlags := append(app.Flags, []cli.Flag{
		cli.BoolFlag{
			Name:   "no-upload, noupload",
			Usage:  "no-upload means don't actually upload files",
			EnvVar: "PHOTOSYNC_RETRO_TAGS",
		},
		cli.BoolFlag{
			Name:   "retro-tags",
			Usage:  "retroactively set the tags for images found in a folder with tags in the config",
			EnvVar: "PHOTOSYNC_RETRO_TAGS",
		},
		cli.BoolFlag{
			Name:   "retro-albums",
			Usage:  "retroactively set the albums for images found in a folder with albums in the config",
			EnvVar: "PHOTOSYNC_RETRO_ALBUMS",
		},
	}...)

	app.Commands = []cli.Command{
		{
			Name:    "version",
			Aliases: []string{"v"},
			Usage:   "print out the version",
			Action:  version,
		},
		{
			Name:    "rename",
			Aliases: []string{"r"},
			Usage:   "rename matching files with date",
			Flags:   renameFlags,
			Action:  rename,
		},
		{
			Name:    "sync",
			Aliases: []string{"s"},
			Usage:   "sync the files to flickr",
			Flags:   syncFlags,
			Action:  sync,
		},
	}

	app.Run(os.Args)
}

func version(c *cli.Context) {
	println("syncphotos version ", syncphotos_version_string)
}

func parseOptions(c *cli.Context) *photosync.Options {
	return &photosync.Options{c.String("config"), c.Bool("dry-run"), c.Bool("no-upload"), c.Bool("daemon"), c.Bool("retro-tags"), c.Bool("retro-albums")}
}

func rename(c *cli.Context) {
	opts := parseOptions(c)
	opts.NoUpload = true
	run(opts)
}

func sync(c *cli.Context) {
	opts := parseOptions(c)
	run(opts)
}
