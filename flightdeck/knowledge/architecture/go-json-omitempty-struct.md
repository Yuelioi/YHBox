# ⚠ Go JSON 的 omitempty 不会省略空值 struct
Go `encoding/json` 对值类型 struct 不会按“所有字段为零”将其视为空，因此 `PackageLink` 这类字段写成 `PackageLink json:"...,omitempty"` 时，零值仍序列化为 `{}`。文件虽然是合法 JSON，但 `package.json` 的编辑器 schema 会把空 `repository`、`bugs` 等对象判为类型或结构错误。

可选 object 应使用 `*PackageLink`：`nil` 表示字段不存在，非 nil 表示确实提供链接。回归测试应从默认模型走到最终 JSON bytes，断言可选字段完全不存在；只测 marshal/unmarshal 成功抓不到 schema 诊断。

已落盘的 Container 不能只手改 `package.json`，否则 manifest hash 与 `yotta-lock.json` 立即失配。应让修复后的 `Store.Save` 按 package → graph → installation → lock 的协议整代重写。
