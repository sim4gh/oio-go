package util

import (
	"fmt"
	"math"
	"strings"
)

// FormatBytes converts bytes to human-readable string (e.g., "1.5 MB")
func FormatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}

	units := []string{"B", "KB", "MB", "GB"}
	k := float64(1024)
	i := int(math.Floor(math.Log(float64(bytes)) / math.Log(k)))

	if i >= len(units) {
		i = len(units) - 1
	}

	value := float64(bytes) / math.Pow(k, float64(i))

	if i > 0 {
		return fmt.Sprintf("%.1f %s", value, units[i])
	}
	return fmt.Sprintf("%.0f %s", value, units[i])
}

// Truncate truncates text to specified length with ellipsis
func Truncate(text string, length int) string {
	if text == "" {
		return ""
	}
	if len(text) <= length {
		return text
	}
	if length <= 3 {
		return text[:length]
	}
	return text[:length-3] + "..."
}

// CreateProgressBar creates a progress bar string
func CreateProgressBar(current, total int64, width int) string {
	if total == 0 {
		return strings.Repeat(" ", width+10)
	}

	percent := int(math.Min(100, float64(current)/float64(total)*100))
	filled := int(float64(percent) / 100 * float64(width))
	empty := width - filled

	return fmt.Sprintf("[%s%s] %d%%", strings.Repeat("=", filled), strings.Repeat(" ", empty), percent)
}

// MaskToken masks a token for display, showing only first and last 4 chars
func MaskToken(token string) string {
	if token == "" {
		return "(not set)"
	}
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

// ReplaceNewlines replaces newlines with spaces for single-line display
func ReplaceNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
