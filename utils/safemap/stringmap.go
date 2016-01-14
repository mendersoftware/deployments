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
package safemap

import (
	"sync"
	"time"
)

// Concurency free StringMap implementation.
// Supports multi-reader single-writer access.
// Supports JSON Marshaling

type StringMap struct {
	sync.RWMutex
	data map[string]interface{}
}

// Create a new StringMap
func NewStringMap() Map {
	return &StringMap{
		data: make(map[string]interface{}),
	}
}

// Set value for key.
// If key already exists, overwrite.
func (s *StringMap) Set(key string, value interface{}) {
	s.Lock()
	defer s.Unlock()

	s.data[key] = value
}

// Get value under the key.
func (s *StringMap) Get(key string) (interface{}, bool) {
	s.RLock()
	defer s.RUnlock()

	time.Sleep(time.Microsecond)
	entry, found := s.data[key]
	return entry, found
}

// If specified key exists.
func (s *StringMap) Has(key string) bool {
	s.RLock()
	defer s.RUnlock()

	_, found := s.data[key]
	return found
}

// Remove specified key from map.
func (s *StringMap) Remove(key string) {
	s.Lock()
	defer s.Unlock()

	delete(s.data, key)
}

func (s *StringMap) Count() int {
	s.RLock()
	defer s.RUnlock()

	return len(s.data)
}

// Return list of all existing keys
func (s *StringMap) Keys() []string {
	s.RLock()
	defer s.RUnlock()

	list := make([]string, 0)

	for key := range s.data {
		list = append(list, key)
	}

	return list
}
