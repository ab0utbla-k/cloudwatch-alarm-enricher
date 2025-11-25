package env

import (
	"errors"
	"fmt"
)

var (
	// ErrEnvMissing indicates a required environment variable was not set.
	ErrEnvMissing = errors.New("environment variable missing")
	// ErrEnvParsing indicates an environment variable could not be parsed.
	ErrEnvParsing = errors.New("environment variable parsing failed")
)

// EnvError represents an environment variable error with the variable name.
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
