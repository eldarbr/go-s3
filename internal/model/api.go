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
	BucketName  string
	File
	RequesterUUID uuid.UUID
}

type UploadResult string

const (
	UploadResultOk    = "ok"
	UploadResultError = "error"
)

type UploadedFileInfo struct {
	IDstr    string       `json:"id,omitempty"`
	FileName string       `json:"name"`
	Error    string       `json:"error,omitempty"`
	Result   UploadResult `json:"result"`
}

type UploadFileResponse struct {
	Results []UploadedFileInfo `json:"results"`
}
