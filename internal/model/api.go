package model

import (
	"io"

	"github.com/google/uuid"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type CreateBucketRequest struct {
	Name         string             `json:"name"`
	Availability BucketAvailability `json:"availability"`
}

type CreateBucketResponse struct {
	Name      string  `json:"name"`
	SizeQuota float64 `json:"sizeQuota"`
}

type UploadFileRequest struct {
	FileContent io.Reader
	File
	RequesterUUID uuid.UUID
}
