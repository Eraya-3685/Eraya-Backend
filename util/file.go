package util

import (
	"errors"
	"strings"
)

var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
	"image/svg+xml": true,
}

// IsValidImage checks if the given content-type is a supported image format.
func IsValidImage(contentType string) bool {
	return allowedImageTypes[strings.ToLower(contentType)]
}

// ValidateImage returns an error if the content-type is not a supported image format.
func ValidateImage(contentType string) error {
	if !IsValidImage(contentType) {
		return errors.New("unsupported file format. please upload an image (JPG, PNG, WebP, GIF, or SVG)")
	}
	return nil
}

// ValidateImageSize returns an error if the file size exceeds the max limit (default 2MB).
func ValidateImageSize(size int64, maxMB int) error {
	maxSize := int64(maxMB) * 1024 * 1024
	if size > maxSize {
		return errors.New("file is too large. maximum allowed size is 2MB")
	}
	return nil
}
