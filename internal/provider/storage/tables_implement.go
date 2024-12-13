package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/eldarbr/go-auth/pkg/database"
	"github.com/eldarbr/go-s3/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type implTableBuckets struct{}

type implTableFiles struct{}

func (implTableBuckets) Add(ctx context.Context, querier database.Querier, bucket *model.Bucket) error {
	if querier == nil || bucket == nil {
		return database.ErrNilArgument
	}

	query := `
INSERT INTO "buckets"
  ("name",
   "owner_id",
   "availability",
   "size_quota")
VALUES
  ($1, $2, $3, $4)
RETURNING "id"
	`

	queryResult := querier.QueryRow(ctx, query, bucket.Name, bucket.OwnerID, bucket.Availability, bucket.SizeQuota)
	err := queryResult.Scan(&bucket.ID)

	if err != nil && strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		return database.ErrUniqueKeyViolation
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return database.ErrNoRows
	}

	if err != nil {
		return fmt.Errorf("implTableBuckets.Add failed on INSERT: %w", err)
	}

	return nil
}

func (implTableBuckets) UpdateByID(ctx context.Context, querier database.Querier, bucket *model.Bucket) error {
	if querier == nil || bucket == nil {
		return database.ErrNilArgument
	}

	query := `
UPDATE "buckets"
SET
  "name" = $1,
  "owner_id" = $2,
  "availability" = $3,
  "size_quota" = $4
WHERE "id" = $5
	`

	result, err := querier.Exec(ctx, query, bucket.Name, bucket.OwnerID, bucket.Availability, bucket.SizeQuota, bucket.ID)
	if err != nil && strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		return database.ErrUniqueKeyViolation
	}

	if err != nil {
		return fmt.Errorf("implTableBuckets.UpdateByID failed on UPDATE: %w", err)
	}

	if result.RowsAffected() == 0 {
		return database.ErrNoRows
	}

	return nil
}

func (implTableBuckets) GetByID(ctx context.Context, querier database.Querier, bucketID int64) (*model.Bucket, error) {
	if querier == nil {
		return nil, database.ErrNilArgument
	}

	query := `
SELECT
  "id",
  "name",
  "owner_id",
  "availability",
  "size_quota"
FROM "buckets"
WHERE "id" = $1
	`

	var dst model.Bucket

	queryResult := querier.QueryRow(ctx, query, bucketID)
	err := queryResult.Scan(&dst.ID, &dst.Name, &dst.OwnerID, &dst.Availability, &dst.SizeQuota)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, database.ErrNoRows
	}

	if err != nil {
		return nil, fmt.Errorf("implTableBuckets.GetByID failed on SELECT: %w", err)
	}

	return &dst, nil
}

func (implTableBuckets) GetByName(ctx context.Context, querier database.Querier, name string) (*model.Bucket, error) {
	if querier == nil {
		return nil, database.ErrNilArgument
	}

	query := `
SELECT
  "id",
  "name",
  "owner_id",
  "availability",
  "size_quota"
FROM "buckets"
WHERE "name" = $1
	`

	var dst model.Bucket

	queryResult := querier.QueryRow(ctx, query, name)
	err := queryResult.Scan(&dst.ID, &dst.Name, &dst.OwnerID, &dst.Availability, &dst.SizeQuota)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, database.ErrNoRows
	}

	if err != nil {
		return nil, fmt.Errorf("implTableBuckets.GetByName failed on SELECT: %w", err)
	}

	return &dst, nil
}

func (implTableBuckets) DeleteByID(ctx context.Context, querier database.Querier, bucketID int64) error {
	if querier == nil {
		return database.ErrNilArgument
	}

	query := `
DELETE FROM "buckets"
WHERE "id" = $1
	`

	result, err := querier.Exec(ctx, query, bucketID)
	if err != nil {
		return fmt.Errorf("implTableBuckets.DeleteByID failed on DELETE: %w", err)
	}

	if result.RowsAffected() == 0 {
		return database.ErrNoRows
	}

	return nil
}

func (implTableFiles) Add(ctx context.Context, querier database.Querier, file *model.File) error {
	if querier == nil || file == nil {
		return database.ErrNilArgument
	}

	query := `
INSERT INTO "files"
  ("filename",
   "mime",
   "bucket_id",
   "access",
   "size_bytes",
   "filename_suffix")
VALUES
  ($1, $2, $3, $4, $5, $6)
RETURNING "id", "created_ts"
	`

	queryResult := querier.QueryRow(ctx, query, file.Filename, file.MIME, file.BucketID, file.Access,
		file.SizeBytes, file.FilenameSuffix)
	err := queryResult.Scan(&file.ID, &file.CreatedTS)

	if err != nil && strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		return database.ErrUniqueKeyViolation
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return database.ErrNoRows
	}

	if err != nil {
		return fmt.Errorf("implTableFiles.Add failed on INSERT: %w", err)
	}

	return nil
}

func (implTableFiles) InsertID(ctx context.Context, querier database.Querier, file *model.File) error {
	if querier == nil || file == nil {
		return database.ErrNilArgument
	}

	query := `
INSERT INTO "files"
  ("id",
	 "filename",
   "mime",
   "bucket_id",
   "access",
   "size_bytes",
   "filename_suffix")
VALUES
  ($1, $2, $3, $4, $5, $6, $7)
	`

	result, err := querier.Exec(ctx, query, file.ID, file.Filename, file.MIME, file.BucketID, file.Access,
		file.SizeBytes, file.FilenameSuffix)
	if err != nil && strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		return database.ErrUniqueKeyViolation
	}

	if err != nil {
		return fmt.Errorf("implTableFiles.UpdateByID failed on UPDATE: %w", err)
	}

	if result.RowsAffected() == 0 {
		return database.ErrNoRows
	}

	return nil
}

func (implTableFiles) UpdateByID(ctx context.Context, querier database.Querier, file *model.File) error {
	if querier == nil || file == nil {
		return database.ErrNilArgument
	}

	query := `
UPDATE "files"
SET
  "filename" = $1,
  "mime" = $2,
  "bucket_id" = $3,
  "access" = $4,
  "size_bytes" = $5,
  "filename_suffix" = $6
WHERE "id" = $7 AND "is_deleted" = FALSE
	`

	result, err := querier.Exec(ctx, query, file.Filename, file.MIME, file.BucketID, file.Access, file.SizeBytes,
		file.FilenameSuffix, file.ID)
	if err != nil && strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		return database.ErrUniqueKeyViolation
	}

	if err != nil {
		return fmt.Errorf("implTableFiles.UpdateByID failed on UPDATE: %w", err)
	}

	if result.RowsAffected() == 0 {
		return database.ErrNoRows
	}

	return nil
}

func (implTableFiles) GetByID(ctx context.Context, querier database.Querier, fileID uuid.UUID) (*model.File, error) {
	if querier == nil {
		return nil, database.ErrNilArgument
	}

	query := `
SELECT
  "id",
  "filename",
  "mime",
  "created_ts",
  "bucket_id",
  "access",
  "size_bytes",
  "filename_suffix"
FROM "files"
WHERE "id" = $1 AND "is_deleted" = FALSE
	`

	var dst model.File

	queryResult := querier.QueryRow(ctx, query, fileID)
	err := queryResult.Scan(&dst.ID, &dst.Filename, &dst.MIME, &dst.CreatedTS, &dst.BucketID, &dst.Access,
		&dst.SizeBytes, &dst.FilenameSuffix)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, database.ErrNoRows
	}

	if err != nil {
		return nil, fmt.Errorf("implTableFiles.GetByID failed on SELECT: %w", err)
	}

	return &dst, nil
}

func (implTableFiles) GetFilesOfABucket(ctx context.Context, querier database.Querier,
	bucketID int64) ([]model.File, error) {
	if querier == nil {
		return nil, database.ErrNilArgument
	}

	query := `
SELECT
  "id",
  "filename",
  "mime",
  "created_ts",
  "bucket_id",
  "access",
  "size_bytes",
  "filename_suffix"
FROM "files"
WHERE "bucket_id" = $1 AND "is_deleted" = FALSE
ORDER BY "filename", "filename_suffix"
	`

	var (
		dst     []model.File
		nextDst model.File
		err     error
	)

	queryResult, err := querier.Query(ctx, query, bucketID)
	if err != nil {
		return nil, fmt.Errorf("implTableFiles.GetFilesOfABucket failed on SELECT: %w", err)
	}

	dst, err = pgx.CollectRows(queryResult, func(row pgx.CollectableRow) (model.File, error) {
		err = row.Scan(&nextDst.ID, &nextDst.Filename, &nextDst.MIME, &nextDst.CreatedTS,
			&nextDst.BucketID, &nextDst.Access, &nextDst.SizeBytes, &nextDst.FilenameSuffix)

		return nextDst, err //nolint:wrapcheck // not an actual return
	})
	if err != nil {
		return nil, fmt.Errorf("implTableFiles.GetFilesOfABucket failed on Scan: %w", err)
	}

	return dst, nil
}

func (implTableFiles) DeleteByID(ctx context.Context, querier database.Querier, fileID uuid.UUID) error {
	if querier == nil {
		return database.ErrNilArgument
	}

	query := `
DELETE FROM "files"
WHERE "id" = $1
	`

	result, err := querier.Exec(ctx, query, fileID)
	if err != nil {
		return fmt.Errorf("implTableFiles.DeleteByID failed on DELETE: %w", err)
	}

	if result.RowsAffected() == 0 {
		return database.ErrNoRows
	}

	return nil
}

func (implTableFiles) LockFilename(ctx context.Context, querier database.Querier, filename string) error {
	if querier == nil {
		return database.ErrNilArgument
	}

	query := `
SELECT 1
FROM "files"
WHERE "filename" = $1 AND "is_deleted" = FALSE
FOR UPDATE
	`

	_, err := querier.Exec(ctx, query, filename)

	if errors.Is(err, pgx.ErrNoRows) {
		return database.ErrNoRows
	}

	if err != nil {
		return fmt.Errorf("implTableFiles.GetByID failed on SELECT: %w", err)
	}

	return nil
}

func (filesTable implTableFiles) PrepareNewFilenameSuffix(ctx context.Context, querier database.Querier,
	filename string,
) (int32, error) {
	if querier == nil {
		return 0, database.ErrNilArgument
	}

	// only makes sense if the querire is a tx.
	err := filesTable.LockFilename(ctx, querier, filename)
	if err != nil {
		return 0, fmt.Errorf("implTableFiles.PrepareNewFilenameSuffix couldn't lock the filename: %w", err)
	}

	query := `
SELECT COALESCE(MAX("filename_suffix"), -1) + 1
FROM "files"
WHERE "filename" = $1
	`

	var dst int32

	queryResult := querier.QueryRow(ctx, query, filename)
	err = queryResult.Scan(&dst)

	if errors.Is(err, pgx.ErrNoRows) {
		return 0, database.ErrNoRows
	}

	if err != nil {
		return 0, fmt.Errorf("implTableFiles.PrepareNewFilenameSuffix failed on SELECT: %w", err)
	}

	return dst, nil
}

func (implTableFiles) MarkDeleted(ctx context.Context, querier database.Querier, fileID uuid.UUID) error {
	if querier == nil {
		return database.ErrNilArgument
	}

	query := `
UPDATE "files"
SET
  "is_deleted" = TRUE
WHERE "id" = $1
	`

	result, err := querier.Exec(ctx, query, fileID)
	if err != nil {
		return fmt.Errorf("implTableFiles.MarkDeleted failed on UPDATE: %w", err)
	}

	if result.RowsAffected() == 0 {
		return database.ErrNoRows
	}

	return nil
}
