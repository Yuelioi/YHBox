# 错误契约系统整改计划

## 1. P0 transport 与注册 seam

- [x] 所有 Wails service 显式安装 `apperr.Marshal`，并以注册测试阻止遗漏。
- [x] 建立 Wails `Bindings.Add/Call` 框架级失败门禁，并用真实 WebView 探针验证 Settings、Hotkey、Workflow。
- [x] `transport.unstructured_failure` 记录脱敏 transport shape 与 operation。
- [ ] 核心验收遇到 `transport.unstructured_failure` 时直接失败。

## 2. Workflow 保存与诊断投影

- [x] 保留 compiler diagnostics，不再用缺少 cause 的 `INVALID_RESULT` 压扁。
- [x] 保存前检查草稿阻断诊断，支持定位 graph/node/field 并保留输入。
- [x] 覆盖旧节点不可升级、revision conflict、无效配置和提交后同步失败。

## 3. 领域 Problem 收敛

- [x] Hotkey 冲突/保留/非法/OS 注册/持久化改为 stable Problem，用户界面不再展示 raw `LastError`。
- [x] Settings、Calibration、Schedule 补全 RPC boundary 的 domain mapping 和恢复动作。
- [x] InputClip、Macro、Snippet、Workflow、Tools、Asset 完成 error inventory 与有限 allowlist。

## 4. 异步与持久失败

- [x] Recording、窗口捕获等事件删除 `payload.error` 旁路，只接受 canonical `problem`。
- [x] Run/Conversation 前置失败、取消和 adapter 错误保持 durable identity 与 stable Problem。
- [x] 清点被静默吞掉的 Promise，仅保留有明确幂等清理语义的忽略点。

## 5. 治理与完成验收

- [x] CI 检查所有注册 service 的 marshaler、业务错误映射和 i18n parity。
- [x] Wails framework call 门禁断言 canonical cause/operation ID；真实隔离 WebView journey 通过。
- [x] 错误 inventory 无未解释缺口，`task check`、WebView smoke 与 `task check:full` 全部通过。
