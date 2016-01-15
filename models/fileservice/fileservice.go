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
package fileservice

import (
	"time"
)

const (
	DefaultLinkExpire = time.Hour * 24
)

type FileServiceModelI interface {
	// Delete stored object. If not found return error.
	Delete(customerId, objectId string) error
	Exists(customerId, objectId string) (bool, error)
	LastModified(customerId, objectId string) (time.Time, error)
	PutRequest(customerId, objectId string, duration time.Duration) (*Link, error)
	GetRequest(customerId, objectId string, duration time.Duration) (*Link, error)
}
