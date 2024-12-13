package model

import (
	"time"

	"github.com/google/uuid"
)

type BucketAvailability string

type FileAccess string

type Bucket struct {
	Name         string
	Availability BucketAvailability
	ID           int64
	OwnerID      uuid.UUID
	SizeQuota    float64
}

type File struct {
	CreatedTS      time.Time  `json:"createdTs"`
	Filename       string     `json:"filename"`
	MIME           string     `json:"mime"`
	Access         FileAccess `json:"access"`
	BucketID       int64      `json:"-"`
	SizeBytes      int64      `json:"sizeBytes"`
	FilenameSuffix int32      `json:"-"`
	ID             uuid.UUID  `json:"id"`
}

const (
	BucketAvailabilityClosed     BucketAvailability = "closed"
	BucketAvailabilityAccessible BucketAvailability = "accessible"
)

const (
	FileAccessPrivate FileAccess = "private"
	FileAccessPublic  FileAccess = "public"
)
