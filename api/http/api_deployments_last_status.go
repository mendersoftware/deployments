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
	"net/http"

	"github.com/pkg/errors"

	"github.com/ant0ine/go-json-rest/rest"

	"github.com/mendersoftware/go-lib-micro/identity"
	"github.com/mendersoftware/go-lib-micro/requestlog"
	"github.com/mendersoftware/go-lib-micro/rest_utils"
)

// device deployments last status handler
func (d *DeploymentsApiHandlers) GetDeviceDeploymentLastStatus(
	w rest.ResponseWriter,
	r *rest.Request,
) {
	l := requestlog.GetRequestLogger(r)

	l.Debugf("starting")

	tenantId := r.PathParam("tenant")
	if tenantId == "" {
		l.Error("tenant id cannot be empty")
		rest_utils.RestErrWithLog(
			w,
			r,
			l,
			errors.New("empty tenant id"),
			http.StatusBadRequest,
		)
	}
	var devicesIds []string
	if err := r.DecodeJsonPayload(&devicesIds); err != nil {
		l.Errorf("error during DecodeJsonPayload: %s.", err.Error())
		rest_utils.RestErrWithLog(
			w,
			r,
			l,
			errors.Wrap(err, "cannot parse device ids array"),
			http.StatusBadRequest,
		)
		return
	}

	l.Debugf("querying %d devices ids", len(devicesIds))
	ctx := r.Context()
	ctx = identity.WithContext(
		ctx,
		&identity.Identity{
			Tenant: tenantId,
		},
	)
	lastDeployments, err := d.app.GetDeviceDeploymentLastStatus(ctx, devicesIds)
	switch err {
	default:
		d.view.RenderInternalError(w, r, err, l)
	case nil:
		l.Infof("outputting: %+v", lastDeployments)
		w.WriteHeader(http.StatusOK)
		_ = w.WriteJson(lastDeployments)
	}
}
