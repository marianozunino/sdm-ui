package main

import (
	"fmt"
	"os"

	"github.com/marianozunino/sdm-ui/internal/program"
)

// execute sdm status and capture all the output
func main() {

	p := program.NewProgram()

	if err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
