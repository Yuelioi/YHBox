package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yottaapp/yotta/internal/nodecontract"
	workflowschema "github.com/yottaapp/yotta/internal/workflow/schema"
)

func main() {
	output := flag.String("output", "contracts/workflow/v3/workflow-source.schema.json", "JSON Schema output path")
	contractName := flag.String("contract", "workflow", "contract to generate: workflow, diagnostic, or node")
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
	return workflowschema.GenerateContract(name)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
