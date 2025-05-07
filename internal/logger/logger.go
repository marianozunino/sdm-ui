package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ConfigureLogger sets up zerolog with the specified debug level and formatting
func ConfigureLogger(debug bool) {
	// Set the global logging level based on the debug flag
	level := zerolog.InfoLevel
	if debug {
		level = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(level)

	// Configure the console writer with customized formatting
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "", // Empty TimeFormat for default formatting
		FormatMessage: func(i interface{}) string {
			if i == nil {
				return ""
			}
			return fmt.Sprintf("%s", i)
		},
		FormatLevel: func(i interface{}) string {
			if i == nil {
				return ""
			}
			if debug {
				return strings.ToUpper(fmt.Sprintf("[%s]", i))
			}
			// In non-debug mode, only show level for warnings and errors
			level := strings.ToUpper(fmt.Sprintf("%s", i))
			if level == "WARN" || level == "ERROR" {
				return fmt.Sprintf("[%s]", level)
			}
			return ""
		},
		FormatCaller: func(i interface{}) string {
			if i == nil {
				return ""
			}
			return filepath.Base(fmt.Sprintf("%s >", i))
		},
		// Exclude timestamps in non-debug mode
		PartsExclude: []string{"time"},
	}

	// Create the logger
	logger := log.Output(consoleWriter)

	// Configure logger based on debug mode
	if debug {
		logger = logger.With().Caller().Timestamp().Logger()
	} else {
		// Don't include timestamp in non-debug mode
		logger = logger.With().Logger()
	}

	// Replace the global logger
	log.Logger = logger
}
