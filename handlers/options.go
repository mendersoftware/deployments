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
package handlers

import (
	"github.com/ant0ine/go-json-rest/rest"
)

type CreateOptionsHandler func(methods ...string) rest.HandlerFunc

type OptionsHandler struct {
	// Shared  reads, need locking of any write mathod is introduced.
	methods map[string]bool
}

// NewOptionsHandler creates http handler object that will server OPTIONS method requests,
// Accepts a list of http methods.
// Adds information that it serves OPTIONS method automatically.
func NewOptionsHandler(methods ...string) rest.HandlerFunc {
	handler := &OptionsHandler{
		methods: make(map[string]bool, len(methods)+1),
	}

	for _, method := range methods {
		handler.methods[method] = true
	}

	if _, ok := handler.methods[HttpMethodOptions]; !ok {
		handler.methods[HttpMethodOptions] = true
	}

	return handler.handle
}

// Handle is a method for handling OPTIONS method requests.
// This method is called concurently while serving requests and should not modify self.
func (o *OptionsHandler) handle(w rest.ResponseWriter, r *rest.Request) {
	for method, _ := range o.methods {
		w.Header().Add(HttpHeaderAllow, method)
	}
}
