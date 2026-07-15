package mcpserver

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/yottaapp/yotta/internal/nodeauthoring"
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

var builtinAuthoring31 = sync.OnceValues(func() (nodeauthoring.Snapshot, error) {
	builtins, err := builtinCatalog31()
	if err != nil {
		return nodeauthoring.Snapshot{}, err
	}
	return nodeauthoring.Project(nodeauthoring.Input{
		Catalog: builtins.Catalog, Types: builtins.Types, Capabilities: builtins.Capabilities,
		Contracts: builtins.Contracts, GeneratorVersion: nodes31.GeneratorVersion,
	})
})

func searchCatalog31JSON(query string) []byte {
	builtins, err := builtinCatalog31()
	if err != nil {
		return marshalMCP31(map[string]any{"error": err.Error()})
	}
	projection, err := builtinAuthoring31()
	if err != nil {
		return marshalMCP31(map[string]any{"error": err.Error()})
	}
	query = strings.ToLower(strings.TrimSpace(query))
	items := []catalogSearchItem31{}
	for _, contract := range builtins.Contracts {
		projected, ok := projection.Node(contract.NodeRef().NodeTypeID)
		if !ok {
			continue
		}
		haystack := strings.ToLower(strings.Join([]string{projected.NodeRef.NodeTypeID, projected.TitleKey, projected.DescriptionKey, strings.Join(projected.Tags, " ")}, " "))
		if query == "" || strings.Contains(haystack, query) {
			items = append(items, catalogSearchItem31{
				NodeTypeID: projected.NodeRef.NodeTypeID, NodeRef: projected.NodeRef.SemanticDigest.String(),
				TitleKey: projected.TitleKey, DescriptionKey: projected.DescriptionKey,
				ExecutionClass: string(projected.Execution.Class),
			})
		}
	}
	return marshalMCP31(map[string]any{"format": "yotta.catalog-search", "version": "3.1", "projectionDigest": projection.Digest(), "generatorVersion": nodes31.GeneratorVersion, "items": items})
}

func describeCatalog31JSON(nodeTypeID string) []byte {
	projection, err := builtinAuthoring31()
	if err != nil {
		return marshalMCP31(map[string]any{"error": err.Error()})
	}
	projected, ok := projection.Node(nodeTypeID)
	if !ok {
		return marshalMCP31(map[string]any{"error": "UNKNOWN_NODE_TYPE", "nodeTypeId": nodeTypeID})
	}
	return marshalMCP31(map[string]any{"format": "yotta.catalog-description", "version": "3.1", "projectionDigest": projection.Digest(), "generatorVersion": nodes31.GeneratorVersion, "node": projected})
}

func marshalMCP31(value any) []byte {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return []byte(`{"error":"ENCODE_FAILED"}`)
	}
	return raw
}
