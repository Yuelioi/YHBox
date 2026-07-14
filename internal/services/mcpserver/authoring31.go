package mcpserver

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/yottaapp/yotta/internal/nodes31"
)

type catalogSearchItem31 struct {
	NodeTypeID     string `json:"nodeTypeId"`
	NodeRef        string `json:"semanticDigest"`
	TitleKey       string `json:"titleKey"`
	DescriptionKey string `json:"descriptionKey"`
	ExecutionClass string `json:"executionClass"`
}

var builtinCatalog31 = sync.OnceValues(func() (nodes31.Builtins, error) {
	return nodes31.Build()
})

var builtinPresentationMetadata31 = sync.OnceValues(func() (map[string]any, error) {
	artifacts, err := nodes31.GenerateArtifacts()
	if err != nil {
		return nil, err
	}
	var document struct {
		PresentationDigest string `json:"presentationDigest"`
		Body               struct {
			GeneratorVersion string `json:"generatorVersion"`
		} `json:"body"`
	}
	if err := json.Unmarshal(artifacts.Presentation, &document); err != nil {
		return nil, err
	}
	return map[string]any{"presentationDigest": document.PresentationDigest, "generatorVersion": document.Body.GeneratorVersion}, nil
})

func searchCatalog31JSON(query string) []byte {
	builtins, err := builtinCatalog31()
	if err != nil {
		return marshalMCP31(map[string]any{"error": err.Error()})
	}
	contract := builtins.ConcatContract
	machine, authoring := contract.Machine(), contract.Authoring()
	query = strings.ToLower(strings.TrimSpace(query))
	haystack := strings.ToLower(strings.Join([]string{machine.NodeTypeID, authoring.TitleKey, authoring.DescriptionKey, strings.Join(authoring.Tags, " ")}, " "))
	items := []catalogSearchItem31{}
	if query == "" || strings.Contains(haystack, query) {
		items = append(items, catalogSearchItem31{
			NodeTypeID: machine.NodeTypeID, NodeRef: contract.NodeRef().SemanticDigest.String(),
			TitleKey: authoring.TitleKey, DescriptionKey: authoring.DescriptionKey,
			ExecutionClass: string(machine.Execution.Class),
		})
	}
	metadata, err := builtinPresentationMetadata31()
	if err != nil {
		return marshalMCP31(map[string]any{"error": err.Error()})
	}
	return marshalMCP31(map[string]any{"format": "yotta.catalog-search", "version": "3.1", "presentationDigest": metadata["presentationDigest"], "generatorVersion": metadata["generatorVersion"], "items": items})
}

func describeCatalog31JSON(nodeTypeID string) []byte {
	builtins, err := builtinCatalog31()
	if err != nil {
		return marshalMCP31(map[string]any{"error": err.Error()})
	}
	if nodeTypeID != builtins.ConcatContract.NodeRef().NodeTypeID {
		return marshalMCP31(map[string]any{"error": "UNKNOWN_NODE_TYPE", "nodeTypeId": nodeTypeID})
	}
	var contract any
	if err := json.Unmarshal(builtins.ConcatContract.Bytes(), &contract); err != nil {
		return marshalMCP31(map[string]any{"error": err.Error()})
	}
	metadata, err := builtinPresentationMetadata31()
	if err != nil {
		return marshalMCP31(map[string]any{"error": err.Error()})
	}
	return marshalMCP31(map[string]any{"format": "yotta.catalog-description", "version": "3.1", "presentationDigest": metadata["presentationDigest"], "generatorVersion": metadata["generatorVersion"], "contract": contract})
}

func marshalMCP31(value any) []byte {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return []byte(`{"error":"ENCODE_FAILED"}`)
	}
	return raw
}
