package program

import (
	"fmt"
	"os/exec"

	"github.com/rs/zerolog/log"
)

var dependencies = []string{
	"sdm",
	"zenity",
}

func mustHaveDependencies(dmenuCommand DMenuCommand) {
	log.Debug().Msg("Checking dependencies...")

	for _, dependency := range dependencies {
		log.Debug().Msg(fmt.Sprintf("Checking dependency: %s", dependency))
		_, err := exec.LookPath(dependency)
		if err != nil {
			log.Fatal().Msg(fmt.Sprintf("Dependency not found: %s", dependency))
		}
	}

	log.Debug().Msg(fmt.Sprintf("Checking dmenu command: %s", dmenuCommand.String()))
	_, err := exec.LookPath(dmenuCommand.String())
	if err != nil {
		log.Fatal().Msg(fmt.Sprintf("Dependency not found: %s", dmenuCommand.String()))
	}

	log.Debug().Msg("Dependencies OK")
}
