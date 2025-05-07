package sdm

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

// SDMErrorCode represents the type of error encountered during SDM operations
type SDMErrorCode int

// Error code constants
const (
	Unauthorized SDMErrorCode = iota
	InvalidCredentials
	ResourceNotFound
	ConnectionFailed
	PermissionDenied
	Unknown
)

// String returns a string representation of the error code
func (c SDMErrorCode) String() string {
	switch c {
	case Unauthorized:
		return "Unauthorized"
	case InvalidCredentials:
		return "InvalidCredentials"
	case ResourceNotFound:
		return "ResourceNotFound"
	case ConnectionFailed:
		return "ConnectionFailed"
	case PermissionDenied:
		return "PermissionDenied"
	default:
		return "Unknown"
	}
}

// SDMError represents an error returned by the SDM CLI
type SDMError struct {
	Code SDMErrorCode
	Msg  string
	Err  error // Original error
}

// Error implements the error interface
func (e SDMError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code.String(), e.Msg, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code.String(), e.Msg)
}

// Unwrap implements the errors.Unwrap interface
func (e SDMError) Unwrap() error {
	return e.Err
}

// parseSdmError parses the output and error to return an SDMError with the appropriate code
func parseSdmError(output string, err error) error {
	if err == nil {
		return nil
	}

	log.Debug().
		Str("output", output).
		Err(err).
		Msg("Parsing SDM error")

	// Define error matchers with their respective error codes
	errorMatchers := []struct {
		pattern string
		code    SDMErrorCode
	}{
		{"You are not authenticated", Unauthorized},
		{"Authentication required", Unauthorized},
		{"Cannot find datasource named", ResourceNotFound},
		{"Resource not found", ResourceNotFound},
		{"access denied", InvalidCredentials},
		{"Invalid credentials", InvalidCredentials},
		{"Permission denied", PermissionDenied},
		{"Connection refused", ConnectionFailed},
		{"Could not connect", ConnectionFailed},
		{"Timed out", ConnectionFailed},
	}

	// Check for known error patterns
	for _, matcher := range errorMatchers {
		if strings.Contains(output, matcher.pattern) {
			return SDMError{
				Code: matcher.code,
				Msg:  output,
				Err:  err,
			}
		}
	}

	// If no specific pattern is matched, return an unknown error
	return SDMError{
		Code: Unknown,
		Msg:  output,
		Err:  err,
	}
}
