package main

import (
	"flag"
	"fmt"
	"os"
)

// execute sdm status and capture all the output
func main() {

	flag.Parse()
	if len(flag.Args()) < 1 {
		fmt.Println(usage)
		os.Exit(1)
	}

	if err := checkDependencies(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	command := flag.Args()[0]

	p := newProgram()

	if err := p.Run(command, flag.Args()[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
