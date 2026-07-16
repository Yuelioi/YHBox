package nodeoptions

import (
	"fmt"
	"strings"

	"github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/internal/services/container"
)

const AsyncSourceSubgraphIDs = "subgraphIDs"

func RegisterSubgraphAsyncSource(nodeSvc *node.NodeService, subgraphSvc *container.SubgraphService) {
	node.RegisterAsyncSource(nodeSvc, AsyncSourceSubgraphIDs, func(_, specKind string, _ map[string]any) ([]node.EnumOption, error) {
		opts := []node.EnumOption{}
		if subgraphSvc == nil {
			return opts, nil
		}
		for _, sg := range subgraphSvc.List() {
			if specKind == "Subgraph" && sg.IsAnonymous {
				continue
			}
			opts = append(opts, node.EnumOption{
				Value: sg.ID,
				Label: formatSubgraphLabel(sg),
				Meta:  subgraphMeta(sg),
			})
		}
		return opts, nil
	})
}

func formatSubgraphLabel(sg container.Subgraph) string {
	label := strings.TrimSpace(sg.Label)
	if label == "" {
		return sg.ID
	}
	return fmt.Sprintf("%s (%s)", label, sg.ID)
}

func subgraphMeta(sg container.Subgraph) map[string]any {
	meta := map[string]any{}
	if sg.Label != "" {
		meta["label"] = sg.Label
	}
	if sg.Category != "" {
		meta["category"] = sg.Category
	}
	if len(sg.Tags) > 0 {
		meta["tags"] = sg.Tags
	}
	if sg.IsAnonymous {
		meta["anonymous"] = true
	}
	return meta
}
