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

package main

import (
	"mime"
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/mendersoftware/deployments/config"
	"github.com/mendersoftware/go-lib-micro/accesslog"
	"github.com/mendersoftware/go-lib-micro/requestid"
	"github.com/mendersoftware/go-lib-micro/requestlog"
)

const (
	HttpHeaderContentType                 string = "Content-type"
	HttpHeaderOrigin                      string = "Origin"
	HttpHeaderAuthorization               string = "Authorization"
	HttpHeaderAcceptEncoding              string = "Accept-Encoding"
	HttpHeaderAccessControlRequestHeaders string = "Access-Control-Request-Headers"
	HttpHeaderAccessControlRequestMethod  string = "Access-Control-Request-Method"
	HttpHeaderLastModified                string = "Last-Modified"
	HttpHeaderExpires                     string = "Expires"
	HttpHeaderLocation                    string = "Location"
	HttpHeaderAllow                       string = "Allow"
	HttpHeaderAccept                      string = "Accept"

	EnvProd string = "prod"
	EnvDev  string = "dev"
)

var DefaultDevStack = []rest.Middleware{

	// logging
	&requestlog.RequestLogMiddleware{},
	&accesslog.AccessLogMiddleware{Format: accesslog.SimpleLogFormat},
	&rest.TimerMiddleware{},
	&rest.RecorderMiddleware{},

	// catches the panic errors that occur with stack trace
	&rest.RecoverMiddleware{
		EnableResponseStackTrace: true,
	},

	// json pretty print
	&rest.JsonIndentMiddleware{},
	&requestid.RequestIdMiddleware{},
}

var DefaultProdStack = []rest.Middleware{

	// logging
	&requestlog.RequestLogMiddleware{},
	&accesslog.AccessLogMiddleware{Format: accesslog.SimpleLogFormat},
	&rest.TimerMiddleware{},
	&rest.RecorderMiddleware{},

	// catches the panic errorsx
	&rest.RecoverMiddleware{},

	// response compression
	&rest.GzipMiddleware{},
	&requestid.RequestIdMiddleware{},
}

func SetupMiddleware(c config.ConfigReader, api *rest.Api) {
	api.Use(DefaultDevStack...)

	// Verifies the request Content-Type header if the content is non-null.
	// For the POST /api/0.0.1/images request expected Content-Type is 'multipart/form-data'.
	// For the rest of the requests expected Content-Type is 'application/json'.
	api.Use(&rest.IfMiddleware{
		Condition: func(r *rest.Request) bool {
			if r.URL.Path == "/api/0.0.1/images" && r.Method == http.MethodPost {
				return true
			} else {
				return false
			}
		},
		IfTrue: rest.MiddlewareSimple(func(handler rest.HandlerFunc) rest.HandlerFunc {
			return func(w rest.ResponseWriter, r *rest.Request) {
				mediatype, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
				if r.ContentLength > 0 && !(mediatype == "multipart/form-data") {
					rest.Error(w, "Bad Content-Type, expected 'multipart/form-data'", http.StatusUnsupportedMediaType)
					return
				}
				// call the wrapped handler
				handler(w, r)
			}
		}),
		IfFalse: &rest.ContentTypeCheckerMiddleware{},
	})

	api.Use(&rest.CorsMiddleware{
		RejectNonCorsRequests: false,

		// Should be tested with some list
		OriginValidator: func(origin string, request *rest.Request) bool {
			// Accept all requests
			return true
		},

		// Preflight request cache lenght
		AccessControlMaxAge: 60,

		// Allow authentication requests
		AccessControlAllowCredentials: true,

		// Allowed headers
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
		},

		// Allowed heardes
		AllowedHeaders: []string{
			HttpHeaderAccept,
			HttpHeaderAllow,
			HttpHeaderContentType,
			HttpHeaderOrigin,
			HttpHeaderAuthorization,
			HttpHeaderAcceptEncoding,
			HttpHeaderAccessControlRequestHeaders,
			HttpHeaderAccessControlRequestMethod,
		},

		// Headers that can be exposed to JS
		AccessControlExposeHeaders: []string{
			HttpHeaderLocation,
		},
	})
}
