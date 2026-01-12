// Package env provides type-safe environment variable parsing with validation.
package env

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

var (
	// ErrMissing indicates a required environment variable was not set.
	ErrMissing = errors.New("environment variable missing")
	// ErrParsing indicates an environment variable could not be parsed.
	ErrParsing = errors.New("environment variable parsing failed")
)

// Error represents an environment variable error with the variable name.
type Error struct {
	Key string
	Err error
}

func (e *Error) Error() string {
	return fmt.Sprintf("environment variable %s: %v", e.Key, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// Get retrieves an environment variable with a default value.
// If the variable is not set or parsing fails, returns the default value.
func Get[T any](key string, defaultValue T, parser func(string) (T, error)) T {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}

	parsed, err := parser(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

// GetRequired retrieves a required environment variable.
// Returns an error if the variable is not set or parsing fails.
func GetRequired[T any](key string, parser func(string) (T, error)) (T, error) {
	var zero T

	value, ok := os.LookupEnv(key)
	if !ok {
		return zero, &Error{Key: key, Err: ErrMissing}
	}

	parsed, err := parser(value)
	if err != nil {
		return zero, &Error{Key: key, Err: errors.Join(ErrParsing, err)}
	}
	return parsed, nil
}

// ParseString returns the input string as-is without validation.
func ParseString(s string) (string, error) {
	return s, nil
}

// ParseNonEmptyString validates that the input string is not empty.
func ParseNonEmptyString(s string) (string, error) {
	if s == "" {
		return "", errors.New("empty string not allowed")
	}
	return s, nil
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
