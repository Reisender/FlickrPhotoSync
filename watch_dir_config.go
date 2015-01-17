package photosync

import (
	"bytes"
	"text/template"
)

type WatchDirConfig struct {
	Dir string
	Tags string
	tagsTmpl *template.Template
}

func (this *WatchDirConfig) CreateTemplates() {
	this.tagsTmpl = template.Must(template.New("tagsTmpl").Parse(this.Tags))
}

func (this *WatchDirConfig) GetTags(context DymanicValueContext) (string, error) {
	tags := new(bytes.Buffer)

	if err := this.tagsTmpl.Execute(tags,context); err != nil {
		return this.Tags, err
	}

	return tags.String(), nil
}
