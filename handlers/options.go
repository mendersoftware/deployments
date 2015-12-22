package handlers

import (
	"github.com/ant0ine/go-json-rest/rest"
)

type OptionsHandler struct {
	// Shared  reads, need locking of any write mathod is introduced.
	methods map[string]bool
}

// NewOptionsHandler creates http handler object that will server OPTIONS method requests,
// Accepts a list of http methods.
// Adds information that it serves OPTIONS method automatically.
func NewOptionsHandler(methods ...string) *OptionsHandler {
	handler := &OptionsHandler{
		methods: make(map[string]bool, len(methods)+1),
	}

	for _, method := range methods {
		handler.methods[method] = true
	}

	if _, ok := handler.methods[HttpMethodOptions]; !ok {
		handler.methods[HttpMethodOptions] = true
	}

	return handler
}

// Handle is a method for handling OPTIONS method requests.
// This method is called concurently while serving requests and should not modify self.
func (o *OptionsHandler) Handle(w rest.ResponseWriter, r *rest.Request) {
	for method, _ := range o.methods {
		w.Header().Add(HttpHeaderAllow, method)
	}
}
