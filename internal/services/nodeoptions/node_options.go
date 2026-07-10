package nodeoptions

import (
	"fmt"
	"strings"

	"github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/internal/services/asset"
	"github.com/yottaapp/yotta/internal/services/container"
)

const (
	AsyncSourceClipIDs     = "clipIDs"
	AsyncSourceSubgraphIDs = "subgraphIDs"
)

func RegisterAssetAsyncSources(nodeSvc *node.NodeService, assetSvc *asset.Service, subgraphSvc *container.SubgraphService) {
	nodeSvc.RegisterAsyncSource(AsyncSourceClipIDs, func(_, _ string, _ map[string]any) ([]node.EnumOption, error) {
		opts := []node.EnumOption{}
		if assetSvc == nil {
			return opts, nil
		}
		for _, item := range assetSvc.List() {
			if item.Kind != asset.KindClip {
				continue
			}
			opts = append(opts, node.EnumOption{
				Value: item.GUID,
				Label: formatAssetLabel(item.Name, item.GUID),
				Meta:  assetMeta(item),
			})
		}
		return opts, nil
	})

	nodeSvc.RegisterAsyncSource(AsyncSourceSubgraphIDs, func(_, specKind string, _ map[string]any) ([]node.EnumOption, error) {
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

func formatAssetLabel(name, guid string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return guid
	}
	return fmt.Sprintf("%s (%s)", name, guid)
}

func assetMeta(item asset.AssetSummary) map[string]any {
	meta := map[string]any{}
	if item.Name != "" {
		meta["name"] = item.Name
	}
	if item.Category != "" {
		meta["category"] = item.Category
	}
	if len(item.Tags) > 0 {
		meta["tags"] = item.Tags
	}
	return meta
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
