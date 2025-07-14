package storages

import (
	"context"
	"io"
)


type StorageInf interface {
	GetType() string
	Upload(ctx context.Context,objtectKey string,data io.Reader) error
	Download(ctx context.Context,objtectKey string) (io.ReadCloser,error)
}

type StorageMangerInf interface {
	AddStorage(storage StorageInf) error
	GetStorage(storageType string) StorageInf
	RemoveStorage(storageType string) error
}