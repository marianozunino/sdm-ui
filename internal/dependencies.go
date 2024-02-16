package internal

import (
	"fmt"
	"os/exec"
)

var dependencies = []string{
	"sdm", "zenity",
}

func checkDependencies() error {
	printDebug("Checking dependencies...")
	for _, dependency := range dependencies {
		printDebug(fmt.Sprintf("Checking dependency: %s", dependency))
		_, err := exec.LookPath(dependency)
		if err != nil {
			return err
		}
	}
	return nil
}

func printDebug(msg string) {
	if *debugMode {
		fmt.Printf("[DEBUG]: %s\n", msg)
	}
}
