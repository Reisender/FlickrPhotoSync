package photosync

import (
	"fmt"
	"log"
	"net/url"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"strconv"
	"sync"
	"github.com/garyburd/go-oauth/oauth"
)


type Photo struct {
	Id string
	Owner string
	Secret string
	Title string
	Ispublic int `json:"string"`
	Isfriend int `json:"string"`
	Isfamily int `json:"string"`
}

type PhotoInfo struct {
	Rotation int
	Originalformat string
	Media string
}

type PhotoSize struct {
	Label string
	Source string
}

type FlickrApiResponse struct {
	Stat string
	Data struct {
		Page int
		Pages int
		Perpage int
		Total string
		Photos []Photo `json:"photo"`
	} `json:"photos"`
	User FlickrUser `json:"user"`
	PhotoDetails PhotoInfo `json:"photo"`
	SizeData struct {
		Sizes []PhotoSize `json:"size"`
	} `json:"sizes"`
}

type FlickrUser struct {
	Id string
	Username struct {
		Content string `json:"_content"`
	} `json:"username"`
}

type FlickrAPI struct {
	config PhotosyncConfig
	FlickrUserId string `json:"flickr_user_id"`
	apiBase string
	form url.Values
	oauthClient oauth.Client
}


// ***** Public Functions *****


func NewFlickrAPI() *FlickrAPI {
	return &FlickrAPI{
		config: config, // config the value is set in photosync.go
		apiBase: "https://api.flickr.com/services/rest",
		form: url.Values{ // default querystring values
			"format": {"json"},
			"nojsoncallback": {"1"},
		},
		oauthClient: oauth.Client {
			TemporaryCredentialRequestURI: "https://api.flickr.com/services/oauth/request_token",
			ResourceOwnerAuthorizationURI: "https://api.flickr.com/services/oauth/authorize",
			TokenRequestURI:               "https://api.flickr.com/services/oauth/access_token",
			Credentials: config.Consumer, // setup the consumer key and secret from the confis
		},
	}
}

func (this *FlickrAPI) GetPhotos(user *FlickrUser) (*PhotosMap, error) {
	this.form.Set("method", "flickr.photos.search")
	this.form.Set("user_id", user.Id)
	defer this.form.Del("user_id") // remove from form values when done

	// needed for getAllPages
	this.form.Set("per_page", "500") // max page size
	defer this.form.Del("per_page") // remove from form values when done

	photos := make(PhotosMap)

	err := this.getAllPages(func(data *FlickrApiResponse) {
		// extract into photos map
		for _, img := range data.Data.Photos {
			photos[img.Title] = img
		}
	})

	return &photos, err
}

func (this *FlickrAPI) GetLogin() (*FlickrUser, error) {
	this.form.Set("method", "flickr.test.login")

	data, err := this.apiGet()
	if err != nil {
		return nil, err
	}

	return &data.User, nil
}

func (this *FlickrAPI) GetExtention(info *PhotoInfo) (string, error) {
	switch info.Media {
	case "photo":
		return "jpg", nil
	case "video":
		return "mp4", nil
	default:
		return "", Error{"Unable to find file extention."}
	}
}

func (this *FlickrAPI) GetInfo(p *Photo) (*PhotoInfo, error) {
	this.form.Set("method", "flickr.photos.getInfo")

	this.form.Set("photo_id", p.Id)
	defer this.form.Del("photo_id") // remove from form values when done

	data, err := this.apiGet()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return &data.PhotoDetails, nil
}

func (this *FlickrAPI) GetSizes(p *Photo) (*[]PhotoSize, error) {
	this.form.Set("method", "flickr.photos.getSizes")

	this.form.Set("photo_id", p.Id)
	defer this.form.Del("photo_id") // remove from form values when done

	data, err := this.apiGet()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return &data.SizeData.Sizes, nil
}

func (this *FlickrAPI) SetTitle(p *Photo, title string) error {
	this.form.Set("method", "flickr.photos.setMeta")

	this.form.Set("photo_id", string(p.Id))
	defer this.form.Del("photo_id") // remove from form values when done

	this.form.Set("title", title)
	defer this.form.Del("title")

	_, err := this.apiGet()

	return err
}


// ***** Private Functions *****


func (this *FlickrAPI) apiGet() (*FlickrApiResponse, error) {
	resp := FlickrApiResponse{}
	r, err := this.oauthClient.Get(http.DefaultClient, &this.config.Access, this.apiBase, this.form)
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

func (this *FlickrAPI) getAllPages(fn func(*FlickrApiResponse)) error {
	var wg sync.WaitGroup

	data, err := this.apiGet()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print("\rloading: ",int((float32(1)/float32(data.Data.Pages))*100),"%")
	wg.Add(data.Data.Pages)
	//go func() {
	func() {
		defer wg.Done()
		fn(data)
	}()

	// get the rest of the pages
	for page := 2; page <= data.Data.Pages; page++ {
		// comment out the parallel requesting as the flickr api seems occasionally return a dup page response
		//go func(page int) { 
		func(page int) { 
			defer wg.Done()

			this.form.Set("page", strconv.Itoa(page))

			data, err := this.apiGet()
			if err != nil {
				log.Fatal(err)
			}

			fmt.Print("\rloading: ",int((float32(page)/float32(data.Data.Pages))*100),"%")

			fn(data)
		}(page)
	}

	wg.Wait()
	fmt.Println("")

	return nil
}
