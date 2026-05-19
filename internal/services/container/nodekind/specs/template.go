// internal/services/container/nodekind/specs/template.go
package specs

import "yhbox/internal/services/container/nodekind"

func init() {
	nodekind.Register(&nodekind.Spec{
		Kind:   "WaitTemplate", Group: "detect",
		ExecIn: []string{"in"}, ExecOut: []string{"found", "timeout"},
		DataIn: map[string]nodekind.PinType{
			"timeoutMs": nodekind.PinNumber, "threshold": nodekind.PinNumber,
		},
		DataOut:  map[string]nodekind.PinType{"point": nodekind.PinPoint},
		Defaults: map[string]any{"template": "", "literal": map[string]any{"timeoutMs": 5000.0, "threshold": 0.85}},
		IsYield:  true,
	})
	nodekind.Register(&nodekind.Spec{
		Kind:   "CheckTemplate", Group: "detect",
		ExecIn: []string{"in"}, ExecOut: []string{"yes", "no"},
		DataIn:   map[string]nodekind.PinType{"threshold": nodekind.PinNumber},
		DataOut:  map[string]nodekind.PinType{"point": nodekind.PinPoint},
		Defaults: map[string]any{"template": "", "literal": map[string]any{"threshold": 0.85}},
		IsYield:  true,
	})
	nodekind.Register(&nodekind.Spec{
		Kind:   "ClickTemplate", Group: "detect",
		ExecIn: []string{"in"}, ExecOut: []string{"done", "timeout"},
		DataIn: map[string]nodekind.PinType{
			"timeoutMs": nodekind.PinNumber, "threshold": nodekind.PinNumber,
		},
		DataOut:  map[string]nodekind.PinType{"point": nodekind.PinPoint},
		Defaults: map[string]any{"template": "", "button": "left", "literal": map[string]any{"timeoutMs": 5000.0, "threshold": 0.85}},
		IsYield:  true,
	})
	nodekind.Register(&nodekind.Spec{
		Kind:   "DetectColor", Group: "detect",
		ExecIn: []string{"in"}, ExecOut: []string{"yes", "no"},
		DataIn:  map[string]nodekind.PinType{"minPixels": nodekind.PinNumber},
		DataOut: map[string]nodekind.PinType{"point": nodekind.PinPoint},
		Defaults: map[string]any{
			"region": "0.4,0.55,0.2,0.05", "mode": "hsv",
			"range":   "50,60,67,127,253,255",
			"literal": map[string]any{"minPixels": 5.0},
		},
		IsYield: true,
	})
	nodekind.Register(&nodekind.Spec{
		Kind:   "DetectColorHSV", Group: "detect",
		ExecIn: []string{"in"}, ExecOut: []string{"yes", "no", "timeout"},
		DataIn: map[string]nodekind.PinType{
			"minPixelRatio": nodekind.PinNumber, "pollIntervalMs": nodekind.PinNumber, "timeoutMs": nodekind.PinNumber,
		},
		DataOut: map[string]nodekind.PinType{
			"pixelCount": nodekind.PinNumber, "pixelRatio": nodekind.PinNumber,
		},
		Defaults: map[string]any{
			"roi":     map[string]any{"x": 0.0, "y": 0.0, "w": 100.0, "h": 100.0},
			"hsv":     map[string]any{"hMin": 0.0, "hMax": 180.0, "sMin": 0.0, "sMax": 255.0, "vMin": 0.0, "vMax": 255.0},
			"literal": map[string]any{"minPixelRatio": 0.05, "pollIntervalMs": 100.0, "timeoutMs": 5000.0},
		},
		IsYield: true,
	})
	nodekind.Register(&nodekind.Spec{
		Kind:   "ROIColorScan", Group: "detect",
		ExecIn: []string{"in"}, ExecOut: []string{"found", "notFound", "timeout"},
		DataIn: map[string]nodekind.PinType{
			"minClusterPx": nodekind.PinNumber, "maxClusterPx": nodekind.PinNumber, "minClusterCount": nodekind.PinNumber,
			"pollIntervalMs": nodekind.PinNumber, "timeoutMs": nodekind.PinNumber,
		},
		DataOut: map[string]nodekind.PinType{
			"clusters": nodekind.PinAny, "clusterCount": nodekind.PinNumber,
		},
		Defaults: map[string]any{
			"roi":      map[string]any{"x": 0.0, "y": 0.0, "w": 100.0, "h": 100.0},
			"hsv":      map[string]any{"hMin": 0.0, "hMax": 180.0, "sMin": 0.0, "sMax": 255.0, "vMin": 0.0, "vMax": 255.0},
			"scanAxis": "x",
			"literal": map[string]any{
				"minClusterPx": 2.0, "maxClusterPx": 0.0, "minClusterCount": 1.0,
				"pollIntervalMs": 100.0, "timeoutMs": 5000.0,
			},
		},
		IsYield: true,
	})
	nodekind.Register(&nodekind.Spec{
		Kind:   "Screenshot", Group: "detect",
		ExecIn: []string{"in"}, ExecOut: []string{"done"},
		DataOut:  map[string]nodekind.PinType{"path": nodekind.PinString},
		Defaults: map[string]any{"pathTemplate": "screenshots/{ts}.png"},
	})
	nodekind.Register(&nodekind.Spec{
		Kind:    "ColorBarTrack",
		Group:   "detect",
		ExecIn:  []string{"in"},
		ExecOut: []string{"found", "missing"},
		DataIn:  nil, // roi 走 config (复合 object, 跟 DetectColorHSV/ROIColorScan 同款)
		DataOut: map[string]nodekind.PinType{
			"cursorX":    nodekind.PinNumber,
			"targetX":    nodekind.PinNumber,
			"targetW":    nodekind.PinNumber,
			"confidence": nodekind.PinNumber,
			"yellowPx":   nodekind.PinNumber,
			"greenPx":    nodekind.PinNumber,
		},
		Defaults: map[string]any{
			"roi": map[string]any{"x": 0.3, "y": 0.55, "w": 0.4, "h": 0.05},
		},
		// IsYield: false — 单次抓帧 + 分析, 无阻塞
	})
}
