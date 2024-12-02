package storage_test

import (
	"context"
	"flag"
	"testing"

	"github.com/eldarbr/go-auth/pkg/database"
	"github.com/eldarbr/go-s3/internal/model"
	"github.com/eldarbr/go-s3/internal/provider/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func clearTables(t *testing.T) {
	t.Helper()

	_, err := testDB.GetPool().Exec(context.Background(), "TRUNCATE TABLE files RESTART IDENTITY CASCADE")
	require.NoError(t, err)

	_, err = testDB.GetPool().Exec(context.Background(), "TRUNCATE TABLE buckets RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

var testDBUri = flag.String("t-db-uri", "", "perform sql tests on the `t-db-uri` database")

var testDB *database.Database

func TestMain(m *testing.M) {
	flag.Parse()

	if testDBUri != nil && *testDBUri != "" {
		testDB, _ = database.Setup(context.Background(), *testDBUri, "file://./sql")

		defer testDB.ClosePool()
	}

	m.Run()
}

func checkDB(t *testing.T) {
	t.Helper()

	pool := testDB.GetPool()
	if pool == nil {
		t.Skip("database was not initialized")
	}
}

func TestTableBucketsIntegration(t *testing.T) {
	checkDB(t)
	clearTables(t)

	ctx := context.Background()
	querier := testDB.GetPool()

	// Add
	bucket := &model.Bucket{
		Name:         "TestBucket",
		Availability: model.BucketAvailabilityAccessible,
		OwnerID:      uuid.New(),
		SizeQuota:    1024,
	}
	err := storage.TableBuckets.Add(ctx, querier, bucket)
	require.NoError(t, err)

	// Add - duplicate name
	duplicateBucket := &model.Bucket{
		Name:         "TestBucket",
		Availability: model.BucketAvailabilityAccessible,
		OwnerID:      uuid.New(),
		SizeQuota:    1024,
	}
	err = storage.TableBuckets.Add(ctx, querier, duplicateBucket)
	require.ErrorIs(t, err, database.ErrUniqueKeyViolation)

	// GetByID
	retrievedBucket, err := storage.TableBuckets.GetByID(ctx, querier, bucket.ID)
	require.NoError(t, err)
	assert.Equal(t, bucket.Name, retrievedBucket.Name)

	// GetByID - not found
	_, err = storage.TableBuckets.GetByID(ctx, querier, -1)
	require.ErrorIs(t, err, database.ErrNoRows)

	// GetByName
	retrievedBucketByName, err := storage.TableBuckets.GetByName(ctx, querier, bucket.Name)
	require.NoError(t, err)
	assert.Equal(t, bucket.ID, retrievedBucketByName.ID)

	// GetByName - not found
	_, err = storage.TableBuckets.GetByName(ctx, querier, "NonExistentBucket")
	require.ErrorIs(t, err, database.ErrNoRows)

	// UpdateByID
	bucket.SizeQuota = 2048
	err = storage.TableBuckets.UpdateByID(ctx, querier, bucket)
	require.NoError(t, err)

	updatedBucket, err := storage.TableBuckets.GetByID(ctx, querier, bucket.ID)
	require.NoError(t, err)
	assert.InEpsilon(t, float64(2048), updatedBucket.SizeQuota, 0.1)

	// UpdateByID - not found
	nonExistentBucket := &model.Bucket{ID: -1, Name: "NonExistentBucket", Availability: model.BucketAvailabilityClosed}
	err = storage.TableBuckets.UpdateByID(ctx, querier, nonExistentBucket)
	require.ErrorIs(t, err, database.ErrNoRows)

	// DeleteByID
	err = storage.TableBuckets.DeleteByID(ctx, querier, bucket.ID)
	require.NoError(t, err)

	deletedBucket, err := storage.TableBuckets.GetByID(ctx, querier, bucket.ID)
	require.ErrorIs(t, err, database.ErrNoRows)
	assert.Nil(t, deletedBucket)

	// DeleteByID - not found
	err = storage.TableBuckets.DeleteByID(ctx, querier, bucket.ID)
	require.ErrorIs(t, err, database.ErrNoRows)
}

func TestTableFilesIntegration(t *testing.T) {
	checkDB(t)
	clearTables(t)

	ctx := context.Background()
	querier := testDB.GetPool()

	bucket := &model.Bucket{
		Name:         "TestBucketFiles",
		Availability: model.BucketAvailabilityAccessible,
		OwnerID:      uuid.New(),
		SizeQuota:    1024,
	}
	err := storage.TableBuckets.Add(ctx, querier, bucket)
	require.NoError(t, err)

	// Add
	file := &model.File{
		Filename:       "TestFile",
		MIME:           "text/plain",
		BucketID:       bucket.ID,
		Access:         model.FileAccessPublic,
		SizeBytes:      123,
		FilenameSuffix: 0,
	}
	err = storage.TableFiles.Add(ctx, querier, file)
	require.NoError(t, err)

	// InsertID
	fileID := uuid.New()
	file2 := &model.File{
		ID:             fileID,
		Filename:       "TestFile2",
		MIME:           "text/plain",
		BucketID:       bucket.ID,
		Access:         model.FileAccessPublic,
		SizeBytes:      123,
		FilenameSuffix: 0,
	}
	err = storage.TableFiles.InsertID(ctx, querier, file2)
	require.NoError(t, err)

	retrievedFile2, err := storage.TableFiles.GetByID(ctx, querier, fileID)
	require.NoError(t, err)
	assert.Equal(t, file2.Filename, retrievedFile2.Filename)

	// GetByID
	retrievedFile, err := storage.TableFiles.GetByID(ctx, querier, file.ID)
	require.NoError(t, err)
	assert.Equal(t, file.Filename, retrievedFile.Filename)

	// GetByID - not found
	_, err = storage.TableFiles.GetByID(ctx, querier, uuid.New())
	require.ErrorIs(t, err, database.ErrNoRows)

	// GetFilesOfABucket
	files, err := storage.TableFiles.GetFilesOfABucket(ctx, querier, bucket.ID)
	require.NoError(t, err)
	require.Len(t, files, 2)

	// UpdateByID
	file.SizeBytes = 456
	err = storage.TableFiles.UpdateByID(ctx, querier, file)
	require.NoError(t, err)

	updatedFile, err := storage.TableFiles.GetByID(ctx, querier, file.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(456), updatedFile.SizeBytes)

	// UpdateByID - not found
	nonExistentFile := &model.File{ID: uuid.New(), Access: model.FileAccessPrivate}
	err = storage.TableFiles.UpdateByID(ctx, querier, nonExistentFile)
	require.ErrorIs(t, err, database.ErrNoRows)

	// DeleteByID
	err = storage.TableFiles.DeleteByID(ctx, querier, file.ID)
	require.NoError(t, err)

	deletedFile, err := storage.TableFiles.GetByID(ctx, querier, file.ID)
	require.ErrorIs(t, err, database.ErrNoRows)
	assert.Nil(t, deletedFile)

	// DeleteByID - not found
	err = storage.TableFiles.DeleteByID(ctx, querier, file.ID)
	require.ErrorIs(t, err, database.ErrNoRows)

	tx1, err := testDB.GetPool().Begin(ctx)
	require.NoError(t, err)
	defer tx1.Rollback(ctx)

	// PrepareNewFilenameSuffix
	suffix, err := storage.TableFiles.PrepareNewFilenameSuffix(ctx, tx1, "TestFile3")
	require.NoError(t, err)
	assert.Equal(t, int32(1), suffix)

	tx2, err := testDB.GetPool().Begin(ctx)
	require.NoError(t, err)
	defer tx2.Rollback(ctx)

	// Subsequent call should increment suffix
	suffix, err = storage.TableFiles.PrepareNewFilenameSuffix(ctx, tx2, "TestFile3")
	require.NoError(t, err)
	assert.Equal(t, int32(1), suffix)
}
