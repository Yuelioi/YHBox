// internal/nodes/io/play_clip.go
// PlayClip — 回放录制的 InputClip. 老 runtime: nodes.go::execPlayClip + pkg/inputclip
// ClipPlayer + RuntimeContext.{ClipResolver, InputBus, InputBackend, ClipPolicy,
// MouseCounts360, Window.ClientW/H}. 整套 clip 子系统 Phase 4 框架 Ctx 还没有
// 对应 service — Phase 5 wire 时新增 ClipService interface (Resolve + Play
// 阻塞跑完 + InputBus 独占 + ctx cancel → Cancel).
//
// 本 Phase: 注册 Spec (FE 节点列表可见), Run 返 errPlayClipPhase5. validator
// + dependency.RegisterExtractor 已在老 specs/input.go 配齐, 新框架 Spec 这边
// 仅声明 metadata.
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

// errPlayClipPhase5 sentinel returned by PlayClip.Run. Phase 5 wire 加 ClipService 后
// Run 走真回放; 现在节点放进 graph 跑会立即报错, validator 同时检 clipID 是否存在.
var errPlayClipPhase5 = errors.New("PlayClip — Phase 5 wire 需要 ClipService (Resolve + Play + InputBus 独占)")

func (PlayClip) Spec() node.Spec {
	return node.Spec{
		Kind:        "PlayClip",
		Category:    "IO",
		DisplayName: "回放录像",
		Description: "(Phase 5 stub) 回放录制的 InputClip. 老 runtime 走 ClipPlayer + InputBus 独占; 框架 Ctx 还没 ClipService.",
		Inputs: []node.InputSpec{
			{Name: pcInExec, Type: "Exec"},
			{Name: pcInClipID, Type: "String", Required: true, Semantic: "ClipID",
				DisplayName: "录像 ID",
				Doc:         "clips/ 目录下文件名 (不含扩展名)",
				Widget: node.WidgetSpec{Kind: "async-dropdown",
					Props: node.MarshalProps(node.AsyncDropdownProps{AsyncSource: "clipIDs"})}},
		},
		Outputs: []node.OutputSpec{
			{Name: pcOutDone, Type: "Exec", DisplayName: "完成 (Phase 5)"},
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
