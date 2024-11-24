package business

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/eldarbr/go-auth/pkg/database"
	"github.com/eldarbr/go-s3/internal/model"
	"github.com/eldarbr/go-s3/internal/provider/storage"
)

type BusinessModule struct {
	dbInstance  *database.Database
	fileStorage FileStorage
}

type FileStorage interface {
	CreateFolder(bucketID string) error
	ReadFile(bucketID, fileID string, dst io.Writer) error
	WriteFile(bucketID, fileID string, src io.Reader) error
}

func NewBusinessModule(dbInstance *database.Database, fileStorage FileStorage) *BusinessModule {
	return &BusinessModule{
		dbInstance:  dbInstance,
		fileStorage: fileStorage,
	}
}

func (business BusinessModule) CreateBucket(ctx context.Context, bucket *model.Bucket) error {
	tx, err := business.dbInstance.GetPool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("business.UploadFile begin tx: %w", err)
	}

	defer tx.Rollback(ctx) //nolint:errcheck // won't check

	err = storage.TableBuckets.Add(ctx, business.dbInstance.GetPool(), bucket)
	if err != nil {
		return fmt.Errorf("business.CreateBucket TableBuckets.Add: %w", err)
	}

	err = business.fileStorage.CreateFolder(strconv.FormatInt(bucket.ID, 10))
	if err != nil {
		return fmt.Errorf("business.CreateBucket fileStorage.CreateFolder: %w", err)
	}

	err = tx.Commit(ctx)

	return err
}

func (business BusinessModule) UploadFile(ctx context.Context, request model.UploadFileRequest) error {
	tx, err := business.dbInstance.GetPool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("business.UploadFile begin tx: %w", err)
	}

	defer tx.Rollback(ctx) //nolint:errcheck // won't check

	err = storage.TableFiles.Add(ctx, tx, &request.File)
	if err != nil {
		return fmt.Errorf("business.UploadFile TableFiles.Add: %w", err)
	}

	err = business.fileStorage.WriteFile(strconv.FormatInt(request.BucketID, 10),
		strconv.FormatInt(request.File.ID, 10),
		request.FileContent)
	if err != nil {
		return fmt.Errorf("business.UploadFile fileStorage.WriteFile: %w", err)
	}

	err = tx.Commit(ctx)

	return err
}
