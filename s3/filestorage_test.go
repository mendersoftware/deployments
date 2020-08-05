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

package s3

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/assert"
)

func TestListBuckets(t *testing.T) {
	t.Parallel()

	expiredContext, cancel := context.WithTimeout(
		context.TODO(), -time.Second,
	)
	defer cancel()

	// Need to redefine s3/ListBucketsOutput in order to get the correct
	// xml encoding.
	type Owner struct {
		DisplayName string
		ID          string
	}
	type Bucket struct {
		Name         string
		CreationDate time.Time
	}
	type BucketList []struct {
		Bucket Bucket
	}
	type ListBucketsOutput struct {
		Owner   Owner
		Buckets BucketList
	}

	testCases := []struct {
		Name string

		Ctx context.Context

		ResponseCode  int
		ResponseBody  interface{}
		ResponseHdrs  http.Header
		OutputBuckets []string

		Error error
	}{{
		Name: "ok",

		ResponseCode: http.StatusOK,
		ResponseBody: ListBucketsOutput{
			Buckets: BucketList{
				{Bucket: Bucket{
					Name:         "test",
					CreationDate: time.Now().UTC(),
				}},
			},
			Owner: Owner{
				DisplayName: "test-name",
				ID:          "123",
			},
		},
		OutputBuckets: []string{"test"},
	}, {
		Name:  "error context deadline exceeded",
		Ctx:   expiredContext,
		Error: context.DeadlineExceeded,
	}}
	responseChan := make(chan *http.Response, 1)
	handler := func(w http.ResponseWriter, r *http.Request) {
		var rsp *http.Response
		select {
		case rsp = <-responseChan:

		default:
			panic(fmt.Sprintf(
				"[PROG ERR]: I don't know what to respond! %v",
				*r,
			))
		}
		for k, v := range rsp.Header {
			for _, vv := range v {
				w.Header().Add(k, vv)
			}
		}
		w.WriteHeader(rsp.StatusCode)
		if rsp.Body != nil {
			_, err := io.Copy(w, rsp.Body)
			if err != nil {
				panic(fmt.Sprintf("[PROG ERR]: %s", err))
			}
		}
	}
	srv := httptest.NewTLSServer(http.HandlerFunc(handler))
	srvURL, _ := url.Parse(srv.URL)

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			httpClient := http.Client{
				Transport: &http.Transport{
					DialTLS: func(network, addr string) (net.Conn, error) {
						return tls.Dial(network, srvURL.Host, &tls.Config{
							InsecureSkipVerify: true,
						})
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
			if tc.ResponseCode != 0 {
				var rsp = http.Response{
					StatusCode: tc.ResponseCode,
				}
				if tc.ResponseBody != nil {
					b, _ := xml.Marshal(tc.ResponseBody)
					b = append([]byte(xml.Header), b...)
					rsp.Body = ioutil.NopCloser(bytes.NewReader(b))
					rsp.Header = http.Header{
						"Content-Type": []string{"text/xml"},
					}
				}
				responseChan <- &rsp
			}

			buckets, err := sss.ListBuckets(tc.Ctx)
			if tc.Error != nil {
				if assert.Error(t, err) {
					assert.Contains(t, err.Error(), tc.Error.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, buckets, tc.OutputBuckets)
			}
		})
	}
}
