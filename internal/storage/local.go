package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

// LocalStorage stores objects on the local filesystem.
type LocalStorage struct {
	Dir string
}

func NewLocalStorage(dir string) *LocalStorage {
	return &LocalStorage{Dir: dir}
}

func (l *LocalStorage) Name() string {
	return "local"
}

func (l *LocalStorage) Put(_ context.Context, objectKey string, body io.Reader, _ string) error {
	path := filepath.Join(l.Dir, filepath.FromSlash(objectKey))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, body)
	return err
}

func (l *LocalStorage) Delete(_ context.Context, objectKey string) error {
	path := filepath.Join(l.Dir, filepath.FromSlash(objectKey))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
