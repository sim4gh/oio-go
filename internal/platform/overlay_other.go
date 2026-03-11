//go:build !darwin

package platform

// OverlayProcess is a no-op on non-darwin platforms.
type OverlayProcess struct{}

// HasSwift returns false on non-darwin platforms.
func HasSwift() bool {
	return false
}

// StartOverlay is a no-op on non-darwin platforms.
func StartOverlay(mode string, duration int) *OverlayProcess {
	return nil
}

// Stop is a no-op on non-darwin platforms.
func (o *OverlayProcess) Stop() {}
