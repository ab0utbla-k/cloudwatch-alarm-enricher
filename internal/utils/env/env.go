package env

import "os"

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
