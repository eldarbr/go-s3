package business

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/eldarbr/go-auth/pkg/database"
	"github.com/eldarbr/go-s3/internal/model"
	"github.com/eldarbr/go-s3/internal/provider/storage"
	"github.com/google/uuid"
)

type BusinessModule struct {
	dbInstance  *database.Database
	fileStorage FileStorage
}

var (
	ErrBadRequest   = errors.New("malformed request")
	ErrNoPermission = errors.New("the user has no permissions")
	ErrNoBucket     = errors.New("bucket not found")
)

type FileStorage interface {
	CreateFolder(bucketID string) error
	OpenFile(bucketID, fileID string) (io.ReadSeekCloser, error)
	WriteFile(bucketID, fileID string, src io.Reader) (int64, error)
}

func NewBusinessModule(dbInstance *database.Database, fileStorage FileStorage) *BusinessModule {
	return &BusinessModule{
		dbInstance:  dbInstance,
		fileStorage: fileStorage,
	}
}

func (business BusinessModule) CreateBucket(ctx context.Context, bucket *model.Bucket) error {
	transaction, err := business.dbInstance.GetPool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("business.UploadFile begin transaction: %w", err)
	}

	defer transaction.Rollback(ctx) //nolint:errcheck // won't check

	err = storage.TableBuckets.Add(ctx, transaction, bucket)
	if err != nil {
		return fmt.Errorf("business.CreateBucket TableBuckets.Add: %w", err)
	}

	err = business.fileStorage.CreateFolder(strconv.FormatInt(bucket.ID, 10))
	if err != nil {
		return fmt.Errorf("business.CreateBucket fileStorage.CreateFolder: %w", err)
	}

	err = transaction.Commit(ctx)

	return err
}

func (business BusinessModule) UploadFile(ctx context.Context, request model.UploadFileRequest) (*uuid.UUID, error) {
	bucketInfo, err := storage.TableBuckets.GetByName(ctx, business.dbInstance.GetPool(), request.BucketName)
	if errors.Is(err, database.ErrNoRows) {
		return nil, ErrNoBucket
	}

	if err != nil {
		return nil, fmt.Errorf("business.TableBuckets.GetByName: %w", err)
	}

	if bucketInfo.OwnerID != request.RequesterUUID {
		return nil, ErrNoPermission
	}

	newFileUUID, uuidErr := uuid.NewRandom()
	if uuidErr != nil {
		return nil, fmt.Errorf("business.uuid.NewRandom: %w", uuidErr)
	}

	request.File.ID = newFileUUID
	request.File.BucketID = bucketInfo.ID

	bytesWritten, err := business.fileStorage.WriteFile(strconv.FormatInt(request.BucketID, 10),
		request.File.ID.String(),
		request.FileContent)
	if err != nil {
		return nil, fmt.Errorf("business.UploadFile fileStorage.WriteFile: %w", err)
	}

	request.File.SizeBytes = bytesWritten

	err = storage.TableFiles.InsertID(ctx, business.dbInstance.GetPool(), &request.File)
	if err != nil {
		return nil, fmt.Errorf("business.UploadFile TableFiles.Add: %w", err)
	}

	return &newFileUUID, nil
}

func (business BusinessModule) FetchFile(ctx context.Context, request model.FetchFileRequest) error {
	bucketInfo, err := storage.TableBuckets.GetByName(ctx, business.dbInstance.GetPool(), request.BucketName)
	if errors.Is(err, database.ErrNoRows) {
		return ErrNoBucket
	}

	if err != nil {
		return fmt.Errorf("business.TableBuckets.GetByName: %w", err)
	}

	fileInfo, err := storage.TableFiles.GetByID(ctx, business.dbInstance.GetPool(), request.FileID)
	if errors.Is(err, database.ErrNoRows) {
		return ErrNoBucket
	}

	if err != nil {
		return fmt.Errorf("business.TableFiles.GetByID: %w", err)
	}

	mayServe := (fileInfo.BucketID == bucketInfo.ID) &&
		((bucketInfo.Availability == model.BucketAvailabilityAccessible && fileInfo.Access == model.FileAccessPublic) ||
			(request.RequestingUserID != nil && *request.RequestingUserID == bucketInfo.OwnerID))

	if !mayServe {
		return ErrNoPermission
	}

	request.RespWriter.Header().Set("Content-Type", fileInfo.MIME)
	request.RespWriter.Header().Set("Content-Disposition", "attachment; filename=" + fileInfo.Filename)

	file, fileErr := business.fileStorage.OpenFile(strconv.FormatInt(bucketInfo.ID, 10), fileInfo.ID.String())
	if fileErr != nil {
		return fileErr
	}

	defer file.Close()

	http.ServeContent(request.RespWriter, request.RawRequest, fileInfo.Filename, fileInfo.CreatedTS, file)

	return err
}
