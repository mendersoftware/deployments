// Copyright 2023 Northern.tech AS
//
//	Licensed under the Apache License, Version 2.0 (the "License");
//	you may not use this file except in compliance with the License.
//	You may obtain a copy of the License at
//
//	    http://www.apache.org/licenses/LICENSE-2.0
//
//	Unless required by applicable law or agreed to in writing, software
//	distributed under the License is distributed on an "AS IS" BASIS,
//	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	See the License for the specific language governing permissions and
//	limitations under the License.
package rest_utils

import (
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/mendersoftware/go-lib-micro/requestid"
)

// return selected http code + error message directly taken from error
// log error
func RestErrWithLog(w rest.ResponseWriter, r *rest.Request, l *log.Logger, e error, code int) {
	l = l.WithCallerContext(1)
	restErrWithLogMsg(w, r, l, e, code, "", logrus.ErrorLevel)
}

// return http 500, with an "internal error" message
// log full error
func RestErrWithLogInternal(w rest.ResponseWriter, r *rest.Request, l *log.Logger, e error) {
	msg := "internal error"
	l = l.WithCallerContext(1)
	restErrWithLogMsg(w, r, l, e, http.StatusInternalServerError, msg, logrus.ErrorLevel)
}

// return an error code with an overriden message (to avoid exposing the details)
// log full error as debug
func RestErrWithDebugMsg(w rest.ResponseWriter, r *rest.Request, l *log.Logger, e error, code int, msg string) {
	l = l.WithCallerContext(1)
	restErrWithLogMsg(w, r, l, e, code, msg, logrus.DebugLevel)
}

// return an error code with an overriden message (to avoid exposing the details)
// log full error as info
func RestErrWithInfoMsg(w rest.ResponseWriter, r *rest.Request, l *log.Logger, e error, code int, msg string) {
	l = l.WithCallerContext(1)
	restErrWithLogMsg(w, r, l, e, code, msg, logrus.InfoLevel)
}

// return an error code with an overriden message (to avoid exposing the details)
// log full error as warning
func RestErrWithWarningMsg(w rest.ResponseWriter, r *rest.Request, l *log.Logger, e error, code int, msg string) {
	l = l.WithCallerContext(1)
	restErrWithLogMsg(w, r, l, e, code, msg, logrus.WarnLevel)
}

// same as RestErrWithErrorMsg - for backward compatibility purpose
// return an error code with an overriden message (to avoid exposing the details)
// log full error as error
func RestErrWithLogMsg(w rest.ResponseWriter, r *rest.Request, l *log.Logger, e error, code int, msg string) {
	l = l.WithCallerContext(1)
	restErrWithLogMsg(w, r, l, e, code, msg, logrus.ErrorLevel)
}

// return an error code with an overriden message (to avoid exposing the details)
// log full error as error
func RestErrWithErrorMsg(w rest.ResponseWriter, r *rest.Request, l *log.Logger, e error, code int, msg string) {
	l = l.WithCallerContext(1)
	restErrWithLogMsg(w, r, l, e, code, msg, logrus.ErrorLevel)
}

// return an error code with an overriden message (to avoid exposing the details)
// log full error as fatal
func RestErrWithFatalMsg(w rest.ResponseWriter, r *rest.Request, l *log.Logger, e error, code int, msg string) {
	l = l.WithCallerContext(1)
	restErrWithLogMsg(w, r, l, e, code, msg, logrus.FatalLevel)
}

// return an error code with an overriden message (to avoid exposing the details)
// log full error as panic
func RestErrWithPanicMsg(w rest.ResponseWriter, r *rest.Request, l *log.Logger, e error, code int, msg string) {
	l = l.WithCallerContext(1)
	restErrWithLogMsg(w, r, l, e, code, msg, logrus.PanicLevel)
}

// return an error code with an overriden message (to avoid exposing the details)
// log full error with given log level
func restErrWithLogMsg(w rest.ResponseWriter, r *rest.Request, l *log.Logger,
	e error, code int, msg string, logLevel logrus.Level) {

	if msg != "" {
		e = errors.Wrap(e, msg)
	} else {
		msg = e.Error()
	}

	w.WriteHeader(code)
	err := w.WriteJson(ApiError{
		Err:   msg,
		ReqId: requestid.GetReqId(r),
	})
	if err != nil {
		panic(err)
	}
	switch logLevel {
	case logrus.DebugLevel:
		l.Debug(e.Error())
	case logrus.InfoLevel:
		l.Info(e.Error())
	case logrus.WarnLevel:
		l.Warn(e.Error())
	case logrus.ErrorLevel:
		l.Error(e.Error())
	case logrus.FatalLevel:
		l.Fatal(e.Error())
	case logrus.PanicLevel:
		l.Panic(e.Error())
	}
}
