package tools

// MousePosInfo describes the pointer in screen and target-client coordinates.
type MousePosInfo struct {
	ScreenX int     `json:"screenX"`
	ScreenY int     `json:"screenY"`
	HasGame bool    `json:"hasGame"`
	ClientX int     `json:"clientX"`
	ClientY int     `json:"clientY"`
	XRatio  float64 `json:"xRatio"`
	YRatio  float64 `json:"yRatio"`
	ClientW int     `json:"clientW"`
	ClientH int     `json:"clientH"`
}
