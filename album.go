package photosync

import (
)

type AlbumsMap map[string]Album

type Album struct {
	Id string
	Title struct {
		Content string `json:"_content"`
	} `json:"title"`
	PhotoIds []string
}

func (this Album) GetTitle() string {
	return this.Title.Content
}

func (this *Album) Prepend(photoId string) {
	this.PhotoIds = append([]string{photoId}, this.PhotoIds...)
}

func (this *Album) Append(photoId string) {
	this.PhotoIds = append(this.PhotoIds,photoId)
}

func (this *Album) Reverse() {
	var newOrder []string
	for i := len(this.PhotoIds)-1; i >= 0; i-- {
		newOrder = append(newOrder, this.PhotoIds[i])
	}
	this.PhotoIds = newOrder
}
