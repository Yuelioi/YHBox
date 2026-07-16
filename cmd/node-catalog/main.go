// node-catalog exports the exact Node Contract 3.1 artifacts compiled into Yotta.
package main

import (
	"fmt"
	"os"

	"github.com/yottaapp/yotta/internal/nodes"
)

const usage = "usage: node-catalog catalog | authoring | docs"

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	raw, err := render(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if _, err := os.Stdout.Write(raw); err != nil {
		fmt.Fprintf(os.Stderr, "write node artifact: %v\n", err)
		os.Exit(1)
	}
	if len(raw) == 0 || raw[len(raw)-1] != '\n' {
		fmt.Println()
	}
}

func render(command string) ([]byte, error) {
	artifacts, err := nodes.GenerateArtifacts()
	if err != nil {
		return nil, fmt.Errorf("generate Node Contract 3.1 artifacts: %w", err)
	}
	switch command {
	case "catalog":
		return artifacts.Catalog, nil
	case "authoring":
		return artifacts.Authoring, nil
	case "docs":
		return artifacts.Documentation, nil
	default:
		return nil, fmt.Errorf("unknown command %q; %s", command, usage)
	}
}
