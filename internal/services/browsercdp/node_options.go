package browsercdp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yotta/internal/node"
)

const AsyncSourceTargets = "browserCDPTargets"

func RegisterNodeAsyncSource(nodeSvc *node.NodeService, svc *Service) {
	nodeSvc.RegisterAsyncSource(AsyncSourceTargets, func(_, _ string, params map[string]any) ([]node.EnumOption, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		endpoint := stringParam(params, "Endpoint")
		targets, err := svc.ListTargets(ctx, endpoint)
		if err != nil {
			return nil, err
		}
		opts := make([]node.EnumOption, 0, len(targets))
		for _, t := range targets {
			opts = append(opts, node.EnumOption{
				Value: t.ID,
				Label: formatTargetLabel(t),
			})
		}
		return opts, nil
	})
}

func stringParam(params map[string]any, key string) string {
	if params == nil {
		return ""
	}
	if v, ok := params[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func formatTargetLabel(t TargetInfo) string {
	title := strings.TrimSpace(t.Title)
	if title == "" {
		title = strings.TrimSpace(t.URL)
	}
	if title == "" {
		title = t.ID
	}
	if t.URL != "" && t.URL != title {
		return fmt.Sprintf("%s (%s)", title, t.URL)
	}
	return title
}
