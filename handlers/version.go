package handlers

import (
	"github.com/ant0ine/go-json-rest/rest"
)

type VersionHandlerI interface {
	Get(w rest.ResponseWriter, r *rest.Request)
}

type Version struct {
	Version string `json:"version,omitempty"`
	Build   string `json:"build,omitempty"`
}

func NewVersion(version, build string) *Version {
	return &Version{
		Version: version,
		Build:   build,
	}
}

func (v *Version) Get(w rest.ResponseWriter, r *rest.Request) {
	w.WriteJson(v)
}
