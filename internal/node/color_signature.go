// internal/node/color_signature.go
// ColorSignature — FindColorSignature 用: 锚点 + N 偏移点的颜色签名。spec §节点1。
package node

// ColorPoint 签名里一个点。DX/DY 相对锚点的像素偏移 (锚点本身 DX=DY=0)。
// Tol *int 纯 nullable: nil → 用节点默认容差; 非 nil (含 0=严格) → 显式。R/G/B ∈ [0,255]。
type ColorPoint struct {
	DX, DY  int
	R, G, B int
	Tol     *int
}

// ColorSignature 点列表, Points[0] = 锚点。
type ColorSignature struct {
	Points []ColorPoint
}
