package main

import (
	"fmt"
	"log"
	"flag"
	"github.com/Reisender/photosync"
)

var configPath = flag.String("config", "config.json", "Path to configuration file containing the application's credentials.")

func main() {
	if err := photosync.LoadConfig(configPath); err != nil {
		log.Fatalf("Error reading configuration, %v", err)
	}

	user := photosync.GetLogin()

	photos := photosync.GetPhotos(user.Id)

	fmt.Println("length = ", len(*photos))
}
