package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/nodes31"
	workflowschema "github.com/yottaapp/yotta/internal/workflow/schema"
)

func main() {
	output := flag.String("output", "contracts/workflow/3.1/workflow-source.schema.json", "JSON Schema output path")
	contractName := flag.String("contract", "workflow", "contract to generate: workflow, diagnostic, node, authoring, builtin-catalog, builtin-authoring, or builtin-docs")
	flag.Parse()

	formatted, err := generate(*contractName)
	if err != nil {
		fail(err)
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fail(err)
	}
	if err := os.WriteFile(*output, formatted, 0o644); err != nil {
		fail(err)
	}
}

func generate(name string) ([]byte, error) {
	if name == "node" {
		return nodecontract.GenerateSchema()
	}
	if name == "authoring" {
		return nodeauthoring.GenerateSchema()
	}
	if name == "builtin-catalog" || name == "builtin-authoring" || name == "builtin-docs" {
		artifacts, err := nodes31.GenerateArtifacts()
		if err != nil {
			return nil, err
		}
		switch name {
		case "builtin-catalog":
			return artifacts.Catalog, nil
		case "builtin-authoring":
			return artifacts.Authoring, nil
		default:
			return artifacts.Documentation, nil
		}
	}
	return workflowschema.GenerateContract(name)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
