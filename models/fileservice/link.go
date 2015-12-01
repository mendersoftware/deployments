package fileservice

import (
	"time"
)

type Link struct {
	Uri    string    `json:"uri"`
	Expire time.Time `json:"expire,omitempty"`
}

func NewLink(uri string, expire time.Time) *Link {
	return &Link{
		Uri:    uri,
		Expire: expire,
	}
}
