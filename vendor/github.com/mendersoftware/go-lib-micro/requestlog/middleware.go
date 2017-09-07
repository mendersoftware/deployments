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
package requestlog

import (
	"github.com/Sirupsen/logrus"
	"github.com/ant0ine/go-json-rest/rest"

	"github.com/mendersoftware/go-lib-micro/log"
)

// name of the per-request log in the request's context
const ReqLog = "request_log"

// RequestLogMiddleware creates a per-request logger and sticks it into Env.
// The logger will be ready to use in the handler (less boilerplate).
// Other middlewares (notably requestid) may add context to the log.
// Per-request loggers will by default be derived from the global log.Log,
// unless BaseLogger is specified. In that case, it will serve as the root logger.
// Additional context can be attached by setting LogContext field.
type RequestLogMiddleware struct {
	BaseLogger *logrus.Logger
	LogContext log.Ctx
}

// MiddlewareFunc makes RequestLogMiddleware implement the Middleware interface.
func (mw *RequestLogMiddleware) MiddlewareFunc(h rest.HandlerFunc) rest.HandlerFunc {
	return func(w rest.ResponseWriter, r *rest.Request) {
		var l *log.Logger
		if mw.BaseLogger == nil {
			l = log.New(mw.LogContext)
		} else {
			l = log.NewFromLogger(mw.BaseLogger, mw.LogContext)
		}

		r.Env[ReqLog] = l
		h(w, r)
	}
}

func GetRequestLogger(env map[string]interface{}) *log.Logger {
	return env[ReqLog].(*log.Logger)
}
