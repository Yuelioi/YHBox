package runtime

import (
	"context"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lxn/win"

	"yhbox/internal/services/container"
)

// execScreenshot 截图节点。
//
// config:
//   - pathTemplate string  (默认 "screenshots/{ts}.png")
//   - roi          map     (可选 {x,y,w,h}；缺省 → 全窗口帧)
//
// 路径安全规则: 拒绝 ".." 片段、绝对路径（以 `/`、`\` 或盘符 `C:` 开头）。
//
// 模板占位符: {ts} UnixMilli, {nodeId}, {containerId}, {date} YYYYMMDD。
//
// 输出根目录: 优先 YHBOX_DATA_DIR 环境变量，否则 bin/data/screenshots/。
//
// $sys 输出: lastScreenshot.path (写入的绝对路径)。
//
// 出口: done。
func (r *ContainerRunner) execScreenshot(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	tmpl := configString(n, "pathTemplate")
	if tmpl == "" {
		tmpl = "screenshots/{ts}.png"
	}

	// --- 路径安全检查 ---
	if err := checkSafePath(n.ID, tmpl); err != nil {
		return nil, err
	}

	// --- 模板展开 ---
	now := time.Now()
	rel := expandTemplate(tmpl, n.ID, r.rt.Container.ID, now)

	// --- 解析输出根目录 ---
	root := screenshotRoot()
	abs, err := filepath.Abs(filepath.Join(root, rel))
	if err != nil {
		return nil, fmt.Errorf("Screenshot %s: resolve path: %w", n.ID, err)
	}

	// --- 确保目录存在 ---
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return nil, fmt.Errorf("Screenshot %s: mkdir %s: %w", n.ID, filepath.Dir(abs), err)
	}

	// --- 截图：ROI（可选）或全帧 ---
	hwnd := win.HWND(r.rt.Window.HWND)
	roiCfg, roiErr := configROI(n, "roi")
	var captureErr error
	var pngErr error

	fh, err := os.Create(abs)
	if err != nil {
		return nil, fmt.Errorf("Screenshot %s: create %s: %w", n.ID, abs, err)
	}
	defer fh.Close()

	if roiErr == nil {
		frame, cErr := r.rt.Capture.FrameROI(hwnd, roiCfg.X, roiCfg.Y, roiCfg.W, roiCfg.H)
		captureErr = cErr
		if captureErr == nil && frame != nil {
			pngErr = png.Encode(fh, frame)
		}
	} else {
		frame, cErr := r.rt.Capture.Frame(hwnd)
		captureErr = cErr
		if captureErr == nil && frame != nil {
			pngErr = png.Encode(fh, frame)
		}
	}

	if captureErr != nil {
		return nil, fmt.Errorf("Screenshot %s: capture: %w", n.ID, captureErr)
	}
	if pngErr != nil {
		return nil, fmt.Errorf("Screenshot %s: encode PNG: %w", n.ID, pngErr)
	}

	// --- 存储路径到 SysState ---
	r.rt.UpdateSys(func(s *SysState) {
		s.LastScreenshot.Path = abs
	})

	return r.edges.next(n.ID+".done", tok.LoopStack), nil
}

// checkSafePath 验证 pathTemplate 不含危险片段。
// 拒绝：绝对路径（/、\、盘符 C:）、".." 路径段。
func checkSafePath(nodeID, tmpl string) error {
	// 拒绝绝对路径（以 / 或 \ 开头）
	if strings.HasPrefix(tmpl, "/") || strings.HasPrefix(tmpl, "\\") {
		return fmt.Errorf("Screenshot %s: unsafe pathTemplate", nodeID)
	}
	// 拒绝 Windows 盘符 (e.g. C:)
	if len(tmpl) >= 2 && tmpl[1] == ':' {
		return fmt.Errorf("Screenshot %s: unsafe pathTemplate", nodeID)
	}
	// 拒绝 .. 路径段
	for _, part := range strings.FieldsFunc(tmpl, func(c rune) bool { return c == '/' || c == '\\' }) {
		if part == ".." {
			return fmt.Errorf("Screenshot %s: unsafe pathTemplate", nodeID)
		}
	}
	return nil
}

// expandTemplate 展开 pathTemplate 中的占位符。
func expandTemplate(tmpl, nodeID, containerID string, now time.Time) string {
	r := strings.NewReplacer(
		"{ts}", fmt.Sprintf("%d", now.UnixMilli()),
		"{nodeId}", nodeID,
		"{containerId}", containerID,
		"{date}", now.Format("20060102"),
	)
	return r.Replace(tmpl)
}

// screenshotRoot 返回截图根目录：优先 YHBOX_DATA_DIR 环境变量，否则 bin/data/screenshots。
func screenshotRoot() string {
	if dir := os.Getenv("YHBOX_DATA_DIR"); dir != "" {
		return dir
	}
	return filepath.Join("bin", "data", "screenshots")
}
