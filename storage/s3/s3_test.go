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

package s3

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			return
		},
	))
	defer srv.Close()

	srvURL, err := url.Parse(srv.URL)
	if err != nil {
		panic(err)
	}

	httpClient := http.Client{
		Transport: &http.Transport{
			DialTLS: func(network, addr string) (net.Conn, error) {
				return net.Dial(network, srvURL.Host)
			},
		},
	}
	sess, _ := session.NewSession()
	sss := SimpleStorageService{
		client: s3.New(
			sess,
			aws.NewConfig().
				WithHTTPClient(&httpClient).
				WithRegion("test").
				WithCredentials(credentials.
					NewStaticCredentials(
						"test-id",
						"test-secret",
						"test-token",
					)),
		),
		bucket: "test",
	}

	err = sss.HealthCheck(context.Background())
	assert.NoError(t, err)
}
