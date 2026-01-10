package services

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/photosync/server/internal/models"
)

// PhotoStorageService handles file storage with Year/Month organization
type PhotoStorageService struct {
	basePath          string
	allowedExtensions map[string]bool
	maxFileSizeBytes  int64
}

// NewPhotoStorageService creates a new PhotoStorageService
func NewPhotoStorageService(basePath string, allowedExtensions []string, maxFileSizeMB int64) (*PhotoStorageService, error) {
	if strings.TrimSpace(basePath) == "" {
		return nil, fmt.Errorf("base path cannot be empty")
	}

	absPath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, err
	}

	// Ensure directory exists
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return nil, err
	}

	// Build extension set
	extSet := make(map[string]bool)
	if len(allowedExtensions) == 0 {
		// Defaults
		for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".heic", ".heif", ".bmp", ".tiff", ".tif"} {
			extSet[strings.ToLower(ext)] = true
		}
	} else {
		for _, ext := range allowedExtensions {
			extSet[strings.ToLower(ext)] = true
		}
	}

	return &PhotoStorageService{
		basePath:          absPath,
		allowedExtensions: extSet,
		maxFileSizeBytes:  maxFileSizeMB * 1024 * 1024,
	}, nil
}

// Store saves a file and returns the relative storage path
func (s *PhotoStorageService) Store(reader io.Reader, originalFilename string, dateTaken time.Time, fileSize int64) (string, error) {
	// Validate file size
	if fileSize > s.maxFileSizeBytes {
		return "", models.ErrFileTooLarge
	}

	// Sanitize and validate filename
	sanitizedFilename := sanitizeFilename(originalFilename)
	ext := strings.ToLower(filepath.Ext(sanitizedFilename))

	if !s.allowedExtensions[ext] {
		return "", models.ErrInvalidExtension
	}

	// Create Year/Month folder structure
	year := dateTaken.Format("2006")
	month := dateTaken.Format("01")
	relativeFolderPath := filepath.Join(year, month)
	absoluteFolderPath := filepath.Join(s.basePath, relativeFolderPath)

	if err := os.MkdirAll(absoluteFolderPath, 0755); err != nil {
		return "", err
	}

	// Generate unique filename
	uniqueFilename := generateUniqueFilename(sanitizedFilename, absoluteFolderPath)
	relativeFilePath := filepath.Join(relativeFolderPath, uniqueFilename)
	absoluteFilePath := filepath.Join(s.basePath, relativeFilePath)

	// Security check: ensure path is within base path
	if !strings.HasPrefix(absoluteFilePath, s.basePath) {
		return "", models.ErrPathTraversal
	}

	// Write file
	file, err := os.OpenFile(absoluteFilePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		os.Remove(absoluteFilePath) // Clean up on error
		return "", err
	}

	// Return path with forward slashes for consistency
	return strings.ReplaceAll(relativeFilePath, string(os.PathSeparator), "/"), nil
}

// Delete removes a file by its stored path
func (s *PhotoStorageService) Delete(storedPath string) bool {
	if strings.TrimSpace(storedPath) == "" {
		return false
	}

	fullPath, err := s.GetFullPath(storedPath)
	if err != nil {
		return false
	}

	if err := os.Remove(fullPath); err != nil {
		return false
	}

	return true
}

// GetFullPath returns the absolute path for a stored path
func (s *PhotoStorageService) GetFullPath(storedPath string) (string, error) {
	if strings.TrimSpace(storedPath) == "" {
		return "", fmt.Errorf("stored path cannot be empty")
	}

	// Normalize path separators
	normalizedPath := filepath.FromSlash(storedPath)
	fullPath := filepath.Join(s.basePath, normalizedPath)

	// Security check
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(absPath, s.basePath) {
		return "", models.ErrPathTraversal
	}

	return absPath, nil
}

// Exists checks if a file exists at the given stored path
func (s *PhotoStorageService) Exists(storedPath string) bool {
	fullPath, err := s.GetFullPath(storedPath)
	if err != nil {
		return false
	}

	_, err = os.Stat(fullPath)
	return err == nil
}

// sanitizeFilename removes path components and invalid characters
func sanitizeFilename(filename string) string {
	// Get just the filename
	name := filepath.Base(filename)

	// Replace dangerous characters
	replacer := strings.NewReplacer(
		"..", "",
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	name = replacer.Replace(name)

	// Limit length
	const maxLength = 200
	if len(name) > maxLength {
		ext := filepath.Ext(name)
		nameWithoutExt := strings.TrimSuffix(name, ext)
		if len(nameWithoutExt) > maxLength-len(ext) {
			nameWithoutExt = nameWithoutExt[:maxLength-len(ext)]
		}
		name = nameWithoutExt + ext
	}

	return name
}

// MoveFile moves a file from one location to another within storage
// Returns the new relative path
func (s *PhotoStorageService) MoveFile(currentPath, newRelativeFolder, newFilename string) (string, error) {
	// Get full path of current file
	currentFullPath, err := s.GetFullPath(currentPath)
	if err != nil {
		return "", fmt.Errorf("invalid current path: %w", err)
	}

	// Check current file exists
	if _, err := os.Stat(currentFullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("source file does not exist: %s", currentPath)
	}

	// Create new folder
	newFolderFullPath := filepath.Join(s.basePath, newRelativeFolder)
	if err := os.MkdirAll(newFolderFullPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create destination folder: %w", err)
	}

	// Generate unique filename in new location
	uniqueFilename := generateUniqueFilename(newFilename, newFolderFullPath)
	newRelativePath := filepath.Join(newRelativeFolder, uniqueFilename)
	newFullPath := filepath.Join(s.basePath, newRelativePath)

	// Security check
	absNewPath, err := filepath.Abs(newFullPath)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(absNewPath, s.basePath) {
		return "", models.ErrPathTraversal
	}

	// Move the file
	if err := os.Rename(currentFullPath, newFullPath); err != nil {
		// If rename fails (cross-device), try copy+delete
		if err := copyFile(currentFullPath, newFullPath); err != nil {
			return "", fmt.Errorf("failed to move file: %w", err)
		}
		os.Remove(currentFullPath)
	}

	// Return path with forward slashes for consistency
	return strings.ReplaceAll(newRelativePath, string(os.PathSeparator), "/"), nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// generateUniqueFilename creates a unique filename if collision exists
func generateUniqueFilename(filename, folderPath string) string {
	nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))
	ext := filepath.Ext(filename)
	candidate := filename
	counter := 1

	for {
		fullPath := filepath.Join(folderPath, candidate)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			break
		}

		candidate = fmt.Sprintf("%s_%03d%s", nameWithoutExt, counter, ext)
		counter++

		if counter > 9999 {
			// Fall back to timestamp
			candidate = fmt.Sprintf("%s_%d%s", nameWithoutExt, time.Now().UnixNano(), ext)
			break
		}
	}

	return candidate
}
