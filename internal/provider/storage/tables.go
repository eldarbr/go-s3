package storage

import (
	"context"

	"github.com/eldarbr/go-auth/pkg/database"
	"github.com/eldarbr/go-s3/internal/model"
	"github.com/google/uuid"
)

//nolint:gochecknoinits // Set default implementations.
func init() {
	TableBuckets = implTableBuckets{}
	TableFiles = implTableFiles{}
}

var TableBuckets interface {
	Add(ctx context.Context, querier database.Querier, bucket *model.Bucket) error
	UpdateByID(ctx context.Context, querier database.Querier, bucket *model.Bucket) error
	// UpdateByName(ctx context.Context, querier database.Querier, bucket *model.Bucket, name string) error
	GetByID(ctx context.Context, querier database.Querier, id int64) (*model.Bucket, error)
	GetByName(ctx context.Context, querier database.Querier, name string) (*model.Bucket, error)
	DeleteByID(ctx context.Context, querier database.Querier, id int64) error
	// DeleteByName(ctx context.Context, querier database.Querier, name string) error
}

var TableFiles interface {
	Add(ctx context.Context, querier database.Querier, file *model.File) error
	InsertID(ctx context.Context, querier database.Querier, file *model.File) error
	UpdateByID(ctx context.Context, querier database.Querier, file *model.File) error
	GetByID(ctx context.Context, querier database.Querier, fileID uuid.UUID) (*model.File, error)
	GetFilesOfABucket(ctx context.Context, querier database.Querier, bucketID int64) ([]model.File, error)
	DeleteByID(ctx context.Context, querier database.Querier, fileID uuid.UUID) error
	LockFilename(ctx context.Context, querier database.Querier, filename string) error
	PrepareNewFilenameSuffix(ctx context.Context, querier database.Querier, filename string) (int32, error)
}
