package models

import (
	"time"

	"github.com/google/uuid"
)

// CollectionPhoto represents the junction between collections and photos
type CollectionPhoto struct {
	ID           string    `json:"id"`
	CollectionID string    `json:"collectionId"`
	PhotoID      string    `json:"photoId"`
	Position     int       `json:"position"`
	AddedAt      time.Time `json:"addedAt"`
}

// NewCollectionPhoto creates a new collection-photo association
func NewCollectionPhoto(collectionID, photoID string, position int) *CollectionPhoto {
	return &CollectionPhoto{
		ID:           uuid.New().String(),
		CollectionID: collectionID,
		PhotoID:      photoID,
		Position:     position,
		AddedAt:      time.Now().UTC(),
	}
}

// CollectionPhotoWithDetails includes photo metadata for API responses
type CollectionPhotoWithDetails struct {
	CollectionPhoto
	Photo *Photo `json:"photo,omitempty"`
}
