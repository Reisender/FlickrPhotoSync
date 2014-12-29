package photosync

import (
	"github.com/garyburd/go-oauth/oauth"
	"io/ioutil"
	"encoding/json"
)

type PhotosMap map[string]Photo

type OauthConfig struct {
	Consumer oauth.Credentials
	Access oauth.Credentials
}

var config OauthConfig

// Load the consumer key and secret in from the config file
func LoadConfig(configPath *string) error {
	b, err := ioutil.ReadFile(*configPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, &config)
}
