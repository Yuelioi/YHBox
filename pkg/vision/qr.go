// pkg/vision/qr.go
// QR 解码封装 — 基于 github.com/makiuchi-d/gozxing (Apache-2.0, 纯 Go).
//
// 返回本地基元类型 QRHit (像素坐标), 不依赖 internal/node.
// adapter 层负责坐标归一化 + ROI 偏移换算.
package vision

import (
	"image"

	"github.com/makiuchi-d/gozxing"
	multiqr "github.com/makiuchi-d/gozxing/multi/qrcode"
)

// QRHit 一次成功解码的二维码。Points = 解码器定位点像素坐标 (相对传入子图左上角).
// gozxing QRCodeMultiReader 实测 DetectMulti 输出 3 个定位块中心点 (QR 三个 finder-pattern
// 角), 但点数/顺序以库为准, 调用方不应假设数量.
type QRHit struct {
	Text   string
	Points [][2]int // [x, y] 像素, 相对子图原点
}

var multiReader = multiqr.NewQRCodeMultiReader()

// DecodeQRFromImage 在 img 中解码所有 QR 码. 解码失败 (NotFoundException/FormatException/
// ChecksumException 等 ReaderException) → 返空 slice + nil error (节点据此走 NotFound 出口).
// 其余 error → 透传.
func DecodeQRFromImage(img image.Image) ([]QRHit, error) {
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		// NewBinaryBitmapFromImage 几乎不会出错 (nil binarizer 才报错); 若出错透传.
		return nil, err
	}

	results, err := multiReader.DecodeMultipleWithoutHint(bmp)
	if err != nil {
		// ReaderException (NotFoundException/FormatException/ChecksumException 等) → 空结果
		if _, ok := err.(gozxing.ReaderException); ok {
			return nil, nil
		}
		return nil, err
	}

	hits := make([]QRHit, 0, len(results))
	for _, r := range results {
		pts := r.GetResultPoints()
		pixPts := make([][2]int, len(pts))
		for i, p := range pts {
			pixPts[i] = [2]int{int(p.GetX()), int(p.GetY())}
		}
		hits = append(hits, QRHit{
			Text:   r.GetText(),
			Points: pixPts,
		})
	}
	return hits, nil
}
