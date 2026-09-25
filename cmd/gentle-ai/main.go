package main

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// version is set by GoReleaser via ldflags at build time.
var version = "dev"

const retiredNotice = "gentle-ai CLI ha sido retirado definitivamente; usa 'axiom'"

func run(args []string, stdout, stderr io.Writer) error {
	fmt.Fprintln(stderr, retiredNotice)
	return errors.New(retiredNotice)
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}
