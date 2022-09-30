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

package model

import (
	"time"
)

type Link struct {
	Uri    string            `json:"uri"`
	Expire time.Time         `json:"expire,omitempty"`
	Method string            `json:"method,omitempty"`
	Header map[string]string `json:"header,omitempty"`
}

func NewLink(uri string, expire time.Time) *Link {
	return &Link{
		Uri:    uri,
		Expire: expire,
	}
}
