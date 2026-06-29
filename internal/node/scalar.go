package node

// ResolveScalar 把"一值两单位"标量解析成客户区比例。
// 约定: |v|<=1 → 比例 (1=100%, 直接用); |v|>1 → 像素 (v/fullPx)。负值保留符号。
// fullPx<=0 无法换算像素 → 退回原值。
func ResolveScalar(v float64, fullPx int) float64 {
	if v >= -1 && v <= 1 {
		return v
	}
	if fullPx <= 0 {
		return v
	}
	return v / float64(fullPx)
}
