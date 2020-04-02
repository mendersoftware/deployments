// Copyright 2020 Northern.tech AS
//
//    All Rights Reserved

package mgoutils

import (
	"bytes"
	"encoding/json"

	"go.mongodb.org/mongo-driver/mongo"
	"gopkg.in/yaml.v2"
)

// IndexError wraps a parsed "duplicate key error" (E11000) with the key/value
// conflicts contained in the IndexConflict property.
type IndexError struct {
	IndexConflict map[string]interface{}
	err           error
}

func NewIndexError(err error) *IndexError {
	var idxErr IndexError
	var errs mongo.WriteErrors

	if we, ok := err.(mongo.WriteException); ok {
		errs = we.WriteErrors
	} else if we, ok := err.(mongo.BulkWriteException); ok {
		errs = make([]mongo.WriteError, len(we.WriteErrors))
		for i, bwe := range we.WriteErrors {
			errs[i] = bwe.WriteError
		}
	} else {
		return nil
	}

	for _, err := range errs {
		b := []byte(err.Message)
		start := bytes.Index(b, []byte{'{'})
		if start == -1 {
			return nil
		}
		idxErr.IndexConflict = make(map[string]interface{})
		err := yaml.Unmarshal(b[start:], &idxErr.IndexConflict)
		if err != nil {
			return nil
		}
	}
	return &idxErr
}

func (ie *IndexError) Error() string {
	return ie.String()
}

func (ie *IndexError) String() string {
	out, err := json.Marshal(ie.IndexConflict)
	if err != nil {
		return ""
	}
	return string(out)
}
