package photosync

import (
	"os"
	"io"
	"fmt"
	"log"
	"net/url"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"encoding/xml"
	"strconv"
	"sync"
	"github.com/garyburd/go-oauth/oauth"
	"mime/multipart"
	"bytes"
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

type FlickrUploadResponse struct {
	XMLName xml.Name `xml:"rsp"`
	Status string `xml:"stat,attr"`
	PhotoId string `xml:"photoid"`
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


func NewFlickrAPI(config *PhotosyncConfig) *FlickrAPI {
	return &FlickrAPI{
		config: *config, // config the value is set in photosync.go
		apiBase: "https://api.flickr.com/services",
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

func (this *FlickrAPI) GetFilenamesConfig() []FilenameConfig {
	return this.config.Filenames
}

func (this *FlickrAPI) GetPhotos(user *FlickrUser) (*PhotosMap, error) {
	this.form.Set("user_id", user.Id)
	defer this.form.Del("user_id") // remove from form values when done

	this.form.Set("media", "photos")
	defer this.form.Del("media") // remove from form values when done

	return this.Search(&this.form)
}

func (this *FlickrAPI) GetVideos(user *FlickrUser) (*PhotosMap, error) {
	this.form.Set("user_id", user.Id)
	defer this.form.Del("user_id") // remove from form values when done

	this.form.Set("media", "videos")
	defer this.form.Del("media") // remove from form values when done

	return this.Search(&this.form)
}

func (this *FlickrAPI) Search(form *url.Values) (*PhotosMap, error) {
	form.Set("method", "flickr.photos.search")

	// needed for getAllPages
	form.Set("per_page", "500") // max page size
	defer form.Del("per_page") // remove from form values when done

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

	data, err := this.get()
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

	data, err := this.get()
	if err != nil {
		return nil, err
	}

	return &data.PhotoDetails, nil
}

func (this *FlickrAPI) GetSizes(p *Photo) (*[]PhotoSize, error) {
	this.form.Set("method", "flickr.photos.getSizes")

	this.form.Set("photo_id", p.Id)
	defer this.form.Del("photo_id") // remove from form values when done

	data, err := this.get()
	if err != nil {
		return nil, err
	}

	return &data.SizeData.Sizes, nil
}

func (this *FlickrAPI) AddTags(photoId, tags string) error {
	this.form.Set("method", "flickr.photos.addTags")

	this.form.Set("photo_id", photoId)
	defer this.form.Del("photo_id") // remove from form values when done

	this.form.Set("tags", tags)
	defer this.form.Del("tags")

	_, err := this.post()

	return err
}

func (this *FlickrAPI) SetTitle(photo_id, title string) error {
	this.form.Set("method", "flickr.photos.setMeta")

	this.form.Set("photo_id", photo_id)
	defer this.form.Del("photo_id") // remove from form values when done

	this.form.Set("title", title)
	defer this.form.Del("title")

	_, err := this.post()

	return err
}

func (this *FlickrAPI) SetDate(photoId, date string) error {
	this.form.Set("method", "flickr.photos.setDates")

	this.form.Set("photo_id", photoId)
	defer this.form.Del("photo_id") // remove from form values when done

	this.form.Set("date_taken", date)
	defer this.form.Del("date_taken")

	_, err := this.get()

	return err
}

func (this *FlickrAPI) Upload(path string, file os.FileInfo) (*FlickrUploadResponse, error) {
	// Prepare a form that you will submit to that URL.
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add your image file
	f, err := os.Open(path)
	if err != nil { return nil, err }

	fw, err := w.CreateFormFile("photo", file.Name())
	if err != nil { return nil, err }

	if _, err = io.Copy(fw, f); err != nil { return nil, err }

	// close this to get the terminating boundary
	w.Close()

	// create the request
	req, err := http.NewRequest("POST", this.apiBase+"/upload/", &b)
	if err != nil { return nil, err }

	// set the content type for the mutlipart
	req.Header.Set("Content-Type", w.FormDataContentType())

	// add the oauth sig as well
	req.Header.Set("Authorization", this.oauthClient.AuthorizationHeader(&this.config.Access, "POST", req.URL, url.Values{}))

	// do the actual post
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil { return nil, err }

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil { return nil, err }

	// Check the response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}


	xr := FlickrUploadResponse{}
	if err := xml.Unmarshal(body, &xr); err != nil { return nil, err }

	if xr.Status != "ok" {
		return nil, Error{"failed status on upload"}
	}

	return &xr, nil
}

func (this *FlickrAPI) Download(info *PhotoInfo, p *Photo) error {
	sizes, _ := this.GetSizes(p)
	ext, _ := this.GetExtention(info)

	for _, v := range *sizes {
		if (info.Media == "video" && v.Label == "Video Original") || (info.Media == "photo" && v.Label == "Original") {
			out, err := os.Create(p.Title+"."+ext)
			if err != nil { return err }

			r, err := http.Get(v.Source)
			if err != nil { return err }

			defer r.Body.Close()

			n, err := io.Copy(out, r.Body)

			fmt.Println("written ",n)
		}
	}

	return nil
}



// ***** Private Functions *****

func (this *FlickrAPI) get() (*FlickrApiResponse, error) {
	return this.do("GET")
}

func (this *FlickrAPI) post() (*FlickrApiResponse, error) {
	return this.do("POST")
}

func (this *FlickrAPI) put() (*FlickrApiResponse, error) {
	return this.do("PUT")
}

func (this *FlickrAPI) del() (*FlickrApiResponse, error) {
	return this.do("DELETE")
}

func (this *FlickrAPI) do(method string) (*FlickrApiResponse, error) {
	resp := FlickrApiResponse{}
	methodFunc := this.oauthClient.Get
	switch method { // override the default method of get
		case "POST":
			methodFunc = this.oauthClient.Post
		case "PUT":
			methodFunc = this.oauthClient.Put
		case "DELETE":
			methodFunc = this.oauthClient.Delete
	}
	r, err := methodFunc(http.DefaultClient, &this.config.Access, this.apiBase+"/rest", this.form)
	if err != nil { return nil, err }

	defer r.Body.Close()

	if r.StatusCode != 200 {
		return nil, &Error{r.Status}
	}

	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
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

	data, err := this.get()
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
			defer this.form.Del("page")

			data, err := this.get()
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
