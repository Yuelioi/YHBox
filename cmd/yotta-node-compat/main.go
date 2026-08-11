// Command yotta-node-compat freezes and verifies built-in Catalog identities
// published by stable Yotta releases.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/releasecompat"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	flags := flag.NewFlagSet("yotta-node-compat", flag.ContinueOnError)
	write := flags.Bool("write", false, "freeze the current product release before checking")
	requireCurrent := flags.Bool("require-current", false, "require a snapshot for the current product version")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: yotta-node-compat [--write] [--require-current]")
	}
	root, err := repositoryRoot()
	if err != nil {
		return err
	}
	rawVersion, err := os.ReadFile(filepath.Join(root, "VERSION"))
	if err != nil {
		return err
	}
	productVersion := strings.TrimSpace(string(rawVersion))
	builtins, err := nodes.Build()
	if err != nil {
		return fmt.Errorf("build current Node Catalog: %w", err)
	}
	refs := builtins.Catalog.NodeRefs()
	typeRefs := builtins.Catalog.TypeRefs()
	capabilityRefs := builtins.Catalog.CapabilityRefs()
	nodeReleases := releasecompat.NodeReleases{Root: filepath.Join(root, "contracts", "node", "releases")}
	catalogReleases := releasecompat.CatalogReleases{Root: filepath.Join(root, "contracts", "catalog", "releases")}
	if *write {
		if err := nodeReleases.Freeze(productVersion, refs); err != nil {
			return err
		}
		if err := catalogReleases.Freeze(productVersion, typeRefs, capabilityRefs); err != nil {
			return err
		}
	}
	nodeFloors, err := nodeReleases.Check(productVersion, refs, *requireCurrent || *write)
	if err != nil {
		return err
	}
	catalogFloors, err := catalogReleases.Check(productVersion, typeRefs, capabilityRefs, *requireCurrent || *write)
	if err != nil {
		return err
	}
	fmt.Printf(
		"built-in Catalog compatibility OK: %d NodeRef floor(s), %d Catalog floor(s), %d nodes, %d data types, %d capabilities\n",
		nodeFloors, catalogFloors, len(refs), len(typeRefs), len(capabilityRefs),
	)
	return nil
}

func repositoryRoot() (string, error) {
	current, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if info, statErr := os.Stat(filepath.Join(current, "VERSION")); statErr == nil && info.Mode().IsRegular() {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("repository root containing VERSION was not found")
		}
		current = parent
	}
}
