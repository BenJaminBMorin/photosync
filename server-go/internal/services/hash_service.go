package services

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"regexp"
	"strings"
)

// HashService handles file hashing
type HashService struct {
	sha256Regex *regexp.Regexp
}

// NewHashService creates a new HashService
func NewHashService() *HashService {
	return &HashService{
		sha256Regex: regexp.MustCompile(`^[a-f0-9]{64}$`),
	}
}

// ComputeHash computes the SHA256 hash of a reader
func (s *HashService) ComputeHash(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ComputeHashBytes computes the SHA256 hash of bytes
func (s *HashService) ComputeHashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// NormalizeHash normalizes a hash string to lowercase
func (s *HashService) NormalizeHash(hash string) string {
	normalized := strings.TrimSpace(hash)

	// Remove "sha256:" prefix if present
	if strings.HasPrefix(strings.ToLower(normalized), "sha256:") {
		normalized = normalized[7:]
	}

	return strings.ToLower(normalized)
}

// IsValidHash checks if a string is a valid SHA256 hash
func (s *HashService) IsValidHash(hash string) bool {
	if strings.TrimSpace(hash) == "" {
		return false
	}

	normalized := s.NormalizeHash(hash)
	return s.sha256Regex.MatchString(normalized)
}
