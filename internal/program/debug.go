package program

import "fmt"

func printDebug(msg string) {
	if *debugMode {
		fmt.Printf("[DEBUG]: %s\n", msg)
	}
}
