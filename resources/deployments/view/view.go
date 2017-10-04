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

package view

import (
	"net/http"
	"strings"

	"github.com/ant0ine/go-json-rest/rest"

	"github.com/mendersoftware/deployments/resources/deployments"
	"github.com/mendersoftware/deployments/utils/restutil/view"
)

type DeploymentsView struct {
	view.RESTView
}

func (d *DeploymentsView) RenderNoUpdateForDevice(w rest.ResponseWriter) {
	d.RenderEmptySuccessResponse(w)
}

// Success response with no data aka. 204 No Content
func (d *DeploymentsView) RenderEmptySuccessResponse(w rest.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func (d *DeploymentsView) RenderDeploymentLog(w rest.ResponseWriter, dlog deployments.DeploymentLog) {
	h, _ := w.(http.ResponseWriter)

	h.Header().Set("Content-Type", "text/plain")
	h.WriteHeader(http.StatusOK)

	for _, m := range dlog.Messages {
		as := m.String()
		h.Write([]byte(as))
		if !strings.HasSuffix(as, "\n") {
			h.Write([]byte("\n"))
		}
	}
}
