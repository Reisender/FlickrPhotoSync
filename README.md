PhotoSync
===============

Test bed for me to play with go. This will watch a folder and uploading anything new to Flickr, including videos.


# photosync
--
    import "github.com/Reisender/photosync"


## Usage

#### func  LoadConfig

```go
func LoadConfig(configPath *string) error
```
Load the consumer key and secret in from the config file

#### func  Sync

```go
func Sync(api *FlickrAPI, photos *PhotosMap, dryrun bool) (int, int, error)
```

#### type Error

```go
type Error struct {
}
```

API Error type

#### func (Error) Error

```go
func (e Error) Error() string
```

#### type FlickrAPI

```go
type FlickrAPI struct {
	FlickrUserId string `json:"flickr_user_id"`
}
```


#### func  NewFlickrAPI

```go
func NewFlickrAPI() *FlickrAPI
```

#### func (*FlickrAPI) Download

```go
func (this *FlickrAPI) Download(info *PhotoInfo, p *Photo)
```

#### func (*FlickrAPI) GetExtention

```go
func (this *FlickrAPI) GetExtention(info *PhotoInfo) (string, error)
```

#### func (*FlickrAPI) GetInfo

```go
func (this *FlickrAPI) GetInfo(p *Photo) (*PhotoInfo, error)
```

#### func (*FlickrAPI) GetLogin

```go
func (this *FlickrAPI) GetLogin() (*FlickrUser, error)
```

#### func (*FlickrAPI) GetPhotos

```go
func (this *FlickrAPI) GetPhotos(user *FlickrUser) (*PhotosMap, error)
```

#### func (*FlickrAPI) GetSizes

```go
func (this *FlickrAPI) GetSizes(p *Photo) (*[]PhotoSize, error)
```

#### func (*FlickrAPI) SetTitle

```go
func (this *FlickrAPI) SetTitle(p *Photo, title string) error
```

#### func (*FlickrAPI) Upload

```go
func (this *FlickrAPI) Upload(path string, file os.FileInfo) (*FlickrUploadResponse, error)
```

#### type FlickrApiResponse

```go
type FlickrApiResponse struct {
	Stat string
	Data struct {
		Page    int
		Pages   int
		Perpage int
		Total   string
		Photos  []Photo `json:"photo"`
	} `json:"photos"`
	User         FlickrUser `json:"user"`
	PhotoDetails PhotoInfo  `json:"photo"`
	SizeData     struct {
		Sizes []PhotoSize `json:"size"`
	} `json:"sizes"`
}
```


#### type FlickrUploadResponse

```go
type FlickrUploadResponse struct {
	XMLName xml.Name `xml:"rsp"`
	Status  string   `xml:"stat,attr"`
	PhotoId string   `xml:"photoid"`
}
```


#### type FlickrUser

```go
type FlickrUser struct {
	Id       string
	Username struct {
		Content string `json:"_content"`
	} `json:"username"`
}
```


#### type OauthConfig

```go
type OauthConfig struct {
	Consumer oauth.Credentials
	Access   oauth.Credentials
}
```


#### type Photo

```go
type Photo struct {
	Id       string
	Owner    string
	Secret   string
	Title    string
	Ispublic int `json:"string"`
	Isfriend int `json:"string"`
	Isfamily int `json:"string"`
}
```


#### type PhotoInfo

```go
type PhotoInfo struct {
	Rotation       int
	Originalformat string
	Media          string
}
```


#### type PhotoSize

```go
type PhotoSize struct {
	Label  string
	Source string
}
```


#### type PhotosMap

```go
type PhotosMap map[string]Photo
```


#### type PhotosyncConfig

```go
type PhotosyncConfig struct {
	OauthConfig
	WatchDir string `json:"watch_dir"`
}
```
