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

package azblob

const (
	BufferSizeMin     = 4 * 1024          // 4KiB
	BufferSizeDefault = 8 * BufferSizeMin // 32KiB - same default as used in io.Copy
)

type Options struct {
	ConnectionString *string
	SharedKey        *SharedKeyCredentials

	BufferSize int

	ContentType    *string
	FilenameSuffix *string
}

func NewOptions(opts ...*Options) *Options {
	opt := &Options{
		BufferSize: BufferSizeDefault,
	}
	for _, o := range opts {
		if o == nil {
			continue
		}
		if o.ConnectionString != nil {
			opt.ConnectionString = o.ConnectionString
		}
		if o.SharedKey != nil {
			opt.SharedKey = o.SharedKey
		}
		if o.ContentType != nil {
			opt.ContentType = o.ContentType
		}
		if o.FilenameSuffix != nil {
			opt.FilenameSuffix = o.FilenameSuffix
		}
		if o.BufferSize >= BufferSizeMin {
			opt.BufferSize = o.BufferSize
		}
	}
	return opt
}

func (opts *Options) SetConnectionString(connStr string) *Options {
	opts.ConnectionString = &connStr
	return opts
}

func (opts *Options) SetSharedKey(sk SharedKeyCredentials) *Options {
	opts.SharedKey = &sk
	return opts
}

func (opts *Options) SetContentType(typ string) *Options {
	opts.ContentType = &typ
	return opts
}

func (opts *Options) SetFilenameSuffix(suffix string) *Options {
	opts.FilenameSuffix = &suffix
	return opts
}

func (opts *Options) SetBufferSize(size int) *Options {
	opts.BufferSize = size
	return opts
}
