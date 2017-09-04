// Copyright 2017 Northern.tech AS
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
	"strings"

	"github.com/pkg/errors"
)

const (
	IdentityContextKey = "github.com/mendersoftware/go-lib-micro/identity.Identity"
)

// Token field names
const (
	subjectClaim = "sub"
	tenantClaim  = "mender.tenant"
)

type Identity struct {
	Subject string
	Tenant  string
}

type rawClaims map[string]interface{}

func decodeClaims(token string) (rawClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("incorrect token format")
	}

	b64claims := parts[1]
	// add padding as needed
	if pad := len(b64claims) % 4; pad != 0 {
		b64claims += strings.Repeat("=", 4-pad)
	}

	rawclaims, err := base64.StdEncoding.DecodeString(b64claims)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode raw claims %v",
			b64claims)
	}

	var claims rawClaims
	err = json.Unmarshal(rawclaims, &claims)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode claims")
	}

	return claims, nil
}

func getTenant(claims rawClaims) (string, error) {
	rawTenant, ok := claims[tenantClaim]
	if !ok {
		return "", nil
	}

	tenant, ok := rawTenant.(string)
	if !ok {
		return "", errors.Errorf("invalid tenant ID format")
	}
	return tenant, nil
}

func getSubject(claims rawClaims) (string, error) {
	rawsub, ok := claims[subjectClaim]
	if !ok {
		return "", errors.Errorf("subject claim not found")
	}

	sub, ok := rawsub.(string)
	if !ok {
		return "", errors.Errorf("invalid subject format")
	}
	return sub, nil
}

// Generate identity information from given JWT by extracting subject and tenant claims.
// Note that this function does not perform any form of token signature
// verification.
func ExtractIdentity(token string) (Identity, error) {
	claims, err := decodeClaims(token)
	if err != nil {
		return Identity{}, err
	}

	sub, err := getSubject(claims)
	if err != nil {
		return Identity{}, err
	}

	tenant, err := getTenant(claims)
	if err != nil {
		return Identity{}, err
	}

	return Identity{Subject: sub, Tenant: tenant}, nil
}

// Extract identity information from HTTP Authorization header. The header is
// assumed to contain data in format: `Bearer <token>`
func ExtractIdentityFromHeaders(headers http.Header) (Identity, error) {
	auth := strings.Split(headers.Get("Authorization"), " ")

	if len(auth) != 2 {
		return Identity{}, errors.Errorf("malformed authorization data")
	}

	if auth[0] != "Bearer" {
		return Identity{}, errors.Errorf("unknown authorization method %v", auth[0])
	}

	return ExtractIdentity(auth[1])
}
