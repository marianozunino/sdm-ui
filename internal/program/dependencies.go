package program

import (
	"fmt"
	"os/exec"
)

var dependencies = []string{
	"sdm", "zenity", "rofi",
}

func mustHaveDependencies() {
	printDebug("Checking dependencies...")
	for _, dependency := range dependencies {
		printDebug(fmt.Sprintf("Checking dependency: %s", dependency))
		_, err := exec.LookPath(dependency)
		if err != nil {
			panic(err)
		}
	}
	printDebug("Dependencies OK")
}
