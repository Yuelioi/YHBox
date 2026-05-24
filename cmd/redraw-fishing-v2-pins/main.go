// cmd/redraw-fishing-v2-pins — Phase 5.6 atomic #4.
//
// 批量改 fishing-v2 14 子图 + container.json 的 pin name 跟新 framework Spec 对齐:
//   - CheckTemplate/ClickTemplate/WaitTemplate/ColorBarTrack/Sleep/KeyPress/KeyHold*: lowercase → PascalCase
//     (template→Template / threshold→Threshold / yes→Found / no→NotFound / found→Found 等)
//   - If: condition→Condition / then→True / else→False
//   - Subgraph/CollapsedNode/Try: Config["subgraphId"] → Config["SubgraphID"]
//   - Switch: Config["cases"]: [str...] → Config["Case1Value":..., "Case2Value":...] +
//     edge pin "swNode.<caseName>" → "swNode.Case<N>" + "swNode.default" → "swNode.Default"
//
// 不变 (新 spec 跟老一致): Loop / GetVar / SetVar / GetParam / GetSys / Expr / Throw / Break /
// Continue / Stop / Start / SubgraphInput / SubgraphOutput.
//
// Usage:
//
//	go run ./cmd/redraw-fishing-v2-pins             # 默认改 bin/data/containers/fishing-v2/
//	go run ./cmd/redraw-fishing-v2-pins --dry-run   # 只打印 diff 摘要不写盘
//	go run ./cmd/redraw-fishing-v2-pins --dir other # 改指定目录
//
// 跑完用 cmd/validate-fishing-v2 验证 0 errors.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// pinRenames per-kind: oldPin → newPin (input + output + Config 键 同 namespace).
// 来源: internal/nodes/*/ const tables (PascalCase 是 new spec ground truth).
var pinRenames = map[string]map[string]string{
	"CheckTemplate": {
		"template": "Template", "threshold": "Threshold",
		"yes": "Found", "no": "NotFound",
		"point": "Point", "conf": "Conf",
	},
	"ClickTemplate": {
		"template": "Template", "timeoutMs": "TimeoutMs",
		"threshold": "Threshold", "button": "Button",
		"done": "Done", "timeout": "Timeout",
		"point": "Point", "conf": "Conf",
	},
	"WaitTemplate": {
		"template": "Template", "timeoutMs": "TimeoutMs",
		"threshold": "Threshold",
		"found":     "Found", "timeout": "Timeout",
		"point": "Point", "conf": "Conf",
	},
	"DetectColor": {
		"region": "Region", "mode": "Mode", "range": "Range", "minPixels": "MinPixels",
		"yes": "Yes", "no": "No",
		"count": "Count", "center": "Center",
	},
	"DetectColorHSV": {
		"roi": "ROI", "hsv": "HSV",
		"minPixelRatio":  "MinPixelRatio",
		"pollIntervalMs": "PollIntervalMs",
		"timeoutMs":      "TimeoutMs",
		"yes":            "Yes", "no": "No", "timeout": "Timeout",
		"pixelCount": "PixelCount", "pixelRatio": "PixelRatio",
	},
	"ROIColorScan": {
		"roi": "ROI", "hsv": "HSV", "axis": "Axis",
		"minClusterPx":    "MinClusterPx",
		"maxClusterPx":    "MaxClusterPx",
		"minClusterCount": "MinClusterCount",
		"pollIntervalMs":  "PollIntervalMs",
		"timeoutMs":       "TimeoutMs",
		"found":           "Found", "notFound": "NotFound", "timeout": "Timeout",
	},
	"Screenshot": {
		"pathTemplate": "PathTemplate",
		"roi":          "ROI",
		"done":         "Done",
		"path":         "Path",
	},
	"ColorBarTrack": {
		"rois":  "Rois",
		"found": "Found", "missing": "Missing",
		"cursorX": "CursorX", "targetX": "TargetX", "targetW": "TargetW",
		"confidence": "Confidence",
		"yellowPx":   "YellowPx", "greenPx": "GreenPx",
	},
	"Sleep": {
		"duration":   "Duration",
		"durationMs": "Duration", // legacy fishing-v2 JSON stored as ms number; Duration() coerces
		"done":       "Done",
		"out":        "Done", // 老 runtime Sleep 用 .out 出口, 新 spec 用 .Done
	},
	"KeyPress": {
		"vk": "VK", "durationMs": "DurationMs",
		"done": "Done",
		"out":  "Done", // 老 runtime KeyPress 用 .out, 新 spec 用 .Done
	},
	"KeyHoldStart": {
		"vk":  "VK",
		"out": "Out",
	},
	"KeyHoldStop": {
		"vk":  "VK",
		"out": "Out",
	},
	"If": {
		"condition": "Condition",
		"then":      "True",
		"else":      "False",
	},
	"Subgraph": {
		"subgraphId": "SubgraphID",
		"done":       "Done",
	},
	"CollapsedNode": {
		"subgraphId": "SubgraphID",
		"done":       "Done",
	},
	"Try": {
		"subgraphId": "SubgraphID",
		// Try out/catch lowercase — 不动
	},
	// Switch — Value input PascalCase (新 nodepkg Spec swInValue="Value"); cases[]/caseN→Case1Value etc. 在主逻辑处理.
	"Switch": {
		"value": "Value",
	},
}

type fileChange struct {
	Path        string
	NodeRenames int
	EdgeRenames int
	SwitchCases int
}

func main() {
	dryRun := flag.Bool("dry-run", false, "print diff summary without writing")
	dir := flag.String("dir", "bin/data/containers/fishing-v2", "container root dir")
	flag.Parse()

	files := []string{filepath.Join(*dir, "container.json")}
	sgDir := filepath.Join(*dir, "subgraphs")
	entries, err := os.ReadDir(sgDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read subgraphs dir: %v\n", err)
		os.Exit(1)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		files = append(files, filepath.Join(sgDir, e.Name()))
	}
	sort.Strings(files)

	var totals fileChange
	for _, f := range files {
		ch, err := processFile(f, *dryRun)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", f, err)
			os.Exit(1)
		}
		totals.NodeRenames += ch.NodeRenames
		totals.EdgeRenames += ch.EdgeRenames
		totals.SwitchCases += ch.SwitchCases
		if ch.NodeRenames+ch.EdgeRenames+ch.SwitchCases > 0 {
			fmt.Printf("%s: %d config keys / %d edges / %d switch cases renamed\n",
				f, ch.NodeRenames, ch.EdgeRenames, ch.SwitchCases)
		}
	}
	mode := "wrote"
	if *dryRun {
		mode = "would write"
	}
	fmt.Printf("\nTotal: %s %d files, %d config keys, %d edges, %d switch cases renamed.\n",
		mode, len(files), totals.NodeRenames, totals.EdgeRenames, totals.SwitchCases)
}

func processFile(path string, dryRun bool) (fileChange, error) {
	var ch fileChange
	ch.Path = path

	data, err := os.ReadFile(path)
	if err != nil {
		return ch, err
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return ch, fmt.Errorf("unmarshal: %w", err)
	}

	graph, ok := doc["graph"].(map[string]any)
	if !ok {
		return ch, fmt.Errorf("no graph in doc")
	}

	// Pass 1: rewrite node configs + collect (nodeID → kind) and Switch case maps.
	nodesByID := map[string]string{}
	switchCaseMaps := map[string]map[string]string{}
	if nodesRaw, ok := graph["nodes"].([]any); ok {
		for _, n := range nodesRaw {
			node, ok := n.(map[string]any)
			if !ok {
				continue
			}
			nodeID, _ := node["id"].(string)
			kind, _ := node["kind"].(string)
			nodesByID[nodeID] = kind

			cfg, _ := node["config"].(map[string]any)
			if cfg == nil {
				continue
			}

			// Switch 特殊: cases[] (legacy 字符串数组) → Case1Value/Case2Value/...
			if kind == "Switch" {
				cm := map[string]string{"default": "Default"}
				if casesAny, hasCases := cfg["cases"]; hasCases {
					cases, _ := casesAny.([]any)
					for i, c := range cases {
						cv, _ := c.(string)
						cfg[fmt.Sprintf("Case%dValue", i+1)] = cv
						cm[cv] = fmt.Sprintf("Case%d", i+1)
						ch.SwitchCases++
					}
					delete(cfg, "cases")
				}
				switchCaseMaps[nodeID] = cm
			}

			// 通用 rename per kind
			if rename, ok := pinRenames[kind]; ok {
				renameMapKeys(cfg, rename, &ch.NodeRenames)
				// literal sub-map (config["literal"] = inline pin defaults) 也 rename
				if lit, ok := cfg["literal"].(map[string]any); ok {
					renameMapKeys(lit, rename, &ch.NodeRenames)
				}
			}
		}
	}

	// Pass 2: rewrite edges (from / to pin parts).
	if edgesRaw, ok := graph["edges"].([]any); ok {
		for _, e := range edgesRaw {
			edge, ok := e.(map[string]any)
			if !ok {
				continue
			}
			if from, ok := edge["from"].(string); ok {
				newFrom := renamePinRef(from, nodesByID, switchCaseMaps)
				if newFrom != from {
					edge["from"] = newFrom
					ch.EdgeRenames++
				}
			}
			if to, ok := edge["to"].(string); ok {
				newTo := renamePinRef(to, nodesByID, switchCaseMaps)
				if newTo != to {
					edge["to"] = newTo
					ch.EdgeRenames++
				}
			}
		}
	}

	if dryRun {
		return ch, nil
	}

	if ch.NodeRenames+ch.EdgeRenames+ch.SwitchCases == 0 {
		return ch, nil // 文件无变化 — 不重写避免改 mtime / 顺序差异.
	}

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return ch, fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		return ch, fmt.Errorf("write: %w", err)
	}
	return ch, nil
}

func renameMapKeys(m map[string]any, rename map[string]string, counter *int) {
	for old, new := range rename {
		if v, has := m[old]; has {
			if _, conflict := m[new]; conflict && old != new {
				// 已有新 key — skip 避免覆盖 (用户已部分迁移过)
				continue
			}
			m[new] = v
			if old != new {
				delete(m, old)
				*counter++
			}
		}
	}
}

func renamePinRef(ref string, nodesByID map[string]string, switchCaseMaps map[string]map[string]string) string {
	idx := strings.Index(ref, ".")
	if idx < 0 {
		return ref
	}
	nodeID := ref[:idx]
	pin := ref[idx+1:]

	kind, ok := nodesByID[nodeID]
	if !ok {
		return ref
	}

	// Switch case rename 优先 (case 名跟 cases[] 内容一致, 不在通用 rename 表)
	if cm, ok := switchCaseMaps[nodeID]; ok {
		if newPin, has := cm[pin]; has {
			return nodeID + "." + newPin
		}
	}

	// 通用 rename
	if rename, ok := pinRenames[kind]; ok {
		if newPin, has := rename[pin]; has {
			return nodeID + "." + newPin
		}
	}

	return ref
}
