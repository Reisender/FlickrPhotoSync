package photosync

import (
	"time"
)

// Context for dynamic values in the config
type DymanicValueContext struct {
	Foo string
	File FilenameConfig
	Exif ExifToolOutput
}

func (this DymanicValueContext) ExifDate() (string, error) {
	layout := "20060102_150405"
	t, err := time.Parse(ExifTimeLayout, this.Exif.Ifd.ModifyDate)
	if err != nil { return "", nil }

	return t.Format(layout), nil
}
