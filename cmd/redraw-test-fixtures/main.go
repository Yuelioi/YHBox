// cmd/redraw-test-fixtures — atomic #4 follow-up.
//
// 把 internal/services/container/runtime/*_test.go 内的老 lowercase pin name
// (subgraphId / durationMs / template / threshold / vk / condition / then / else / done / yes / no /
//  found / notFound / timeout / cases / ...) 一锅改成新 framework Spec 用的 PascalCase.
//
// 跟 cmd/redraw-fishing-v2-pins (改 JSON) 配套, 一气推完 Phase 5.6 redraw.
//
// 关键: per-kind 感知 — 解析 test file 里 `ID: "xxx", Kind: "Yyy"` 字面值建 nodeID→kind 映射,
// 再 per-edge 根据 source kind 决定要不要 rename. 否则 `loop.done`→`loop.Done` 会破 Loop
// (Loop 用 lowercase done) 但 `sleep.done`→`sleep.Done` 是对的 (Sleep 用 Done).
//
// Usage:
//
//	go run ./cmd/redraw-test-fixtures
//	go run ./cmd/redraw-test-fixtures --dry-run
//	go run ./cmd/redraw-test-fixtures --dir other
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// pinRenames per-kind: oldPin → newPin (对 edge ref `<nodeID>.<oldPin>"` 跟 Config keys 都过).
// 跟 cmd/redraw-fishing-v2-pins 的 pinRenames 表对齐 (来源 internal/nodes/*/ Spec 常量).
//
// 不在此表的 kind: pin 名跟老一致 (e.g. Loop body/done 全 lowercase, SetVar varName/scope/value
// 全 lowercase, Start out 全 lowercase). 不动.
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
		"durationMs": "Duration", // legacy ms number, Duration() coerces
		"done":       "Done",
		"out":        "Done", // 老 runtime Sleep 用 .out 出口
	},
	"KeyPress": {
		"vk": "VK", "durationMs": "DurationMs",
		"done": "Done",
		"out":  "Done", // 老 runtime KeyPress 用 .out
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
		// Try out/catch 仍 lowercase — 不动
	},
	"Switch": {
		"value": "Value",
	},
	"Loop": {
		"complete": "done",
	},
}

// nodeDeclRE 抓 `ID: "xxx", Kind: "Yyy"` 形式. Test fixture 普遍这个写法.
// Kind 可能在 ID 之后或之前同一 struct literal 里 — 用 line-window 内查找.
//
// 例: `{ID: "loop", Kind: "Loop", Config: ...}` → nodeID=loop, kind=Loop.
// 多行写法 (ID 一行 Kind 下行) 不支持 — fishing-v2 测试都单行.
var nodeDeclRE = regexp.MustCompile(`\bID:\s*"([^"]+)"[^}]*?\bKind:\s*"([^"]+)"`)

// edgeRefRE 抓边 `From: "<nodeID>.<pin>"` / `To: "<nodeID>.<pin>"` / `from: "..."` / `out: ["<nodeID>.<pin>"...]` form.
// 单 group 1 = nodeID, group 2 = pin.
var edgeRefRE = regexp.MustCompile(`"([A-Za-z_][A-Za-z0-9_]*)\.([A-Za-z_][A-Za-z0-9_]*)"`)

// switchCasesRE: `"cases": []any{"A", "B"}` → 拆 Case1Value/Case2Value Config keys.
var switchCasesRE = regexp.MustCompile(`"cases":\s*\[\]any\{([^}]*)\}`)

// configKeyRE 抓 `"<lowercase>":` (Config key 形式). 跟 Go 字符串 literal 重合.
var configKeyRE = regexp.MustCompile(`"([a-z][A-Za-z0-9]*)":`)

func main() {
	dryRun := flag.Bool("dry-run", false, "print summary without writing")
	dir := flag.String("dir", "internal/services/container/runtime", "test dir")
	flag.Parse()

	entries, err := os.ReadDir(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read dir: %v\n", err)
		os.Exit(1)
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		files = append(files, filepath.Join(*dir, e.Name()))
	}
	sort.Strings(files)

	var totalFiles, totalEdges, totalConfigs, totalSwitches int
	for _, f := range files {
		ed, cf, sw, err := processFile(f, *dryRun)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", f, err)
			os.Exit(1)
		}
		if ed+cf+sw > 0 {
			fmt.Printf("%s: %d edges, %d configs, %d switch-cases\n", f, ed, cf, sw)
			totalFiles++
		}
		totalEdges += ed
		totalConfigs += cf
		totalSwitches += sw
	}
	mode := "wrote"
	if *dryRun {
		mode = "would write"
	}
	fmt.Printf("\nTotal: %s %d files, %d edges, %d configs, %d switch-cases\n",
		mode, totalFiles, totalEdges, totalConfigs, totalSwitches)
}

func processFile(path string, dryRun bool) (edges, configs, switches int, err error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, 0, err
	}
	out, ed, cf, sw := rewrite(string(src))
	if string(src) == out {
		return 0, 0, 0, nil
	}
	if dryRun {
		return ed, cf, sw, nil
	}
	if err := os.WriteFile(path, []byte(out), 0644); err != nil {
		return 0, 0, 0, fmt.Errorf("write: %w", err)
	}
	return ed, cf, sw, nil
}

func rewrite(src string) (out string, edges, configs, switches int) {
	// Pass 1: build nodeID → kind map by scanning whole file. Handle multi-occurrence (same ID
	// reused across test funcs — re-declares overwrite earlier; fine for our purpose).
	nodesByID := map[string]string{}
	for _, m := range nodeDeclRE.FindAllStringSubmatch(src, -1) {
		nodesByID[m[1]] = m[2]
	}

	// Pass 2: Switch "cases" → Case1Value/2Value keys + collect switchCaseMaps for edge rename.
	switchCaseMaps := map[string]map[string]string{}
	// Find Switch node IDs + nearby "cases" array (within same line group).
	// Heuristic: line-by-line — if we see `Kind: "Switch"` capture node ID, then next `"cases":`
	// belongs to it. Test fixtures all in same struct literal lines.
	lines := strings.Split(src, "\n")
	currentSwitchID := ""
	for i, ln := range lines {
		// Detect Switch node declaration on this line.
		if strings.Contains(ln, `Kind: "Switch"`) {
			if mm := nodeDeclRE.FindStringSubmatch(ln); mm != nil && mm[2] == "Switch" {
				currentSwitchID = mm[1]
			}
		}
		// Handle "cases": [...] on this line.
		if m := switchCasesRE.FindStringSubmatchIndex(ln); m != nil {
			inside := ln[m[2]:m[3]]
			vals := splitCases(inside)
			cm := map[string]string{"default": "Default"}
			var parts []string
			for j, v := range vals {
				parts = append(parts, fmt.Sprintf(`"Case%dValue": %s`, j+1, v))
				// unquote literal to map case value → CaseN exit pin name
				if strings.HasPrefix(v, `"`) && strings.HasSuffix(v, `"`) {
					cm[v[1:len(v)-1]] = fmt.Sprintf("Case%d", j+1)
				}
			}
			lines[i] = ln[:m[0]] + strings.Join(parts, ", ") + ln[m[1]:]
			switches++
			if currentSwitchID != "" {
				switchCaseMaps[currentSwitchID] = cm
			}
			currentSwitchID = ""
		}
	}
	src = strings.Join(lines, "\n")

	// Pass 3: Edge pin refs `nodeID.pin"` — rename per source node kind.
	src = edgeRefRE.ReplaceAllStringFunc(src, func(m string) string {
		sub := edgeRefRE.FindStringSubmatch(m)
		nodeID, pin := sub[1], sub[2]
		// Switch case rename first.
		if cm, ok := switchCaseMaps[nodeID]; ok {
			if newPin, has := cm[pin]; has {
				edges++
				return fmt.Sprintf(`"%s.%s"`, nodeID, newPin)
			}
		}
		kind, ok := nodesByID[nodeID]
		if !ok {
			return m
		}
		if renames, ok := pinRenames[kind]; ok {
			if newPin, has := renames[pin]; has && newPin != pin {
				edges++
				return fmt.Sprintf(`"%s.%s"`, nodeID, newPin)
			}
		}
		return m
	})

	// Pass 4: Config key renames (global, unambiguous lowercase → PascalCase).
	// 这些 key 名在新 spec 中 PascalCase 是唯一选择 — 不同 kind 不冲突.
	// "durationMs" 例外: Sleep→Duration, KeyPress→DurationMs. 当前 tests 没 KeyPress
	// "durationMs" literal (grep 验证), 全局 Sleep 改 Duration 安全.
	globalConfigRenames := map[string]string{
		`"subgraphId":`:     `"SubgraphID":`,
		`"vk":`:             `"VK":`,
		`"condition":`:      `"Condition":`,
		`"threshold":`:      `"Threshold":`,
		`"template":`:       `"Template":`,
		`"timeoutMs":`:      `"TimeoutMs":`,
		`"pollIntervalMs":`: `"PollIntervalMs":`,
		`"minPixelRatio":`:  `"MinPixelRatio":`,
		`"minClusterPx":`:   `"MinClusterPx":`,
		`"maxClusterPx":`:   `"MaxClusterPx":`,
		`"minClusterCount":`: `"MinClusterCount":`,
		`"minPixels":`:      `"MinPixels":`,
		`"durationMs":`:     `"Duration":`, // Sleep only; tests have no KeyPress literal
		`"region":`:         `"Region":`,
		`"range":`:          `"Range":`,
		`"roi":`:            `"ROI":`,
		`"hsv":`:            `"HSV":`,
		`"rois":`:           `"Rois":`,
	}
	for old, new := range globalConfigRenames {
		if strings.Contains(src, old) {
			cnt := strings.Count(src, old)
			src = strings.ReplaceAll(src, old, new)
			configs += cnt
		}
	}

	// "subgraphId" type-assertion form too: `["subgraphId"]` / `"subgraphId",`.
	src = strings.ReplaceAll(src, `["subgraphId"]`, `["SubgraphID"]`)
	src = strings.ReplaceAll(src, `"subgraphId",`, `"SubgraphID",`)

	return src, edges, configs, switches
}

func splitCases(s string) []string {
	var out []string
	var cur strings.Builder
	inStr := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '"' {
			inStr = !inStr
			cur.WriteByte(c)
			continue
		}
		if c == ',' && !inStr {
			v := strings.TrimSpace(cur.String())
			if v != "" {
				out = append(out, v)
			}
			cur.Reset()
			continue
		}
		cur.WriteByte(c)
	}
	if v := strings.TrimSpace(cur.String()); v != "" {
		out = append(out, v)
	}
	return out
}
