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

	// GetByID
	retrievedBucket, err := storage.TableBuckets.GetByID(ctx, querier, bucket.ID)

	require.NoError(t, err)
	assert.Equal(t, bucket.Name, retrievedBucket.Name)

	// GetByName
	retrievedBucketByName, err := storage.TableBuckets.GetByName(ctx, querier, bucket.Name)

	require.NoError(t, err)
	assert.Equal(t, bucket.ID, retrievedBucketByName.ID)

	// UpdateByID
	bucket.SizeQuota = 2048
	err = storage.TableBuckets.UpdateByID(ctx, querier, bucket)

	require.NoError(t, err)

	updatedBucket, err := storage.TableBuckets.GetByID(ctx, querier, bucket.ID)

	require.NoError(t, err)
	assert.InEpsilon(t, float64(2048), updatedBucket.SizeQuota, 0.1)

	// DeleteByID
	err = storage.TableBuckets.DeleteByID(ctx, querier, bucket.ID)

	require.NoError(t, err)

	deletedBucket, err := storage.TableBuckets.GetByID(ctx, querier, bucket.ID)

	require.Error(t, err)
	assert.Nil(t, deletedBucket)
}

func TestTableFilesIntegration(t *testing.T) {
	checkDB(t)
	clearTables(t)

	ctx := context.Background()
	querier := testDB.GetPool()

	bucket := &model.Bucket{
		Name:         "FileTestBucket",
		Availability: model.BucketAvailabilityAccessible,
		OwnerID:      uuid.New(),
		SizeQuota:    2048,
	}
	err := storage.TableBuckets.Add(ctx, querier, bucket)

	require.NoError(t, err)

	// Add
	file := &model.File{
		Filename: "testfile.txt",
		MIME:     "text/plain",
		BucketID: bucket.ID,
	}
	err = storage.TableFiles.Add(ctx, querier, file)

	require.NoError(t, err)

	// GetByID
	retrievedFile, err := storage.TableFiles.GetByID(ctx, querier, file.ID)

	require.NoError(t, err)
	assert.Equal(t, file.Filename, retrievedFile.Filename)

	// GetFilesOfABucket
	filesInBucket, err := storage.TableFiles.GetFilesOfABucket(ctx, querier, bucket.ID)

	require.NoError(t, err)
	assert.Len(t, filesInBucket, 1)
	assert.Equal(t, file.ID, filesInBucket[0].ID)

	// UpdateByID
	file.Filename = "updatedfile.txt"
	err = storage.TableFiles.UpdateByID(ctx, querier, file)

	require.NoError(t, err)

	updatedFile, err := storage.TableFiles.GetByID(ctx, querier, file.ID)

	require.NoError(t, err)
	assert.Equal(t, "updatedfile.txt", updatedFile.Filename)

	// DeleteByID
	err = storage.TableFiles.DeleteByID(ctx, querier, file.ID)

	require.NoError(t, err)

	deletedFile, err := storage.TableFiles.GetByID(ctx, querier, file.ID)

	require.Error(t, err)
	assert.Nil(t, deletedFile)
}
