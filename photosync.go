package photosync

import (
	"fmt"
	"os"
	"github.com/garyburd/go-oauth/oauth"
	"io/ioutil"
	"encoding/json"
	"path/filepath"
	"strings"
)

type PhotosMap map[string]Photo

type OauthConfig struct {
	Consumer oauth.Credentials
	Access oauth.Credentials
}

type PhotosyncConfig struct {
	OauthConfig
	WatchDir string `json:"watch_dir"`
}

var config PhotosyncConfig

// Load the consumer key and secret in from the config file
func LoadConfig(configPath *string) error {
	b, err := ioutil.ReadFile(*configPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, &config)
}

func checkFile(path string, f os.FileInfo, err error) error {
	ext := strings.ToUpper(filepath.Ext(f.Name()))

	if ext == ".JPG" || ext == ".MOV" || ext == ".MP4" {
		fname := strings.Split(f.Name(),ext)
		key := strings.Join(fname[:len(fname)-1],ext)
		fmt.Print("checking:", key)



		fmt.Println("")
	}
	return nil
}

func Sync(api *FlickrAPI, photos *PhotosMap) error {
	// walk the directory
	fmt.Println(config.WatchDir)

	err := filepath.Walk(config.WatchDir, func(path string, f os.FileInfo, err error) error {
		ext := strings.ToUpper(filepath.Ext(f.Name()))

		if ext == ".JPG" || ext == ".MOV" || ext == ".MP4" {
			fname := strings.Split(f.Name(),ext)
			key := strings.Join(fname[:len(fname)-1],ext)
			fmt.Print("checking: ", key)

			_, exists := (*photos)[key]

			if exists {
				fmt.Print(" exists")
			} else {
				fmt.Print(" need to upload")
			}

			fmt.Println("")
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
