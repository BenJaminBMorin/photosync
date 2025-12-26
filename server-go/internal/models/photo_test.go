package models

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPhoto(t *testing.T) {
	t.Run("creates photo with valid parameters", func(t *testing.T) {
		filename := "test_photo.jpg"
		storedPath := "2024/03/test_photo.jpg"
		hash := "abc123def456abc123def456abc123def456abc123def456abc123def456abcd"
		fileSize := int64(1024)
		dateTaken := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)

		photo, err := NewPhoto(filename, storedPath, hash, fileSize, dateTaken)

		require.NoError(t, err)
		assert.NotEmpty(t, photo.ID)
		assert.Equal(t, filename, photo.OriginalFilename)
		assert.Equal(t, storedPath, photo.StoredPath)
		assert.Equal(t, strings.ToLower(hash), photo.FileHash)
		assert.Equal(t, fileSize, photo.FileSize)
		assert.Equal(t, dateTaken, photo.DateTaken)
		assert.WithinDuration(t, time.Now().UTC(), photo.UploadedAt, time.Second*5)
	})

	t.Run("rejects empty filename", func(t *testing.T) {
		_, err := NewPhoto("", "path", "hash", 1024, time.Now())
		assert.ErrorIs(t, err, ErrEmptyFilename)
	})

	t.Run("rejects empty stored path", func(t *testing.T) {
		_, err := NewPhoto("file.jpg", "", "hash", 1024, time.Now())
		assert.ErrorIs(t, err, ErrEmptyStoredPath)
	})

	t.Run("rejects empty hash", func(t *testing.T) {
		_, err := NewPhoto("file.jpg", "path", "", 1024, time.Now())
		assert.ErrorIs(t, err, ErrEmptyHash)
	})

	t.Run("rejects zero file size", func(t *testing.T) {
		_, err := NewPhoto("file.jpg", "path", "hash", 0, time.Now())
		assert.ErrorIs(t, err, ErrInvalidFileSize)
	})

	t.Run("rejects negative file size", func(t *testing.T) {
		_, err := NewPhoto("file.jpg", "path", "hash", -100, time.Now())
		assert.ErrorIs(t, err, ErrInvalidFileSize)
	})

	t.Run("sanitizes filename with path components", func(t *testing.T) {
		malicious := "../../../etc/passwd.jpg"

		photo, err := NewPhoto(malicious, "safe/path.jpg", "hash", 1024, time.Now())

		require.NoError(t, err)
		assert.NotContains(t, photo.OriginalFilename, "..")
		assert.NotContains(t, photo.OriginalFilename, "/")
	})

	t.Run("normalizes hash to lowercase", func(t *testing.T) {
		upperHash := "ABC123DEF456ABC123DEF456ABC123DEF456ABC123DEF456ABC123DEF456ABCD"

		photo, err := NewPhoto("file.jpg", "path", upperHash, 1024, time.Now())

		require.NoError(t, err)
		assert.Equal(t, strings.ToLower(upperHash), photo.FileHash)
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		photo1, err := NewPhoto("a.jpg", "path1", "hash1", 100, time.Now())
		require.NoError(t, err)

		photo2, err := NewPhoto("b.jpg", "path2", "hash2", 100, time.Now())
		require.NoError(t, err)

		assert.NotEqual(t, photo1.ID, photo2.ID)
	})
}
