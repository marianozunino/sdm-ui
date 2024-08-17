package program

import (
	"fmt"
	"os/exec"

	"github.com/rs/zerolog/log"
)

var dependencies = [3]string{
	"sdm",
	"zenity",
	"rofi",
}

func mustHaveDependencies() {
	log.Debug().Msg("Checking dependencies...")
	for _, dependency := range dependencies {
		log.Debug().Msg(fmt.Sprintf("Checking dependency: %s", dependency))
		_, err := exec.LookPath(dependency)
		if err != nil {
			log.Fatal().Msg(fmt.Sprintf("Dependency not found: %s", dependency))
		}
	}
	log.Debug().Msg("Dependencies OK")
}
