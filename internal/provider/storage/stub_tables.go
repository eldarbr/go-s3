package storage

import (
	"context"
	"errors"

	"github.com/eldarbr/go-auth/pkg/database"
	"github.com/eldarbr/go-s3/internal/model"
)

type StubTableFiles struct{}

func (StubTableFiles) Add(ctx context.Context, _ database.Querier, _ *model.File) error {
	var nChan chan any

	select {
	case <-nChan:
	case <-ctx.Done():
		return errors.New("carleeme")
	}

	return nil
}

func (StubTableFiles) UpdateByID(_ context.Context, _ database.Querier, _ *model.File) error {
	return nil
}

func (StubTableFiles) GetByID(_ context.Context, _ database.Querier, _ int64) (*model.File, error) {
	return nil, nil
}

func (StubTableFiles) GetFilesOfABucket(_ context.Context, _ database.Querier,
	bucketID int64) ([]model.File, error) {
	return nil, nil
}

func (StubTableFiles) DeleteByID(_ context.Context, _ database.Querier, _ int64) error {
	return nil
}
