package env

import (
	"fmt"
	"strconv"
	"time"
)

// ParseString returns the input string as-is without validation.
// Empty strings are considered valid values.
func ParseString(s string) (string, error) {
	return s, nil
}

// ParseNonEmptyString validates that the input string is not empty.
// Returns an error if the string is empty.
func ParseNonEmptyString(s string) (string, error) {
	if s == "" {
		return "", fmt.Errorf("empty string not allowed")
	}
	return ParseString(s)
}

func ParseInt(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func ParseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

func ParseBool(s string) (bool, error) {
	return strconv.ParseBool(s)
}
