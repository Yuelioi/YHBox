// Package version 是 YHBox 版本号的"唯一权威来源"。
//
// 改版本步骤：
//  1. 改下面 Version 常量
//  2. 跑 `task version:sync` 把版本号传播到：
//     - build/config.yml（wails3 build/打包用）
//     - build/windows/info.json（exe 资源 file_version + ProductVersion）
//     - frontend/package.json
//  3. commit 时把以上 4 个文件一起带上
//
// Go 代码里要拿版本号 → import "yhbox/pkg/version" → `version.Version`。
// 前端要拿版本号 → import 不到 Go，要通过 wails service 或在 build 时 inject。
package version

const Version = "2.0.0"
