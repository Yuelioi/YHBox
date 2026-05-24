// internal/node/outputs.go
package node

import "fmt"

type outputsImpl struct {
	exitName string
	data     map[string]any
}

func (o *outputsImpl) exit() (string, map[string]any) {
	return o.exitName, o.data
}

type outBuilderImpl struct {
	exitName string
	spec     *Spec
	data     map[string]any
	fired    bool
}

func (b *outBuilderImpl) Set(field string, value any) OutBuilder {
	if b.fired {
		panic(fmt.Sprintf("Set after Fire (node %q, exit %q)", b.spec.Kind, b.exitName))
	}
	b.data[field] = value
	return b
}

func (b *outBuilderImpl) Fire() Outputs {
	if b.fired {
		panic(fmt.Sprintf("multiple Fire() in single Run (node %q)", b.spec.Kind))
	}
	b.fired = true
	return &outputsImpl{exitName: b.exitName, data: b.data}
}

// validateExitName 在 Out(name) 调用时检查 name 是否在 Spec.Outputs.
// 错 → 立即 panic (author bug, fail fast — Claude r4 #2).
func validateExitName(spec *Spec, name string) {
	for _, o := range spec.Outputs {
		if o.Name == name {
			return
		}
	}
	panic(fmt.Sprintf("ctx.Out(%q) — name not in Spec.Outputs (node %q)", name, spec.Kind))
}
