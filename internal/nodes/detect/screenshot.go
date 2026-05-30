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
		NeedsWindow: true,
		Inputs: []node.InputSpec{
			{Name: ssInExec, Type: "Exec"},
			{Name: ssInPathTemplate, Type: "String", Default: "screenshots/{ts}.png",
				Widget: node.WidgetSpec{Kind: "text"}},
			{Name: ssInROI, Type: "Geometry", Schema: node.GeometrySchema()},
		},
		Outputs: []node.OutputSpec{
			{Name: ssOutDone, Type: "Exec",
				Data: []node.DataField{
					{Name: ssDataPath, Type: "String"},
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
	// Ctx 不暴露 containerId/nodeId, 这两个 placeholder 传空; 路径靠 timestamp 区分.
	rel := expandScreenshotTemplate(tmpl, "", "", ctx.Now())

	root := screenshotOutputRoot()
	abs, err := filepath.Abs(filepath.Join(root, rel))
	if err != nil {
		return nil, fmt.Errorf("Screenshot resolve path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return nil, fmt.Errorf("Screenshot mkdir %s: %w", filepath.Dir(abs), err)
	}

	// Geometry 零值 = 全帧; adapter 内 ResolveGeometry 统一处理, 无需节点侧区分.
	roi := in.Geometry(ssInROI)
	pngData, err := ctx.Capture().CaptureROI(roi)
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

// checkSafeScreenshotPath 拒绝绝对路径 / 盘符 / ".." 段, 防止写出沙箱根目录.
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
