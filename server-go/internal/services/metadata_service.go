package services

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
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

// PhotoMetadata represents the full set of PhotoSync metadata embedded in image files
type PhotoMetadata struct {
	PhotoID    string
	UserID     string
	DeviceID   string
	FileHash   string
	UploadedAt time.Time
}

// EmbedFullMetadata writes all PhotoSync metadata to the image's EXIF/XMP tags
// This includes PhotoID, UserID, DeviceID, FileHash, and UploadedAt
func (s *MetadataService) EmbedFullMetadata(storedPath string, metadata PhotoMetadata) error {
	fullPath := fmt.Sprintf("%s/%s", s.basePath, storedPath)

	// Build exiftool arguments
	args := []string{
		"-overwrite_original",
		fmt.Sprintf("-ImageUniqueID=%s", metadata.PhotoID),
		fmt.Sprintf("-XMP-photosync:PhotoID=%s", metadata.PhotoID),
		fmt.Sprintf("-XMP-photosync:UserID=%s", metadata.UserID),
		fmt.Sprintf("-XMP-photosync:DeviceID=%s", metadata.DeviceID),
		fmt.Sprintf("-XMP-photosync:FileHash=%s", metadata.FileHash),
		fmt.Sprintf("-XMP-photosync:UploadedAt=%s", metadata.UploadedAt.Format(time.RFC3339)),
		fullPath,
	}

	cmd := exec.Command("exiftool", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Warning: failed to embed full metadata in %s: %v (output: %s)", storedPath, err, string(output))
		return err
	}

	return nil
}

// ReadFullMetadata extracts all PhotoSync metadata from an image file
// Returns nil if no PhotoSync metadata is found
func (s *MetadataService) ReadFullMetadata(storedPath string) (*PhotoMetadata, error) {
	fullPath := fmt.Sprintf("%s/%s", s.basePath, storedPath)

	// Read all PhotoSync XMP fields at once
	cmd := exec.Command("exiftool",
		"-s3",
		"-XMP-photosync:PhotoID",
		"-XMP-photosync:UserID",
		"-XMP-photosync:DeviceID",
		"-XMP-photosync:FileHash",
		"-XMP-photosync:UploadedAt",
		"-t", // Tab-separated output
		fullPath,
	)

	output, err := cmd.Output()
	if err != nil {
		// Try reading just ImageUniqueID as fallback for legacy files
		return s.readLegacyMetadata(fullPath)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return nil, nil
	}

	metadata := &PhotoMetadata{}

	for _, line := range lines {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		value := strings.TrimSpace(parts[1])
		if value == "" {
			continue
		}

		switch {
		case strings.Contains(parts[0], "PhotoID"):
			metadata.PhotoID = value
		case strings.Contains(parts[0], "UserID"):
			metadata.UserID = value
		case strings.Contains(parts[0], "DeviceID"):
			metadata.DeviceID = value
		case strings.Contains(parts[0], "FileHash"):
			metadata.FileHash = value
		case strings.Contains(parts[0], "UploadedAt"):
			if t, err := time.Parse(time.RFC3339, value); err == nil {
				metadata.UploadedAt = t
			}
		}
	}

	// If no metadata was found, return nil
	if metadata.PhotoID == "" {
		return nil, nil
	}

	return metadata, nil
}

// readLegacyMetadata reads just the PhotoID from ImageUniqueID for legacy files
func (s *MetadataService) readLegacyMetadata(fullPath string) (*PhotoMetadata, error) {
	cmd := exec.Command("exiftool",
		"-s3",
		"-ImageUniqueID",
		fullPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, nil
	}

	photoID := strings.TrimSpace(string(output))
	if photoID == "" {
		return nil, nil
	}

	return &PhotoMetadata{
		PhotoID: photoID,
	}, nil
}

// HasEmbeddedMetadata checks if a file has any PhotoSync metadata embedded
func (s *MetadataService) HasEmbeddedMetadata(storedPath string) bool {
	metadata, err := s.ReadFullMetadata(storedPath)
	return err == nil && metadata != nil && metadata.PhotoID != ""
}

// UpdateEmbeddedMetadata updates specific fields in the embedded metadata
// This is useful when only certain fields need to change (e.g., after user assignment)
func (s *MetadataService) UpdateEmbeddedMetadata(storedPath string, updates map[string]string) error {
	fullPath := fmt.Sprintf("%s/%s", s.basePath, storedPath)

	args := []string{"-overwrite_original"}

	for field, value := range updates {
		switch field {
		case "PhotoID":
			args = append(args, fmt.Sprintf("-ImageUniqueID=%s", value))
			args = append(args, fmt.Sprintf("-XMP-photosync:PhotoID=%s", value))
		case "UserID":
			args = append(args, fmt.Sprintf("-XMP-photosync:UserID=%s", value))
		case "DeviceID":
			args = append(args, fmt.Sprintf("-XMP-photosync:DeviceID=%s", value))
		case "FileHash":
			args = append(args, fmt.Sprintf("-XMP-photosync:FileHash=%s", value))
		case "UploadedAt":
			args = append(args, fmt.Sprintf("-XMP-photosync:UploadedAt=%s", value))
		}
	}

	args = append(args, fullPath)

	cmd := exec.Command("exiftool", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Warning: failed to update metadata in %s: %v (output: %s)", storedPath, err, string(output))
		return err
	}

	return nil
}
