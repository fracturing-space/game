package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve working directory: %w", err)
	}
	g := newGenerator(rootDir)
	switch len(args) {
	case 0:
		return g.generateAll(registry())
	case 1:
		if args[0] == "check" {
			return g.checkAll(registry())
		}
	}
	return fmt.Errorf("usage: go run ./tools/normgen [check]")
}
