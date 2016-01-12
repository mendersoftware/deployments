package handlers

import (
	"github.com/ant0ine/go-json-rest/rest"
)

type VersionHandlerI interface {
	Get(w rest.ResponseWriter, r *rest.Request)
}

type Version struct {
	version string
}

func NewVersion(version string) *Version {
	return &Version{
		version: version,
	}
}

func (v *Version) Get(w rest.ResponseWriter, r *rest.Request) {
	w.WriteJson(v.version)
}
