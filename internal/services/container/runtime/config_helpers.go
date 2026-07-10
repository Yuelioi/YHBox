package runtime

// hsvRange HSV 阈值区间。H ∈ [0,360]，S/V ∈ [0,100]。
type hsvRange struct {
	hMin, hMax int
	sMin, sMax int
	vMin, vMax int
}
