package model

import (
	"time"

	"github.com/google/uuid"
)

type BucketAvailability string

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
	ID        int64
	BucketID  int64
}

const (
	BucketAvailabilityClosed     BucketAvailability = "closed"
	BucketAvailabilityAccessible BucketAvailability = "accessible"
)
