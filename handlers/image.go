package handlers

import (
	"net/http"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/mendersoftware/artifacts/controllers"
	"github.com/mendersoftware/artifacts/models/images"
	"github.com/mendersoftware/artifacts/models/users"
)

const (
	QueryExpireName = "expire"
	// 7 days in minutes
	QueryExpireMaxLimit = 60 * 7 * 24
	// 1 minute
	QueryExpireMinLimit = 1
)

// Takes care of input processing, responce building, calling appropriate controller
type ImageMeta struct {
	controler controllers.ImagesControllerI
}

func NewImageMeta(controler controllers.ImagesControllerI) *ImageMeta {

	return &ImageMeta{
		controler: controler,
	}
}

func (m *ImageMeta) Lookup(w rest.ResponseWriter, r *rest.Request) {

	u := users.NewDummyUser(r.Env["REMOTE_USER"].(string))

	images, err := m.controler.Lookup(u)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteJson(images)
}

func (m *ImageMeta) Get(w rest.ResponseWriter, r *rest.Request) {

	u := users.NewDummyUser(r.Env["REMOTE_USER"].(string))
	id := r.PathParam("id")

	image, err := m.controler.Get(u, id)
	if err != nil {
		rest.NotFound(w, r)
		return
	}

	w.Header().Set("Last-Modified", image.LastUpdated.UTC().Format(http.TimeFormat))
	w.WriteJson(image)
}

// Location for GET object is hardcoded here.
func (m *ImageMeta) Create(w rest.ResponseWriter, r *rest.Request) {

	u := users.NewDummyUser(r.Env["REMOTE_USER"].(string))

	// Validate incomming request

	imagePub := &images.ImageMetaPublic{}

	if err := r.DecodeJsonPayload(&imagePub); err != nil {
		rest.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := imagePub.Valid(); err != nil {
		rest.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Pass to controller
	imgNew, err := m.controler.Create(u, imagePub)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Location", "/api/0.0.1/images/"+imgNew.Id)
	w.WriteHeader(http.StatusCreated)
	w.WriteJson(imgNew)

}

func (m *ImageMeta) Edit(w rest.ResponseWriter, r *rest.Request) {

	u := users.NewDummyUser(r.Env["REMOTE_USER"].(string))
	id := r.PathParam("id")

	// Validate incomming request

	imagePub := &images.ImageMetaPublic{}

	if err := r.DecodeJsonPayload(&imagePub); err != nil {
		rest.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := imagePub.Valid(); err != nil {
		rest.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Pass to controller
	if err := m.controler.Edit(u, id, imagePub); err != nil {
		if err == controllers.ErrNotFound {
			rest.NotFound(w, r)
			return
		}

		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Location", r.RequestURI)
	w.WriteHeader(http.StatusNoContent)
}

func (m *ImageMeta) Delete(w rest.ResponseWriter, r *rest.Request) {

	u := users.NewDummyUser(r.Env["REMOTE_USER"].(string))
	id := r.PathParam("id")

	if err := m.controler.Delete(u, id); err != nil {
		if err == controllers.ErrNotFound {
			rest.NotFound(w, r)
			return
		}

		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (m *ImageMeta) UploadLink(w rest.ResponseWriter, r *rest.Request) {

	u := users.NewDummyUser(r.Env["REMOTE_USER"].(string))
	id := r.PathParam("id")

	minutes, err := ParseAndValidateUIntQuery(QueryExpireName,
		r.URL.Query().Get(QueryExpireName),
		QueryExpireMinLimit, QueryExpireMaxLimit)

	if err != nil {
		rest.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	link, err := m.controler.UploadLink(u, id, time.Duration(minutes)*time.Minute)

	if err != nil {
		if err == controllers.ErrNotFound {
			rest.NotFound(w, r)
			return
		}

		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Expires", link.Expire.UTC().Format(http.TimeFormat))
	w.WriteJson(link)
}

func (m *ImageMeta) DownloadLink(w rest.ResponseWriter, r *rest.Request) {

	u := users.NewDummyUser(r.Env["REMOTE_USER"].(string))
	id := r.PathParam("id")

	minutes, err := ParseAndValidateUIntQuery(QueryExpireName,
		r.URL.Query().Get(QueryExpireName),
		QueryExpireMinLimit, QueryExpireMaxLimit)

	if err != nil {
		rest.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	link, err := m.controler.DownloadLink(u, id, time.Duration(minutes)*time.Minute)

	if err != nil {
		if err == controllers.ErrNotFound {
			rest.NotFound(w, r)
			return
		}

		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Expires", link.Expire.UTC().Format(http.TimeFormat))
	w.WriteJson(link)

}
