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

	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/deployments/storage"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/errors"
)

type settings model.StorageSettings

func settingsFromContext(ctx context.Context) *settings {
	s, _ := storage.SettingsFromContext(ctx)
	return (*settings)(s)
}

func (s settings) endpointResolver(presign bool) (resolver s3.EndpointResolver) {
	if s.Uri != "" {
		uri := s.Uri
		if s.ExternalUri != "" && presign {
			uri = s.ExternalUri
		}
		resolver = s3.EndpointResolverFromURL(
			uri,
			func(ep *aws.Endpoint) {
				ep.HostnameImmutable = s.ForcePathStyle
				if s.Region != "" {
					ep.SigningRegion = s.Region
				}
			},
		)
	}
	return resolver
}

func (s settings) credentials() StaticCredentials {
	return StaticCredentials{
		Key:    s.Key,
		Secret: s.Secret,
		Token:  s.Token,
	}
}

func (s settings) getOptions(presign bool) (func(*s3.Options), error) {
	if err := model.StorageSettings(s).Validate(); err != nil {
		return nil, errors.WithMessage(err, "s3: invalid settings")
	}
	return func(opts *s3.Options) {
		opts.Region = s.Region
		opts.Credentials = s.credentials()
		opts.UsePathStyle = s.ForcePathStyle
		opts.UseAccelerate = s.UseAccelerate
		opts.EndpointResolver = s.endpointResolver(presign)
	}, nil
}
