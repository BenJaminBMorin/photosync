package models

import (
	"time"

	"github.com/google/uuid"
)

// CollectionShare represents a collection shared with a specific user
type CollectionShare struct {
	ID           string    `json:"id"`
	CollectionID string    `json:"collectionId"`
	UserID       string    `json:"userId"`
	CreatedAt    time.Time `json:"createdAt"`
}

// NewCollectionShare creates a new share association
func NewCollectionShare(collectionID, userID string) *CollectionShare {
	return &CollectionShare{
		ID:           uuid.New().String(),
		CollectionID: collectionID,
		UserID:       userID,
		CreatedAt:    time.Now().UTC(),
	}
}

// CollectionShareWithUser includes user details for API responses
type CollectionShareWithUser struct {
	CollectionShare
	User *User `json:"user,omitempty"`
}
