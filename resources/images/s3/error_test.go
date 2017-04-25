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
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func bytesBuffer(data string) io.ReadCloser {
	return ioutil.NopCloser(bytes.NewBufferString(data))
}

func TestGetS3Error(t *testing.T) {

	t.Parallel()

	tcs := []struct {
		rsp *http.Response
		err error
	}{
		{
			rsp: &http.Response{
				StatusCode: http.StatusBadRequest,
				Header: http.Header{
					"Content-Type": []string{"pets"},
				},
				Body: nil,
			},
			err: errors.New("unexpected S3 error response, status: 400, type: pets"),
		},
		{
			rsp: &http.Response{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type": []string{"application/xml"},
				},
				Body: nil,
			},
			err: errors.New("unexpected S3 error response, status: 200, type: application/xml"),
		},
		{
			rsp: &http.Response{
				StatusCode: http.StatusBadRequest,
				Header: http.Header{
					"Content-Type": []string{"application/xml"},
				},
				Body: bytesBuffer("foo-bar-bar"),
			},
			err: errors.New("failed to decode XML encoded error response: EOF"),
		},
		{
			rsp: &http.Response{
				StatusCode: http.StatusBadRequest,
				Header: http.Header{
					"Content-Type": []string{"application/xml"},
				},
				Body: bytesBuffer(`<?xml version="1.0" encoding="UTF-8"?>
<Error>
  <Code>NoSuchKey</Code>
  <Message>The resource you requested does not exist</Message>
  <Resource>/mybucket/myfoto.jpg</Resource>
  <RequestId>4442587FB7D0A2F9</RequestId>
</Error>
`),
			},
			err: errors.New("S3 request failed with code NoSuchKey: The resource you requested does not exist, request ID: 4442587FB7D0A2F9"),
		},
	}

	for idx, tc := range tcs {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {

			err := getS3Error(tc.rsp)
			assert.EqualError(t, err, tc.err.Error())
		})
	}
}
