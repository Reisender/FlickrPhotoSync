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

// Load the consumer key and secret in from the config file
func readCredentials() error {
	b, err := ioutil.ReadFile(*credPath)
	if err != nil {
		return err
	}
	//return json.Unmarshal(b, &oauthClient.Credentials)
	return json.Unmarshal(b, &config)
}

func getPhotos() {
	urlStr := "https://api.flickr.com/services/rest"
	form := url.Values{}
	form.Set("method", "flickr.photos.getUntagged")
	form.Add("format", "json")
	form.Add("nojsoncallback", "1")
	form.Add("per_page", "500") // max page size
	//form.Add("page", "1")


	r, err := oauthClient.Get(http.DefaultClient, &config.Access, urlStr, form)
	if err != nil {
		log.Fatal(err)
	}

	defer r.Body.Close()

	if r.StatusCode != 200 {
		log.Fatal(r.Status)
	}

	var data Response
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}

	json.Unmarshal(contents, &data)
	if data.Stat != "ok" {
		log.Fatal("dang:\n",string(contents))
	}

	fmt.Println(data.Data.Pages)

	/*
	for _, img := range data.Data.Photos {
		fmt.Println("Image title: ",img.Title)
	}
	*/
}

func main() {
	if err := readCredentials(); err != nil {
		log.Fatalf("Error reading configuration, %v", err)
	}

	// setup the consumer key and secret from the confis
	oauthClient.Credentials = config.Consumer

	getPhotos()
}
