package services

import "github.com/yottaapp/yotta/pkg/version"

// AppInfoService 暴露 app 元数据给前端（版本号 / 作者 / 仓库 URL 等）。
// 前端 about 页通过这个 RPC 读，避免硬编码字符串再跟 backend 飘。
type AppInfoService struct{}

func NewAppInfoService() *AppInfoService { return &AppInfoService{} }

// AppInfo 前端展示用的 app 元数据。
type AppInfo struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Author   string `json:"author"`
	Repo     string `json:"repo"`
	Bilibili string `json:"bilibili"`
}

// Info 返当前 app 元数据。版本号源 = pkg/version.Version。
func (s *AppInfoService) Info() AppInfo {
	return AppInfo{
		Name:     "Yotta",
		Version:  version.Version,
		Author:   "Yueli",
		Repo:     "https://github.com/yottaapp/yotta",
		Bilibili: "https://space.bilibili.com/4279370",
	}
}
