package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/invopop/jsonschema"
	workflowschema "github.com/yottaapp/yotta/internal/workflow/schema"
)

func main() {
	output := flag.String("output", "contracts/workflow/v3/workflow-source.schema.json", "JSON Schema output path")
	contractName := flag.String("contract", "workflow", "contract to generate: workflow or diagnostic")
	flag.Parse()

	reflector := &jsonschema.Reflector{Anonymous: true, ExpandedStruct: true}
	var contract *jsonschema.Schema
	var id, title string
	switch *contractName {
	case "workflow":
		contract = reflector.Reflect(&workflowschema.WorkflowSource{})
		id = "https://yottaapp.dev/contracts/workflow/v3/workflow-source.schema.json"
		title = "Yotta Workflow Source v3"
	case "diagnostic":
		contract = reflector.Reflect(&workflowschema.Diagnostic{})
		id = "https://yottaapp.dev/contracts/workflow/v3/diagnostic.schema.json"
		title = "Yotta Compiler Diagnostic v3"
	default:
		fail(fmt.Errorf("unknown contract %q", *contractName))
	}
	raw, err := json.Marshal(contract)
	if err != nil {
		fail(err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		fail(err)
	}
	document["$id"] = id
	document["title"] = title

	formatted, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		fail(err)
	}
	formatted = append(formatted, '\n')
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fail(err)
	}
	if err := os.WriteFile(*output, formatted, 0o644); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
