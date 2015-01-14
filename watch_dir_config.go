package photosync

import (
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
