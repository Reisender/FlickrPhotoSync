package photosync

import (
	"fmt"
	"log"
	"time"
	"os"
	"os/exec"
	"github.com/garyburd/go-oauth/oauth"
	"io/ioutil"
	"encoding/json"
	"path/filepath"
	"strings"
	"github.com/go-fsnotify/fsnotify"
)

// time formats
const FilenameTimeLayout = "20060102_150405"
const ExifTimeLayout = "2006:01:02 15:04:05"
const FlickrTimeLayout = "2006-01-02 15:04:05"

type Options struct {
	ConfigPath string
	Dryrun bool
	Daemon bool
	RetroTags bool
}

type PhotosMap map[string]Photo

type OauthConfig struct {
	Consumer oauth.Credentials
	Access oauth.Credentials
}

type PhotosyncConfig struct {
	OauthConfig
	Filenames []FilenameConfig `json:"filenames"`
	WatchDir []WatchDirConfig `json:"directories"`
	FilenameTimeFormats []FilenameTimeFormat `json:"filename_time_formats"`
}

type FilenameTimeFormat struct {
	Format string
	Prefix []string
	Postfix []string
}

type ExifToolOutput struct {
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

	if err := json.Unmarshal(b, config); err != nil {
		return err
	}

	// precompile the filename regexps
	for i := 0; i < len(config.Filenames); i++ {
		config.Filenames[i].Load()
	}

	// create the templates
	for i := 0; i < len(config.WatchDir); i++ {
		config.WatchDir[i].CreateTemplates()
	}

	return nil
}

func Sync(api *FlickrAPI, photos *PhotosMap, videos *PhotosMap, opt *Options) (int, int, int, error) {
	exCnt := 0
	upCnt := 0
	errCnt := 0

	for _, dir := range api.config.WatchDir {
		// ensure the path exists
		if _, err := os.Stat(dir.Dir); os.IsNotExist(err) {
				fmt.Printf("no such file or directory: %s", dir.Dir)
				continue
		}

		err := filepath.Walk(dir.Dir, func (path string, f os.FileInfo, err error) error {
			return processFile(api, &dir, path, f, photos, videos, &exCnt, &upCnt, &errCnt, opt)
		})

		if err != nil {
			return exCnt, upCnt, errCnt, err
		}
	}

	// start the daemon
	if opt.Daemon {
		log.Println("starting...")

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatal(err)
		}

		done := make(chan bool)
		dirCfgs := make(map[string]WatchDirConfig)

		go func() {
			for {
				select {
				case event := <-watcher.Events:
					//log.Println("event:", event)
					if event.Op & fsnotify.Create == fsnotify.Create {
						log.Println("created file:", event.Name)
						f, err := os.Stat(event.Name)
						if err != nil {
							log.Println("error getting file info for ", event.Name)
							continue
						}

						// get the dir config
						var cfg *WatchDirConfig
						for dir, dirCfg := range dirCfgs {
							if event.Name[:len(dir)] == dir {
								cfg = &dirCfg
							}
						}


						processFile(api, cfg, event.Name, f, photos, videos, &exCnt, &upCnt, &errCnt, opt)
					}
				case err := <-watcher.Errors:
					log.Println("error:", err)
				}
			}
		}()

		for _, dir := range api.config.WatchDir {
			// ensure the path exists
			if _, err := os.Stat(dir.Dir); os.IsNotExist(err) {
					fmt.Printf("no such file or directory: %s", dir.Dir)
					continue
			}

			dirCfgs[dir.Dir] = dir // add to map of configs for the event handler to use

			err = watcher.Add(dir.Dir)
			if err != nil {
				log.Fatal(err)
			}

		}

		<-done

	}

	return exCnt, upCnt, errCnt, nil
}

func processFile(api *FlickrAPI, dir *WatchDirConfig, path string, f os.FileInfo, photos, videos *PhotosMap, exCnt, upCnt, errCnt *int, opt *Options) error {
	if !f.IsDir() { // make sure we aren't operating on a directory

		ext := filepath.Ext(f.Name())
		extUpper := strings.ToUpper(ext)

		// rename file if needed
		// check again all filename configs
		for _, fncfg := range api.GetFilenamesConfig() {
			newPath, changed := fncfg.GetNewPath(path)
			if changed {
				fmt.Println("rename to:", newPath)

				if !opt.Dryrun {
					os.Rename(path,newPath)
				}

				break // found our match to bail
			}
		}

		if extUpper == ".JPG" || extUpper == ".MOV" || extUpper == ".MP4" {
			fname := strings.Split(f.Name(),ext)
			key := strings.Join(fname[:len(fname)-1],ext)
			fmt.Println(path)

			var exists bool
			var exPhoto Photo

			if extUpper == ".JPG" {
				exPhoto, exists = (*photos)[key]
			} else if extUpper == ".MOV" || extUpper == ".MP4" {
				exPhoto, exists = (*videos)[key]
			}

			if !exists {
				fmt.Print("|=====")

				tmppath, done, er := FixExif(key, path, f)

				if !opt.Dryrun {

					path = tmppath // update the path to the potentially new path
					if er != nil { *errCnt++; return nil }
					res, err := api.Upload(path, f)
					if err != nil { *errCnt++; return nil }

					defer done(api, res.PhotoId)

					// set the tags in config
					if len(dir.Tags) > 0 {
						api.AddTags(res.PhotoId, dir.Tags)
					}

					fmt.Println("=====| 100%")
				} else {
					fmt.Println("=====| 100% --+ dry run +--")
				}

				*upCnt++
			} else {
				// still apply retroactive tags
				if opt.RetroTags && len(dir.Tags) > 0 {
					fmt.Print("assign tags: ",dir.Tags)
					if !opt.Dryrun {
						api.AddTags(exPhoto.Id, dir.Tags)
					} else {
						fmt.Print(" --+ dry run +--")
					}
					fmt.Println()
				}

				*exCnt++
			}
		}
	}

	return nil
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
		//fmt.Println("using title",tmp)

		// parse what's left
		t, err := time.Parse(tf.Format, tmp)
		if err == nil { return &t, nil }
	}
	return nil, Error{"no timestamp in title"}
}


func GetExifData(path string) (*ExifToolOutput, error) {
	out, err := exec.Command("exiftool","-a","-u","-g1","-json",path).CombinedOutput()
	if err != nil { return nil, err }

	exif := make([]ExifToolOutput,1)
	if err := json.Unmarshal(out, &exif); err != nil { return nil, err }

	return &exif[0], nil
}

//
// Checks the EXIF data for JPGs and returns the path to either the original or the fixed JPG file.
// The 2nd return value should be called when use of the JPG is complete.
// workingFile, done, err := FixExif(...)
// defer done()
//
func FixExif(title string, path string, f os.FileInfo) (string, func(api *FlickrAPI, photoId string), error) {
	ext := filepath.Ext(f.Name())
	extUpper := strings.ToUpper(ext)
	var timeFromFilename *time.Time

	_setDateTaken := func(api *FlickrAPI, photoId string) {
		var err error
		timeFromFilename, err = getTimeFromTitle(api, title)
		if err != nil { timeFromFilename = nil }

		// check the file name
		if timeFromFilename != nil {
			fmt.Printf("set time from file name: %s\n",timeFromFilename.Format(FlickrTimeLayout))
			api.SetDate(photoId, timeFromFilename.Format(FlickrTimeLayout)) // eat the error as this is optional
		}
	}

	_setDateTakenMov := func(api *FlickrAPI, photoId string) {
		// they are done uploading the file so let's set it's date
		_setDateTaken(api, photoId)

		if timeFromFilename == nil {
			// fall back to the mod time
			// we do this for MOV's because there isn't exif data to use
			fmt.Printf("set time to: %s\n",f.ModTime().Format(FlickrTimeLayout))
			api.SetDate(photoId, f.ModTime().Format(FlickrTimeLayout)) // eat the error as this is optional
		}
	}


	if extUpper == ".JPG" {
		// check for valid exif data
		exif, err := GetExifData(path)
		if err != nil { return "", _setDateTaken, err }

		if len(exif.ExifTool.Warning) > 0 {
			// we have an exif error
			if len(exif.Ifd.ModifyDate) > 0 {
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
