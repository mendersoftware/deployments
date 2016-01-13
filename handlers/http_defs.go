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

const (
	// HTTP Methods
	HttpMethodGet     string = "GET"
	HttpMethodPut     string = "PUT"
	HttpMethodPost    string = "POST"
	HttpMethodHead    string = "HEAD"
	HttpMethodOptions string = "OPTIONS"
	HttpMethodDelete  string = "DELETE"
	HttpMethodPatch   string = "PATCH"
	HttpMethodTrace   string = "TRACE"

	// HTTP HEADERS
	HttpHeaderAllow                       string = "Allow"
	HttpHeaderContentType                 string = "Content-type"
	HttpHeaderOrigin                      string = "Origin"
	HttpHeaderAuthorization               string = "Authorization"
	HttpHeaderAcceptEncoding              string = "Accept-Encoding"
	HttpHeaderAccessControlRequestHeaders string = "Access-Control-Request-Headers"
	HttpHeaderAccessControlRequestMethod  string = "Access-Control-Request-Method"
	HttpHeaderLastModified                string = "Last-Modified"
	HttpHeaderExpires                     string = "Expires"
	HttpHeaderLocation                    string = "Location"
)
