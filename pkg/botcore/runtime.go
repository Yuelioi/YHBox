package botcore

import "io/fs"

// RuntimeAssets 是 bot 启动后所有 locale-bound 视觉/资源型 资产的统一句柄。
//
// 每个 bot package 暴露：
//
//	func Assets() RuntimeAssets
//
// caller (main.go / 集成测试 / debug 工具) 通过这一个返回值拿到 bot 当前 locale
// 下需要在 runtime 持续读文件的资产。
//
// 字段：
//   - Templates: 视觉模板文件系统（PNG 等）；bot 没有视觉模板就给 nil
//   - TemplatesTOML: templates 元数据 bytes（喂给 vision.LoadTemplatesConfig）；同上
//
// **重要边界**：config.yaml 解析后的强类型 state 不进 RuntimeAssets。Config 数据
// 在 bot.LoadConfig 里反序列化进包级变量或 bot 自己的 Config struct，caller 直接
// 访问那些 typed state。不要在 RuntimeAssets 里暴露 ConfigFS / RawConfigBytes —— 这种
// 字段会诱导 caller 绕过 LoadConfig 自行解析 yaml，造成"加载路径分裂"、override
// 合并语义被绕过、Validate 失效。
//
// 未来扩展（OCR 字典 / ML 模型 / locale-specific masks / audio assets）按需加字段
// 即可，不动 API 形状。
type RuntimeAssets struct {
	Templates     fs.FS
	TemplatesTOML []byte
}
