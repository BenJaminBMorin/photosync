package services

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// MetadataService handles embedding metadata into image files
type MetadataService struct {
	basePath string
}

// NewMetadataService creates a new MetadataService
func NewMetadataService(basePath string) *MetadataService {
	return &MetadataService{
		basePath: basePath,
	}
}

// EmbedPhotoID writes the photo UUID to the image's EXIF/XMP metadata
// Uses ImageUniqueID (EXIF) and XMP-photosync:PhotoID (custom XMP) for redundancy
func (s *MetadataService) EmbedPhotoID(storedPath string, photoID string) error {
	fullPath := fmt.Sprintf("%s/%s", s.basePath, storedPath)

	// Use exiftool to write the photo ID to multiple metadata fields
	// - ImageUniqueID: Standard EXIF field for unique image identifiers
	// - XMP-photosync:PhotoID: Custom XMP field specific to our application
	cmd := exec.Command("exiftool",
		"-overwrite_original",
		fmt.Sprintf("-ImageUniqueID=%s", photoID),
		fmt.Sprintf("-XMP-photosync:PhotoID=%s", photoID),
		fullPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Log but don't fail - metadata embedding is optional
		log.Printf("Warning: failed to embed metadata in %s: %v (output: %s)", storedPath, err, string(output))
		return err
	}

	return nil
}

// ReadPhotoID reads the embedded photo ID from an image file
func (s *MetadataService) ReadPhotoID(storedPath string) (string, error) {
	fullPath := fmt.Sprintf("%s/%s", s.basePath, storedPath)

	// Try to read ImageUniqueID first, fall back to XMP field
	cmd := exec.Command("exiftool",
		"-s3",
		"-ImageUniqueID",
		fullPath,
	)

	output, err := cmd.Output()
	if err == nil {
		id := strings.TrimSpace(string(output))
		if id != "" {
			return id, nil
		}
	}

	// Try XMP field
	cmd = exec.Command("exiftool",
		"-s3",
		"-XMP-photosync:PhotoID",
		fullPath,
	)

	output, err = cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// IsExiftoolAvailable checks if exiftool is installed
func IsExiftoolAvailable() bool {
	_, err := exec.LookPath("exiftool")
	return err == nil
}
