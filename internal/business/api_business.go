package business

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

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
	DeleteFile(bucketID, fileID string) error
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
		return fmt.Errorf("business.CreateBucket begin transaction: %w", err)
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
	if err != nil {
		return fmt.Errorf("business.CreateBucket transaction.Commit: %w", err)
	}

	return nil
}

func (business BusinessModule) UploadFile(ctx context.Context, request model.UploadFileRequest) (*uuid.UUID, error) {
	bucketInfo, err := storage.TableBuckets.GetByName(ctx, business.dbInstance.GetPool(), request.BucketName)
	if errors.Is(err, database.ErrNoRows) {
		return nil, ErrNoBucket
	}

	if err != nil {
		return nil, fmt.Errorf("business.UploadFile business.TableBuckets.GetByName: %w", err)
	}

	if bucketInfo.OwnerID != request.RequesterUUID {
		return nil, ErrNoPermission
	}

	newFileUUID, uuidErr := uuid.NewRandom()
	if uuidErr != nil {
		return nil, fmt.Errorf("business.uuid.NewRandom: %w", uuidErr)
	}

	bytesWritten, err := business.fileStorage.WriteFile(strconv.FormatInt(bucketInfo.ID, 10),
		newFileUUID.String(),
		request.FileContent)
	if err != nil {
		return nil, fmt.Errorf("business.UploadFile fileStorage.WriteFile: %w", err)
	}

	transaction, err := business.dbInstance.GetPool().Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("business.UploadFile begin transaction: %w", err)
	}

	defer transaction.Rollback(ctx) //nolint:errcheck // won't check

	newSuffix, err := storage.TableFiles.PrepareNewFilenameSuffix(ctx, transaction, request.Filename)
	if err != nil {
		return nil, fmt.Errorf("business.UploadFile storage.TableFiles.PrepareNewFilenameSuffix: %w", err)
	}

	request.File.ID = newFileUUID
	request.File.BucketID = bucketInfo.ID
	request.FilenameSuffix = newSuffix
	request.File.SizeBytes = bytesWritten

	err = storage.TableFiles.InsertID(ctx, transaction, &request.File)
	if err != nil {
		return nil, fmt.Errorf("business.UploadFile TableFiles.Add: %w", err)
	}

	err = transaction.Commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("business.CreateBucket transaction.Commit: %w", err)
	}

	createdID := request.File.ID

	return &createdID, nil
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
	request.RespWriter.Header().Set("Content-Disposition", "inline; filename="+fileInfo.Filename)

	file, fileErr := business.fileStorage.OpenFile(strconv.FormatInt(bucketInfo.ID, 10), fileInfo.ID.String())
	if fileErr != nil {
		return fmt.Errorf("business.FetchFile fileStorage.OpenFile: %w", fileErr)
	}

	defer file.Close()

	http.ServeContent(request.RespWriter, request.RawRequest, fileInfo.Filename, fileInfo.CreatedTS, file)

	return nil
}

func (business BusinessModule) ListFiles(ctx context.Context, requesterUUID uuid.UUID, bucketName string,
) ([]model.File, error) {
	bucketInfo, err := storage.TableBuckets.GetByName(ctx, business.dbInstance.GetPool(), bucketName)
	if err != nil {
		return nil, fmt.Errorf("ListFiles couldn't get the bucket entry: %s, %w", bucketName, err)
	}

	if bucketInfo.OwnerID != requesterUUID {
		return nil, ErrNoPermission
	}

	files, err := storage.TableFiles.GetFilesOfABucket(ctx, business.dbInstance.GetPool(), bucketInfo.ID)
	if err != nil {
		return nil, fmt.Errorf("ListFiles couldn't list files of the bucket: %s, %w", bucketName, err)
	}

	for fileIndex := range files {
		if files[fileIndex].FilenameSuffix != 0 {
			var builder strings.Builder

			lastDotIdx := strings.LastIndex(files[fileIndex].Filename, ".")

			if lastDotIdx != -1 {
				builder.WriteString(files[fileIndex].Filename[0:lastDotIdx])
			} else {
				builder.WriteString(files[fileIndex].Filename)
			}

			builder.WriteString("_")
			builder.WriteString(strconv.FormatInt(int64(files[fileIndex].FilenameSuffix), 10))

			if lastDotIdx != -1 {
				builder.WriteString(files[fileIndex].Filename[lastDotIdx:])
			}

			files[fileIndex].Filename = builder.String()
		}
	}

	return files, nil
}

func (business BusinessModule) EditFile(ctx context.Context, request model.File, bucketName string,
	requesterID uuid.UUID,
) error {
	dbFile, err := storage.TableFiles.GetByID(ctx, business.dbInstance.GetPool(), request.ID)
	if err != nil {
		return fmt.Errorf("EditFile couldn't get the file entry: %s, %w", request.ID.String(), err)
	}

	bucketInfo, err := storage.TableBuckets.GetByName(ctx, business.dbInstance.GetPool(), bucketName)
	if err != nil {
		return fmt.Errorf("EditFile couldn't get the bucket entry: %s, %w", bucketName, err)
	}

	// check if actually the file is in the bucket.
	if dbFile.BucketID != bucketInfo.ID {
		return ErrBadRequest
	}

	// check if the user has permissions to edit.
	if bucketInfo.OwnerID != requesterID {
		return ErrNoPermission
	}

	if request.Filename != "" {
		dbFile.Filename = request.Filename
	}

	if request.Access != "" {
		dbFile.Access = request.Access
	}

	err = storage.TableFiles.UpdateByID(ctx, business.dbInstance.GetPool(), dbFile)
	if err != nil {
		return fmt.Errorf("EditFile couldn't update the file entry: %w", err)
	}

	return nil
}

func (business BusinessModule) DeleteFile(ctx context.Context, fileID uuid.UUID, bucketName string,
	requesterID uuid.UUID,
) error {
	dbFile, err := storage.TableFiles.GetByID(ctx, business.dbInstance.GetPool(), fileID)
	if err != nil {
		return fmt.Errorf("DeleteFile couldn't get the file entry: %s, %w", fileID.String(), err)
	}

	bucketInfo, err := storage.TableBuckets.GetByName(ctx, business.dbInstance.GetPool(), bucketName)
	if err != nil {
		return fmt.Errorf("DeleteFile couldn't get the bucket entry: %s, %w", bucketName, err)
	}

	// check if actually the file is in the bucket.
	if dbFile.BucketID != bucketInfo.ID {
		return ErrBadRequest
	}

	// check if the user has permissions to edit.
	if bucketInfo.OwnerID != requesterID {
		return ErrNoPermission
	}

	err = storage.TableFiles.MarkDeleted(ctx, business.dbInstance.GetPool(), fileID)
	if err != nil {
		return fmt.Errorf("DeleteFile couldn't mark the db entry: %w", err)
	}

	err = business.fileStorage.DeleteFile(strconv.FormatInt(bucketInfo.ID, 10), fileID.String())
	if err != nil {
		return fmt.Errorf("DeleteFile couldn't delete the db file entry: %w", err)
	}

	err = storage.TableFiles.DeleteByID(ctx, business.dbInstance.GetPool(), fileID)
	if err != nil {
		return fmt.Errorf("DeleteFile couldn't delete the db file entry: %w", err)
	}

	return nil
}
