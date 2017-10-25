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
	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/mendersoftware/go-lib-micro/rest_utils"

	"github.com/mendersoftware/deployments/resources/tenants/model"
)

type Controller struct {
	model model.Model
}

func NewController(model model.Model) *Controller {
	return &Controller{
		model: model,
	}
}

func (c *Controller) ProvisionTenantsHandler(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Context()
	l := log.FromContext(ctx)

	defer r.Body.Close()

	tenant, err := ParseNewTenantReq(r.Body)
	if err != nil {
		rest_utils.RestErrWithLog(w, r, l, err, http.StatusBadRequest)
		return
	}

	err = c.model.ProvisionTenant(ctx, tenant.TenantId)
	if err != nil {
		rest_utils.RestErrWithLogInternal(w, r, l, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
