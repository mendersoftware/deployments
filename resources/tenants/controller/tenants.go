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
	"fmt"
	"mime"
	"mime/multipart"
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
	deploymentsController "github.com/mendersoftware/deployments/resources/deployments/controller"
	deploymentsModel "github.com/mendersoftware/deployments/resources/deployments/model"
	"github.com/pkg/errors"

	imageController "github.com/mendersoftware/deployments/resources/images/controller"

	"github.com/mendersoftware/deployments/app"
	"github.com/mendersoftware/go-lib-micro/identity"
	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/mendersoftware/go-lib-micro/rest_utils"
)

type Controller struct {
	model      app.App
	depsModel  deploymentsModel.DeploymentsModel
	imageModel imageController.ImagesModel
	imageCtrl  imageController.SoftwareImagesController
	restView   imageController.RESTView
}

func NewController(model app.App, depsModel *deploymentsModel.DeploymentsModel,
	imgModel imageController.ImagesModel, imgCtrl *imageController.SoftwareImagesController,
	restView imageController.RESTView) *Controller {

	return &Controller{
		model:      model,
		depsModel:  *depsModel,
		imageModel: imgModel,
		imageCtrl:  *imgCtrl,
		restView:   restView,
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

func (c *Controller) DeploymentsPerTenantHandler(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Context()
	l := log.FromContext(ctx)
	defer r.Body.Close()

	tenantID := r.PathParam("tenant")

	if tenantID == "" {
		rest_utils.RestErrWithLog(w, r, l, nil, http.StatusBadRequest)
	}

	query, err := deploymentsController.ParseLookupQuery(r.URL.Query())

	if err != nil {
		rest_utils.RestErrWithLog(w, r, l, err, http.StatusBadRequest)
	}

	ident := &identity.Identity{Tenant: tenantID}
	ctx = identity.WithContext(r.Context(), ident)

	if deps, err := c.depsModel.LookupDeployment(ctx, query); err != nil {
		rest_utils.RestErrWithLog(w, r, l, err, http.StatusBadRequest)
	} else {
		w.WriteJson(deps)
	}
}

func (c *Controller) NewImageForTenantHandler(w rest.ResponseWriter, r *rest.Request) {
	l := log.FromContext(r.Context())

	tenantID := r.PathParam("tenant")

	if tenantID == "" {
		rest_utils.RestErrWithLog(w, r, l, fmt.Errorf("missing tenant id in path"), http.StatusBadRequest)
		return
	}

	ident := &identity.Identity{Tenant: tenantID}
	ctx := identity.WithContext(r.Context(), ident)

	// parse content type and params according to RFC 1521
	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		rest_utils.RestErrWithLog(w, r, l, err, http.StatusBadRequest)
		return
	}

	mr := multipart.NewReader(r.Body, params["boundary"])
	// parse multipart message

	multipartUploadMsg, err := c.imageCtrl.ParseMultipart(mr, imageController.DefaultMaxMetaSize)
	if err != nil {
		c.restView.RenderError(w, r, err, http.StatusBadRequest, l)
		return
	}

	imgID, err := c.imageModel.CreateImage(ctx, multipartUploadMsg)
	cause := errors.Cause(err)
	switch cause {
	default:
		c.restView.RenderInternalError(w, r, err, l)
	case nil:
		c.restView.RenderSuccessPost(w, r, imgID)
	case imageController.ErrModelArtifactNotUnique:
		l.Error(err.Error())
		c.restView.RenderError(w, r, cause, http.StatusUnprocessableEntity, l)
	case imageController.ErrModelMissingInputMetadata, imageController.ErrModelMissingInputArtifact,
		imageController.ErrModelInvalidMetadata, imageController.ErrModelMultipartUploadMsgMalformed,
		imageController.ErrModelArtifactFileTooLarge, imageController.ErrModelParsingArtifactFailed:
		l.Error(err.Error())
		c.restView.RenderError(w, r, cause, http.StatusBadRequest, l)
	}
}
