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

func Sync(api *FlickrAPI, photos *PhotosMap, dryrun bool) (int, int, error) {
	existingCount := 0
	uploadedCount := 0

	return existingCount, uploadedCount, filepath.Walk(config.WatchDir, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() { // make sure we aren't operating on a directory

			ext := strings.ToUpper(filepath.Ext(f.Name()))
			if ext == ".JPG" || ext == ".MOV" || ext == ".MP4" {
				fname := strings.Split(f.Name(),ext)
				key := strings.Join(fname[:len(fname)-1],ext)
				fmt.Println(path)

				_, exists := (*photos)[key]

				if !exists {
					fmt.Println("|==========| 100%")

					if !dryrun {
						if _, err := api.Upload(path, f); err != nil { return err }
					}

					uploadedCount++
				} else {
					existingCount++
				}
			}
		}

		return nil
	})
}
