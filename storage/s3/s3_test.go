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
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/deployments/storage"
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

	httpClient := &http.Client{
		Transport: &http.Transport{
			DialTLS: func(network, addr string) (net.Conn, error) {
				return net.Dial(network, srvURL.Host)
			},
		},
	}

	sss := SimpleStorageService{
		client: s3.New(s3.Options{
			Region:     "test",
			HTTPClient: httpClient,
			Credentials: StaticCredentials{
				Key:    "test",
				Secret: "secret",
				Token:  "token",
			},
		}),
		bucket: "test",
	}

	err = sss.HealthCheck(context.Background())
	assert.NoError(t, err)
}

func newTestServerAndClient(handler http.Handler) (*SimpleStorageService, *httptest.Server) {
	srv := httptest.NewServer(handler)
	var d net.Dialer
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return d.DialContext(
					ctx,
					srv.Listener.Addr().Network(),
					srv.Listener.Addr().String(),
				)
			},
			DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return d.DialContext(
					ctx,
					srv.Listener.Addr().Network(),
					srv.Listener.Addr().String(),
				)
			},
		},
	}

	s3c := s3.New(s3.Options{
		Region:     "region",
		HTTPClient: httpClient,
		Credentials: StaticCredentials{
			Key:    "test",
			Secret: "secret",
			Token:  "token",
		},
	})

	sss := &SimpleStorageService{
		client:        s3c,
		presignClient: s3.NewPresignClient(s3c),
		bucket:        "bucket",
	}
	return sss, srv
}

func TestGetObject(t *testing.T) {
	t.Parallel()

	type testCase struct {
		Name string

		CTX        context.Context
		ObjectPath string

		Handler func(t *testing.T) http.HandlerFunc
		Body    []byte
		Error   assert.ErrorAssertionFunc
	}

	testCases := []testCase{{
		Name: "ok",

		ObjectPath: "foo/bar",
		Handler: func(t *testing.T) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/foo/bar", r.URL.Path)
				assert.Equal(t, "bucket.s3.region.amazonaws.com", r.Host)

				w.WriteHeader(http.StatusOK)
				w.Write([]byte("imagine artifacts"))
			}
		},
		Body: []byte("imagine artifacts"),
	}, {
		Name: "error/object not found",

		ObjectPath: "foo/bar",
		Handler: func(t *testing.T) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/foo/bar", r.URL.Path)
				assert.Equal(t, "bucket.s3.region.amazonaws.com", r.Host)

				w.WriteHeader(http.StatusNotFound)
			}
		},
		Error: func(t assert.TestingT, err error, _ ...interface{}) bool {
			var apiErr smithy.APIError
			t1 := assert.ErrorAs(t, err, &apiErr)
			return t1 && assert.Equal(t, "NotFound", apiErr.ErrorCode())
		},
	}, {
		Name: "error/invalid settings from context",

		CTX: storage.SettingsWithContext(
			context.Background(),
			&model.StorageSettings{},
		),
		ObjectPath: "foo/bar",
		Handler: func(t *testing.T) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				assert.Fail(t, "the test was not supposed to make a request")
				w.WriteHeader(http.StatusInternalServerError)
			}
		},
		Error: func(t assert.TestingT, err error, _ ...interface{}) bool {
			var verr validation.Errors
			return assert.Error(t, err) &&
				assert.ErrorAs(t, err, &verr) &&
				assert.Contains(t, verr, "key") &&
				assert.Contains(t, verr, "secret") &&
				assert.Contains(t, verr, "bucket") &&
				assert.Contains(t, verr, "region")
		},
	}}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			s3c, srv := newTestServerAndClient(tc.Handler(t))
			defer srv.Close()
			var ctx context.Context
			if tc.CTX != nil {
				ctx = tc.CTX
			} else {
				ctx = context.Background()
			}
			obj, err := s3c.GetObject(ctx, tc.ObjectPath)
			if tc.Error != nil {
				tc.Error(t, err)
			} else if assert.NoError(t, err) {
				b, _ := io.ReadAll(obj)
				obj.Close()
				assert.Equal(t, tc.Body, b)
			}
		})
	}
}
