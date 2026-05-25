// internal/nodes/detect/screenshot.go
// Screenshot — 抓全帧或 ROI, 按 pathTemplate 展开后写到 YHBOX_DATA_DIR (默认 bin/data/screenshots).
// 路径安全检查: 拒绝绝对路径 / 盘符 / ".." 路径段.
package detect

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"yhbox/internal/node"
)

func init() { node.Register(&Screenshot{}) }

type Screenshot struct{}

const (
	ssInExec         = "In"
	ssInPathTemplate = "PathTemplate"
	ssInROI          = "ROI"
	ssOutDone        = "Done"
	ssDataPath       = "Path"
)

func (Screenshot) Spec() node.Spec {
	return node.Spec{
		Kind:        "Screenshot",
		Category:    "Detect",
		DisplayName: "截图",
		Description: "抓帧并写文件. pathTemplate 支持 {ts} / {nodeId} / {containerId} / {date}. ROI 缺省 = 全帧.",
		Inputs: []node.InputSpec{
			{Name: ssInExec, Type: "Exec"},
			{Name: ssInPathTemplate, Type: "String", Default: "screenshots/{ts}.png",
				DisplayName: "路径模板",
				Doc:         "相对路径, 不含 '..' / 盘符 / 开头 '/' '\\'. {ts}/{nodeId}/{containerId}/{date} 自动展开.",
				Widget:      node.WidgetSpec{Kind: "text"}},
			{Name: ssInROI, Type: "JSON", DisplayName: "ROI (像素, 可选)",
				Doc:    `{"x":0,"y":0,"w":100,"h":100} — 缺省/全 0 = 全帧`,
				Widget: node.WidgetSpec{Kind: "json",
					Props: node.MarshalProps(node.JSONProps{Rows: 2})}},
		},
		Outputs: []node.OutputSpec{
			{Name: ssOutDone, Type: "Exec", DisplayName: "完成",
				Data: []node.DataField{
					{Name: ssDataPath, Type: "String", Doc: "写入的绝对路径"},
				}},
		},
	}
}

func (Screenshot) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	tmpl := in.String(ssInPathTemplate)
	if tmpl == "" {
		tmpl = "screenshots/{ts}.png"
	}
	if err := checkSafeScreenshotPath(tmpl); err != nil {
		return nil, err
	}
	// containerId/nodeId 节点拿不到 (Phase 4 framework 没暴露), 老 runtime 走 r.rt.Container.ID 跟 node.ID;
	// 这里用 timestamp + 空 placeholder. Phase 5 wire 时 Ctx 可加 NodeID() / ContainerID() 暴露.
	rel := expandScreenshotTemplate(tmpl, "", "", ctx.Now())

	root := screenshotOutputRoot()
	abs, err := filepath.Abs(filepath.Join(root, rel))
	if err != nil {
		return nil, fmt.Errorf("Screenshot resolve path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return nil, fmt.Errorf("Screenshot mkdir %s: %w", filepath.Dir(abs), err)
	}

	var pngData []byte
	roi, hasROI := pickScreenshotROI(in.JSON(ssInROI))
	if hasROI {
		pngData, err = ctx.Capture().CaptureROI(roi.X, roi.Y, roi.W, roi.H)
	} else {
		pngData, err = ctx.Capture().Capture()
	}
	if err != nil {
		return nil, fmt.Errorf("Screenshot capture: %w", err)
	}
	if pngData == nil {
		// stub / capture 返 nil bytes (e.g. test 环境). 写空文件以保持 Done 出口语义.
		// production capture 真返 PNG, 不会到这.
		pngData = []byte{}
	}
	if err := os.WriteFile(abs, pngData, 0o644); err != nil {
		return nil, fmt.Errorf("Screenshot write %s: %w", abs, err)
	}
	return ctx.Out(ssOutDone).Set(ssDataPath, abs).Fire(), nil
}

// ssROI Screenshot 用的 ROI 像素整数. 不复用 detect_color_hsv 的 parseROIRect (那个用 Rect 浮点).
type ssROI struct{ X, Y, W, H int }

func pickScreenshotROI(m map[string]any) (ssROI, bool) {
	if m == nil {
		return ssROI{}, false
	}
	f := func(k string) int {
		switch v := m[k].(type) {
		case float64:
			return int(v)
		case int:
			return v
		}
		return 0
	}
	r := ssROI{X: f("x"), Y: f("y"), W: f("w"), H: f("h")}
	if r.W < 1 || r.H < 1 {
		return ssROI{}, false
	}
	return r, true
}

func (Screenshot) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return fmt.Sprintf("↧ %s", out.String(ssDataPath))
}

func (Screenshot) Validate(in node.Inputs) []node.ValidationError {
	tmpl := in.String(ssInPathTemplate)
	if tmpl == "" {
		return nil
	}
	if checkSafeScreenshotPath(tmpl) != nil {
		return []node.ValidationError{{
			Code:    "UNSAFE_SCREENSHOT_PATH",
			Message: "pathTemplate must be relative with no '..' / 盘符 / 开头 '/' '\\'",
			Field:   ssInPathTemplate,
		}}
	}
	return nil
}

// checkSafeScreenshotPath 跟老 runtime checkSafePath 一致 — 拒绝绝对路径 / 盘符 / ".." 段.
func checkSafeScreenshotPath(tmpl string) error {
	if strings.HasPrefix(tmpl, "/") || strings.HasPrefix(tmpl, "\\") {
		return fmt.Errorf("Screenshot: unsafe pathTemplate (absolute)")
	}
	if len(tmpl) >= 2 && tmpl[1] == ':' {
		return fmt.Errorf("Screenshot: unsafe pathTemplate (drive letter)")
	}
	for _, part := range strings.FieldsFunc(tmpl, func(c rune) bool { return c == '/' || c == '\\' }) {
		if part == ".." {
			return fmt.Errorf("Screenshot: unsafe pathTemplate ('..')")
		}
	}
	return nil
}

func expandScreenshotTemplate(tmpl, nodeID, containerID string, now time.Time) string {
	r := strings.NewReplacer(
		"{ts}", fmt.Sprintf("%d", now.UnixMilli()),
		"{nodeId}", nodeID,
		"{containerId}", containerID,
		"{date}", now.Format("20060102"),
	)
	return r.Replace(tmpl)
}

func screenshotOutputRoot() string {
	if dir := os.Getenv("YHBOX_DATA_DIR"); dir != "" {
		return dir
	}
	return filepath.Join("bin", "data", "screenshots")
}
