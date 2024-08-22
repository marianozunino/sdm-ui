package app

import (
	"fmt"
	"os/exec"

	"github.com/rs/zerolog/log"
)

var dependencies = []string{
	"sdm",
}

func (app *App) mustHaveDependencies() {
	log.Debug().Msg("Checking dependencies...")

	if app.passwordCommand == PasswordCommandZenity {
		dependencies = append(dependencies, "zenity")
	}

	if app.dmenuCommand != DMenuCommandNoop {
		dependencies = append(dependencies, app.dmenuCommand.String())
	}

	for _, dependency := range dependencies {
		log.Debug().Msg(fmt.Sprintf("Checking dependency: %s", dependency))
		_, err := exec.LookPath(dependency)
		if err != nil {
			log.Fatal().Msg(fmt.Sprintf("Dependency not found: %s", dependency))
		}
	}

	log.Debug().Msg("Dependencies OK")
}
