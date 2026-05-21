// internal/services/container/nodekind/specs/verify.go
package specs

import (
	"yhbox/internal/services/container/dependency"
	"yhbox/internal/services/container/nodekind"
)

// VerifyDetectExtractorsRegistered 启动期检查所有 detect group 节点都注册了 DependencyExtractor.
// main.go / test 调用一次, 漏注册 → fail-fast.
func VerifyDetectExtractorsRegistered() error {
	var detectKinds []string
	for _, k := range nodekind.Kinds() {
		spec, _ := nodekind.Get(k)
		if spec != nil && spec.Group == "detect" {
			detectKinds = append(detectKinds, k)
		}
	}
	return dependency.VerifyDetectGroupHasExtractors(detectKinds)
}
