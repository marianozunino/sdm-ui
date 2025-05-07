package app

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/rs/zerolog/log"
)

// ErrDependencyNotFound is returned when a required dependency is not found
var ErrDependencyNotFound = errors.New("dependency not found")

// mustHaveDependencies checks if all required dependencies are available
func (app *App) mustHaveDependencies() error {
	log.Debug().Msg("Checking dependencies...")

	// Create a fresh slice to avoid modifying the static dependencies slice
	requiredDeps := []string{"sdm"}

	// Add password command dependency if needed
	if app.passwordCommand == PasswordCommandZenity {
		requiredDeps = append(requiredDeps, "zenity")
	}

	// Add dmenu command dependency if needed
	if app.dmenuCommand != DMenuCommandNoop {
		requiredDeps = append(requiredDeps, app.dmenuCommand.String())
	}

	// Check for each dependency
	for _, dependency := range requiredDeps {
		log.Debug().Str("dependency", dependency).Msg("Checking dependency")

		path, err := exec.LookPath(dependency)
		if err != nil {
			log.Error().
				Err(err).
				Str("dependency", dependency).
				Msg("Dependency not found")

			return fmt.Errorf("%w: %s", ErrDependencyNotFound, dependency)
		}

		log.Debug().
			Str("dependency", dependency).
			Str("path", path).
			Msg("Dependency found")
	}

	log.Debug().Msg("All dependencies available")
	return nil
}
