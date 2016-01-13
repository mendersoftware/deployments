// Copyright 2016 Mender Software AS
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
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
