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


type StorageManger struct {
	storages map[string]StorageInf
}

func NewStorageManger() *StorageManger {
	return &StorageManger{
		storages: make(map[string]StorageInf),
	}
}

func (s *StorageManger) AddStorage(storage StorageInf) error {
	s.storages[storage.GetType()] = storage
	return nil
}

func (s *StorageManger) GetStorage(storageType string) StorageInf {
	return s.storages[storageType]
}

func (s *StorageManger) RemoveStorage(storageType string) error {
	delete(s.storages, storageType)
	return nil
}

