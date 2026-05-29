// internal/nodes/io/play_clip.go
// PlayClip — 回放录制的 InputClip. 目前是 stub: 注册 Spec (FE 节点列表可见),
// Run 直接报错, 因为还没有 ClipService (Resolve + Play 阻塞跑完 + InputBus 独占).
package io

import (
	"errors"

	"yhbox/internal/node"
)

func init() { node.Register(&PlayClip{}) }

type PlayClip struct{}

const (
	pcInExec   = "In"
	pcInClipID = "ClipID"
	pcOutDone  = "Done"
)

// errPlayClipPhase5 — PlayClip 没有 ClipService 时 Run 立即返回此 sentinel;
// 放进 graph 跑会报错, validator 同时检 clipID 是否存在.
var errPlayClipPhase5 = errors.New("PlayClip — Phase 5 wire 需要 ClipService (Resolve + Play + InputBus 独占)")

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
	return nil, errPlayClipPhase5
}

func (PlayClip) Dependencies(in node.Inputs) []node.Dependency {
	id := in.String(pcInClipID)
	if id == "" {
		return nil
	}
	return []node.Dependency{{Kind: "clip", Key: id}}
}
