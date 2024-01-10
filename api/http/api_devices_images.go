// Copyright 2024 Northern.tech AS
//
//	Licensed under the Apache License, Version 2.0 (the "License");
//	you may not use this file except in compliance with the License.
//	You may obtain a copy of the License at
//
//	    http://www.apache.org/licenses/LICENSE-2.0
//
//	Unless required by applicable law or agreed to in writing, software
//	distributed under the License is distributed on an "AS IS" BASIS,
//	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	See the License for the specific language governing permissions and
//	limitations under the License.

package http

import (
	"net/http"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/asaskevich/govalidator"

	dconfig "github.com/mendersoftware/deployments/config"
	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/go-lib-micro/config"
	"github.com/mendersoftware/go-lib-micro/requestlog"
)

const MaxImagesForDevice = 100

func (d *DeploymentsApiHandlers) GetImagesForDevice(w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r)

	defer redactReleaseName(r)
	q := r.URL.Query()

	filter := &model.ReleaseOrImageFilter{
		Name:       q.Get(ParamName),
		DeviceType: q.Get(ParamDeviceType),
		Page:       1,
		PerPage:    MaxImagesForDevice,
	}

	list, _, err := d.app.ListImages(r.Context(), filter)
	if err != nil {
		d.view.RenderInternalError(w, r, err, l)
		return
	}

	d.view.RenderSuccessGet(w, list)
}

func (d *DeploymentsApiHandlers) GetImageForDevice(w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r)

	id := r.PathParam("id")
	if !govalidator.IsUUID(id) {
		d.view.RenderError(w, r, ErrIDNotUUID, http.StatusBadRequest, l)
		return
	}

	image, err := d.app.GetImage(r.Context(), id)
	if err != nil {
		d.view.RenderInternalError(w, r, err, l)
		return
	}

	if image == nil {
		d.view.RenderErrorNotFound(w, r, l)
		return
	}

	d.view.RenderSuccessGet(w, image)
}

func (d *DeploymentsApiHandlers) DownloadImageForDevice(w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r)

	id := r.PathParam("id")

	if !govalidator.IsUUID(id) {
		d.view.RenderError(w, r, ErrIDNotUUID, http.StatusBadRequest, l)
		return
	}

	expireSeconds := config.Config.GetInt(dconfig.SettingsStorageDownloadExpireSeconds)
	link, err := d.app.DownloadLink(r.Context(), id, time.Duration(expireSeconds)*time.Second)
	if err != nil {
		d.view.RenderInternalError(w, r, err, l)
		return
	}

	if link == nil {
		d.view.RenderErrorNotFound(w, r, l)
		return
	}

	d.view.RenderSuccessGet(w, link)
}
