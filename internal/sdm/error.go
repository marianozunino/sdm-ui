package sdm

import (
	"strings"

	"github.com/rs/zerolog/log"
)

type SDMErrorCode int

const (
	Unauthorized SDMErrorCode = iota
	InvalidCredentials
	Unknown
	ResourceNotFound
)

type SDMError struct {
	Code SDMErrorCode
	Msg  string
}

func (e SDMError) Error() string {
	return e.Msg
}

// parseSdmError parses the output and error to return an SDMError with the appropriate code.
func parseSdmError(output string, err error) error {
	log.Debug().Msgf("Parsing error: %s", output)
	if err == nil {
		return nil
	}
	switch {
	case strings.Contains(output, "You are not authenticated"):
		return SDMError{Code: Unauthorized, Msg: output}
	case strings.Contains(output, "Cannot find datasource named"):
		return SDMError{Code: ResourceNotFound, Msg: output}
	case strings.Contains(output, "access denied"):
		return SDMError{Code: InvalidCredentials, Msg: output}
	case strings.Contains(output, "Invalid credentials"):
		return SDMError{Code: InvalidCredentials, Msg: output}
	default:
		return SDMError{Code: Unknown, Msg: output}
	}
}
