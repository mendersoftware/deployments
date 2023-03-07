// Copyright 2023 Northern.tech AS
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

package main

import (
	"mime"
	"net/http"
	"regexp"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/mendersoftware/go-lib-micro/accesslog"
	"github.com/mendersoftware/go-lib-micro/config"
	"github.com/mendersoftware/go-lib-micro/identity"
	"github.com/mendersoftware/go-lib-micro/requestid"
	"github.com/mendersoftware/go-lib-micro/requestlog"

	api_http "github.com/mendersoftware/deployments/api/http"
	dconfig "github.com/mendersoftware/deployments/config"
)

var commonLoggingAccessStack = []rest.Middleware{
	// logging
	&requestlog.RequestLogMiddleware{},
	&accesslog.AccessLogMiddleware{Format: accesslog.SimpleLogFormat},
	&rest.TimerMiddleware{},
	&rest.RecorderMiddleware{},
}

var defaultDevStack = []rest.Middleware{

	// catches the panic errors that occur with stack trace
	&rest.RecoverMiddleware{
		EnableResponseStackTrace: true,
	},

	// json pretty print
	&rest.JsonIndentMiddleware{},
}

var defaultProdStack = []rest.Middleware{
	// catches the panic errorsx
	&rest.RecoverMiddleware{},
}

var regexInternalArtifacts = regexp.MustCompile(
	"^" + api_http.ApiUrlInternal + "/tenants/([0-9a-f-]+)/artifacts",
)

func SetupMiddleware(c config.Reader, api *rest.Api) {

	api.Use(commonLoggingAccessStack...)

	mwtype := c.GetString(dconfig.SettingMiddleware)
	if mwtype == dconfig.EnvDev {
		api.Use(defaultDevStack...)
	} else {
		api.Use(defaultProdStack...)
	}

	api.Use(&requestid.RequestIdMiddleware{},
		&identity.IdentityMiddleware{
			UpdateLogger: true,
		})

	// Verifies the request Content-Type header if the content is non-null.
	// For the POST /api/0.0.1/images request expected Content-Type is 'multipart/form-data'.
	// For the rest of the requests expected Content-Type is 'application/json'.
	api.Use(&rest.IfMiddleware{
		Condition: func(r *rest.Request) bool {
			if r.Method == http.MethodPost {
				if r.URL.Path == api_http.ApiUrlManagementArtifacts ||
					r.URL.Path == api_http.ApiUrlManagementArtifactsGenerate ||
					regexInternalArtifacts.MatchString(r.URL.Path) {
					return true
				}
			}
			return false
		},
		IfTrue: rest.MiddlewareSimple(func(handler rest.HandlerFunc) rest.HandlerFunc {
			return func(w rest.ResponseWriter, r *rest.Request) {
				mediatype, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
				if r.ContentLength > 0 && !(mediatype == "multipart/form-data") {
					rest.Error(w,
						"Bad Content-Type, expected 'multipart/form-data'",
						http.StatusUnsupportedMediaType)
					return
				}
				// call the wrapped handler
				handler(w, r)
			}
		}),
		IfFalse: &rest.ContentTypeCheckerMiddleware{},
	})
}
