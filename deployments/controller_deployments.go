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

package deployments

import (
	"errors"
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/asaskevich/govalidator"
)

// Errors
var (
	ErrIDNotUUIDv4 = errors.New("ID is not UUIDv4")
)

type DeploymentsModeler interface {
	CreateDeployment(constructor *DeploymentConstructor) (string, error)
	GetDeployment(deploymentID string) (*Deployment, error)
	GetDeploymentForDevice(deviceID string) (interface{}, error)
}

type DeploymentsController struct {
	views DeploymentsViews
	model DeploymentsModeler
}

func NewDeploymentsController(model DeploymentsModeler, views DeploymentsViews) *DeploymentsController {
	return &DeploymentsController{
		views: views,
		model: model,
	}
}

func (d *DeploymentsController) PostDeployment(w rest.ResponseWriter, r *rest.Request) {

	var constructor *DeploymentConstructor

	if err := r.DecodeJsonPayload(&constructor); err != nil {
		d.views.RenderError(w, err, http.StatusBadRequest)
		return
	}

	if err := constructor.Validate(); err != nil {
		d.views.RenderError(w, err, http.StatusBadRequest)
		return
	}

	id, err := d.model.CreateDeployment(constructor)
	if err != nil {
		d.views.RenderError(w, err, http.StatusInternalServerError)
		return
	}

	d.views.RenderSuccessPost(w, r, id)
}

func (d *DeploymentsController) GetDeployment(w rest.ResponseWriter, r *rest.Request) {

	id := r.PathParam("id")

	if !govalidator.IsUUIDv4(id) {
		d.views.RenderError(w, ErrIDNotUUIDv4, http.StatusBadRequest)
		return
	}

	deployment, err := d.model.GetDeployment(id)
	if err != nil {
		d.views.RenderError(w, err, http.StatusInternalServerError)
		return
	}

	if deployment == nil {
		d.views.RenderErrorNotFound(w)
	}

	d.views.RenderSuccessGet(w, deployment)
}

func (d *DeploymentsController) GetDeploymentForDevice(w rest.ResponseWriter, r *rest.Request) {

	id := r.PathParam("id")

	if !govalidator.IsUUIDv4(id) {
		d.views.RenderError(w, ErrIDNotUUIDv4, http.StatusBadRequest)
		return
	}

	deployment, err := d.model.GetDeploymentForDevice(id)
	if err != nil {
		d.views.RenderError(w, err, http.StatusInternalServerError)
		return
	}

	if deployment == nil {
		d.views.RenderNoUpdateForDevice(w)
	}

	d.views.RenderSuccessGet(w, deployment)
}
