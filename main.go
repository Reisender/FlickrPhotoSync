package main

import (
	"github.com/garyburd/go-oauth/oauth"
	"fmt"
	"net/http"
	"net/url"
	"log"
	"flag"
	"io/ioutil"
	"encoding/json"
	"strconv"
)

type Config struct {
	Consumer oauth.Credentials
	Access oauth.Credentials
}

type Photo struct {
	Id int
	Owner string
	Secret string
	Title string
	Ispublic int
	Isfriend int
	Isfamily int
}

type Response struct {
	Stat string
	Data struct {
		Page int
		Pages int
		Perpage int
		Total string
		Photos []Photo `json:"photo"`
	} `json:"photos"`
}

var oauthClient = oauth.Client {
	TemporaryCredentialRequestURI: "https://api.flickr.com/services/oauth/request_token",
	ResourceOwnerAuthorizationURI: "https://api.flickr.com/services/oauth/authorize",
	TokenRequestURI:               "https://api.flickr.com/services/oauth/access_token",
}

var credPath = flag.String("config", "config.json", "Path to configuration file containing the application's credentials.")
var config Config
var apiBase = "https://api.flickr.com/services/rest"
var form = url.Values{}

// Load the consumer key and secret in from the config file
func readCredentials() error {
	b, err := ioutil.ReadFile(*credPath)
	if err != nil {
		return err
	}
	//return json.Unmarshal(b, &oauthClient.Credentials)
	return json.Unmarshal(b, &config)
}

func apiGet(resp *Response) {
	r, err := oauthClient.Get(http.DefaultClient, &config.Access, apiBase, form)
	if err != nil {
		log.Fatal(err)
	}

	defer r.Body.Close()

	if r.StatusCode != 200 {
		log.Fatal(r.Status)
	}

	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}

	json.Unmarshal(contents, resp)
	if resp.Stat != "ok" {
		log.Fatal("dang:\n",string(contents))
	}
}

func getAllPages(fn func(*Response)) {
	var data Response
	pageCount := 1

	// get the rest of the pages
	for page := 1; page <= pageCount; page++ {
		form.Set("page", strconv.Itoa(page))

		apiGet(&data)

		// update the page count if needed
		if pageCount != data.Data.Pages {
			pageCount = data.Data.Pages
		}

		fn(&data)
	}
}

func getPhotos() {
	form.Set("method", "flickr.photos.getUntagged")
	form.Add("format", "json")
	form.Add("nojsoncallback", "1")
	form.Add("per_page", "500") // max page size

	photos := make(map[string]Photo)

	getAllPages(func(data *Response) {
		// extract into photos map
		for _, img := range data.Data.Photos {
			photos[img.Title] = img
		}
	})

	fmt.Println("length = ", len(photos))
}

func main() {
	if err := readCredentials(); err != nil {
		log.Fatalf("Error reading configuration, %v", err)
	}

	// setup the consumer key and secret from the confis
	oauthClient.Credentials = config.Consumer

	getPhotos()
}
