package model

import (
	"regexp"
)

const (
	minBucketNameLength = 6
	maxBucketNameLength = 30
)

var (
	regexpValidBucketName = regexp.MustCompile(`^[A-z0-9-]+$`)
)

func (req CreateBucketRequest) Valid() bool {
	nameLength := len(req.Name)
	if nameLength < minBucketNameLength || nameLength > maxBucketNameLength {
		return false
	}

	if !(regexpValidBucketName.MatchString(req.Name)) {
		return false
	}

	if (req.Availability != BucketAvailabilityAccessible) &&
		(req.Availability != BucketAvailabilityClosed) {
		return false
	}

	return true
}
