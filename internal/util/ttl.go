package util

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var ttlRegex = regexp.MustCompile(`^(\d+)([smhd])$`)

// ParseTTL parses a TTL string (e.g., "30s", "60m", "24h", "7d") to seconds
func ParseTTL(ttlString string) (int, error) {
	if ttlString == "" {
		return 0, errors.New("invalid TTL format. Use: 30s, 60m, 24h, or 7d")
	}

	matches := ttlRegex.FindStringSubmatch(ttlString)
	if matches == nil {
		return 0, errors.New("invalid TTL format. Use: 30s, 60m, 24h, or 7d")
	}

	value, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, err
	}

	if value <= 0 {
		return 0, errors.New("TTL value must be greater than 0")
	}

	unit := matches[2]

	switch unit {
	case "s":
		if value > 31536000 {
			return 0, errors.New("TTL in seconds cannot exceed 1 year (31536000s)")
		}
		return value, nil
	case "m":
		if value > 525600 {
			return 0, errors.New("TTL in minutes cannot exceed 1 year (525600m)")
		}
		return value * 60, nil
	case "h":
		if value > 8760 {
			return 0, errors.New("TTL in hours cannot exceed 1 year (8760h)")
		}
		return value * 3600, nil
	case "d":
		if value > 365 {
			return 0, errors.New("TTL in days cannot exceed 1 year (365d)")
		}
		return value * 86400, nil
	default:
		return 0, errors.New("invalid unit. Use s (seconds), m (minutes), h (hours), or d (days)")
	}
}

// FormatExpiry formats expiry timestamp to human-readable relative time
func FormatExpiry(expiresAt int64) string {
	if expiresAt == 0 {
		return "unknown"
	}

	now := time.Now().Unix()
	remaining := expiresAt - now

	if remaining < 0 {
		return "expired"
	}
	if remaining < 60 {
		return fmt.Sprintf("%ds", remaining)
	}
	if remaining < 3600 {
		return fmt.Sprintf("%dm", remaining/60)
	}
	if remaining < 86400 {
		return fmt.Sprintf("%dh", remaining/3600)
	}
	return fmt.Sprintf("%dd", remaining/86400)
}

// FormatExpiryTime formats expiry timestamp to full date/time
func FormatExpiryTime(expiresAt int64) string {
	if expiresAt == 0 {
		return "never (permanent)"
	}

	expirationDate := time.Unix(expiresAt, 0)
	now := time.Now()
	diff := expirationDate.Sub(now)

	if diff <= 0 {
		return "expired"
	}

	days := int(diff.Hours() / 24)
	hours := int(diff.Hours()) % 24
	minutes := int(diff.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("in %d day%s (%s)", days, pluralize(days), expirationDate.Format("Jan 2, 2006"))
	} else if hours > 0 {
		return fmt.Sprintf("in %d hour%s", hours, pluralize(hours))
	} else if minutes > 0 {
		return fmt.Sprintf("in %d minute%s", minutes, pluralize(minutes))
	} else {
		seconds := int(diff.Seconds())
		return fmt.Sprintf("in %d second%s", seconds, pluralize(seconds))
	}
}

// IsValidTTL validates a TTL string format
func IsValidTTL(ttlString string) bool {
	return ttlRegex.MatchString(ttlString)
}

// SecondsToTTL converts seconds to human-readable TTL format
func SecondsToTTL(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm", seconds/60)
	}
	if seconds < 86400 {
		return fmt.Sprintf("%dh", seconds/3600)
	}
	return fmt.Sprintf("%dd", seconds/86400)
}

func pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
