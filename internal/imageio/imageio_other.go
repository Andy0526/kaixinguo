//go:build !darwin

// Package imageio decodes the pictures Go's image packages can't, such as
// HEIC covers, with macOS's ImageIO. Elsewhere it decodes none.
package imageio

import (
	"errors"
	"image"
)

// Decode is not available outside macOS.
func Decode(data []byte, maxWidth int) (*image.RGBA, error) {
	return nil, errors.New("imageio: only on macOS")
}
