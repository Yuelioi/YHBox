# Final contract and release acceptance

Status: current

## Outcome

对 Yotta 3.1 的全部实现作一次总审计，集中修复仍属于仓库工程范围的缺口，并给出可复核的 completion verdict。结论必须区分工程完成、可构建 candidate 和公开 stable 发布，不得因外部治理前置而虚报完成或无限延长代码迁移。

## Completion criterion

- Slice registry 中全部 implementation Slice 已 completed，仓库不存在旧 Container runtime、双执行路径或 release-number package/type name。
- stable nodeTypeId、显式 SemVer version、semanticDigest、Catalog/Program/package locks 与 Go/TS/schema/reference/golden 一致且 drift gate 可重建。
- GUI、headless、AI、MCP、schedule/debug 只消费唯一 compiler/Program/Run application path；第三方执行只来自已签名启用 package。
- 对照 plan、design、review 和 AGENTS contract 完成 Standards/Spec review；所有高风险 finding 修复或明确归入公开发布外部前置。
- 最终阶段末一次性运行 task check、race/fuzz 触发组、Windows production build、Process/Wasm smoke、WebView smoke+人工截图，以及 Linux/macOS portable core；原生三平台 GUI 以 CI gui-build matrix 为权威证据。
- candidate staging、artifact manifest、辅助 runner/DLL/ADB inclusion 与 package/sign 入口不发生“测试后重建”。
- 当前 LICENSE 仍是 source-available 时不得宣称 OSI open source；公开 stable 所需许可证、签名证书、canonical repository、维护者权限、owner settings 和真实宿主 smoke 必须列为外部阻塞。
- 给出明确 verdict：major upgrade engineering complete / incomplete；若 incomplete，只保留可执行的真实缺口，不创建泛化 fog。

## Work plan

1. 建立 plan/design/review → code/tests/docs 的 completion matrix，先做静态审计，不重复运行已绿门禁。
2. 完成 Standards 与 Spec review，集中归并 finding。
3. 修复工程范围 finding，并补齐 contract/reference/golden/release staging drift。
4. 阶段末统一运行最终 acceptance matrix。
5. 更新架构、operations、release status 与 Flightdeck；若工程完成则关闭 Topic，公开发布外部前置另行记录。

## Blocked by

无工程设计阻塞。公开 stable 发布有外部前置，但不阻塞开始工程总审计。

## Verification

尚未开始最终总审计。插件阶段的 task check、Windows build/plugin/WebView smoke 与 portable core 证据可作为输入，但最终 verdict 前仍需完成 completion matrix、review 和阶段末最终门禁。

## Out of scope

- 未经用户授权 push、创建公开仓库、改写历史或变更 owner 级设置。
- 在没有证书时伪造签名/timestamp 成功。
- 把当前 source-available LICENSE 描述成 OSI open source。
- 为旧 2.x/3.0 contract、package、ABI 或 workflow 增加兼容 shim。

## Result

进行中。
