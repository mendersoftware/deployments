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
	PutRequest(customerId, objectId string, duration time.Duration) (*Link, error)
	GetRequest(customerId, objectId string, duration time.Duration) (*Link, error)
}
