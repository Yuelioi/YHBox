// tools/rhythm/debug.go
package rhythm

import (
	"encoding/csv"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// Debug 位掩码 flag 决定开哪些调试输出。caller 用 flags & DebugXxx != 0 测试。
//
// bit 0: 每帧 ROI 截图 (4 张 PNG) — 预留，未实现
// bit 1: mask 可视化（二值图）   — 预留，未实现
// bit 2: cnt 时间序列写 logger   — rhythm.Run 内联实现
// bit 3: overlay (ROI 外框 + 中心权重区 + cnt + state) — 预留，未实现（需要 full-frame，Capturer 当前只给 per-ROI）
// bit 4: cnt CSV 导出            — 本文件 CSVWriter 实现
const (
	DebugSaveROI   = 1 << 0
	DebugSaveMask  = 1 << 1
	DebugLogCnts   = 1 << 2
	DebugOverlay   = 1 << 3
	DebugExportCSV = 1 << 4
)

// CSVWriter 为 bit 4 提供：开 → 写 header → 每 tick 写一行 → Close 刷盘。
//
// 文件名 <dir>/rhythm-darkratio-YYYYMMDD_HHMMSS.csv，dir 由 caller 传入。
// 每行 9 列：tick_ms, t{1..4}_dark_ratio (0.0-1.0), t{1..4}_pressed (0/1)。
type CSVWriter struct {
	mu     sync.Mutex
	f      *os.File
	w      *csv.Writer
	t0     time.Time
	closed bool
}

// NewCSVWriter 创建文件并写 header。失败返回 nil + err。
func NewCSVWriter(dir string) (*CSVWriter, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", dir, err)
	}
	name := fmt.Sprintf("rhythm-darkratio-%s.csv", time.Now().Format("20060102_150405"))
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create %s: %w", path, err)
	}
	w := csv.NewWriter(f)
	if err := w.Write([]string{"tick_ms", "t1_ratio", "t2_ratio", "t3_ratio", "t4_ratio", "t1_pressed", "t2_pressed", "t3_pressed", "t4_pressed"}); err != nil {
		f.Close()
		return nil, err
	}
	return &CSVWriter{f: f, w: w, t0: time.Now()}, nil
}

// Write 写一行。pressed=true 即 prev 边沿状态（持续音符也算 pressed）。
func (c *CSVWriter) Write(now time.Time, ratios [4]float32, pressed [4]bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("csv writer closed")
	}
	tickMs := now.Sub(c.t0).Milliseconds()
	row := []string{
		strconv.FormatInt(tickMs, 10),
		strconv.FormatFloat(float64(ratios[0]), 'f', 4, 32),
		strconv.FormatFloat(float64(ratios[1]), 'f', 4, 32),
		strconv.FormatFloat(float64(ratios[2]), 'f', 4, 32),
		strconv.FormatFloat(float64(ratios[3]), 'f', 4, 32),
		boolChar(pressed[0]),
		boolChar(pressed[1]),
		boolChar(pressed[2]),
		boolChar(pressed[3]),
	}
	return c.w.Write(row)
}

// Path 返回 CSV 文件路径（log 用）。
func (c *CSVWriter) Path() string {
	if c.f != nil {
		return c.f.Name()
	}
	return ""
}

// Close 刷缓冲 + 关文件。多次调用安全。
func (c *CSVWriter) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	c.w.Flush()
	return c.f.Close()
}

func boolChar(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// SavePNG 简单 helper：把 *image.RGBA 写到 path（自动 mkdir）。
// 给未来 bit 0/1 实现用。
func SavePNG(path string, img *image.RGBA) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
