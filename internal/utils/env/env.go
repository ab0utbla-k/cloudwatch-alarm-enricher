// Package env provides type-safe environment variable parsing with validation.
package env

import "os"

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
		return zero, &EnvError{
			Key: key,
			Err: ErrEnvMissing,
		}
	}

	parsed, err := parser(value)
	if err != nil {
		return zero, &EnvError{
			Key: key,
			Err: ErrEnvParsing,
		}
	}
	return parsed, nil
}
