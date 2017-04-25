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

package s3

import (
	"encoding/xml"
	"net/http"

	"github.com/pkg/errors"
)

// getS3Error tries to extract S3 error information from HTTP response. Response
// body is partially consumed. Returns an error with whatever error information returned
// by S3 or just a generic description of a problem in case the response is not
// a correct error response.
//
// See https://docs.aws.amazon.com/AmazonS3/latest/API/ErrorResponses.html for
// example error response returned by S3.
func getS3Error(r *http.Response) error {
	s3rsp := struct {
		XMLName   xml.Name `xml:"Error"`
		Code      string   `xml:"Code"`
		Message   string   `xml:"Message"`
		RequestId string   `xml:"RequestId"`
		Resource  string   `xml:"Resource"`
	}{}

	if r.StatusCode < 300 ||
		r.Header.Get("Content-Type") != "application/xml" {

		return errors.Errorf("unexpected S3 error response, status: %v, type: %s",
			r.StatusCode, r.Header.Get("Content-Type"))
	}

	dec := xml.NewDecoder(r.Body)
	err := dec.Decode(&s3rsp)
	if err != nil {
		return errors.Wrap(err, "failed to decode XML encoded error response")
	}

	return errors.Errorf("S3 request failed with code %s: %s, request ID: %s",
		s3rsp.Code, s3rsp.Message, s3rsp.RequestId)
}
