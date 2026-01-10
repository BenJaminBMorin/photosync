package services

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/jdeng/goheif"
)

// ThumbnailSize represents a thumbnail size configuration
type ThumbnailSize struct {
	Name      string
	MaxDim    int  // Maximum dimension (width or height)
	Quality   int  // JPEG quality (1-100)
}

var (
	// ThumbSmall is 200px max dimension
	ThumbSmall = ThumbnailSize{Name: "small", MaxDim: 200, Quality: 80}
	// ThumbMedium is 500px max dimension
	ThumbMedium = ThumbnailSize{Name: "medium", MaxDim: 500, Quality: 85}
	// ThumbLarge is 1000px max dimension
	ThumbLarge = ThumbnailSize{Name: "large", MaxDim: 1000, Quality: 85}
)

// ThumbnailResult contains paths to generated thumbnails
type ThumbnailResult struct {
	SmallPath  string
	MediumPath string
	LargePath  string
	Width      int
	Height     int
}

// ThumbnailService handles thumbnail generation
type ThumbnailService struct {
	basePath string
}

// NewThumbnailService creates a new ThumbnailService
func NewThumbnailService(basePath string) *ThumbnailService {
	return &ThumbnailService{basePath: basePath}
}

// GenerateThumbnails creates thumbnails for an image and returns their paths
func (s *ThumbnailService) GenerateThumbnails(imageData []byte, photoID string, storedPath string, orientation int) (*ThumbnailResult, error) {
	var img image.Image
	var format string
	var err error

	// Check if this is a HEIC/HEIF file
	if IsHEIC(storedPath) {
		img, err = decodeHEIC(imageData)
		if err != nil {
			return nil, fmt.Errorf("failed to decode HEIC image: %w", err)
		}
		format = "heic"
	} else {
		// Standard image decode
		img, format, err = image.Decode(bytes.NewReader(imageData))
		if err != nil {
			return nil, fmt.Errorf("failed to decode image: %w", err)
		}
	}

	// Apply EXIF orientation correction
	img = applyOrientation(img, orientation)

	// Get original dimensions (after orientation correction)
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Determine the directory for thumbnails based on stored path
	// storedPath is like "2026/01/IMG_001.jpg"
	dir := filepath.Dir(storedPath)
	thumbDir := filepath.Join(dir, ".thumbs")

	// Create thumbnail directory if it doesn't exist
	fullThumbDir := filepath.Join(s.basePath, thumbDir)
	if err := os.MkdirAll(fullThumbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create thumbnail directory: %w", err)
	}

	result := &ThumbnailResult{
		Width:  width,
		Height: height,
	}

	// Generate each thumbnail size
	sizes := []struct {
		size    ThumbnailSize
		pathPtr *string
	}{
		{ThumbSmall, &result.SmallPath},
		{ThumbMedium, &result.MediumPath},
		{ThumbLarge, &result.LargePath},
	}

	for _, sizeItem := range sizes {
		thumbPath, err := s.generateThumbnail(img, photoID, thumbDir, sizeItem.size, format)
		if err != nil {
			return nil, fmt.Errorf("failed to generate %s thumbnail: %w", sizeItem.size.Name, err)
		}
		*sizeItem.pathPtr = thumbPath
	}

	return result, nil
}

// generateThumbnail creates a single thumbnail and returns its relative path
func (s *ThumbnailService) generateThumbnail(img image.Image, photoID string, thumbDir string, size ThumbnailSize, format string) (string, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate new dimensions maintaining aspect ratio
	var newWidth, newHeight int
	if width > height {
		if width > size.MaxDim {
			newWidth = size.MaxDim
			newHeight = height * size.MaxDim / width
		} else {
			newWidth = width
			newHeight = height
		}
	} else {
		if height > size.MaxDim {
			newHeight = size.MaxDim
			newWidth = width * size.MaxDim / height
		} else {
			newWidth = width
			newHeight = height
		}
	}

	// Resize using high-quality Lanczos filter
	resized := imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)

	// Generate filename: {photoID}_{size}.jpg
	filename := fmt.Sprintf("%s_%s.jpg", photoID, size.Name)
	relativePath := filepath.Join(thumbDir, filename)
	fullPath := filepath.Join(s.basePath, relativePath)

	// Create the file
	out, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create thumbnail file: %w", err)
	}
	defer out.Close()

	// Encode as JPEG
	opts := &jpeg.Options{Quality: size.Quality}
	if err := jpeg.Encode(out, resized, opts); err != nil {
		os.Remove(fullPath) // Clean up on failure
		return "", fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	return relativePath, nil
}

// applyOrientation corrects image orientation based on EXIF data
func applyOrientation(img image.Image, orientation int) image.Image {
	switch orientation {
	case 1:
		// Normal, no transformation needed
		return img
	case 2:
		// Flip horizontal
		return imaging.FlipH(img)
	case 3:
		// Rotate 180
		return imaging.Rotate180(img)
	case 4:
		// Flip vertical
		return imaging.FlipV(img)
	case 5:
		// Transpose (flip horizontal + rotate 270)
		return imaging.Rotate270(imaging.FlipH(img))
	case 6:
		// Rotate 90 CW
		return imaging.Rotate270(img)
	case 7:
		// Transverse (flip horizontal + rotate 90)
		return imaging.Rotate90(imaging.FlipH(img))
	case 8:
		// Rotate 90 CCW
		return imaging.Rotate90(img)
	default:
		return img
	}
}

// GetThumbnailPath returns the full filesystem path for a thumbnail
func (s *ThumbnailService) GetThumbnailPath(relativePath string) string {
	return filepath.Join(s.basePath, relativePath)
}

// DeleteThumbnails removes all thumbnails for a photo
func (s *ThumbnailService) DeleteThumbnails(smallPath, mediumPath, largePath string) error {
	paths := []string{smallPath, mediumPath, largePath}
	for _, p := range paths {
		if p != "" {
			fullPath := filepath.Join(s.basePath, p)
			os.Remove(fullPath) // Ignore errors for non-existent files
		}
	}
	return nil
}

// IsSupportedFormat checks if the file extension is supported for thumbnail generation
func IsSupportedFormat(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	supported := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
		".bmp":  true,
		".tiff": true,
		".tif":  true,
		".heic": true,
		".heif": true,
	}
	return supported[ext]
}

// IsHEIC checks if the file is HEIC/HEIF format (requires special handling)
func IsHEIC(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".heic" || ext == ".heif"
}

// RegenerateThumbnailsFromFile generates thumbnails from an existing file on disk
func (s *ThumbnailService) RegenerateThumbnailsFromFile(photoID string, storedPath string) (*ThumbnailResult, error) {
	fullPath := filepath.Join(s.basePath, storedPath)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", storedPath)
	}

	// Skip unsupported formats
	if !IsSupportedFormat(storedPath) {
		return nil, fmt.Errorf("unsupported format: %s", storedPath)
	}

	// Read file
	imageData, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Generate thumbnails (orientation=1 means no rotation, can be enhanced later)
	return s.GenerateThumbnails(imageData, photoID, storedPath, 1)
}

// decodeHEIC decodes a HEIC/HEIF image using goheif (pure Go)
func decodeHEIC(data []byte) (image.Image, error) {
	img, err := goheif.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode HEIC image: %w", err)
	}
	return img, nil
}

// GenerateSingleThumbnail creates a single thumbnail in memory and returns the JPEG bytes
// This is useful for generating preview thumbnails without saving to disk
func (s *ThumbnailService) GenerateSingleThumbnail(imageData []byte, maxDim int, orientation int) ([]byte, error) {
	var img image.Image
	var err error

	// Try to decode as various formats
	img, _, err = image.Decode(bytes.NewReader(imageData))
	if err != nil {
		// Try HEIC
		img, err = decodeHEIC(imageData)
		if err != nil {
			return nil, fmt.Errorf("failed to decode image: %w", err)
		}
	}

	// Apply EXIF orientation correction
	img = applyOrientation(img, orientation)

	// Get dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate new dimensions maintaining aspect ratio
	var newWidth, newHeight int
	if width > height {
		if width > maxDim {
			newWidth = maxDim
			newHeight = height * maxDim / width
		} else {
			newWidth = width
			newHeight = height
		}
	} else {
		if height > maxDim {
			newHeight = maxDim
			newWidth = width * maxDim / height
		} else {
			newWidth = width
			newHeight = height
		}
	}

	// Resize using high-quality Lanczos filter
	resized := imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)

	// Encode as JPEG to buffer
	var buf bytes.Buffer
	opts := &jpeg.Options{Quality: 80}
	if err := jpeg.Encode(&buf, resized, opts); err != nil {
		return nil, fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	return buf.Bytes(), nil
}
