//go:build !darwin

package platform

import "fmt"

// GetClipboardImage extracts image from clipboard (not supported on this platform)
func GetClipboardImage() ([]byte, error) {
	return nil, fmt.Errorf("clipboard image extraction is only supported on macOS")
}
