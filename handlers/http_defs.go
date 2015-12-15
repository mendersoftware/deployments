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
