// Copyright 2020 Northern.tech AS
//
//    All Rights Reserved

package mgoutils

import (
	"reflect"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
)

// DocumentFromStruct creates a bson document from a struct in the order of the
// underlying data structure. Additional fields can be appended to the struct
// with the appendElements, these fields will be added at the end of the
// document.
func DocumentFromStruct(
	sct interface{},
	appendElements ...bson.E,
) (doc bson.D) {
	s := reflect.ValueOf(sct)
	defer func() {
		if r := recover(); r != nil {
			doc = nil
		}
	}()

	if s.Kind() == reflect.Ptr {
		s = s.Elem()
	}
	if s.Kind() == reflect.Interface {
		s = s.Elem()
	}

	numAppends := len(appendElements)
	numFields := s.NumField()
	doc = make(bson.D, 0, numFields)
	fields := s.Type()
	for i := 0; i < numFields; i++ {
		field := fields.Field(i)
		value := s.Field(i)
		tag, ok := field.Tag.Lookup("bson")
		if !ok {
			tag = strings.ToLower(field.Name)
		}
		if tags := strings.Split(tag, ","); len(tags) > 1 {
			if tags[1] == "omitempty" &&
				value.Interface() == reflect.Zero(
					value.Type()).Interface() {
				continue
			}
			tag = tags[0]
		}
		doc = append(doc, bson.E{Key: tag, Value: value.Interface()})
	}
	for i := 0; i < numAppends; i++ {
		doc = append(doc, appendElements[i])
	}
	return doc
}
