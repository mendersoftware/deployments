// Copyright 2022 Northern.tech AS
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

package accesslog

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/mendersoftware/go-lib-micro/rest.utils"
	"github.com/pkg/errors"
)

const MaxTraceback = 32

func funcname(fn string) string {
	// strip package path
	i := strings.LastIndex(fn, "/")
	fn = fn[i+1:]
	// strip package name.
	i = strings.Index(fn, ".")
	fn = fn[i+1:]
	return fn
}

func panicHandler(c *gin.Context, startTime time.Time) {
	if r := recover(); r != nil {
		l := log.FromContext(c.Request.Context())
		latency := time.Since(startTime)
		trace := [MaxTraceback]uintptr{}
		// Skip 3
		// = runtime.Callers + runtime.extern.Callers + runtime.gopanic
		num := runtime.Callers(3, trace[:])
		var traceback strings.Builder
		for i := 0; i < num; i++ {
			fn := runtime.FuncForPC(trace[i])
			if fn == nil {
				fmt.Fprintf(&traceback, "\n???")
				continue
			}
			file, line := fn.FileLine(trace[i])
			fmt.Fprintf(&traceback, "\n%s(%s):%d",
				file, funcname(fn.Name()), line,
			)
		}

		logCtx := log.Ctx{
			"clientip": c.ClientIP(),
			"method":   c.Request.Method,
			"path":     c.Request.URL.Path,
			"qs":       c.Request.URL.RawQuery,
			"responsetime": fmt.Sprintf("%dus",
				latency.Round(time.Microsecond).Microseconds()),
			"status":    500,
			"ts":        startTime.Format(time.RFC3339),
			"type":      c.Request.Proto,
			"useragent": c.Request.UserAgent(),
			"trace":     traceback.String()[1:],
		}
		l = l.F(logCtx)
		func() {
			// Panic is going to panic, but we
			// immediately want to recover.
			defer func() { recover() }() //nolint:errcheck
			l.Panicf("[request panic] %s", r)
		}()

		// Try to respond with an internal server error.
		// If the connection is broken it might panic again.
		defer func() { recover() }() // nolint:errcheck
		rest.RenderError(c,
			http.StatusInternalServerError,
			errors.New("internal error"),
		)
	}
}

// Middleware provides accesslog middleware for the gin-gonic framework.
// This middleware will recover any panic from the occurring in the API
// handler and log it to panic level.
// If an error status is returned in the response, the middleware tries
// to pop the topmost error from the gin.Context (c.Error) and puts it in
// the "error" context to the final log entry.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		defer panicHandler(c, startTime)

		c.Next()

		l := log.FromContext(c.Request.Context())
		latency := time.Since(startTime)
		code := c.Writer.Status()
		// Add status and response time to log context
		size := c.Writer.Size()
		if size < 0 {
			size = 0
		}
		logCtx := log.Ctx{
			"byteswritten": size,
			"clientip":     c.ClientIP(),
			"method":       c.Request.Method,
			"path":         c.Request.URL.Path,
			"qs":           c.Request.URL.RawQuery,
			"responsetime": fmt.Sprintf("%dus",
				latency.Round(time.Microsecond).Microseconds()),
			"status":    code,
			"ts":        startTime.Format(time.RFC3339),
			"type":      c.Request.Proto,
			"useragent": c.Request.UserAgent(),
		}
		l = l.F(logCtx)

		if code < 400 {
			logged := false
			for pathSuffix, status := range DebugLogsByPathSuffix {
				if code == status && strings.HasSuffix(c.Request.URL.Path, pathSuffix) {
					l.Debug()
					logged = true
					break
				}
			}

			if !logged {
				l.Info()
			}
		} else {
			if len(c.Errors) > 0 {
				errs := c.Errors.Errors()
				var errMsg string
				if len(errs) == 1 {
					errMsg = errs[0]
				} else {
					for i, err := range errs {
						errMsg = errMsg + fmt.Sprintf(
							"#%02d: %s\n", i+1, err,
						)
					}
				}
				l = l.F(log.Ctx{
					"error": errMsg,
				})
			} else {
				l = l.F(log.Ctx{
					"error": http.StatusText(code),
				})
			}
			l.Error()
		}
	}
}
