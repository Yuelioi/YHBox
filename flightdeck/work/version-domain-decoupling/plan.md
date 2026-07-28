# 版本域解耦实施计划

## Outcome

产品版本只需修改一个权威源；各合同和协议独立演进；生成物、前端引用和构建元数据由脚本派生；
裸写错误版本会在本地门禁和 CI 中失败。

## Stage 1 — 研究与决策

- [x] 对照 SemVer、Go、npm、Windows/Wails、JSON Schema 和成熟协议的一手资料。
- [x] 固化产品版本、artifact contract、Host Interface、wire protocol、store layout 的升级规则。
- [x] 改写领域语言与兼容性文档，删除“稳定目标全部统一为 3.1”的约定。

## Stage 2 — 产品版本单一事实源

- [x] 建立唯一产品版本源和可测试的版本读取/校验 module。
- [x] 用脚本生成或同步 Wails、Windows、NSIS 与必要前端元数据；升级命令只修改权威源。
- [x] 提供 inventory、check、bump dry-run，并验证构建元数据一致。

## Stage 3 — 持久化合同 v1 cutover

- [x] Workflow Source、Node Contract、Authoring Projection、Catalog、Data、Program、Run、Schedule 和
  Capability artifact 分别拥有独立 `v1`。
- [x] 生成目录与 Schema ID 使用 `/v1/`；前端通过稳定生成入口消费，不知道物理版本目录。
- [x] 更新 strict parser、fixtures、migration registry 和生成合同，不读取开发期 `3.1`。

## Stage 4 — Host、wire 与 storage 解耦

- [x] 建立独立 Host Interface module，集中当前版本和 range compatibility。
- [x] Script Worker、Plugin、AI schema、MCP document 等协议使用各自 `/1`。
- [x] Blob、Workflow、Program、Run store marker 使用各自 layout v1；保留确有意义的历史路径字面量。

## Stage 5 — 防回归与验收

- [x] `task versions:check` 校验产品版本权威源与生成投影；合同版本由所属 module 和合同门禁校验。
- [x] 产品版本提升不得改变合同 schema/digest；合同变化必须由所属版本域显式表达。
- [x] 运行 `task check`，并执行构建、EXE 元数据和桌面启动 smoke，保存最终 handoff。
