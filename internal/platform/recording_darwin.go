//go:build darwin

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// IsRecordingSupported returns true if screen recording is supported
func IsRecordingSupported() bool {
	return true
}

// HasFFmpeg returns true if ffmpeg is available
func HasFFmpeg() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

// RecordScreen records the screen to a .mov file.
// If selectRegion is true, the user picks a region interactively.
// If duration > 0 and selectRegion is false, it records fullscreen for that many seconds.
// Returns the path to the .mov file (caller must clean up).
func RecordScreen(duration int, selectRegion bool) (string, error) {
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("oio-rec-%d.mov", time.Now().UnixNano()))

	var args []string
	args = append(args, "-v") // video recording mode

	if selectRegion {
		// Interactive region selection — user draws a rectangle, press Ctrl+C or Escape to stop
		args = append(args, "-s")
	} else {
		// Timed fullscreen recording
		if duration <= 0 {
			duration = 10
		}
		args = append(args, "-V", fmt.Sprintf("%d", duration))
	}

	args = append(args, tmpFile)

	cmd := exec.Command("screencapture", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		os.Remove(tmpFile)
		return "", fmt.Errorf("screen recording failed: %w", err)
	}

	// Check if file was created (user might have cancelled)
	info, err := os.Stat(tmpFile)
	if os.IsNotExist(err) || (err == nil && info.Size() == 0) {
		os.Remove(tmpFile)
		return "", nil // user cancelled
	}

	return tmpFile, nil
}

// scaleFilter returns the ffmpeg scale filter string.
// If width is 0, no scaling is applied.
func scaleFilter(width int) string {
	if width <= 0 {
		return ""
	}
	return fmt.Sprintf("scale=%d:-1:flags=lanczos", width)
}

// buildVF joins non-empty video filter parts with commas.
func buildVF(parts ...string) string {
	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	result := ""
	for i, p := range out {
		if i > 0 {
			result += ","
		}
		result += p
	}
	return result
}

// ConvertToGIF converts a .mov file to an optimized GIF using ffmpeg two-pass palette.
// width=0 keeps original resolution.
func ConvertToGIF(movPath string, fps int, width int) (string, error) {
	gifPath := movPath[:len(movPath)-len(filepath.Ext(movPath))] + ".gif"
	palettePath := movPath + "-palette.png"
	defer os.Remove(palettePath)

	vf := buildVF(fmt.Sprintf("fps=%d", fps), scaleFilter(width))

	// Pass 1: generate palette
	pass1 := exec.Command("ffmpeg", "-y",
		"-i", movPath,
		"-vf", vf+",palettegen=stats_mode=diff",
		palettePath,
	)
	pass1.Stderr = nil
	if err := pass1.Run(); err != nil {
		return "", fmt.Errorf("GIF palette generation failed: %w", err)
	}

	// Pass 2: apply palette
	filterComplex := fmt.Sprintf("%s[x];[x][1:v]paletteuse=dither=bayer:bayer_scale=5", vf)
	pass2 := exec.Command("ffmpeg", "-y",
		"-i", movPath,
		"-i", palettePath,
		"-filter_complex", filterComplex,
		gifPath,
	)
	pass2.Stderr = nil
	if err := pass2.Run(); err != nil {
		return "", fmt.Errorf("GIF conversion failed: %w", err)
	}

	return gifPath, nil
}

// ConvertToMP4 converts a .mov file to .mp4 using ffmpeg.
// width=0 keeps original resolution (fast remux). width>0 re-encodes with scaling.
func ConvertToMP4(movPath string, width int) (string, error) {
	mp4Path := movPath[:len(movPath)-len(filepath.Ext(movPath))] + ".mp4"

	var args []string
	args = append(args, "-y", "-i", movPath)

	if width > 0 {
		args = append(args, "-vf", scaleFilter(width), "-c:a", "copy")
	} else {
		args = append(args, "-c", "copy")
	}
	args = append(args, mp4Path)

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("MP4 conversion failed: %w", err)
	}

	return mp4Path, nil
}
