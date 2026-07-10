package main

import (
	"os"

	"github.com/dannyben/mob/cmd"
)

var Version = "0.0.0-dev"

func main() {
	if err := cmd.ExecuteWithIO(os.Args[1:], Version, os.Stdin, os.Stdout, os.Stderr); err != nil {
		cmd.PrintError(err, os.Stderr)
		os.Exit(1)
	}
}
