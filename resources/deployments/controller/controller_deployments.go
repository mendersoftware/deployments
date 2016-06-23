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

package controller

import (
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/asaskevich/govalidator"
	"github.com/mendersoftware/deployments/resources/deployments"
	"github.com/pkg/errors"
)

// Errors
var (
	ErrIDNotUUIDv4 = errors.New("ID is not UUIDv4")
)

type DeploymentsController struct {
	view  RESTView
	model DeploymentsModel
}

func NewDeploymentsController(model DeploymentsModel, view RESTView) *DeploymentsController {
	return &DeploymentsController{
		view:  view,
		model: model,
	}
}

func (d *DeploymentsController) PostDeployment(w rest.ResponseWriter, r *rest.Request) {

	constructor, err := d.getDeploymentConstructorFromBody(r)
	if err != nil {
		d.view.RenderError(w, errors.Wrap(err, "Validating request body"), http.StatusBadRequest)
		return
	}

	id, err := d.model.CreateDeployment(constructor)
	if err != nil {
		d.view.RenderError(w, err, http.StatusInternalServerError)
		return
	}

	d.view.RenderSuccessPost(w, r, id)
}

func (d *DeploymentsController) getDeploymentConstructorFromBody(r *rest.Request) (*deployments.DeploymentConstructor, error) {

	var constructor *deployments.DeploymentConstructor
	if err := r.DecodeJsonPayload(&constructor); err != nil {
		return nil, err
	}

	if err := constructor.Validate(); err != nil {
		return nil, err
	}

	return constructor, nil
}

func (d *DeploymentsController) GetDeployment(w rest.ResponseWriter, r *rest.Request) {

	id := r.PathParam("id")

	if !govalidator.IsUUIDv4(id) {
		d.view.RenderError(w, ErrIDNotUUIDv4, http.StatusBadRequest)
		return
	}

	deployment, err := d.model.GetDeployment(id)
	if err != nil {
		d.view.RenderError(w, err, http.StatusInternalServerError)
		return
	}

	if deployment == nil {
		d.view.RenderErrorNotFound(w)
		return
	}

	d.view.RenderSuccessGet(w, deployment)
}

func (d *DeploymentsController) GetDeploymentForDevice(w rest.ResponseWriter, r *rest.Request) {

	id := r.PathParam("id")

	if !govalidator.IsUUIDv4(id) {
		d.view.RenderError(w, ErrIDNotUUIDv4, http.StatusBadRequest)
		return
	}

	deployment, err := d.model.GetDeploymentForDevice(id)
	if err != nil {
		d.view.RenderError(w, err, http.StatusInternalServerError)
		return
	}

	if deployment == nil {
		d.view.RenderNoUpdateForDevice(w)
		return
	}

	d.view.RenderSuccessGet(w, deployment)
}
