package platform

import (
	"os/exec"
	"runtime"
	"strings"
)

// IsScreenshotSupported returns true if screenshot capture is supported on this platform
func IsScreenshotSupported() bool {
	return runtime.GOOS == "darwin"
}

// ClipboardHasImage checks if clipboard contains image data (macOS only)
func ClipboardHasImage() bool {
	if runtime.GOOS != "darwin" {
		return false
	}

	cmd := exec.Command("osascript", "-e", "clipboard info")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	imageTypes := []string{"PNGf", "JPEG", "TIFF", "GIF", "jp2 ", "BMP", "AVIF"}
	outputStr := string(output)
	for _, t := range imageTypes {
		if strings.Contains(outputStr, t) {
			return true
		}
	}
	return false
}
