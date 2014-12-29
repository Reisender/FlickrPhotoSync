package main

import (
	"fmt"
	"log"
	"flag"
	"strings"
	"github.com/Reisender/photosync"

	"os"
	"io"
	"net/http"
)

func addExtention(api *photosync.FlickrAPI, p *photosync.Photo) {
	info, _ := api.GetInfo(p)
	ext, err := api.GetExtention(info)
	if err != nil {
		log.Fatal(err)
	}
	newTitle := p.Title+"."+ext

	fmt.Println("updating title for ",p.Title, "=>", newTitle)

	//api.SetTitle(p, newTitle)
}

func download(api *photosync.FlickrAPI, info *photosync.PhotoInfo, p *photosync.Photo) {
	sizes, _ := api.GetSizes(p)
	i, _ := api.GetInfo(p)
	ext, _ := api.GetExtention(i)

	for _, v := range *sizes {
		if (info.Media == "video" && v.Label == "Video Original") || (info.Media == "photo" && v.Label == "Original") {
			out, err := os.Create(p.Title+"."+ext)
			if err != nil {
				log.Fatal(err)
			}

			r, err := http.Get(v.Source)
			if err != nil {
				log.Fatal(err)
			}
			defer r.Body.Close()

			n, err := io.Copy(out, r.Body)

			fmt.Println("written ",n)
		}
	}
}

func main() {
	configPath := flag.String("config", "config.json", "Path to configuration file containing the application's credentials.")

	flag.Parse()

	if err := photosync.LoadConfig(configPath); err != nil {
		log.Fatalf("Error reading configuration, %v", err)
	}

	fl := photosync.NewFlickrAPI()

	user, err := fl.GetLogin()
	if err != nil {
		log.Fatal(err)
	}

	photos, err := fl.GetPhotos(user)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(len(*photos),"photos found")


	// see if we should fix the titles with a file extension
	var ans string
	for k, v := range *photos {
		if !strings.Contains(k,".") {
			ask: for {
				switch ans {
				case "y":
					ans = "" // reset the answer for the next image
					addExtention(fl,&v)
					break ask // we are done with this ask
				case "all":
					addExtention(fl,&v)
					break ask
				case "n":
					ans = "" // reset the answer for the next image
					fallthrough
				case "none":
					break ask
				case "quit":
					return
				default:
					fmt.Println(k, " is missing the extension")
					fmt.Print("would you like to add?\n(y/n/all/none/quit) :")
					fmt.Scanln(&ans)
				}
			}
		}
		//break
	}
	fmt.Println("good bye")
}
