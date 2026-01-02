package repository

import (
	"context"

	"github.com/photosync/server/internal/models"
)

// PhotoRepo defines the interface for photo persistence operations
type PhotoRepo interface {
	GetByID(ctx context.Context, id string) (*models.Photo, error)
	GetByHash(ctx context.Context, hash string) (*models.Photo, error)
	GetExistingHashes(ctx context.Context, hashes []string) ([]string, error)
	GetAll(ctx context.Context, skip, take int) ([]*models.Photo, error)
	GetCount(ctx context.Context) (int, error)
	Add(ctx context.Context, photo *models.Photo) error
	Delete(ctx context.Context, id string) (bool, error)
}
