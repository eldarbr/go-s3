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
	CreatedTS time.Time
	Filename  string
	MIME      string
	Access    FileAccess
	ID        int64
	BucketID  int64
}

const (
	BucketAvailabilityClosed     BucketAvailability = "closed"
	BucketAvailabilityAccessible BucketAvailability = "accessible"
)

const (
	FileAccessPrivate FileAccess = "private"
	FileAccessPublic  FileAccess = "public"
)
