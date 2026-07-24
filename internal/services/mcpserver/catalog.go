package mcpserver

import (
	"fmt"
	"strings"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
)

const (
	CatalogSearchFormat       = "yotta.catalog-search"
	CatalogSearchVersion      = "1"
	CatalogDescriptionFormat  = "yotta.catalog-description"
	CatalogDescriptionVersion = "1"
)

type CatalogSearchRequest struct {
	Query  string `json:"query,omitempty" jsonschema:"description=Node type ID title description or tag substring"`
	Offset int    `json:"offset,omitempty" jsonschema:"minimum=0"`
	Limit  int    `json:"limit,omitempty" jsonschema:"minimum=1,maximum=100"`
}

type CatalogSearchItem struct {
	NodeTypeID     string          `json:"nodeTypeId"`
	SemanticDigest artifact.Digest `json:"semanticDigest"`
	TitleKey       string          `json:"titleKey,omitempty"`
	DescriptionKey string          `json:"descriptionKey,omitempty"`
	ExecutionClass string          `json:"executionClass"`
}

type CatalogSearchResult struct {
	Format           string              `json:"format"`
	Version          string              `json:"version"`
	ProjectionDigest artifact.Digest     `json:"projectionDigest"`
	GeneratorVersion string              `json:"generatorVersion"`
	Offset           int                 `json:"offset"`
	Total            int                 `json:"total"`
	Items            []CatalogSearchItem `json:"items"`
}

type CatalogDescribeRequest struct {
	NodeTypeID string `json:"nodeTypeId" jsonschema:"required,description=Absolute versioned Node Type ID"`
}

type CatalogDescribeResult struct {
	Format           string                       `json:"format"`
	Version          string                       `json:"version"`
	ProjectionDigest artifact.Digest              `json:"projectionDigest"`
	GeneratorVersion string                       `json:"generatorVersion"`
	Node             nodeauthoring.NodeProjection `json:"node"`
}

func searchCatalog(projection nodeauthoring.Snapshot, request CatalogSearchRequest) (CatalogSearchResult, error) {
	offset, limit, err := boundedPage(request.Offset, request.Limit)
	if err != nil {
		return CatalogSearchResult{}, err
	}
	query := strings.ToLower(strings.TrimSpace(request.Query))
	items := make([]CatalogSearchItem, 0)
	for _, projected := range projection.Nodes() {
		haystack := strings.ToLower(strings.Join([]string{
			projected.NodeRef.NodeTypeID, projected.TitleKey, projected.DescriptionKey, strings.Join(projected.Tags, " "),
		}, " "))
		if query != "" && !strings.Contains(haystack, query) {
			continue
		}
		items = append(items, CatalogSearchItem{
			NodeTypeID: projected.NodeRef.NodeTypeID, SemanticDigest: projected.NodeRef.SemanticDigest,
			TitleKey: projected.TitleKey, DescriptionKey: projected.DescriptionKey,
			ExecutionClass: string(projected.Execution.Class),
		})
	}
	return CatalogSearchResult{
		Format: CatalogSearchFormat, Version: CatalogSearchVersion, ProjectionDigest: projection.Digest(),
		GeneratorVersion: projection.GeneratorVersion(), Offset: offset, Total: len(items),
		Items: page(items, offset, limit),
	}, nil
}

func describeCatalog(projection nodeauthoring.Snapshot, request CatalogDescribeRequest) (CatalogDescribeResult, error) {
	projected, ok := projection.Node(request.NodeTypeID)
	if !ok {
		return CatalogDescribeResult{}, fmt.Errorf("UNKNOWN_NODE_TYPE: %s", request.NodeTypeID)
	}
	return CatalogDescribeResult{
		Format: CatalogDescriptionFormat, Version: CatalogDescriptionVersion, ProjectionDigest: projection.Digest(),
		GeneratorVersion: projection.GeneratorVersion(), Node: projected,
	}, nil
}
