package files

import (
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
		return 0, err
	}

	defer file.Close()

	err = file.Chmod(container.fileMode)
	if err != nil {
		return 0, err
	}

	written, err := io.Copy(file, src)

	return written, err
}

func (container Container) ReadFile(bucketID, fileID string, dst io.Writer) error {
	file, err := os.Open(path.Join(container.basePath, bucketID, fileID))
	if err != nil {
		return err
	}

	defer file.Close()

	_, err = io.Copy(dst, file)

	return err
}

func (container Container) CreateFolder(bucketID string) error {
	return os.Mkdir(path.Join(container.basePath, bucketID), container.dirMode)
}
