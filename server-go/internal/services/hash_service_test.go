package services

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashService_ComputeHash(t *testing.T) {
	svc := NewHashService()

	t.Run("returns consistent hash for same content", func(t *testing.T) {
		content := []byte("Hello, World!")
		reader1 := bytes.NewReader(content)
		reader2 := bytes.NewReader(content)

		hash1, err := svc.ComputeHash(reader1)
		require.NoError(t, err)

		hash2, err := svc.ComputeHash(reader2)
		require.NoError(t, err)

		assert.Equal(t, hash1, hash2)
		assert.Len(t, hash1, 64) // SHA256 = 64 hex chars
	})

	t.Run("returns different hash for different content", func(t *testing.T) {
		hash1, err := svc.ComputeHash(bytes.NewReader([]byte("Content A")))
		require.NoError(t, err)

		hash2, err := svc.ComputeHash(bytes.NewReader([]byte("Content B")))
		require.NoError(t, err)

		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("returns lowercase hash", func(t *testing.T) {
		hash, err := svc.ComputeHash(bytes.NewReader([]byte("test")))
		require.NoError(t, err)

		assert.Equal(t, strings.ToLower(hash), hash)
	})
}

func TestHashService_ComputeHashBytes(t *testing.T) {
	svc := NewHashService()

	t.Run("returns consistent hash", func(t *testing.T) {
		content := []byte("Hello, World!")

		hash1 := svc.ComputeHashBytes(content)
		hash2 := svc.ComputeHashBytes(content)

		assert.Equal(t, hash1, hash2)
		assert.Len(t, hash1, 64)
	})
}

func TestHashService_IsValidHash(t *testing.T) {
	svc := NewHashService()

	tests := []struct {
		name     string
		hash     string
		expected bool
	}{
		{"valid lowercase", "abc123def456abc123def456abc123def456abc123def456abc123def456abcd", true},
		{"valid uppercase", "ABC123DEF456ABC123DEF456ABC123DEF456ABC123DEF456ABC123DEF456ABCD", true},
		{"valid with prefix", "sha256:abc123def456abc123def456abc123def456abc123def456abc123def456abcd", true},
		{"empty", "", false},
		{"whitespace", "   ", false},
		{"too short", "abc123", false},
		{"invalid char", "abc123def456abc123def456abc123def456abc123def456abc123def456abcZ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.IsValidHash(tt.hash)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHashService_NormalizeHash(t *testing.T) {
	svc := NewHashService()

	t.Run("removes sha256 prefix", func(t *testing.T) {
		input := "sha256:abc123def456abc123def456abc123def456abc123def456abc123def456abcd"
		expected := "abc123def456abc123def456abc123def456abc123def456abc123def456abcd"

		result := svc.NormalizeHash(input)
		assert.Equal(t, expected, result)
	})

	t.Run("converts to lowercase", func(t *testing.T) {
		input := "ABC123DEF456ABC123DEF456ABC123DEF456ABC123DEF456ABC123DEF456ABCD"
		expected := "abc123def456abc123def456abc123def456abc123def456abc123def456abcd"

		result := svc.NormalizeHash(input)
		assert.Equal(t, expected, result)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		input := "  abc123  "
		expected := "abc123"

		result := svc.NormalizeHash(input)
		assert.Equal(t, expected, result)
	})
}
