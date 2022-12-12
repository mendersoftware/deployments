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
	"crypto/tls"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/mendersoftware/deployments/storage"
)

const (
	kib = 1024
	mib = kib * 1024

	DefaultBufferSize = 10 * mib
	DefaultExpire     = 15 * time.Minute
)

var (
	validAtLeast5MiB = validation.Min(MultipartMinSize).
		Error("must be at least 5MiB")
)

type Options struct {
	// StaticCredentials that overrides AWS config.
	StaticCredentials *StaticCredentials `json:"auth"`

	// Region where the bucket lives
	Region *string
	// ContentType of the uploaded objects
	ContentType *string
	// FilenameSuffix adds the suffix to the content-disposition for object downloads>
	FilenameSuffix *string
	// ExternalURI is the URI used for signing requests.
	ExternalURI *string
	// URI is the URI for the s3 API.
	URI *string

	// ForcePathStyle encodes bucket in the API path.
	ForcePathStyle bool
	// UseAccelerate enables s3 Accelerate
	UseAccelerate bool

	// DefaultExpire is the fallback presign expire duration
	// (defaults to 15min).
	DefaultExpire *time.Duration
	// BufferSize sets the buffer size allocated for uploads.
	// This implicitly sets the upper limit for upload size:
	// BufferSize * 10000 (defaults to: 5MiB).
	BufferSize *int
}

func NewOptions(opts ...*Options) *Options {
	defaultBufferSize := DefaultBufferSize
	ret := &Options{
		BufferSize: &defaultBufferSize,
	}
	for _, opt := range opts {
		if opt.StaticCredentials != nil {
			ret.StaticCredentials = opt.StaticCredentials
		}
		if opt.Region != nil {
			ret.Region = opt.Region
		}
		if opt.ContentType != nil {
			ret.ContentType = opt.ContentType
		}
		if opt.ExternalURI != nil {
			ret.ExternalURI = opt.ExternalURI
		}
		if opt.URI != nil {
			ret.URI = opt.URI
		}
		if opt.ForcePathStyle != ret.ForcePathStyle {
			ret.ForcePathStyle = opt.ForcePathStyle
		}
		if opt.UseAccelerate != ret.UseAccelerate {
			ret.UseAccelerate = opt.UseAccelerate
		}
		if opt.DefaultExpire != nil {
			ret.DefaultExpire = opt.DefaultExpire
		}
		if opt.BufferSize != nil {
			ret.BufferSize = opt.BufferSize
		}
	}
	return ret
}

func (opts Options) Validate() error {
	return validation.ValidateStruct(&opts,
		validation.Field(&opts.StaticCredentials),
		validation.Field(&opts.BufferSize, validAtLeast5MiB),
	)
}

func (opts *Options) SetStaticCredentials(key, secret, sessionToken string) *Options {
	opts.StaticCredentials = &StaticCredentials{
		Key:    key,
		Secret: secret,
		Token:  sessionToken,
	}
	return opts
}

func (opts *Options) SetRegion(region string) *Options {
	opts.Region = &region
	return opts
}

func (opts *Options) SetContentType(contentType string) *Options {
	opts.ContentType = &contentType
	return opts
}

func (opts *Options) SetFilenameSuffix(suffix string) *Options {
	opts.FilenameSuffix = &suffix
	return opts
}

func (opts *Options) SetExternalURI(externalURI string) *Options {
	opts.ExternalURI = &externalURI
	return opts
}

func (opts *Options) SetURI(URI string) *Options {
	opts.URI = &URI
	return opts
}

func (opts *Options) SetForcePathStyle(forcePathStyle bool) *Options {
	opts.ForcePathStyle = forcePathStyle
	return opts
}

func (opts *Options) SetUseAccelerate(useAccelerate bool) *Options {
	opts.UseAccelerate = useAccelerate
	return opts
}

func (opts *Options) SetDefaultExpire(defaultExpire time.Duration) *Options {
	opts.DefaultExpire = &defaultExpire
	return opts
}

func (opts *Options) SetBufferSize(bufferSize int) *Options {
	opts.BufferSize = &bufferSize
	return opts
}

func (opts *Options) toS3Options() (
	clientOpts func(*s3.Options),
	presignOpts func(*s3.PresignOptions),
) {
	clientOpts = func(s3Opts *s3.Options) {
		if opts.StaticCredentials != nil {
			s3Opts.Credentials = *opts.StaticCredentials
		}
		if opts.Region != nil {
			s3Opts.Region = *opts.Region
		}
		if opts.URI != nil {
			endpointURI := *opts.URI
			s3Opts.EndpointResolver = s3.EndpointResolverFromURL(endpointURI,
				func(ep *aws.Endpoint) {
					ep.HostnameImmutable = true
				},
			)
		}
		s3Opts.UsePathStyle = opts.ForcePathStyle
		s3Opts.UseAccelerate = opts.UseAccelerate
		s3Opts.HTTPClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: storage.GetRootCAs(),
				},
			},
		}
	}

	expires := DefaultExpire
	if opts.DefaultExpire != nil {
		expires = *opts.DefaultExpire
	}
	presignOpts = func(s3Opts *s3.PresignOptions) {
		s3.WithPresignExpires(expires)(s3Opts)
		if opts.ExternalURI != nil {
			presignURL := *opts.ExternalURI
			resolver := s3.EndpointResolverFromURL(presignURL,
				func(ep *aws.Endpoint) {
					ep.HostnameImmutable = opts.ForcePathStyle
				},
			)
			s3.WithPresignClientFromClientOptions(
				s3.WithEndpointResolver(resolver),
			)(s3Opts)
		}
	}
	return clientOpts, presignOpts
}
