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
	ctx      *ctxImpl // nil 表示无 ctx 守卫 (单 builder 单测用)
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
		panic(fmt.Sprintf("multiple Fire() on same builder (node %q)", b.spec.Kind))
	}
	if b.ctx != nil {
		// ctx 级守卫: 同一 Run 内任何第二次 Fire 都炸 (Run 内调多次 Fire → 第 2 次 panic)
		b.ctx.markFired(b.exitName)
	}
	b.fired = true
	return &outputsImpl{exitName: b.exitName, data: b.data}
}

// validateExitName 在 Out(name) 调用时检查 name 是否在 Spec.Outputs.
// 错 → 立即 panic (author bug, fail fast — Claude r4 #2).
//
// DynamicOutputs 节点 (Switch named-by-value) 放行任意 name — 出口由 config 推导,
// 静态 Outputs 列不全; 合法性由节点 Run + validator 保证.
func validateExitName(spec *Spec, name string) {
	if spec.DynamicOutputs {
		return
	}
	for _, o := range spec.Outputs {
		if o.Name == name {
			return
		}
	}
	panic(fmt.Sprintf("ctx.Out(%q) — name not in Spec.Outputs (node %q)", name, spec.Kind))
}
