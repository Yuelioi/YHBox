package androidadb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yotta/internal/node"
)

const AsyncSourceDevices = "androidADBDevices"

func RegisterNodeAsyncSource(nodeSvc *node.NodeService, svc *Service) {
	nodeSvc.RegisterAsyncSource(AsyncSourceDevices, func(_, _ string, _ map[string]any) ([]node.EnumOption, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		devices, err := svc.ListDevices(ctx)
		if err != nil {
			return nil, err
		}
		opts := make([]node.EnumOption, 0, len(devices))
		for _, d := range devices {
			if d.State != "device" {
				continue
			}
			opts = append(opts, node.EnumOption{
				Value: d.Serial,
				Label: formatDeviceLabel(d),
				Meta:  deviceMeta(d),
			})
		}
		return opts, nil
	})
}

func formatDeviceLabel(d Device) string {
	name := strings.ReplaceAll(d.Model, "_", " ")
	if name == "" {
		name = d.Product
	}
	if name == "" {
		name = d.Serial
	}
	if d.Resolution.W > 0 && d.Resolution.H > 0 {
		return fmt.Sprintf("%s (%s, %dx%d)", name, d.Serial, d.Resolution.W, d.Resolution.H)
	}
	return fmt.Sprintf("%s (%s)", name, d.Serial)
}

func deviceMeta(d Device) map[string]any {
	meta := map[string]any{}
	name := strings.ReplaceAll(d.Model, "_", " ")
	if name == "" {
		name = d.Product
	}
	if name != "" {
		meta["name"] = name
	}
	if d.Resolution.W > 0 && d.Resolution.H > 0 {
		meta["width"] = d.Resolution.W
		meta["height"] = d.Resolution.H
	}
	return meta
}
