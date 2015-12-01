package handlers

// import (
// 	"github.com/mendersoftware/services/Godeps/_workspace/src/github.com/ant0ine/go-json-rest/rest"
// 	"github.com/mendersoftware/services/models/image"
// 	"math/rand"
// 	"net/http"
// 	"strconv"
// )

// type ImageVerifier struct {
// 	img *image.ImageModel
// }

// func NewImageVerifier(imageList *image.ImageModel) *ImageVerifier {

// 	return &ImageVerifier{
// 		img: imageList,
// 	}
// }

// // Dummy planceholder
// func (m *ImageVerifier) CreateJob(w rest.ResponseWriter, r *rest.Request) {

// 	id := r.PathParam("id")

// 	_, found := m.img.Get(id)
// 	if !found {
// 		rest.NotFound(w, r)
// 		return
// 	}

// 	jobId := rand.Int()

// 	w.Header().Add("Location", "/api/0.0.1/images/"+id+"/verify/"+strconv.Itoa(jobId))
// 	w.WriteHeader(http.StatusAccepted)
// }

// // Dummy planceholder
// func (m *ImageVerifier) GetJob(w rest.ResponseWriter, r *rest.Request) {

// 	status := struct {
// 		Status   string `json:"status"`
// 		Progress int    `json:"progress"`
// 	}{
// 		"complete",
// 		100,
// 	}

// 	w.WriteJson(status)
// }

// // Dummy planceholder
// func (m *ImageVerifier) DeleteJob(w rest.ResponseWriter, r *rest.Request) {
// 	w.WriteHeader(http.StatusNoContent)
// }
