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
	"strings"
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

type PhotoSet struct {
	Id string
	Title string `json:"title[_content]"`
}

type PhotoSize struct {
	Label string
	Source string
}

type FlickrResponse interface {
	Success() bool
}

type FlickrBaseApiResponse struct {
	Stat string
}
func (this FlickrBaseApiResponse) Success() bool {
	return this.Stat == "ok"
}

type FlickrPagedResponse interface {
	Page() int
	Pages() int
	PerPage() int
	Total() int
	Reset()
	Success() bool
}

// Be more flexible when un-marshaling from json. Unmarshal from int or string.
type FlexInt int
func (this *FlexInt) UnmarshalJSON(b []byte) (err error) {
	var p int
	var ps string

	// parse page
	if err = json.Unmarshal(b, &p); err == nil {
		*this = FlexInt(p)
	} else if err = json.Unmarshal(b, &ps); err == nil {
		var val int
		if val, err = strconv.Atoi(ps); err == nil {
			*this = FlexInt(val)
		}
	}

	return
}

type flickrResponsePageInfo struct {
	Page FlexInt
	Pages FlexInt
	Perpage FlexInt
	Total FlexInt
}

type FlickrAlbumsResponse struct {
	FlickrBaseApiResponse
	Data struct {
		flickrResponsePageInfo
		Albums []Album `json:"photoset"`
	} `json:"photosets"`
}
func (this FlickrAlbumsResponse) Page() int { return int(this.Data.Page) }
func (this FlickrAlbumsResponse) Pages() int { return int(this.Data.Pages) }
func (this FlickrAlbumsResponse) PerPage() int { return int(this.Data.Perpage) }
func (this FlickrAlbumsResponse) Total() int { return int(this.Data.Total) }
func (this *FlickrAlbumsResponse) Reset() {
	this.Stat = ""
	this.Data.Page = 0
	this.Data.Pages = 0
	this.Data.Perpage = 0
	this.Data.Total = 0
	this.Data.Albums = []Album{}
}

type FlickrAlbumPhotosResponse struct {
	FlickrBaseApiResponse
	Data struct {
		flickrResponsePageInfo
		Photos []Photo `json:"photo"`
	} `json:"photoset"`
}
func (this FlickrAlbumPhotosResponse) Page() int { return int(this.Data.Page) }
func (this FlickrAlbumPhotosResponse) Pages() int { return int(this.Data.Pages) }
func (this FlickrAlbumPhotosResponse) PerPage() int { return int(this.Data.Perpage) }
func (this FlickrAlbumPhotosResponse) Total() int { return int(this.Data.Total) }
func (this *FlickrAlbumPhotosResponse) Reset() {
	this.Stat = ""
	this.Data.Page = 0
	this.Data.Pages = 0
	this.Data.Perpage = 0
	this.Data.Total = 0
	this.Data.Photos = []Photo{}
}

type FlickrApiResponse struct {
	FlickrBaseApiResponse
	Data struct {
		flickrResponsePageInfo
		Photos []Photo `json:"photo"`
	} `json:"photos"`
	User FlickrUser `json:"user"`
	PhotoDetails PhotoInfo `json:"photo"`
	SizeData struct {
		Sizes []PhotoSize `json:"size"`
	} `json:"sizes"`
}
func (this FlickrApiResponse) Page() int { return int(this.Data.Page) }
func (this FlickrApiResponse) Pages() int { return int(this.Data.Pages) }
func (this FlickrApiResponse) PerPage() int { return int(this.Data.Perpage) }
func (this FlickrApiResponse) Total() int { return int(this.Data.Total) }
func (this *FlickrApiResponse) Reset() {
	this.Stat = ""
	this.Data.Page = 0
	this.Data.Pages = 0
	this.Data.Perpage = 0
	this.Data.Total = 0
	this.Data.Photos = []Photo{}
	this.PhotoDetails = PhotoInfo{}
	this.SizeData.Sizes = []PhotoSize{}
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

	page := FlickrApiResponse{}
	err := this.getAllPages(&page, func() {
		// extract into photos map
		for _, img := range page.Data.Photos {
			photos[img.Title] = img
		}
		fmt.Print("\rloading: ",int((float32(page.Page())/float32(page.Pages()))*100),"%")
	})
	fmt.Println()

	return &photos, err
}

func (this *FlickrAPI) GetAlbums(user *FlickrUser) (*AlbumsMap, error) {
	this.form.Set("method", "flickr.photosets.getList")

	this.form.Set("user_id", user.Id)
	defer this.form.Del("user_id") // remove from form values when done

	// needed for getAllPages
	this.form.Set("per_page", "500") // max page size
	defer this.form.Del("per_page") // remove from form values when done

	albums := make(AlbumsMap)

	page := FlickrAlbumsResponse{}
	fmt.Print("\rloading albums: 0%")
	err := this.getAllPages(&page, func() {
		for i, alb := range page.Data.Albums {
			albCopy := alb
			_ = this.LoadAlbumPhotos(&albCopy)
			albums[albCopy.GetTitle()] = &albCopy
			cnt := (page.Page()-1) * page.PerPage() + (i+1)
			fmt.Print("\rloading albums: ",int((float32(cnt)/float32(page.Total()))*100),"%")
		}
	})
	fmt.Println()

	return &albums, err
}

func (this *FlickrAPI) GetLogin() (*FlickrUser, error) {
	this.form.Set("method", "flickr.test.login")

	data := FlickrApiResponse{}
	err := this.get(&this.form, &data)
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

	data := FlickrApiResponse{}
	err := this.get(&this.form, &data)
	if err != nil {
		return nil, err
	}

	return &data.PhotoDetails, nil
}

func (this *FlickrAPI) GetSizes(p *Photo) (*[]PhotoSize, error) {
	this.form.Set("method", "flickr.photos.getSizes")

	this.form.Set("photo_id", p.Id)
	defer this.form.Del("photo_id") // remove from form values when done

	data := FlickrApiResponse{}
	err := this.get(&this.form, &data)
	if err != nil {
		return nil, err
	}

	return &data.SizeData.Sizes, nil
}

func (this *FlickrAPI) LoadAlbumPhotos(album *Album) error {
	this.form.Set("method", "flickr.photosets.getPhotos")

	this.form.Set("photoset_id", album.Id)
	defer this.form.Del("photoset_id") // remove from form values when done

	page := FlickrAlbumPhotosResponse{}

	return this.getAllPages(&page, func() {
		// extract into photos map
		for _, img := range page.Data.Photos {
			// using album.Append with mark as dirty
			album.PhotoIds = append(album.PhotoIds, img.Id)
		}
	})
}

func (this *FlickrAPI) AddTags(photoId, tags string) error {
	this.form.Set("method", "flickr.photos.addTags")

	this.form.Set("photo_id", photoId)
	defer this.form.Del("photo_id") // remove from form values when done

	this.form.Set("tags", tags)
	defer this.form.Del("tags")

	data := FlickrApiResponse{}
	err := this.post(&this.form, &data)

	return err
}

func (this *FlickrAPI) AddToAlbum(photoId string, album *Album) error {
	this.form.Set("method", "flickr.photosets.addPhoto")

	this.form.Set("photo_id", photoId)
	defer this.form.Del("photo_id") // remove from form values when done

	this.form.Set("photoset_id", album.Id)
	defer this.form.Del("photoset_id")

	data := FlickrBaseApiResponse{}
	if err := this.post(&this.form, &data); err != nil { return err }

	// add to album photoIds array
	album.Prepend(photoId)

	// now set it to the album photo
	this.form.Set("method", "flickr.photosets.setPrimaryPhoto")

	ignore := FlickrBaseApiResponse{}
	return this.post(&this.form, &ignore)
}

func (this *FlickrAPI) SetAlbumOrder(photoSetId string, photoIds []string) error {
	this.form.Set("method", "flickr.photosets.reorderPhotos")

	this.form.Set("photoset_id", photoSetId)
	defer this.form.Del("photoset_id")

	this.form.Set("photo_ids", strings.Join(photoIds, ","))
	defer this.form.Del("photo_ids") // remove from form values when done

	ignore := FlickrBaseApiResponse{}
	if err := this.post(&this.form, &ignore); err != nil {
		return err
	}

	if !ignore.Success() {
		return &Error{"Failed to set album order"}
	}

	return nil
}

func (this *FlickrAPI) SetAlbumPhoto(photoId, photoSetId string) error {
	this.form.Set("method", "flickr.photosets.setPrimaryPhoto")

	this.form.Set("photo_id", photoId)
	defer this.form.Del("photo_id") // remove from form values when done

	this.form.Set("photoset_id", photoSetId)
	defer this.form.Del("photoset_id")

	ignore := FlickrBaseApiResponse{}
	if err := this.post(&this.form, &ignore); err != nil {
		return err
	}

	if !ignore.Success() {
		return &Error{"Failed to set album photo"}
	}

	return nil
}

func (this *FlickrAPI) SetTitle(photo_id, title string) error {
	this.form.Set("method", "flickr.photos.setMeta")

	this.form.Set("photo_id", photo_id)
	defer this.form.Del("photo_id") // remove from form values when done

	this.form.Set("title", title)
	defer this.form.Del("title")

	data := FlickrApiResponse{}
	err := this.post(&this.form, &data)

	return err
}

func (this *FlickrAPI) SetDate(photoId, date string) error {
	this.form.Set("method", "flickr.photos.setDates")

	this.form.Set("photo_id", photoId)
	defer this.form.Del("photo_id") // remove from form values when done

	this.form.Set("date_taken", date)
	defer this.form.Del("date_taken")

	data := FlickrApiResponse{}
	err := this.get(&this.form, &data)

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

func (this *FlickrAPI) get(form *url.Values, resp FlickrResponse) error {
	return this.do("GET", form, resp)
}

func (this *FlickrAPI) post(form *url.Values, resp FlickrResponse) error {
	return this.do("POST", form, resp)
}

func (this *FlickrAPI) put(form *url.Values, resp FlickrResponse) error {
	return this.do("PUT", form, resp)
}

func (this *FlickrAPI) del(form *url.Values, resp FlickrResponse) error {
	return this.do("DELETE", form, resp)
}

func (this *FlickrAPI) do(method string, form *url.Values, resp FlickrResponse) error {
	contents, err := this.doRaw(method, form)
	if err != nil { return err }

	err = json.Unmarshal(contents, resp)
	if err != nil {
		fmt.Println(string(contents))
		return err
	}

	if !resp.Success() {
		return &Error{ string(contents) }
	}

	return nil
}
func (this *FlickrAPI) doRaw(method string, form *url.Values) ([]byte, error) {
	methodFunc := this.oauthClient.Get
	switch method { // override the default method of get
		case "POST":
			methodFunc = this.oauthClient.Post
		case "PUT":
			methodFunc = this.oauthClient.Put
		case "DELETE":
			methodFunc = this.oauthClient.Delete
	}
	r, err := methodFunc(http.DefaultClient, &this.config.Access, this.apiBase+"/rest", *form)
	if err != nil { return nil,err }

	defer r.Body.Close()

	if r.StatusCode != 200 {
		return nil,&Error{r.Status}
	}

	return ioutil.ReadAll(r.Body)
}

func (this *FlickrAPI) getAllPages(data FlickrPagedResponse, fn func()) error {
	var wg sync.WaitGroup
	form := this.form

	err := this.get(&form, data)
	if err != nil {
		log.Fatal(err)
	}
	wg.Add(data.Pages())
	//go func() {
	func() {
		defer wg.Done()
		fn()
	}()

	// get the rest of the pages
	for page := 2; page <= data.Pages(); page++ {
		// comment out the parallel requesting as the flickr api seems occasionally return a dup page response
		//go func(page int, data FlickrPagedResponse) { 
		func(page int, data FlickrPagedResponse) {
			defer wg.Done()

			form.Set("page", strconv.Itoa(page))
			defer form.Del("page")

			data.Reset()
			err := this.get(&form, data)
			if err != nil {
				log.Fatal(err)
			}

			fn()
		}(page, data)
	}

	wg.Wait()

	return nil
}
