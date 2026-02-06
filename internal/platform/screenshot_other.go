//go:build !darwin

package platform

import "fmt"

// CaptureScreenshot captures a screenshot (not supported on this platform)
func CaptureScreenshot(window, fullscreen bool) ([]byte, error) {
	return nil, fmt.Errorf("screenshot capture is only supported on macOS")
}
