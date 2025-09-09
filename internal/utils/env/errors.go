package env

import (
	"errors"
	"fmt"
)

// Sentinel errors for common cases
var (
	ErrEnvMissing = errors.New("environment variable missing")
	ErrEnvParsing = errors.New("environment variable parsing failed")
)

// EnvError represents an environment variable related error
type EnvError struct {
	Key string
	Err error
}

func (e *EnvError) Error() string {
	return fmt.Sprintf("environment variable %s: %v", e.Key, e.Err)
}

func (e *EnvError) Unwrap() error {
	return e.Err
}
