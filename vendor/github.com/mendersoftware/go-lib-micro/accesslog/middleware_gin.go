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

package accesslog

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mendersoftware/go-lib-micro/log"
)

type LogParameters struct {
	// Context contains the entire request context as exposed by
	// gin.Context.
	Context *gin.Context

	// Path holds the path as set before the request handler is executed.
	Path string

	// StartTime is the time when the request was received by the
	// middleware, which can be used to compute latency.
	StartTime time.Time
}

type LogHook func(parms LogParameters)

func defaultLogHook(params LogParameters) {
	latency := time.Since(params.StartTime)
	c := params.Context
	l := log.FromContext(c.Request.Context())
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
		"path":         params.Path,
		"responsetime": fmt.Sprintf("%dus",
			latency.Round(time.Microsecond).Microseconds()),
		"status":    code,
		"ts":        params.StartTime.Format(time.RFC3339),
		"type":      c.Request.Proto,
		"useragent": c.Request.UserAgent(),
	}
	l = l.F(logCtx)

	if code < 400 {
		l.Info()
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

type MiddlewareOptions struct {
	BeforeHook LogHook

	AfterHook LogHook
}

func NewMiddlewareOptions() *MiddlewareOptions {
	return &MiddlewareOptions{}
}

func (opt *MiddlewareOptions) SetBeforeHook(logHook LogHook) *MiddlewareOptions {
	opt.BeforeHook = logHook
	return opt
}

func (opt *MiddlewareOptions) SetAfterHook(logHook LogHook) *MiddlewareOptions {
	opt.AfterHook = logHook
	return opt
}

// Middleware provides accesslog middleware for the gin-gonic framework.
// The middleware enriches the log context with "method" and "path" context
// and adds a log entry to all request reporting the response time and status-
// code. If an error status is returned in the response, the middleware tries
// to pop the topmost error from the gin.Context (c.Error) and puts it in
// the "error" context to the final log entry.
func Middleware(opts ...*MiddlewareOptions) gin.HandlerFunc {
	opt := NewMiddlewareOptions().
		SetAfterHook(defaultLogHook)
	for _, o := range opts {
		if o == nil {
			continue
		}
		if o.AfterHook != nil {
			opt.AfterHook = o.AfterHook
		}
		if o.BeforeHook != nil {
			opt.BeforeHook = o.BeforeHook
		}
	}
	return func(c *gin.Context) {
		params := LogParameters{
			Context:   c,
			Path:      c.Request.URL.Path,
			StartTime: time.Now(),
		}
		if c.Request.URL.RawQuery != "" {
			params.Path = params.Path + "?" + c.Request.URL.RawQuery
		}

		if opt.BeforeHook != nil {
			opt.BeforeHook(params)
		}

		// Perform the request
		c.Next()

		opt.AfterHook(params)
	}
}
