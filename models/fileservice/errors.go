package fileservice

import "errors"

var (
	ErrNotFound error = errors.New("File not found")
)
