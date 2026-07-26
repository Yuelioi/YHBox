# ADR 0006: Decouple product and compatibility versions

Status: accepted

Yotta 的产品发布、持久化 artifact、Host Interface、进程协议和本地 store 是不同的兼容性边界。它们由各自 module
拥有并独立提升；禁止再把产品版本复制为全局“合同版本”。

产品发布版本使用根目录 `VERSION` 中的 numeric SemVer。构建脚本把它投影到 Wails、Windows manifest、
VERSIONINFO 和 NSIS，并通过 Go linker `-X` 注入运行时。`frontend/package.json` 是 private workspace package
元数据，不承担产品版本职责。

持久化 artifact 使用 `format + version` 选择 strict reader。当前未发布的开发合同一次性切换到各域 `1`；
Schema 路径使用 `v1`，前端通过生成的 `current` alias 消费。开发期旧 `3.1` 数据不迁移、不 dual-read；
稳定发布后，任何 shape 变化必须先提升所属合同版本，再提供确定性的相邻版本 migration。

Host Interface 使用独立 major/minor 支持范围。Script Worker、Plugin 和 MCP document 等 wire contract 使用各自协议
身份与版本。Blob、Workflow Source、Program 和 Run store 各自拥有 layout marker。版本 inventory 汇总这些 owner
的常量，但不要求数值相等。

这样产品可以频繁发版而不制造虚假的数据迁移或插件不兼容。代价是版本域数量增加，因此仓库提供
`yotta-versions` 命令、生成投影和 `task versions:check` 门禁来验证所有权与漂移。
