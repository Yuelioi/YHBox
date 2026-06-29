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

// MatchHit 单模板匹配结果 (Match/WaitMatch 返回, 值语义)。
// Found=false 时 Point/BBox 为零值, Conf = 轮询期间见过的最高匹配度 (诊断"差多少")。
type MatchHit struct {
	Found bool       `json:"found"`
	Point Point      `json:"point"` // 命中中心 (= BBox 中心)
	BBox  [4]float64 `json:"bbox"`  // [x, y(左上), w, h] 全帧归一化
	Conf  float64    `json:"conf"`  // CCOEFF_NORMED 匹配度
}
