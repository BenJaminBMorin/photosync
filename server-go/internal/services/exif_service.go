package services

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

// EXIFData contains extracted EXIF metadata from an image
type EXIFData struct {
	// Camera info
	CameraMake  *string
	CameraModel *string
	LensModel   *string

	// Capture settings
	FocalLength  *string
	Aperture     *string
	ShutterSpeed *string
	ISO          *int
	Orientation  int

	// GPS coordinates
	Latitude  *float64
	Longitude *float64
	Altitude  *float64

	// Image info
	Width     *int
	Height    *int
	DateTaken *time.Time
}

// EXIFService extracts EXIF metadata from images
type EXIFService struct{}

// NewEXIFService creates a new EXIFService
func NewEXIFService() *EXIFService {
	return &EXIFService{}
}

// ExtractFromBytes extracts EXIF data from image bytes
func (s *EXIFService) ExtractFromBytes(data []byte) (*EXIFData, error) {
	return s.ExtractFromReader(bytes.NewReader(data))
}

// ExtractFromReader extracts EXIF data from an io.Reader
func (s *EXIFService) ExtractFromReader(r io.Reader) (*EXIFData, error) {
	x, err := exif.Decode(r)
	if err != nil {
		// No EXIF data or unsupported format - return empty data with defaults
		return &EXIFData{Orientation: 1}, nil
	}

	result := &EXIFData{
		Orientation: 1, // Default orientation
	}

	// Extract camera make
	if tag, err := x.Get(exif.Make); err == nil {
		if val, err := tag.StringVal(); err == nil && val != "" {
			result.CameraMake = &val
		}
	}

	// Extract camera model
	if tag, err := x.Get(exif.Model); err == nil {
		if val, err := tag.StringVal(); err == nil && val != "" {
			result.CameraModel = &val
		}
	}

	// Extract lens model
	if tag, err := x.Get(exif.LensModel); err == nil {
		if val, err := tag.StringVal(); err == nil && val != "" {
			result.LensModel = &val
		}
	}

	// Extract focal length
	if tag, err := x.Get(exif.FocalLength); err == nil {
		if rat, err := tag.Rat(0); err == nil {
			fl := float64(rat.Num().Int64()) / float64(rat.Denom().Int64())
			val := fmt.Sprintf("%.1fmm", fl)
			result.FocalLength = &val
		}
	}

	// Extract aperture (F-number)
	if tag, err := x.Get(exif.FNumber); err == nil {
		if rat, err := tag.Rat(0); err == nil {
			aperture := float64(rat.Num().Int64()) / float64(rat.Denom().Int64())
			val := fmt.Sprintf("f/%.1f", aperture)
			result.Aperture = &val
		}
	}

	// Extract shutter speed (exposure time)
	if tag, err := x.Get(exif.ExposureTime); err == nil {
		if rat, err := tag.Rat(0); err == nil {
			num := rat.Num().Int64()
			denom := rat.Denom().Int64()
			var val string
			if denom == 1 {
				val = fmt.Sprintf("%ds", num)
			} else if num == 1 {
				val = fmt.Sprintf("1/%ds", denom)
			} else {
				// Simplify fraction
				val = fmt.Sprintf("%d/%ds", num, denom)
			}
			result.ShutterSpeed = &val
		}
	}

	// Extract ISO
	if tag, err := x.Get(exif.ISOSpeedRatings); err == nil {
		if val, err := tag.Int(0); err == nil {
			result.ISO = &val
		}
	}

	// Extract orientation
	if tag, err := x.Get(exif.Orientation); err == nil {
		if val, err := tag.Int(0); err == nil && val >= 1 && val <= 8 {
			result.Orientation = val
		}
	}

	// Extract image dimensions
	if tag, err := x.Get(exif.PixelXDimension); err == nil {
		if val, err := tag.Int(0); err == nil {
			result.Width = &val
		}
	} else if tag, err := x.Get(exif.ImageWidth); err == nil {
		if val, err := tag.Int(0); err == nil {
			result.Width = &val
		}
	}

	if tag, err := x.Get(exif.PixelYDimension); err == nil {
		if val, err := tag.Int(0); err == nil {
			result.Height = &val
		}
	} else if tag, err := x.Get(exif.ImageLength); err == nil {
		if val, err := tag.Int(0); err == nil {
			result.Height = &val
		}
	}

	// Extract date taken
	if tm, err := x.DateTime(); err == nil {
		result.DateTaken = &tm
	}

	// Extract GPS coordinates
	lat, lng, err := x.LatLong()
	if err == nil {
		result.Latitude = &lat
		result.Longitude = &lng
	}

	// Extract altitude
	if tag, err := x.Get(exif.GPSAltitude); err == nil {
		if rat, err := tag.Rat(0); err == nil {
			alt := float64(rat.Num().Int64()) / float64(rat.Denom().Int64())
			// Check altitude reference (0 = above sea level, 1 = below)
			if refTag, err := x.Get(exif.GPSAltitudeRef); err == nil {
				if ref, err := refTag.Int(0); err == nil && ref == 1 {
					alt = -alt
				}
			}
			result.Altitude = &alt
		}
	}

	return result, nil
}

// DMSToDecimal converts degrees, minutes, seconds to decimal degrees
// This is a helper for cases where manual GPS parsing is needed
func DMSToDecimal(degrees, minutes, seconds float64, ref string) float64 {
	decimal := degrees + minutes/60 + seconds/3600
	if ref == "S" || ref == "W" {
		decimal = -decimal
	}
	return decimal
}

// FormatCoordinates formats lat/lng as a readable string
func FormatCoordinates(lat, lng float64) string {
	latDir := "N"
	if lat < 0 {
		latDir = "S"
		lat = math.Abs(lat)
	}
	lngDir := "E"
	if lng < 0 {
		lngDir = "W"
		lng = math.Abs(lng)
	}
	return fmt.Sprintf("%.6f°%s, %.6f°%s", lat, latDir, lng, lngDir)
}

// GoogleMapsURL generates a Google Maps URL for coordinates
func GoogleMapsURL(lat, lng float64) string {
	return fmt.Sprintf("https://www.google.com/maps?q=%.6f,%.6f", lat, lng)
}

// Custom tag for lens model (not always in standard library)
func init() {
	// Register additional EXIF tags if needed
	exif.RegisterParsers()
}
