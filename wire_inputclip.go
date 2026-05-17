package main

import (
	"path/filepath"

	"yhbox/internal/services/inputclip"
)

// newInputClipServices 构造 container 级 + library 级 InputClip Service.
//
// 数据根:
//   - container 级 → <dataDir>/_clips/        (容器内部录制产物 / v1 临时共享池)
//   - library   级 → <dataDir>/library/clips/ (分享 / 跨容器复用)
//
// v1: container Service 通过 wails RPC 暴露给前端; library Service 暂保留
// 构造点占位, 后续 library 集成时挂上.
func newInputClipServices(dataDir string) (containerSvc, librarySvc *inputclip.Service) {
	containerRoot := filepath.Join(dataDir, "_clips")
	libraryRoot := filepath.Join(dataDir, "library", "clips")
	containerSvc = inputclip.NewService(inputclip.NewStore(containerRoot))
	librarySvc = inputclip.NewService(inputclip.NewStore(libraryRoot))
	return
}
