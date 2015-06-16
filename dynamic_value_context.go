package photosync

import (
	"time"
	"strings"
	"path/filepath"
)

// Context for dynamic values in the config
type DymanicValueContext struct {
	path string
	dir string
	ext string
	title string
	fileCfg FilenameConfig
	dirCfg WatchDirConfig
	exif ExifToolOutput
}

func (this *DymanicValueContext) ExifDate() (string, error) {
	layout := "20060102_150405"
	t, err := time.Parse(ExifTimeLayout, this.exif.Ifd.ModifyDate)
	if err != nil { return "", nil }

	return t.Format(layout), nil
}

func (this *DymanicValueContext) Folders() (string, error) {
	rel, err := filepath.Rel(this.dirCfg.Dir, this.dir)
	if err != nil { return "", err }

	return strings.Join( strings.Split(rel,"/"), " "), nil
}
