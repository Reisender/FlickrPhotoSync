package photosync

import (
	"bytes"
	"path/filepath"
	"regexp"
	"text/template"
)


type FilenameConfig struct {
	Match string `json:"match"`
	matchRegexp regexp.Regexp
	Append string
	Prepend string
	appendTmpl *template.Template
	prependTmpl *template.Template
}

func (this *FilenameConfig) Load() error {
	rgxp, err := regexp.Compile(this.Match)
	if err != nil { return err }
	this.matchRegexp = *rgxp
	this.prependTmpl = template.Must(template.New("prependTmpl").Parse(this.Prepend))
	this.appendTmpl = template.Must(template.New("appendTmpl").Parse(this.Append))
	return nil
}

func (this *FilenameConfig) getModifiedTitle(title string, context DymanicValueContext) (string, error) {
	tp := new(bytes.Buffer)
	ta := new(bytes.Buffer)

	if err := this.prependTmpl.Execute(tp,context); err != nil {
		return title, err
	}
	if err := this.appendTmpl.Execute(ta,context); err != nil {
		return title, err
	}

	return tp.String() + title + ta.String(), nil
}


func (this *FilenameConfig) GetNewPath(path string) (string, string, bool) {
	// pull out the filename and ext
	dir, fname := filepath.Split(path)
	ext := filepath.Ext(fname)
	title := fname[:len(fname)-len(ext)]

	exif, err := GetExifData(path)
	if err != nil {
		return path, title, false
	}

	if this.matchRegexp.MatchString(fname) {
		//t, err := time.Parse( ExifTimeLayout, exif.Ifd.ModifyDate )
		if err != nil {
			return path, title, false
		}
		context := DymanicValueContext{
			*this,
			*exif,
		}
		newTitle, _ := this.getModifiedTitle(title,context)
		return dir+newTitle+ext, newTitle, true
	} else {
		return path, title, false
	}
}
