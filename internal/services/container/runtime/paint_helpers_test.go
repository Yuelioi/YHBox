package runtime

import (
	"image"
	"image/color"
)

// paintCursorBar / paintTargetBar: 合成 ColorBarTrack 测试帧 (黄 cursor + 青 target).
// inspect_phase / state_fishing / state_waiting / try_hook_f 测试用.

func paintCursorBar(img *image.RGBA, x int) {
	cursor := color.RGBA{R: 255, G: 240, B: 60, A: 255} // H=55 S=195 V=255; H[45,70] S>=40 V>=200
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for dx := -1; dx <= 1; dx++ {
			img.Set(x+dx, y, cursor)
		}
	}
}

func paintTargetBar(img *image.RGBA, x0, x1 int) {
	target := color.RGBA{R: 0, G: 230, B: 200, A: 255} // H=172 S=255 V=230; H[160,180] S>=140 V>=100
	bounds := img.Bounds()
	midY := (bounds.Min.Y + bounds.Max.Y) / 2
	for y := midY - 5; y <= midY+5; y++ {
		for x := x0; x <= x1; x++ {
			img.Set(x, y, target)
		}
	}
}
