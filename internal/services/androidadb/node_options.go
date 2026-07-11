package androidadb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yottaapp/yotta/internal/node"
)

const (
	AsyncSourceDevices = "androidADBDevices"
	AsyncSourceApps    = "androidADBApps"
)

func RegisterNodeAsyncSource(nodeSvc *node.NodeService, svc *Service) {
	node.RegisterAsyncSource(nodeSvc, AsyncSourceDevices, func(_, _ string, _ map[string]any) ([]node.EnumOption, error) {
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

	node.RegisterAsyncSource(nodeSvc, AsyncSourceApps, func(_, _ string, params map[string]any) ([]node.EnumOption, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		apps, err := svc.ListApps(ctx, stringParam(params, "Serial"))
		if err != nil {
			return nil, err
		}
		opts := make([]node.EnumOption, 0, len(apps))
		for _, app := range apps {
			opts = append(opts, node.EnumOption{
				Value: app.Package,
				Label: formatAppLabel(app),
				Meta:  appMeta(app),
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

func formatAppLabel(app App) string {
	label := strings.TrimSpace(app.Label)
	if label == "" {
		label = app.Package
	}
	if label != app.Package {
		label = fmt.Sprintf("%s (%s)", label, app.Package)
	}
	if app.Foreground {
		return "当前前台: " + label
	}
	return label
}

func appMeta(app App) map[string]any {
	meta := map[string]any{"package": app.Package}
	if app.Label != "" {
		meta["label"] = app.Label
	}
	if app.Foreground {
		meta["foreground"] = true
	}
	return meta
}

func stringParam(params map[string]any, key string) string {
	if params == nil {
		return ""
	}
	value, _ := params[key].(string)
	return strings.TrimSpace(value)
}
