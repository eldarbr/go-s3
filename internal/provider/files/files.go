package files

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
)

type Container struct {
	basePath string
	fileMode fs.FileMode
	dirMode  fs.FileMode
}

func NewContainer(basePath string, fileMode, dirMode fs.FileMode) *Container {
	return &Container{
		basePath: basePath,
		fileMode: fileMode,
		dirMode:  dirMode,
	}
}

func (container Container) WriteFile(bucketID, fileID string, src io.Reader) (int64, error) {
	fullPath := path.Join(container.basePath, bucketID, fileID)

	file, err := os.Create(fullPath)
	if err != nil {
		return 0, fmt.Errorf("WriteFile os.Create %w", err)
	}

	defer file.Close()

	err = file.Chmod(container.fileMode)
	if err != nil {
		return 0, fmt.Errorf("WriteFile file.Chmod %w", err)
	}

	written, err := io.Copy(file, src)

	return written, fmt.Errorf("WriteFile io.Copy %w", err)
}

func (container Container) OpenFile(bucketID, fileID string) (io.ReadSeekCloser, error) {
	reader, err := os.Open(path.Join(container.basePath, bucketID, fileID))
	if err != nil {
		return nil, fmt.Errorf("OpenFile os.Open %w", err)
	}

	return reader, nil
}

func (container Container) CreateFolder(bucketID string) error {
	err := os.Mkdir(path.Join(container.basePath, bucketID), container.dirMode)
	if err != nil {
		return fmt.Errorf("CreateFolder os.Mkdir %w", err)
	}

	return nil
}
