// Copyright 2023 Northern.tech AS
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
	"crypto/tls"
	"net/http"
	"net/textproto"
	"net/url"
	"time"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/mendersoftware/deployments/model"
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

// Subset of model.StorageSettings applicable to s3.
type storageSettings struct {
	// BucketName defines the bucket name.
	BucketName *string

	// StaticCredentials that overrides AWS config.
	StaticCredentials *StaticCredentials `json:"auth"`

	// Region where the bucket lives
	Region *string
	// ExternalURI is the URI used for signing requests.
	ExternalURI *string
	// URI is the URI for the s3 API.
	URI *string
	// ProxyURI is used for rewriting presigned requests, pointing the
	// requests to the proxy URL instead of the direct URL to s3.
	ProxyURI *url.URL

	// ForcePathStyle encodes bucket in the API path.
	ForcePathStyle bool
	// UseAccelerate enables s3 Accelerate
	UseAccelerate bool
}

func newFromParent(defaults *storageSettings, parent *model.StorageSettings) *storageSettings {
	ret := new(storageSettings)
	ret.patch(defaults)
	if parent.Bucket != "" {
		ret.BucketName = &parent.Bucket
	}
	if parent.Region != "" {
		ret.Region = &parent.Region
	}
	if parent.Key != "" && parent.Secret != "" {
		ret.StaticCredentials = &StaticCredentials{
			Key:    parent.Key,
			Secret: parent.Secret,
			Token:  parent.Token,
		}
	}
	if parent.ExternalUri != "" {
		ret.ExternalURI = &parent.ExternalUri
	}
	if parent.Uri != "" {
		ret.URI = &parent.Uri
	}
	if parent.ForcePathStyle != ret.ForcePathStyle {
		ret.ForcePathStyle = parent.ForcePathStyle
	}
	if parent.UseAccelerate != ret.UseAccelerate {
		ret.UseAccelerate = parent.UseAccelerate
	}
	return ret
}

func (s storageSettings) Validate() error {
	return validation.ValidateStruct(&s,
		validation.Field(&s.StaticCredentials),
	)
}

func (s storageSettings) options(opts *s3.Options) {
	if s.StaticCredentials != nil {
		opts.Credentials = *s.StaticCredentials
	}
	if s.Region != nil {
		opts.Region = *s.Region
	}
	if s.URI != nil {
		endpointURI := *s.URI
		opts.BaseEndpoint = &endpointURI
	}
	opts.UsePathStyle = s.ForcePathStyle
	opts.UseAccelerate = s.UseAccelerate
}

func (s storageSettings) presignOptions(opts *s3.PresignOptions) {
	if s.ExternalURI != nil {
		presignURL := *s.ExternalURI
		applyURL := func(opts *s3.Options) {
			opts.BaseEndpoint = &presignURL
		}
		s3.WithPresignClientFromClientOptions(
			s.options,
			applyURL,
		)(opts)
	}
}

func (s *storageSettings) patch(setting *storageSettings) *storageSettings {
	if setting == nil {
		return s
	}
	if setting.BucketName != nil {
		s.BucketName = setting.BucketName
	}
	if setting.StaticCredentials != nil {
		s.StaticCredentials = setting.StaticCredentials
	}
	if setting.Region != nil {
		s.Region = setting.Region
	}
	if setting.ExternalURI != nil {
		s.ExternalURI = setting.ExternalURI
	}
	if setting.URI != nil {
		s.URI = setting.URI
	}
	if setting.ProxyURI != nil {
		s.ProxyURI = setting.ProxyURI
	}
	if setting.ForcePathStyle != s.ForcePathStyle {
		s.ForcePathStyle = setting.ForcePathStyle
	}
	if setting.UseAccelerate != s.UseAccelerate {
		s.UseAccelerate = setting.UseAccelerate
	}
	return s
}

type Options struct {
	storageSettings

	// ContentType of the uploaded objects
	ContentType *string
	// FilenameSuffix adds the suffix to the content-disposition for object downloads
	FilenameSuffix *string
	// DefaultExpire is the fallback presign expire duration
	// (defaults to 15min).
	DefaultExpire *time.Duration
	// BufferSize sets the buffer size allocated for uploads.
	// This implicitly sets the upper limit for upload size:
	// BufferSize * 10000 (defaults to: 5MiB).
	BufferSize *int

	// UnsignedHeaders forces the driver to skip the named headers from the
	// being signed.
	UnsignedHeaders []string

	// Transport sets an alternative RoundTripper used by the Go HTTP
	// client.
	Transport http.RoundTripper
}

func NewOptions(opts ...*Options) *Options {
	defaultBufferSize := DefaultBufferSize
	ret := &Options{
		BufferSize: &defaultBufferSize,
	}
	for _, opt := range opts {
		ret.storageSettings.patch(&opt.storageSettings)
		if opt.DefaultExpire != nil {
			ret.DefaultExpire = opt.DefaultExpire
		}
		if opt.ContentType != nil {
			ret.ContentType = opt.ContentType
		}
		if opt.BufferSize != nil {
			ret.BufferSize = opt.BufferSize
		}
		if opt.UnsignedHeaders != nil {
			ret.UnsignedHeaders = opt.UnsignedHeaders
		}
		if opt.Transport != nil {
			ret.Transport = opt.Transport
		}
	}
	return ret
}

func (opts Options) Validate() error {
	return validation.ValidateStruct(&opts,
		validation.Field(&opts.storageSettings),
		validation.Field(&opts.BufferSize, validAtLeast5MiB),
	)
}

func (opts *Options) SetBucketName(bucketName string) *Options {
	opts.storageSettings.BucketName = &bucketName
	return opts
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

func (opts *Options) SetProxyURI(proxyURI *url.URL) *Options {
	opts.ProxyURI = proxyURI
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

func (opts *Options) SetUnsignedHeaders(unsignedHeaders []string) *Options {
	opts.UnsignedHeaders = unsignedHeaders
	return opts
}

func (opts *Options) SetTransport(transport http.RoundTripper) *Options {
	opts.Transport = transport
	return opts
}

type apiOptions func(*middleware.Stack) error

// Google Cloud Storage does not tolerate signing the Accept-Encoding header
func unsignedHeadersMiddleware(headers []string) apiOptions {
	signMiddlewareID := (&v4.SignHTTPRequestMiddleware{}).ID()
	for i := range headers {
		headers[i] = textproto.CanonicalMIMEHeaderKey(headers[i])
	}
	return func(stack *middleware.Stack) error {
		if _, ok := stack.Finalize.Get("Signing"); !ok {
			// If the operation does not invoke signing, we're done.
			return nil
		}
		// ... -> RemoveUnsignedHeaders -> Signing -> AddUnsignedHeaders
		var unsignedHeaders http.Header = make(http.Header)
		err := stack.Finalize.Insert(middleware.FinalizeMiddlewareFunc(
			"RemoveUnsignedHeaders", func(
				ctx context.Context,
				in middleware.FinalizeInput,
				next middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				if req, ok := in.Request.(*smithyhttp.Request); ok {
					for _, hdr := range headers {
						if value, ok := req.Header[hdr]; ok {
							unsignedHeaders[hdr] = value
							req.Header.Del(hdr)
						}
					}
				}
				return next.HandleFinalize(ctx, in)
			}), signMiddlewareID, middleware.Before)
		if err != nil {
			return err
		}
		return stack.Finalize.Insert(middleware.FinalizeMiddlewareFunc(
			"AddUnsignedHeaders", func(
				ctx context.Context,
				in middleware.FinalizeInput,
				next middleware.FinalizeHandler,
			) (middleware.FinalizeOutput, middleware.Metadata, error) {
				if req, ok := in.Request.(*smithyhttp.Request); ok {
					for key, value := range unsignedHeaders {
						req.Header[key] = value
					}
				}
				return next.HandleFinalize(ctx, in)
			}), signMiddlewareID, middleware.After)
	}
}

func (opts *Options) toS3Options() (
	clientOpts func(*s3.Options),
	presignOpts func(*s3.PresignOptions),
) {
	clientOpts = func(s3Opts *s3.Options) {
		opts.options(s3Opts)
		if len(opts.UnsignedHeaders) > 0 {
			s3Opts.APIOptions = append(
				s3Opts.APIOptions,
				unsignedHeadersMiddleware(opts.UnsignedHeaders),
			)
		}
		roundTripper := opts.Transport
		if roundTripper == nil {
			roundTripper = &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: storage.GetRootCAs(),
				},
			}
		}
		s3Opts.HTTPClient = &http.Client{
			Transport: roundTripper,
		}
	}

	expires := DefaultExpire
	if opts.DefaultExpire != nil {
		expires = *opts.DefaultExpire
	}
	presignOpts = func(s3Opts *s3.PresignOptions) {
		opts.presignOptions(s3Opts)
		s3.WithPresignExpires(expires)(s3Opts)
	}
	return clientOpts, presignOpts
}
