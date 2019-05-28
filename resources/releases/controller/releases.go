// Copyright 2019 Northern.tech AS
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
package controller

import (
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/mendersoftware/go-lib-micro/requestlog"

	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/deployments/store"
)

type ReleasesController struct {
	view  RESTView
	store store.DataStore
}

func NewReleasesController(store store.DataStore, view RESTView) *ReleasesController {
	return &ReleasesController{
		store: store,
		view:  view,
	}
}

func (c *ReleasesController) GetReleases(w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r)

	var filt *model.ReleaseFilter

	name := r.URL.Query().Get("name")

	if name != "" {
		filt = &model.ReleaseFilter{
			Name: name,
		}
	}

	releases, err := c.store.GetReleases(r.Context(), filt)
	if err != nil {
		c.view.RenderInternalError(w, r, err, l)
		return
	}

	c.view.RenderSuccessGet(w, releases)
}
