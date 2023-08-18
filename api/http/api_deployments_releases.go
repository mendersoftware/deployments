// Copyright 2023 Northern.tech AS
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
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/pkg/errors"

	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/mendersoftware/go-lib-micro/requestlog"
	"github.com/mendersoftware/go-lib-micro/rest_utils"

	"github.com/mendersoftware/deployments/app"
	"github.com/mendersoftware/deployments/model"
)

func redactReleaseName(r *rest.Request) {
	q := r.URL.Query()
	if q.Get(ParamName) != "" {
		q.Set(ParamName, Redacted)
		r.URL.RawQuery = q.Encode()
	}
}

func (d *DeploymentsApiHandlers) GetReleases(w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r)

	defer redactReleaseName(r)
	filter := getReleaseOrImageFilter(r, false)
	releases, _, err := d.store.GetReleases(r.Context(), filter)
	if err != nil {
		d.view.RenderInternalError(w, r, err, l)
		return
	}

	d.view.RenderSuccessGet(w, releases)
}

func (d *DeploymentsApiHandlers) ListReleases(w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r)

	defer redactReleaseName(r)
	filter := getReleaseOrImageFilter(r, true)
	releases, totalCount, err := d.store.GetReleases(r.Context(), filter)
	if err != nil {
		d.view.RenderInternalError(w, r, err, l)
		return
	}

	hasNext := totalCount > int(filter.Page*filter.PerPage)
	links := rest_utils.MakePageLinkHdrs(r, uint64(filter.Page), uint64(filter.PerPage), hasNext)
	for _, l := range links {
		w.Header().Add("Link", l)
	}
	w.Header().Add(hdrTotalCount, strconv.Itoa(totalCount))

	d.view.RenderSuccessGet(w, releases)
}

func (d *DeploymentsApiHandlers) PatchRelease(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Context()
	l := log.FromContext(ctx)

	releaseName := r.PathParam(ParamName)
	if releaseName == "" {
		err := errors.New("path parameter 'release_name' cannot be empty")
		rest_utils.RestErrWithLog(w, r, l, err, http.StatusNotFound)
		return
	}

	var release model.ReleasePatch
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&release); err != nil {
		rest_utils.RestErrWithLog(w, r, l,
			errors.WithMessage(err,
				"malformed JSON in request body"),
			http.StatusBadRequest)
		return
	}
	if err := release.Validate(); err != nil {
		rest_utils.RestErrWithLog(w, r, l,
			errors.WithMessage(err,
				"invalid request body"),
			http.StatusBadRequest)
		return
	}

	err := d.app.UpdateRelease(ctx, releaseName, release)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, app.ErrReleaseNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, model.ErrTooManyUniqueTags) {
			status = http.StatusConflict
		}
		rest_utils.RestErrWithLog(w, r, l, err, status)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (d *DeploymentsApiHandlers) PutReleaseTags(
	w rest.ResponseWriter,
	r *rest.Request,
) {
	ctx := r.Context()
	l := log.FromContext(ctx)

	releaseName := r.PathParam(ParamName)
	if releaseName == "" {
		err := errors.New("path parameter 'release_name' cannot be empty")
		rest_utils.RestErrWithLog(w, r, l, err, http.StatusNotFound)
		return
	}

	var tags model.Tags
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&tags); err != nil {
		rest_utils.RestErrWithLog(w, r, l,
			errors.WithMessage(err,
				"malformed JSON in request body"),
			http.StatusBadRequest)
		return
	}
	if err := tags.Validate(); err != nil {
		rest_utils.RestErrWithLog(w, r, l,
			errors.WithMessage(err,
				"invalid request body"),
			http.StatusBadRequest)
		return
	}

	err := d.app.ReplaceReleaseTags(ctx, releaseName, tags)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, app.ErrReleaseNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, model.ErrTooManyUniqueTags) {
			status = http.StatusConflict
		}
		rest_utils.RestErrWithLog(w, r, l, err, status)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (d *DeploymentsApiHandlers) GetReleaseTagKeys(
	w rest.ResponseWriter,
	r *rest.Request,
) {
	ctx := r.Context()
	l := log.FromContext(ctx)

	tags, err := d.app.ListReleaseTags(ctx)
	if err != nil {
		rest_utils.RestErrWithLog(w, r, l, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	err = w.WriteJson(tags)
	if err != nil {
		l.Errorf("failed to serialize JSON response: %s", err.Error())
	}
}

func (d *DeploymentsApiHandlers) GetReleasesUpdateTypes(
	w rest.ResponseWriter,
	r *rest.Request,
) {
	ctx := r.Context()
	l := log.FromContext(ctx)

	updateTypes, err := d.store.GetUpdateTypes(ctx)
	if err != nil {
		rest_utils.RestErrWithLog(w, r, l, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	err = w.WriteJson(updateTypes)
	if err != nil {
		l.Errorf("failed to serialize JSON response: %s", err.Error())
	}
}
