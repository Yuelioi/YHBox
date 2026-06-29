package image

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// checkSafePath 拒绝绝对路径 / 盘符 / ".." 段, 防止写出/读出沙箱根目录。
func checkSafePath(tmpl string) error {
	if strings.HasPrefix(tmpl, "/") || strings.HasPrefix(tmpl, "\\") {
		return fmt.Errorf("路径不能以 / 或 \\ 开头(绝对路径)")
	}
	if len(tmpl) >= 2 && tmpl[1] == ':' {
		return fmt.Errorf("路径不能含盘符")
	}
	for _, part := range strings.FieldsFunc(tmpl, func(c rune) bool { return c == '/' || c == '\\' }) {
		if part == ".." {
			return fmt.Errorf("路径不能含 '..'")
		}
	}
	return nil
}

// expandImageTemplate 展开 {ts}(毫秒)/ {date} / {uuid}(防并发同名覆盖)。
func expandImageTemplate(tmpl string, now time.Time) string {
	return strings.NewReplacer(
		"{ts}", fmt.Sprintf("%d", now.UnixMilli()),
		"{date}", now.Format("20060102"),
		"{uuid}", randHex(8),
	).Replace(tmpl)
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// withFormatExt 把文件名扩展名换成与 Image.Format 一致(Format 权威, 覆盖模板里写错的后缀)。
func withFormatExt(rel, format string) string {
	ext := ".png"
	if format == "jpeg" {
		ext = ".jpg"
	}
	if cur := filepath.Ext(rel); cur != "" {
		return rel[:len(rel)-len(cur)] + ext
	}
	return rel + ext
}

// dataRoot 数据根 = YOTTA_DATA_DIR(生产 main.go 设绝对路径), 兜底 bin/data。
func dataRoot() string {
	base := os.Getenv("YOTTA_DATA_DIR")
	if base == "" {
		base = filepath.Join("bin", "data")
	}
	return base
}

// imagesOutputRoot SaveImage 写盘根 = <dataDir>/images(专责图像导出, 区别于 screenshots)。
func imagesOutputRoot() string { return filepath.Join(dataRoot(), "images") }
