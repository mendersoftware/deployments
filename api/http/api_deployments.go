// Copyright 2020 Northern.tech AS
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
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/asaskevich/govalidator"
	"github.com/pkg/errors"

	"github.com/mendersoftware/go-lib-micro/identity"
	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/mendersoftware/go-lib-micro/requestlog"
	"github.com/mendersoftware/go-lib-micro/rest_utils"

	"github.com/mendersoftware/deployments/app"
	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/deployments/store"
)

const (
	// 15 minutes
	DefaultDownloadLinkExpire = 15 * time.Minute

	DefaultMaxMetaSize = 1024 * 1024 * 10
	hdrTotalCount      = "X-Total-Count"
)

// storage keys
const (
	GetDeploymentForDeviceQueryArtifact   = "artifact_name"
	GetDeploymentForDeviceQueryDeviceType = "device_type"
)

// JWT token
const (
	HTTPHeaderAuthorization       = "Authorization"
	HTTPHeaderAuthorizationBearer = "Bearer"
)

const (
	defaultTimeout = time.Second * 10
)

// Errors
var (
	ErrIDNotUUIDv4                          = errors.New("ID is not UUIDv4")
	ErrArtifactUsedInActiveDeployment       = errors.New("Artifact is used in active deployment")
	ErrInvalidExpireParam                   = errors.New("Invalid expire parameter")
	ErrArtifactNameMissing                  = errors.New("request does not contain the name of the artifact")
	ErrArtifactTypeMissing                  = errors.New("request does not contain the type of artifact")
	ErrArtifactDeviceTypesCompatibleMissing = errors.New("request does not contain the list of compatible device types")
	ErrArtifactFileMissing                  = errors.New("request does not contain the artifact file")

	ErrInternal                   = errors.New("Internal error")
	ErrDeploymentAlreadyFinished  = errors.New("Deployment already finished")
	ErrUnexpectedDeploymentStatus = errors.New("Unexpected deployment status")
	ErrMissingIdentity            = errors.New("Missing identity data")
	ErrMissingSize                = errors.New("missing size form-data")
	ErrMissingGroupName           = errors.New("Missing group name")
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

func (u *DeploymentsApiHandlers) AliveHandler(w rest.ResponseWriter, r *rest.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (u *DeploymentsApiHandlers) HealthHandler(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Context()
	l := log.FromContext(ctx)
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	err := u.app.HealthCheck(ctx)
	if err != nil {
		rest_utils.RestErrWithLog(w, r, l, err, http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

// images

func (d *DeploymentsApiHandlers) GetImage(w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r)

	id := r.PathParam("id")

	if !govalidator.IsUUID(id) {
		d.view.RenderError(w, r, ErrIDNotUUIDv4, http.StatusBadRequest, l)
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

func (d *DeploymentsApiHandlers) ListImages(w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r)

	list, err := d.app.ListImages(r.Context(), r.PathParams)
	if err != nil {
		d.view.RenderInternalError(w, r, err, l)
		return
	}

	d.view.RenderSuccessGet(w, list)
}

func (d *DeploymentsApiHandlers) DownloadLink(w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r)

	id := r.PathParam("id")

	if !govalidator.IsUUID(id) {
		d.view.RenderError(w, r, ErrIDNotUUIDv4, http.StatusBadRequest, l)
		return
	}

	link, err := d.app.DownloadLink(r.Context(), id, DefaultDownloadLinkExpire)
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

func (d *DeploymentsApiHandlers) DeleteImage(w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r)

	id := r.PathParam("id")

	if !govalidator.IsUUID(id) {
		d.view.RenderError(w, r, ErrIDNotUUIDv4, http.StatusBadRequest, l)
		return
	}

	if err := d.app.DeleteImage(r.Context(), id); err != nil {
		switch err {
		default:
			d.view.RenderInternalError(w, r, err, l)
		case app.ErrImageMetaNotFound:
			d.view.RenderErrorNotFound(w, r, l)
		case app.ErrModelImageInActiveDeployment:
			d.view.RenderError(w, r, ErrArtifactUsedInActiveDeployment, http.StatusConflict, l)
		}
		return
	}

	d.view.RenderSuccessDelete(w)
}

func (d *DeploymentsApiHandlers) EditImage(w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r)

	id := r.PathParam("id")

	if !govalidator.IsUUID(id) {
		d.view.RenderError(w, r, ErrIDNotUUIDv4, http.StatusBadRequest, l)
		return
	}

	constructor, err := getImageMetaFromBody(r)
	if err != nil {
		d.view.RenderError(w, r, errors.Wrap(err, "Validating request body"), http.StatusBadRequest, l)
		return
	}

	found, err := d.app.EditImage(r.Context(), id, constructor)
	if err != nil {
		if err == app.ErrModelImageUsedInAnyDeployment {
			d.view.RenderError(w, r, err, http.StatusUnprocessableEntity, l)
			return
		}
		d.view.RenderInternalError(w, r, err, l)
		return
	}

	if !found {
		d.view.RenderErrorNotFound(w, r, l)
		return
	}

	d.view.RenderSuccessPut(w)
}

func getImageMetaFromBody(r *rest.Request) (*model.ImageMeta, error) {

	var constructor *model.ImageMeta

	if err := r.DecodeJsonPayload(&constructor); err != nil {
		return nil, err
	}

	if err := constructor.Validate(); err != nil {
		return nil, err
	}

	return constructor, nil
}

// NewImage is the Multipart Image/Meta upload handler.
// Request should be of type "multipart/form-data". The parts are
// key/valyue pairs of metadata information except the last one,
// which must contain the artifact file.
func (d *DeploymentsApiHandlers) NewImage(w rest.ResponseWriter, r *rest.Request) {
	d.newImageWithContext(r.Context(), w, r)
}

func (d *DeploymentsApiHandlers) NewImageForTenantHandler(w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r)

	tenantID := r.PathParam("tenant")

	if tenantID == "" {
		rest_utils.RestErrWithLog(w, r, l, fmt.Errorf("missing tenant id in path"), http.StatusBadRequest)
		return
	}

	var ctx context.Context
	if tenantID != "default" {
		ident := &identity.Identity{Tenant: tenantID}
		ctx = identity.WithContext(r.Context(), ident)
	} else {
		ctx = r.Context()
	}

	d.newImageWithContext(ctx, w, r)
}

func (d *DeploymentsApiHandlers) newImageWithContext(ctx context.Context, w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r)

	formReader, err := r.MultipartReader()
	if err != nil {
		d.view.RenderError(w, r, err, http.StatusBadRequest, l)
		return
	}

	// parse multipart message
	multipartUploadMsg, err := d.ParseMultipart(formReader)
	defer r.MultipartForm.RemoveAll()

	if err != nil {
		d.view.RenderError(w, r, err, http.StatusBadRequest, l)
		return
	}

	imgID, err := d.app.CreateImage(ctx, multipartUploadMsg)
	if err == nil {
		d.view.RenderSuccessPost(w, r, imgID)
		return
	}
	l.Error(err.Error())
	if cErr, ok := err.(*model.ConflictError); ok {
		d.view.RenderError(w, r, cErr, http.StatusConflict, l)
		return
	}
	cause := errors.Cause(err)
	switch cause {
	default:
		d.view.RenderInternalError(w, r, err, l)
		return
	case app.ErrModelArtifactNotUnique:
		l.Error(err.Error())
		d.view.RenderError(w, r, cause, http.StatusUnprocessableEntity, l)
		return
	case app.ErrModelParsingArtifactFailed:
		l.Error(err.Error())
		d.view.RenderError(w, r, formatArtifactUploadError(err), http.StatusBadRequest, l)
		return
	case app.ErrModelMissingInputMetadata, app.ErrModelMissingInputArtifact,
		app.ErrModelInvalidMetadata, app.ErrModelMultipartUploadMsgMalformed,
		app.ErrModelArtifactFileTooLarge:
		l.Error(err.Error())
		d.view.RenderError(w, r, cause, http.StatusBadRequest, l)
		return
	}
}

func formatArtifactUploadError(err error) error {
	// remove generic message
	errMsg := strings.TrimSuffix(err.Error(), ": "+app.ErrModelParsingArtifactFailed.Error())

	// handle specific cases

	if strings.Contains(errMsg, "invalid checksum") {
		return errors.New(errMsg[strings.Index(errMsg, "invalid checksum"):])
	}

	if strings.Contains(errMsg, "unsupported version") {
		return errors.New(errMsg[strings.Index(errMsg, "unsupported version"):] +
			"; supported versions are: 1, 2")
	}

	return errors.New(errMsg)
}

// GenerateImage s the multipart Raw Data/Meta upload handler.
// Request should be of type "multipart/form-data". The parts are
// key/valyue pairs of metadata information except the last one,
// which must contain the file containing the raw data to be processed
// into an artifact.
func (d *DeploymentsApiHandlers) GenerateImage(w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r)

	formReader, err := r.MultipartReader()
	if err != nil {
		d.view.RenderError(w, r, err, http.StatusBadRequest, l)
		return
	}

	// parse multipart message
	multipartMsg, err := d.ParseGenerateImageMultipart(formReader)
	if err != nil {
		d.view.RenderError(w, r, err, http.StatusBadRequest, l)
		return
	}

	tokenFields := strings.Fields(r.Header.Get("Authorization"))
	if len(tokenFields) == 2 && strings.EqualFold(tokenFields[0], "Bearer") {
		multipartMsg.Token = tokenFields[1]
	}

	imgID, err := d.app.GenerateImage(r.Context(), multipartMsg)
	cause := errors.Cause(err)
	switch cause {
	default:
		d.view.RenderInternalError(w, r, err, l)
	case nil:
		d.view.RenderSuccessPost(w, r, imgID)
	case app.ErrModelArtifactNotUnique:
		l.Error(err.Error())
		d.view.RenderError(w, r, cause, http.StatusUnprocessableEntity, l)
	case app.ErrModelParsingArtifactFailed:
		l.Error(err.Error())
		d.view.RenderError(w, r, formatArtifactUploadError(err), http.StatusBadRequest, l)
	case app.ErrModelMissingInputMetadata, app.ErrModelMissingInputArtifact,
		app.ErrModelInvalidMetadata, app.ErrModelMultipartUploadMsgMalformed,
		app.ErrModelArtifactFileTooLarge:
		l.Error(err.Error())
		d.view.RenderError(w, r, cause, http.StatusBadRequest, l)
	}
}

// ParseMultipart parses multipart/form-data message.
func (d *DeploymentsApiHandlers) ParseMultipart(r *multipart.Reader) (*model.MultipartUploadMsg, error) {

	uploadMsg := &model.MultipartUploadMsg{
		MetaConstructor: &model.ImageMeta{},
		ArtifactSize:    app.MaxImageSize,
	}
	// Parse the multipart form sequentially. To remain backward compatible
	// all form names that are not part of the API are ignored.
	for {
		part, err := r.NextPart()
		if err != nil {
			if err == io.EOF {
				// The whole message has been consumed without
				// the "artifact" form part.
				return nil, ErrArtifactFileMissing
			}
			return nil, err
		}
		switch strings.ToLower(part.FormName()) {
		case "description":
			// Add description to the metadata
			dscr, err := ioutil.ReadAll(part)
			if err != nil {
				return nil, err
			}
			uploadMsg.MetaConstructor.Description = string(dscr)

		case "size":
			// Add size limit to the metadata
			sz, err := ioutil.ReadAll(part)
			if err != nil {
				return nil, err
			}
			size, err := strconv.ParseInt(string(sz), 10, 64)
			if err != nil {
				return nil, err
			}
			// Add one since this will impose the upper limit on the
			// artifact size.
			if size > app.MaxImageSize {
				return nil, app.ErrModelArtifactFileTooLarge
			}
			uploadMsg.ArtifactSize = size

		case "artifact_id":
			// Add artifact id to the metadata (must be a valid UUID).
			b, err := ioutil.ReadAll(part)
			if err != nil {
				return nil, err
			}
			id := string(b)
			if !govalidator.IsUUID(id) {
				return nil, errors.New(
					"artifact_id is not a valid UUID",
				)
			}
			uploadMsg.ArtifactID = id

		case "artifact":
			// Assign the form-data payload to the artifact reader
			// and return. The content is consumed elsewhere.
			uploadMsg.ArtifactReader = part
			return uploadMsg, nil

		default:
			// Ignore all non-API sections.
			continue
		}
	}
}

// ParseGenerateImageMultipart parses multipart/form-data message.
func (d *DeploymentsApiHandlers) ParseGenerateImageMultipart(r *multipart.Reader) (*model.MultipartGenerateImageMsg, error) {
	msg := &model.MultipartGenerateImageMsg{}

ParseLoop:
	for {
		part, err := r.NextPart()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch strings.ToLower(part.FormName()) {
		case "args":
			b, err := ioutil.ReadAll(part)
			if err != nil {
				return nil, errors.Wrap(err,
					"failed to read form value 'args'",
				)
			}
			msg.Args = string(b)

		case "description":
			b, err := ioutil.ReadAll(part)
			if err != nil {
				return nil, errors.Wrap(err,
					"failed to read form value 'description'",
				)
			}
			msg.Description = string(b)

		case "device_types_compatible":
			b, err := ioutil.ReadAll(part)
			if err != nil {
				return nil, errors.Wrap(err,
					"failed to read form value 'device_types_compatible'",
				)
			}
			msg.DeviceTypesCompatible = strings.Split(string(b), ",")

		case "file":
			msg.FileReader = part
			break ParseLoop

		case "name":
			b, err := ioutil.ReadAll(part)
			if err != nil {
				return nil, errors.Wrap(err,
					"failed to read form value 'name'",
				)
			}
			msg.Name = string(b)

		case "type":
			b, err := ioutil.ReadAll(part)
			if err != nil {
				return nil, errors.Wrap(err,
					"failed to read form value 'type'",
				)
			}
			msg.Type = string(b)

		default:
			// Ignore non-API sections.
			continue
		}
	}

	return msg, errors.Wrap(msg.Validate(), "api: invalid form parameters")
}

// deployments
func (d *DeploymentsApiHandlers) createDeployment(w rest.ResponseWriter, r *rest.Request, ctx context.Context, l *log.Logger, group string) {
	constructor, err := d.getDeploymentConstructorFromBody(r, group)
	if err != nil {
		d.view.RenderError(w, r, errors.Wrap(err, "Validating request body"), http.StatusBadRequest, l)
		return
	}

	id, err := d.app.CreateDeployment(ctx, constructor)
	switch err {
	case nil:
		// in case of deployment to group remove "/group/{name}" from path before creating location haeder
		r.URL.Path = strings.TrimSuffix(r.URL.Path, "/group/"+constructor.Group)
		d.view.RenderSuccessPost(w, r, id)
	case app.ErrNoArtifact:
		d.view.RenderError(w, r, err, http.StatusUnprocessableEntity, l)
	case app.ErrNoDevices:
		d.view.RenderError(w, r, err, http.StatusBadRequest, l)
	default:
		d.view.RenderInternalError(w, r, err, l)
	}
}

func (d *DeploymentsApiHandlers) PostDeployment(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Context()
	l := requestlog.GetRequestLogger(r)

	d.createDeployment(w, r, ctx, l, "")
}

func (d *DeploymentsApiHandlers) DeployToGroup(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Context()
	l := requestlog.GetRequestLogger(r)

	group := r.PathParam("name")
	if len(group) < 1 {
		d.view.RenderError(w, r, ErrMissingGroupName, http.StatusBadRequest, l)
	}
	d.createDeployment(w, r, ctx, l, group)
}

func (d *DeploymentsApiHandlers) getDeploymentConstructorFromBody(r *rest.Request, group string) (*model.DeploymentConstructor, error) {
	var constructor *model.DeploymentConstructor
	if err := r.DecodeJsonPayload(&constructor); err != nil {
		return nil, err
	}

	constructor.Group = group

	if err := constructor.Validate(); err != nil {
		return nil, err
	}

	return constructor, nil
}

func (d *DeploymentsApiHandlers) GetDeployment(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Context()
	l := requestlog.GetRequestLogger(r)

	id := r.PathParam("id")

	if !govalidator.IsUUID(id) {
		d.view.RenderError(w, r, ErrIDNotUUIDv4, http.StatusBadRequest, l)
		return
	}

	deployment, err := d.app.GetDeployment(ctx, id)
	if err != nil {
		d.view.RenderInternalError(w, r, err, l)
		return
	}

	if deployment == nil {
		d.view.RenderErrorNotFound(w, r, l)
		return
	}

	d.view.RenderSuccessGet(w, deployment)
}

func (d *DeploymentsApiHandlers) GetDeploymentStats(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Context()
	l := requestlog.GetRequestLogger(r)

	id := r.PathParam("id")

	if !govalidator.IsUUID(id) {
		d.view.RenderError(w, r, ErrIDNotUUIDv4, http.StatusBadRequest, l)
		return
	}

	stats, err := d.app.GetDeploymentStats(ctx, id)
	if err != nil {
		d.view.RenderInternalError(w, r, err, l)
		return
	}

	if stats == nil {
		d.view.RenderErrorNotFound(w, r, l)
		return
	}

	d.view.RenderSuccessGet(w, stats)
}

func (d *DeploymentsApiHandlers) GetDeploymentDeviceList(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Context()
	l := requestlog.GetRequestLogger(r)

	id := r.PathParam("id")

	if !govalidator.IsUUID(id) {
		d.view.RenderError(w, r, ErrIDNotUUIDv4, http.StatusBadRequest, l)
		return
	}

	deployment, err := d.app.GetDeployment(ctx, id)
	if err != nil {
		d.view.RenderInternalError(w, r, err, l)
		return
	}

	if deployment == nil {
		d.view.RenderErrorNotFound(w, r, l)
		return
	}

	d.view.RenderSuccessGet(w, deployment.DeviceList)
}

func (d *DeploymentsApiHandlers) AbortDeployment(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Context()
	l := requestlog.GetRequestLogger(r)

	id := r.PathParam("id")

	if !govalidator.IsUUID(id) {
		d.view.RenderError(w, r, ErrIDNotUUIDv4, http.StatusBadRequest, l)
		return
	}

	// receive request body
	var status struct {
		Status model.DeviceDeploymentStatus
	}

	err := r.DecodeJsonPayload(&status)
	if err != nil {
		d.view.RenderError(w, r, err, http.StatusBadRequest, l)
		return
	}
	// "aborted" is the only supported status
	if status.Status != model.DeviceDeploymentStatusAborted {
		d.view.RenderError(w, r, ErrUnexpectedDeploymentStatus, http.StatusBadRequest, l)
	}

	l.Infof("Abort deployment: %s", id)

	// Check if deployment is finished
	isDeploymentFinished, err := d.app.IsDeploymentFinished(ctx, id)
	if err != nil {
		d.view.RenderInternalError(w, r, err, l)
		return
	}
	if isDeploymentFinished {
		d.view.RenderError(w, r, ErrDeploymentAlreadyFinished, http.StatusUnprocessableEntity, l)
		return
	}

	// Abort deployments for devices and update deployment stats
	if err := d.app.AbortDeployment(ctx, id); err != nil {
		d.view.RenderInternalError(w, r, err, l)
	}

	d.view.RenderEmptySuccessResponse(w)
}

func (d *DeploymentsApiHandlers) GetDeploymentForDevice(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Context()
	l := requestlog.GetRequestLogger(r)

	idata := identity.FromContext(ctx)
	if idata == nil {
		d.view.RenderError(w, r, ErrMissingIdentity, http.StatusBadRequest, l)
		return
	}

	q := r.URL.Query()
	installed := &model.InstalledDeviceDeployment{
		ArtifactName: q.Get(GetDeploymentForDeviceQueryArtifact),
		DeviceType:   q.Get(GetDeploymentForDeviceQueryDeviceType),
	}

	if err := installed.Validate(); err != nil {
		d.view.RenderError(w, r, err, http.StatusBadRequest, l)
		return
	}

	deployment, err := d.app.GetDeploymentForDeviceWithCurrent(ctx, idata.Subject, installed)
	if err != nil {
		d.view.RenderInternalError(w, r, err, l)
		return
	}

	if deployment == nil {
		d.view.RenderNoUpdateForDevice(w)
		return
	}

	d.view.RenderSuccessGet(w, deployment)
}

func (d *DeploymentsApiHandlers) PostDeploymentForDevice(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Context()
	l := requestlog.GetRequestLogger(r)

	idata := identity.FromContext(ctx)
	if idata == nil {
		d.view.RenderError(w, r, ErrMissingIdentity, http.StatusBadRequest, l)
		return
	}

	var installed model.InstalledDeviceDeployment
	if err := r.DecodeJsonPayload(&installed); err != nil {
		d.view.RenderError(w, r,
			errors.Wrap(err, "invalid schema"),
			http.StatusBadRequest, l)
		return
	}

	if err := installed.Validate(); err != nil {
		d.view.RenderError(w, r, err, http.StatusBadRequest, l)
		return
	}

	deployment, err := d.app.GetDeploymentForDeviceWithCurrent(ctx, idata.Subject, &installed)
	if err != nil {
		d.view.RenderInternalError(w, r, err, l)
		return
	}

	if deployment == nil {
		d.view.RenderNoUpdateForDevice(w)
		return
	}

	// NOTE: Must use the RenderSuccessGet as the POST variant reports
	//       incorrect status code.
	d.view.RenderSuccessGet(w, deployment)
}

func (d *DeploymentsApiHandlers) PutDeploymentStatusForDevice(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Context()
	l := requestlog.GetRequestLogger(r)

	did := r.PathParam("id")

	idata := identity.FromContext(ctx)
	if idata == nil {
		d.view.RenderError(w, r, ErrMissingIdentity, http.StatusBadRequest, l)
		return
	}

	// receive request body
	var report model.StatusReport

	err := r.DecodeJsonPayload(&report)
	if err != nil {
		d.view.RenderError(w, r, err, http.StatusBadRequest, l)
		return
	}

	l.Infof("status: %+v", report)
	if err := d.app.UpdateDeviceDeploymentStatus(ctx, did,
		idata.Subject, model.DeviceDeploymentState{
			Status:   report.Status,
			SubState: report.SubState,
		}); err != nil {

		if err == app.ErrDeploymentAborted || err == app.ErrDeviceDecommissioned {
			d.view.RenderError(w, r, err, http.StatusConflict, l)
		} else if err == app.ErrStorageNotFound {
			d.view.RenderErrorNotFound(w, r, l)
		} else {
			d.view.RenderInternalError(w, r, err, l)
		}
		return
	}

	d.view.RenderEmptySuccessResponse(w)
}

func (d *DeploymentsApiHandlers) GetDeviceStatusesForDeployment(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Context()
	l := requestlog.GetRequestLogger(r)

	did := r.PathParam("id")

	if !govalidator.IsUUID(did) {
		d.view.RenderError(w, r, ErrIDNotUUIDv4, http.StatusBadRequest, l)
		return
	}

	statuses, err := d.app.GetDeviceStatusesForDeployment(ctx, did)
	if err != nil {
		switch err {
		case app.ErrModelDeploymentNotFound:
			d.view.RenderError(w, r, err, http.StatusNotFound, l)
			return
		default:
			d.view.RenderInternalError(w, r, ErrInternal, l)
			return
		}
	}

	d.view.RenderSuccessGet(w, statuses)
}

func ParseLookupQuery(vals url.Values) (model.Query, error) {
	query := model.Query{}

	search := vals.Get("search")
	if search != "" {
		query.SearchText = search
	}

	createdBefore := vals.Get("created_before")
	if createdBefore != "" {
		if createdBeforeTime, err := parseEpochToTimestamp(createdBefore); err != nil {
			return query, errors.Wrap(err, "timestamp parsing failed for created_before parameter")
		} else {
			query.CreatedBefore = &createdBeforeTime
		}
	}

	createdAfter := vals.Get("created_after")
	if createdAfter != "" {
		if createdAfterTime, err := parseEpochToTimestamp(createdAfter); err != nil {
			return query, errors.Wrap(err, "timestamp parsing failed created_after parameter")
		} else {
			query.CreatedAfter = &createdAfterTime
		}
	}

	status := vals.Get("status")
	switch status {
	case "inprogress":
		query.Status = model.StatusQueryInProgress
	case "finished":
		query.Status = model.StatusQueryFinished
	case "pending":
		query.Status = model.StatusQueryPending
	case "aborted":
		query.Status = model.StatusQueryAborted
	case "":
		query.Status = model.StatusQueryAny
	default:
		return query, errors.Errorf("unknown status %s", status)

	}

	return query, nil
}

func parseEpochToTimestamp(epoch string) (time.Time, error) {
	if epochInt64, err := strconv.ParseInt(epoch, 10, 64); err != nil {
		return time.Time{}, errors.Errorf("invalid timestamp: " + epoch)
	} else {
		return time.Unix(epochInt64, 0).UTC(), nil
	}
}

func (d *DeploymentsApiHandlers) LookupDeployment(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Context()
	l := requestlog.GetRequestLogger(r)

	query, err := ParseLookupQuery(r.URL.Query())
	if err != nil {
		d.view.RenderError(w, r, err, http.StatusBadRequest, l)
		return
	}

	page, perPage, err := rest_utils.ParsePagination(r)
	if err != nil {
		d.view.RenderError(w, r, err, http.StatusBadRequest, l)
		return
	}
	query.Skip = int((page - 1) * perPage)
	query.Limit = int(perPage + 1)

	deps, totalCount, err := d.app.LookupDeployment(ctx, query)
	if err != nil {
		d.view.RenderError(w, r, err, http.StatusBadRequest, l)
		return
	}
	w.Header().Add(hdrTotalCount, strconv.FormatInt(totalCount, 10))

	len := len(deps)
	hasNext := false
	if uint64(len) > perPage {
		hasNext = true
		len = int(perPage)
	}

	links := rest_utils.MakePageLinkHdrs(r, page, perPage, hasNext)
	for _, l := range links {
		w.Header().Add("Link", l)
	}

	d.view.RenderSuccessGet(w, deps[:len])
}

func (d *DeploymentsApiHandlers) PutDeploymentLogForDevice(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Context()
	l := requestlog.GetRequestLogger(r)

	did := r.PathParam("id")

	idata := identity.FromContext(ctx)
	if idata == nil {
		d.view.RenderError(w, r, ErrMissingIdentity, http.StatusBadRequest, l)
		return
	}

	// reuse DeploymentLog, device and deployment IDs are ignored when
	// (un-)marshaling DeploymentLog to/from JSON
	var log model.DeploymentLog

	err := r.DecodeJsonPayload(&log)
	if err != nil {
		d.view.RenderError(w, r, err, http.StatusBadRequest, l)
		return
	}

	if err := d.app.SaveDeviceDeploymentLog(ctx, idata.Subject,
		did, log.Messages); err != nil {

		if err == app.ErrModelDeploymentNotFound {
			d.view.RenderError(w, r, err, http.StatusNotFound, l)
		} else {
			d.view.RenderInternalError(w, r, err, l)
		}
		return
	}

	d.view.RenderEmptySuccessResponse(w)
}

func (d *DeploymentsApiHandlers) GetDeploymentLogForDevice(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Context()
	l := requestlog.GetRequestLogger(r)

	did := r.PathParam("id")
	devid := r.PathParam("devid")

	depl, err := d.app.GetDeviceDeploymentLog(ctx, devid, did)

	if err != nil {
		d.view.RenderInternalError(w, r, err, l)
		return
	}

	if depl == nil {
		d.view.RenderErrorNotFound(w, r, l)
		return
	}

	d.view.RenderDeploymentLog(w, *depl)
}

func (d *DeploymentsApiHandlers) DecommissionDevice(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Context()
	l := requestlog.GetRequestLogger(r)

	id := r.PathParam("id")

	// Decommission deployments for devices and update deployment stats
	err := d.app.DecommissionDevice(ctx, id)

	switch err {
	case nil, app.ErrStorageNotFound:
		d.view.RenderEmptySuccessResponse(w)
	default:
		d.view.RenderInternalError(w, r, err, l)

	}
}

// tenants

func (d *DeploymentsApiHandlers) ProvisionTenantsHandler(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Context()
	l := requestlog.GetRequestLogger(r)

	defer r.Body.Close()

	tenant, err := model.ParseNewTenantReq(r.Body)
	if err != nil {
		rest_utils.RestErrWithLog(w, r, l, err, http.StatusBadRequest)
		return
	}

	err = d.app.ProvisionTenant(ctx, tenant.TenantId)
	if err != nil {
		rest_utils.RestErrWithLogInternal(w, r, l, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (d *DeploymentsApiHandlers) DeploymentsPerTenantHandler(w rest.ResponseWriter, r *rest.Request) {
	tenantID := r.PathParam("tenant")
	if tenantID == "" {
		l := requestlog.GetRequestLogger(r)
		rest_utils.RestErrWithLog(w, r, l, errors.New("missing tenant ID"), http.StatusBadRequest)
		return
	}

	r.Request = r.WithContext(identity.WithContext(
		r.Context(),
		&identity.Identity{Tenant: tenantID},
	))
	d.LookupDeployment(w, r)
}
