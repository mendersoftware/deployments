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
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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
)

func MissingRequiredQueryMsg(name string) string {
	return fmt.Sprintf("Required query parameter missing: '%s'", name)
}

// RestErrorMsg testing function, don,t handle errors
// Used to generate equivalent body as rest.Error (from ant0ine/go-json-rest package) call would
func RestErrorMsg(status error) string {
	msg, _ := json.Marshal(map[string]string{"Error": status.Error()})
	return string(msg)
}

// ParseAndValidateUIntQuery parse and validate uint input as string min and max are included.
func ParseAndValidateUIntQuery(name, value string, min, max uint64) (uint64, error) {

	str := value
	if str == "" {
		return 0, errors.New(MissingRequiredQueryMsg(name))
	}

	uintValue, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
		return 0, err
	}

	if uintValue < min || uintValue > max {
		return 0, errors.New(fmt.Sprintf("Invalid query '%s' value '%d'. Min='%d' Max='%d'.",
			name, uintValue, min, max))
	}

	return uintValue, nil
}
