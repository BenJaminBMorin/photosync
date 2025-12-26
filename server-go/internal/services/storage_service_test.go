package services

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestStorage(t *testing.T) (*PhotoStorageService, string) {
	tempDir, err := os.MkdirTemp("", "photosync-test-*")
	require.NoError(t, err)

	svc, err := NewPhotoStorageService(tempDir, nil, 50)
	require.NoError(t, err)

	return svc, tempDir
}

func cleanupTestStorage(tempDir string) {
	os.RemoveAll(tempDir)
}

func TestPhotoStorageService_Store(t *testing.T) {
	t.Run("stores file in Year/Month folder", func(t *testing.T) {
		svc, tempDir := setupTestStorage(t)
		defer cleanupTestStorage(tempDir)

		content := []byte("fake image content")
		dateTaken := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)

		storedPath, err := svc.Store(
			bytes.NewReader(content),
			"test_photo.jpg",
			dateTaken,
			int64(len(content)),
		)

		require.NoError(t, err)
		assert.True(t, filepath.HasPrefix(storedPath, "2024/03/"))
		assert.True(t, filepath.HasSuffix(storedPath, ".jpg"))

		// Verify file exists
		assert.True(t, svc.Exists(storedPath))
	})

	t.Run("creates unique filename for duplicates", func(t *testing.T) {
		svc, tempDir := setupTestStorage(t)
		defer cleanupTestStorage(tempDir)

		content := []byte("content")
		dateTaken := time.Date(2024, 6, 20, 0, 0, 0, 0, time.UTC)

		path1, err := svc.Store(bytes.NewReader(content), "duplicate.jpg", dateTaken, int64(len(content)))
		require.NoError(t, err)

		path2, err := svc.Store(bytes.NewReader(content), "duplicate.jpg", dateTaken, int64(len(content)))
		require.NoError(t, err)

		assert.NotEqual(t, path1, path2)
		assert.True(t, svc.Exists(path1))
		assert.True(t, svc.Exists(path2))
	})

	t.Run("rejects disallowed extensions", func(t *testing.T) {
		svc, tempDir := setupTestStorage(t)
		defer cleanupTestStorage(tempDir)

		disallowed := []string{".exe", ".bat", ".sh", ".php"}
		for _, ext := range disallowed {
			_, err := svc.Store(
				bytes.NewReader([]byte("content")),
				"file"+ext,
				time.Now(),
				7,
			)
			assert.Error(t, err, "extension %s should be rejected", ext)
		}
	})

	t.Run("sanitizes path traversal attempts", func(t *testing.T) {
		svc, tempDir := setupTestStorage(t)
		defer cleanupTestStorage(tempDir)

		maliciousNames := []string{
			"../../../etc/passwd.jpg",
			"..\\..\\windows\\system32.jpg",
			"/etc/passwd.jpg",
		}

		for _, name := range maliciousNames {
			storedPath, err := svc.Store(
				bytes.NewReader([]byte("content")),
				name,
				time.Now(),
				7,
			)

			require.NoError(t, err)
			assert.NotContains(t, storedPath, "..")
			assert.NotContains(t, storedPath, "/etc/")
			assert.NotContains(t, storedPath, "\\windows\\")
		}
	})
}

func TestPhotoStorageService_Delete(t *testing.T) {
	t.Run("deletes existing file", func(t *testing.T) {
		svc, tempDir := setupTestStorage(t)
		defer cleanupTestStorage(tempDir)

		storedPath, err := svc.Store(
			bytes.NewReader([]byte("content")),
			"delete_me.jpg",
			time.Now(),
			7,
		)
		require.NoError(t, err)
		assert.True(t, svc.Exists(storedPath))

		result := svc.Delete(storedPath)
		assert.True(t, result)
		assert.False(t, svc.Exists(storedPath))
	})

	t.Run("returns false for non-existent file", func(t *testing.T) {
		svc, tempDir := setupTestStorage(t)
		defer cleanupTestStorage(tempDir)

		result := svc.Delete("2024/01/nonexistent.jpg")
		assert.False(t, result)
	})
}

func TestPhotoStorageService_GetFullPath(t *testing.T) {
	t.Run("returns full path for valid stored path", func(t *testing.T) {
		svc, tempDir := setupTestStorage(t)
		defer cleanupTestStorage(tempDir)

		fullPath, err := svc.GetFullPath("2024/03/test.jpg")
		require.NoError(t, err)
		assert.True(t, filepath.HasPrefix(fullPath, tempDir))
	})

	t.Run("rejects path traversal", func(t *testing.T) {
		svc, tempDir := setupTestStorage(t)
		defer cleanupTestStorage(tempDir)

		_, err := svc.GetFullPath("../../../etc/passwd")
		assert.Error(t, err)
	})
}

func TestPhotoStorageService_Exists(t *testing.T) {
	t.Run("returns true for existing file", func(t *testing.T) {
		svc, tempDir := setupTestStorage(t)
		defer cleanupTestStorage(tempDir)

		storedPath, err := svc.Store(
			bytes.NewReader([]byte("content")),
			"exists.jpg",
			time.Now(),
			7,
		)
		require.NoError(t, err)

		assert.True(t, svc.Exists(storedPath))
	})

	t.Run("returns false for non-existent file", func(t *testing.T) {
		svc, tempDir := setupTestStorage(t)
		defer cleanupTestStorage(tempDir)

		assert.False(t, svc.Exists("2024/01/nonexistent.jpg"))
	})
}
