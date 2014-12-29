package photosync

import (
	"github.com/garyburd/go-oauth/oauth"
	"fmt"
	"net/http"
	"net/url"
	"log"
	"io/ioutil"
	"encoding/json"
	"strconv"
	"sync"
)

type Config struct {
	Consumer oauth.Credentials
	Access oauth.Credentials
	FlickrUserId string `json:"flickr_user_id"`
}

type Photo struct {
	Id int `json:"string"`
	Owner string
	Secret string
	Title string
	Ispublic int `json:"string"`
	Isfriend int `json:"string"`
	Isfamily int `json:"string"`
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
	User FlickrUser `json:"user"`
}

type FlickrUser struct {
	Id string
	Username struct {
		Content string `json:"_content"`
	} `json:"username"`
}

// API Error type
type Error struct {
	response string
}
func (e *Error) Error() string {
	return fmt.Sprintf("API fail: %s", e.response)
}


var oauthClient = oauth.Client {
	TemporaryCredentialRequestURI: "https://api.flickr.com/services/oauth/request_token",
	ResourceOwnerAuthorizationURI: "https://api.flickr.com/services/oauth/authorize",
	TokenRequestURI:               "https://api.flickr.com/services/oauth/access_token",
}

var config Config
var apiBase = "https://api.flickr.com/services/rest"
var form = url.Values{}

// Load the consumer key and secret in from the config file
func LoadConfig(configPath *string) error {
	b, err := ioutil.ReadFile(*configPath)
	if err != nil {
		return err
	}

	errr := json.Unmarshal(b, &config)
	if errr != nil {
		return errr
	}

	// setup the consumer key and secret from the confis
	oauthClient.Credentials = config.Consumer

	return nil
}

func apiGet() (*Response, error) {
	resp := Response{}
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

	err = json.Unmarshal(contents, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Stat != "ok" {
		return nil, &Error{ string(contents) }
	}

	return &resp, nil
}

func getAllPages(fn func(*Response)) {
	var wg sync.WaitGroup

	data, err := apiGet()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("got page 1")
	wg.Add(data.Data.Pages)
	go func() {
		defer wg.Done()
		fn(data)
	}()

	// get the rest of the pages
	for page := 2; page <= data.Data.Pages; page++ {
		go func(page int) {
			defer wg.Done()

			form.Set("page", strconv.Itoa(page))

			data, err := apiGet()
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("got page ",page)

			fn(data)
		}(page)
	}

	wg.Wait()
}

func GetPhotos(flickrUserId string) (*map[string]Photo) {
	form.Set("method", "flickr.photos.search")
	form.Set("format", "json")
	form.Set("nojsoncallback", "1")
	form.Set("user_id", flickrUserId)
	form.Set("per_page", "500") // max page size

	photos := make(map[string]Photo)

	getAllPages(func(data *Response) {
		// extract into photos map
		for _, img := range data.Data.Photos {
			photos[img.Title] = img
		}
	})

	return &photos
}

func GetLogin() *FlickrUser {
	form.Set("method", "flickr.test.login")
	form.Set("format", "json")
	form.Set("nojsoncallback", "1")

	data, err := apiGet()
	if err != nil {
		log.Fatal(err)
	}

	return &data.User
}
