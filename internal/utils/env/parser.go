package env

import (
	"fmt"
	"strconv"
	"time"
)

// ParseString returns the input string as-is without validation.
func ParseString(s string) (string, error) {
	return s, nil
}

// ParseNonEmptyString validates that the input string is not empty.
func ParseNonEmptyString(s string) (string, error) {
	if s == "" {
		return "", fmt.Errorf("empty string not allowed")
	}
	return ParseString(s)
}

// ParseInt parses a string as a base-10 int64.
func ParseInt(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// ParseDuration parses a string as a time.Duration (e.g., "30s", "5m").
func ParseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

// ParseBool parses a string as a boolean value.
func ParseBool(s string) (bool, error) {
	return strconv.ParseBool(s)
}
