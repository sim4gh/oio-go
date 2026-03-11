//go:build !darwin

package platform

import "fmt"

// IsRecordingSupported returns true if screen recording is supported
func IsRecordingSupported() bool {
	return false
}

// HasFFmpeg returns true if ffmpeg is available
func HasFFmpeg() bool {
	return false
}

// RecordScreen is not supported on this platform
func RecordScreen(duration int, selectRegion bool) (string, error) {
	return "", fmt.Errorf("screen recording is only supported on macOS")
}

// ConvertToGIF is not supported on this platform
func ConvertToGIF(movPath string, fps int, width int) (string, error) {
	return "", fmt.Errorf("GIF conversion is only supported on macOS")
}

// ConvertToMP4 is not supported on this platform
func ConvertToMP4(movPath string, width int) (string, error) {
	return "", fmt.Errorf("MP4 conversion is only supported on macOS")
}
