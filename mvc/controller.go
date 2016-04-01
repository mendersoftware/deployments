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
package mvc

import (
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
)

func NewCreateController(model CreateModeler, view Viewer) rest.HandlerFunc {
	return func(w rest.ResponseWriter, r *rest.Request) {
		// Validate incomming request
		obj := model.NewObject()

		if err := r.DecodeJsonPayload(&obj); err != nil {
			view.RenderError(w, err, http.StatusBadRequest)
			return
		}

		if err := model.Validate(obj); err != nil {
			view.RenderError(w, err, http.StatusBadRequest)
			return
		}

		id, err := model.Create(obj)
		if err != nil {
			view.RenderError(w, err, http.StatusInternalServerError)
			return
		}

		view.RenderSuccess(w, id)
	}
}

func NewGetObjectController(model GetObjectModeler, view Viewer) rest.HandlerFunc {
	return func(w rest.ResponseWriter, r *rest.Request) {
		response, err := model.GetObject(r.PathParam("id"))
		if err != nil {
			view.RenderError(w, err, http.StatusInternalServerError)
			return
		}

		view.RenderSuccess(w, response)
	}
}
