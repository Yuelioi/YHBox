// internal/node/template_match.go
// TemplateMatch — FindTemplateAll / VisionService.MatchAll 的单个命中实例。spec §节点2。
package node

// TemplateMatch 一个模板命中实例。坐标全帧归一化 0..1。
// VisionService.MatchAll 返回与 FindTemplateAll 节点输出共用此结构体。
type TemplateMatch struct {
	Point       Point      `json:"point"`       // 命中实例中心
	Conf        float64    `json:"conf"`        // CCOEFF_NORMED 匹配度
	BBox        [4]float64 `json:"bbox"`        // [x, y(左上), w, h]; Point = BBox 中心
	TemplateKey string     `json:"templateKey"` // 命中来自哪个模板 GUID (多模板区分)
}
