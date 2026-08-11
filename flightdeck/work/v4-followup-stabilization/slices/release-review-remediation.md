# 发布审查修复

## Goal

修复 4.0 发布审查中已经由代码或门禁证实的本地阻断项；不在此切片改写 Git 历史、切换远端、发布、
替换许可证或配置外部仓库安全策略。

## Findings

- Macro 持久格式已升至 v2，但 v1 资产没有确定性读取与持久迁移，列表会静默隐藏解码失败的旧记录。
- Schedule v1-v4 只在内存升级到 v5，没有原子写回，旧文件会在每次启动重复迁移。
- 初次 review 把 Configured Target 缺少 per-node 权限收窄误判为阻断；既有产品决策明确要求配置即授权、
  per-Run 直接调用，并已删除 capability/grant/租约/arm。重新加入 Scope/Requirement 属于性能回归。
- Move Pointer 对 Desktop/Android/Browser 共用 `linear + 300ms` 默认值，后两种 adapter 只接受 instant/0。
- 内置契约可在同一版本下自动替换 semantic digest，不符合持久工作流的版本迁移边界。
- NewAPI 的 HTTP 200 HTML 响应被归为 `invalid-request`，设置页进一步误报为“供应商拒绝”。
- `task check:full` 被 61 个 staticcheck 问题阻断；`govulncheck` 报告可达的 `GO-2026-5970`。

## Plan

- [x] 用冻结的旧格式 fixture 覆盖 Macro 与 Schedule 的迁移、写回和重启读取。
- [x] 回退误加的 Program Target Requirement、admission 验证和 per-node Target Scope，恢复 Configured Target
  per-Run 直接调用；不得以其它名称恢复逐层权限检查。
- [x] 对跨目标指针节点拆分可执行默认值，并为有语义变化的契约使用新版本与显式迁移。
- [x] 为 HTTP 200 HTML 建立稳定错误分类和中英文可操作提示。
- [x] 清理 staticcheck、升级受影响依赖并运行漏洞门禁。
- [x] 运行定向测试、前端检查与一次 `task check:full`，记录真实退出码。

## Validation

- 2026-08-11 owner 确认同一游戏窗口的“捕鱼”性能黄金验收已通过；该真机结果与自动化编辑器截图门禁
  分开记录，不再列为发布前缺失证据。
- 2026-08-11 owner 纠正安全模型：此前 `task check:full` 只能证明实现内部一致，不能推翻
  [配置直驱运行时](../../config-driven-runtime/context.md)的产品决策；逐层验证已有约 10ms→300ms 的实测回归，
  必须回退。现已删除 Program Requirement、admission 预检、per-node Scope/Borrow 与逐操作白名单，
  恢复 per-Run 直接调用；6 个相关 Go 包回归通过。同一游戏窗口的真实延迟复测仍归入发布前真机证据。

- `task check:full` 于 2026-08-11 完成，真实退出码为 0，耗时 263 秒。
- Go 全包测试、vet、staticcheck 与全局覆盖率门禁通过；最终覆盖率为 65.4%。
- 前端格式、lint、类型、i18n、合同、486 项测试、生产构建与 bundle budget 通过。
- 强制 AI eval 为 8/8 approved、0 safety violation；更新后的 report 由正式 Task 生成。
- `govulncheck ./...` 报告 0 项可达漏洞；`golang.org/x/text` 已升级到修复版本。

## Remaining external release work

- 新公共仓库、remote、历史裁剪、签名、ruleset、私有漏洞报告与许可证选择需要 owner 决策或外部权限。
