---
kind: note
summary: "AI 连接元数据与密钥分离：Windows Credential Manager 保存 Yotta/AI/<connectionID>，设置与状态 RPC 不回传密钥；旧明文仅在全部安全写入成功后清除。"
activation: action
read_when: "修改 AI 连接设置、Provider 缓存、测试连接、Settings schema/RPC、凭据迁移或删除连接时"
recheck_when: "支持新的桌面平台凭据库；改变 connection ID；迁移 settings 格式；新增导入导出、日志或诊断包中的 AI 连接信息时"
---
# AI 凭据存储与迁移边界

## 权威分层

- `AIConnection` 是连接元数据；`apiKey` 字段只为读取旧 settings 做兼容，序列化时 omitempty。
- Windows 的权威密钥存储是 Credential Manager 的 Generic Credential，target 固定为 `Yotta/AI/<connectionID>`。
- 前端只通过 `SecretStatus` 读取 presence；`SetAPIKey` 与 `DeleteAPIKey` 是写/删通道。任何读取设置或状态的 RPC 都不得返回原值。
- 测试连接接受一次性表单密钥；未提供时才按 connection ID 读取已保存密钥。
- Provider cache 指纹包含解析后的密钥，使替换密钥能自动重建 provider，但日志和错误不得包含密钥。

## 旧配置迁移

1. 启动后扫描所有旧 `settings.json` 连接。
2. 先把每一个非空 API Key 写入安全存储。
3. 只有所有写入均成功，才通过 Settings 单写者清除全部旧明文并原子保存。
4. 任一写入失败时保留旧配置；现代前端的 metadata-only 更新也必须保留这份未迁移密钥，避免静默丢失。
5. 迁移失败期间 runtime 可以回退读取旧值；成功迁移后只读安全存储。

部分凭据写入成功但整体失败时允许暂时重复保存，优先保证无数据丢失；后续启动会幂等覆盖并完成清理。

## 生命周期与平台

- 删除 AI 连接后删除对应 credential；失败只记录 connection ID 和结构化错误，不记录密钥。
- Windows 之外当前 secure store 返回 unavailable，符合 GUI 预览级承诺。没有接入 Keychain/Secret Service 前，不得声称跨平台安全凭据已完成。
- 导出、诊断包、日志、toast、settings snapshot 和测试 fixture 都不得携带真实 API Key。
