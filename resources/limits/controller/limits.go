// Copyright 2017 Northern.tech AS
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
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/mendersoftware/go-lib-micro/requestlog"
	"github.com/pkg/errors"

	"github.com/mendersoftware/deployments/resources/limits"
)

var ()

type LimitsController struct {
	view  RESTView
	model LimitsModel
}

func NewLimitsController(model LimitsModel, view RESTView) *LimitsController {
	return &LimitsController{
		model: model,
		view:  view,
	}
}

type limitResponse struct {
	Limit uint64 `json:"limit"`
	Usage uint64 `json:"usage"`
}

func (s *LimitsController) GetLimit(w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r)

	name := r.PathParam("name")

	if !limits.IsValidLimit(name) {
		s.view.RenderError(w, r,
			errors.Errorf("unsupported limit %s", name),
			http.StatusBadRequest, l)
		return
	}

	limit, err := s.model.GetLimit(r.Context(), name)
	if err != nil {
		s.view.RenderInternalError(w, r, err, l)
		return
	}

	s.view.RenderSuccessGet(w, limitResponse{
		Limit: limit.Value,
		Usage: 0, // TODO fill this when ready
	})
}
