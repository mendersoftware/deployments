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
package http

import (
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/pkg/errors"

	"github.com/mendersoftware/go-lib-micro/requestlog"

	"github.com/mendersoftware/deployments/app"
	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/deployments/store"
)

type DeploymentsApiHandlers struct {
	view  RESTView
	store store.DataStore
	app   app.App
}

func NewDeploymentsApiHandlers(store store.DataStore, view RESTView, app app.App) *DeploymentsApiHandlers {
	return &DeploymentsApiHandlers{
		store: store,
		view:  view,
		app:   app,
	}
}

func (d *DeploymentsApiHandlers) GetReleases(w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r)

	var filt *model.ReleaseFilter

	name := r.URL.Query().Get("name")

	if name != "" {
		filt = &model.ReleaseFilter{
			Name: name,
		}
	}

	releases, err := d.store.GetReleases(r.Context(), filt)
	if err != nil {
		d.view.RenderInternalError(w, r, err, l)
		return
	}

	d.view.RenderSuccessGet(w, releases)
}

type limitResponse struct {
	Limit uint64 `json:"limit"`
	Usage uint64 `json:"usage"`
}

func (d *DeploymentsApiHandlers) GetLimit(w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r)

	name := r.PathParam("name")

	if !model.IsValidLimit(name) {
		d.view.RenderError(w, r,
			errors.Errorf("unsupported limit %s", name),
			http.StatusBadRequest, l)
		return
	}

	limit, err := d.app.GetLimit(r.Context(), name)
	if err != nil {
		d.view.RenderInternalError(w, r, err, l)
		return
	}

	d.view.RenderSuccessGet(w, limitResponse{
		Limit: limit.Value,
		Usage: 0, // TODO fill this when ready
	})
}
