package main

import (
	"github.com/yottaapp/yotta/internal/services/asset"
	"github.com/yottaapp/yotta/internal/services/inputclip"
)

// newClipService 构造全局 clip Service — clip 字节落全局 asset 库的 clip kind.
//
// 资产全局化后不再有 container 级 / library 级两套 clip 存储: 单个 Service 共享
// 全局 asset.Store, 录制/回放/分享统一走 GUID + blob 池.
func newClipService(store *asset.Store) *inputclip.Service {
	return inputclip.NewService(store)
}
