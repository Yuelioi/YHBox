package node_test

import (
	"testing"

	nodepkg "yotta/internal/node"

	// 触发所有节点 init().
	_ "yotta/internal/nodes/control"
	_ "yotta/internal/nodes/detect"
	_ "yotta/internal/nodes/input"
	_ "yotta/internal/nodes/io"
	_ "yotta/internal/nodes/purefunc"
	_ "yotta/internal/nodes/random" // RandomInt/RandomFloat/RandomBool
	_ "yotta/internal/nodes/stopwatch"
	_ "yotta/internal/nodes/system"
	_ "yotta/internal/nodes/variable"
)

// validCaptureTypes — Semantic=="capture" 框允许的 CaptureType 值集合.
var validCaptureTypes = map[string]struct{}{
	"number": {},
	"string": {},
	"bool":   {},
	"point":  {},
	"any":    {},
}

// TestCaptureFieldsDeclareValidCaptureType 遍历所有已注册节点 Spec，
// 每个 Semantic=="capture" 的 InputSpec 必须有非空 CaptureType 且值在 validCaptureTypes 内.
func TestCaptureFieldsDeclareValidCaptureType(t *testing.T) {
	for _, rn := range nodepkg.All() {
		spec := rn.Spec
		for _, in := range spec.Inputs {
			if in.Semantic != "capture" {
				continue
			}
			if in.CaptureType == "" {
				t.Errorf("kind=%s pin=%s: Semantic=capture 但 CaptureType 为空，需填 number/string/bool/point/any",
					spec.Kind, in.Name)
				continue
			}
			if _, ok := validCaptureTypes[in.CaptureType]; !ok {
				t.Errorf("kind=%s pin=%s: CaptureType=%q 不在合法集合 {number,string,bool,point,any} 内",
					spec.Kind, in.Name, in.CaptureType)
			}
		}
	}
}
