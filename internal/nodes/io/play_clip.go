// internal/nodes/io/play_clip.go
// PlayClip — 阻塞回放一条录制的 InputClip. clipID 经 ctx.Clip() (runtime 注入的
// ClipResolver + InputBackend + InputBus 独占) 解析并跑完整段, ctx 取消即中断.
package io

import (
	"fmt"

	"yhbox/internal/node"
)

func init() { node.Register(&PlayClip{}) }

type PlayClip struct{}

const (
	pcInExec   = "In"
	pcInClipID = "ClipID"
	pcOutDone  = "Done"
)

func (PlayClip) Spec() node.Spec {
	return node.Spec{
		Kind:     "PlayClip",
		Category: "IO",
		Inputs: []node.InputSpec{
			{Name: pcInExec, Type: "Exec"},
			{Name: pcInClipID, Type: "String", Required: true, Semantic: "ClipID",
				Widget: node.WidgetSpec{Kind: "async-dropdown",
					Props: node.MarshalProps(node.AsyncDropdownProps{AsyncSource: "clipIDs"})}},
		},
		Outputs: []node.OutputSpec{
			{Name: pcOutDone, Type: "Exec"},
		},
	}
}

func (PlayClip) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	clipID := in.String(pcInClipID)
	// ctx.Context() 透传当前 Run 的 cancel — 取消时 Play 内部释放按下键并返 context.Canceled,
	// dispatch 当优雅 halt 不当节点失败.
	if err := ctx.Clip().Play(ctx.Context(), clipID); err != nil {
		return nil, fmt.Errorf("PlayClip clipID=%q: %w", clipID, err)
	}
	return ctx.Out(pcOutDone).Fire(), nil
}

func (PlayClip) Dependencies(in node.Inputs) []node.Dependency {
	id := in.String(pcInClipID)
	if id == "" {
		return nil
	}
	return []node.Dependency{{Kind: "clip", Key: id}}
}
