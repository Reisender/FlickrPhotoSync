package photosync

import (
	"fmt"
	"time"
	"os"
	"os/exec"
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
	WatchDir []string `json:"watch_dir"`
	FilenameTimeFormats []FilenameTimeFormat `json:"filename_time_formats"`
}

type FilenameTimeFormat struct {
	Format string
	Prefix []string
	Postfix []string
}

type exifToolOutput struct {
	SourceFile string
	ExifTool struct {
		Warning string
	}
	Ifd struct {
		Orientation string
		Make string
		Model string
		ModifyDate string
	} `json:"IFD0"`
}


// Load the consumer key and secret in from the config file
func LoadConfig(configPath *string,config *PhotosyncConfig) error {
	b, err := ioutil.ReadFile(*configPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, config)
}

func Sync(api *FlickrAPI, photos *PhotosMap, videos *PhotosMap, dryrun bool) (int, int, int, error) {
	existingCount := 0
	uploadedCount := 0
	errorCount := 0

	for _, dir := range api.config.WatchDir {
		// ensure the path exists
		if _, err := os.Stat(dir); os.IsNotExist(err) {
				fmt.Printf("no such file or directory: %s", dir)
				continue
		}

		err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
			if !f.IsDir() { // make sure we aren't operating on a directory

				ext := filepath.Ext(f.Name())
				extUpper := strings.ToUpper(ext)
				if extUpper == ".JPG" || extUpper == ".MOV" || extUpper == ".MP4" {
					fname := strings.Split(f.Name(),ext)
					key := strings.Join(fname[:len(fname)-1],ext)
					fmt.Println(path)

					var exists bool

					if extUpper == ".JPG" {
						_, exists = (*photos)[key]
					} else if extUpper == ".MOV" || extUpper == ".MP4" {
						_, exists = (*videos)[key]
					}

					if !exists {
						fmt.Print("|=====")

						if !dryrun {

							tmppath, done, er := FixExif(key, path, f, err)
							path = tmppath // update the path to the potentially new path
							if er != nil { errorCount++; return nil }
							res, err := api.Upload(path, f)
							if err != nil { errorCount++; return nil }

							defer done(api, res.PhotoId)

							fmt.Println("=====| 100%")
						} else {
							fmt.Println("=====| 100% --+ dry run +--")
						}

						uploadedCount++
					} else {
						existingCount++
					}
				}
			}

			return nil
		})

		if err != nil {
			return existingCount, uploadedCount, errorCount, err
		}
	}

	return existingCount, uploadedCount, errorCount, nil
}

func getTimeFromTitle(api *FlickrAPI, title string) (*time.Time, error) {
	for _, tf := range api.config.FilenameTimeFormats {
		var tmp = title

		// check prefix
		for _, p := range tf.Prefix {
			if tmp[:len(p)] == p {
				tmp = tmp[len(p):]
				break // we found our prefix
			}
		}

		// check postfix
		for _, p := range tf.Postfix {
			if tmp[len(tmp)-len(p):] == p {
				tmp = tmp[:len(tmp)-len(p)]
				break // we found our postfix
			}
		}
		fmt.Println("using title",tmp)

		// parse what's left
		t, err := time.Parse(tf.Format, tmp)
		if err == nil { return &t, nil }
	}
	return nil, Error{"no timestamp in title"}
}

//
// Checks the EXIF data for JPGs and returns the path to either the original or the fixed JPG file.
// The 2nd return value should be called when use of the JPG is complete.
// workingFile, done, err := FixExif(...)
// defer done()
//
func FixExif(title string, path string, f os.FileInfo, err error) (string, func(api *FlickrAPI, photoId string), error) {
	ext := filepath.Ext(f.Name())
	extUpper := strings.ToUpper(ext)
	var timeFromFilename *time.Time

	_setDateTaken := func(api *FlickrAPI, photoId string) {
		var err error
		timeFromFilename, err = getTimeFromTitle(api, title)
		if err != nil { timeFromFilename = nil }

		// check the file name
		if timeFromFilename != nil {
			fmt.Printf("set time from file name: %s\n",timeFromFilename.Format(flickrTimeLayout))
			api.SetDate(photoId, timeFromFilename.Format(flickrTimeLayout)) // eat the error as this is optional
		}
	}

	_setDateTakenMov := func(api *FlickrAPI, photoId string) {
		// they are done uploading the file so let's set it's date
		_setDateTaken(api, photoId)

		if timeFromFilename == nil {
			// fall back to the mod time
			// we do this for MOV's because there isn't exif data to use
			fmt.Printf("set time to: %s\n",f.ModTime().Format(flickrTimeLayout))
			api.SetDate(photoId, f.ModTime().Format(flickrTimeLayout)) // eat the error as this is optional
		}
	}


	if extUpper == ".JPG" {
		// check for valid exif data
		out, err := exec.Command("exiftool","-a","-u","-g1","-json",path).CombinedOutput()
		if err != nil { return "", _setDateTaken, err }

		exif := make([]exifToolOutput,1)
		if err := json.Unmarshal(out, &exif); err != nil { return "", _setDateTaken, err }

		if len(exif[0].ExifTool.Warning) > 0 {
			// we have an exif error
			if len(exif[0].Ifd.ModifyDate) > 0 {
				// we have a valid date already so just fix exif

				// create tmp file and copy
				tmpfile, err := ioutil.TempFile("",f.Name()+".")
				if err != nil { return "", _setDateTaken, err }

				tmpfilePath := tmpfile.Name() // ensure it's a new file for the sake of
				os.Remove(tmpfile.Name())

				_, errr := exec.Command("exiftool","-exif:all=","-tagsfromfile","@","-all:all","-unsafe","-o",tmpfilePath,path).CombinedOutput()
				if errr != nil { return "", _setDateTaken, errr }

				// return the callback function that should get called when use of this image is complete
				return tmpfilePath, func(api *FlickrAPI, photoId string) {os.Remove(tmpfilePath) }, errr
			}
		}

	} else if extUpper == ".MOV" || extUpper == ".MP4" {
		// always set to the file's modified date
		return path, _setDateTakenMov, nil
	}

	return path, _setDateTaken, nil
}
