package photosync

import (
	"bytes"
	"text/template"
)

type WatchDirConfig struct {
	Dir      string
	Tags     string
	tagsTmpl *template.Template
	Albums   []string
}

func (this *WatchDirConfig) CreateTemplates() {
	this.tagsTmpl = template.Must(template.New("tagsTmpl").Parse(this.Tags))
}

func (this *WatchDirConfig) GetTags(context *DynamicValueContext) (string, error) {
	tags := new(bytes.Buffer)

	if err := this.tagsTmpl.Execute(tags, *context); err != nil {
		return this.Tags, err
	}

	return tags.String(), nil
}

func (this *WatchDirConfig) GetAlbums(context *DynamicValueContext) []string {
	return this.Albums
}
