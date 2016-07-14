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
package identity

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func makeClaimsPart(sub string) string {
	claim := struct {
		Subject string `json:"sub"`
	}{
		Subject: sub,
	}

	data, _ := json.Marshal(&claim)
	rawclaim := base64.StdEncoding.EncodeToString(data)

	return rawclaim
}

func TestExtractIdentity(t *testing.T) {
	_, err := ExtractIdentity("foo")
	assert.Error(t, err)

	_, err = ExtractIdentity("foo.bar")
	assert.Error(t, err)

	_, err = ExtractIdentity("foo.bar.baz")
	assert.Error(t, err)

	// should fail, token is malformed, missing header & signature
	rawclaims := makeClaimsPart("foobar")
	_, err = ExtractIdentity(rawclaims)
	assert.Error(t, err)

	// correct cate
	idata, err := ExtractIdentity("foo." + rawclaims + ".bar")
	assert.NoError(t, err)
	assert.Equal(t, Identity{Subject: "foobar"}, idata)

	// missing subject
	enc := base64.StdEncoding.EncodeToString([]byte(`{"iss": "Mender"}`))
	_, err = ExtractIdentity("foo." + enc + ".bar")
	assert.Error(t, err)

	// bad subject
	enc = base64.StdEncoding.EncodeToString([]byte(`{"sub": 1}`))
	_, err = ExtractIdentity("foo." + enc + ".bar")
	assert.Error(t, err)
}

func TestExtractIdentityFromHeaders(t *testing.T) {
	h := http.Header{}
	_, err := ExtractIdentityFromHeaders(h)
	assert.Error(t, err)

	h.Set("Authorization", "Basic foobar")
	_, err = ExtractIdentityFromHeaders(h)
	assert.Error(t, err)

	h.Set("Authorization", "Bearer")
	_, err = ExtractIdentityFromHeaders(h)
	assert.Error(t, err)

	// correct cate
	rawclaims := makeClaimsPart("foobar")
	h.Set("Authorization", "Bearer foo."+rawclaims+".bar")
	idata, err := ExtractIdentityFromHeaders(h)
	assert.NoError(t, err)
	assert.Equal(t, Identity{Subject: "foobar"}, idata)
}

func TestDecodeClaims(t *testing.T) {
	// malformed tokens
	_, err := decodeClaims("foo")
	assert.Error(t, err)

	_, err = decodeClaims("foo.bar")
	assert.Error(t, err)

	_, err = decodeClaims("foo.bar.baz")
	assert.Error(t, err)

	// should fail, token is malformed, missing header & signature
	rawclaims := makeClaimsPart("foobar")
	_, err = decodeClaims(rawclaims)
	assert.Error(t, err)

	// malformed base64 claims part
	_, err = decodeClaims("foo.00" + rawclaims + ".bar")
	assert.Error(t, err)

	// malformed json
	enc := base64.StdEncoding.EncodeToString([]byte(`"sub": 1}`))
	_, err = ExtractIdentity("foo." + enc + ".bar")

	assert.Error(t, err)
	// correct token
	claims, err := decodeClaims("foo." + rawclaims + ".bar")
	assert.NoError(t, err)
	assert.NotEmpty(t, claims)

	sub, ok := claims[subjectClaim]
	assert.True(t, ok)
	assert.Equal(t, "foobar", sub)
}
