package main

import (
	"os"

	"github.com/kbsink-org/kbsink-cli/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args))
}
