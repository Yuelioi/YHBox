// internal/node/qr.go
// QRResult — DecodeQR 节点的返回类型。spec §节点3。
package node

// QRResult 一个成功解码的二维码。Points = 解码器定位点 (点数/顺序以库为准, 非四角),
// 全帧归一化 (0..1, 原点=全帧左上角)。
type QRResult struct {
	Text   string  `json:"text"`
	Points []Point `json:"points"`
}
