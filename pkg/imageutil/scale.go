// Package imageutil 提供 RGBA image 缩放等通用工具.
package imageutil

import (
	"image"

	"golang.org/x/image/draw"
)

// ScaleRGBA 把 src 缩放到 newW × newH, 用 CatmullRom (bicubic 类) 保细节.
func ScaleRGBA(src *image.RGBA, newW, newH int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}
