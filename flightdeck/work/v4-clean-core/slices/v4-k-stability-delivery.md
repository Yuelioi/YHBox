# V4-K 稳定交付

## Goal

把“旧数据没有丢、用户路径没有写进 Workflow、启动速度可接受”从人工承诺变成可重复的验收入口，
并用 production build、真实 profile 副本和 Windows WebView 完成最终交付验证。

## Status

Completed

## Result

- `task workflow:retention` 只读检查黄金 `fishing-v2` Source，验证 36 个 Blob 的大小与 SHA-256，
  在内存中执行兼容迁移后通过当前节点目录和唯一编译路径生成可执行 Program。
- 保留基线要求至少 7 个 Graph、60 个 Node、18 个 Resource 和 36 个 Blob 引用；当前样本全部通过。
- 黄金 Source 的 Target 默认值数量为 0。每用户应用路径、窗口和设备配置继续由 Settings 管理，
  保存时准备并原子切换 automation generation，失败不改变当前运行代。
- `task webview:smoke:fishing` 将黄金数据复制到隔离 profile 后打开真实编辑器；原始数据不被修改。
- WebView smoke 记录进程启动到调试端点和工作流首屏耗时，默认预算分别为 15 秒和 5 秒。
- production Windows bundle 继续执行入口包和编辑器包体积预算。

## Capability evidence

- 黄金保留报告：Workflow `fishing-v2`，7 Graph、60 Node、18 Resource、36 Blob 引用；
  Source hash 为 `sha256:255af38eca2648738b4f0829345b1df3acf4901dfda4c8a96cb93cb0ce64f3a6`。
- 黄金 WebView 旅程通过 `.task/workflow-editor-smoke/20260726-222540`，首屏 2.506 秒，
  启动 10.504 秒。
- `%LOCALAPPDATA%\Yotta\Yotta` 的 layout 3 health 与迁移计划只读检查通过；正确隔离副本
  `.task/real-profile-validation/20260726-222816` 编译 `fishing-v2` 成功，原 profile 未写入。
- 最终 production build 位于 `.task/v4-production/bin/Yotta.exe`，入口 / 编辑器 gzip 为
  249,341 / 207,006 bytes；完整 Windows WebView 旅程通过
  `.task/workflow-editor-smoke/20260726-225540`，启动 8.693 秒。
- automation generation 测试覆盖同进程 Target 发布、Run 可见、移除和失败回滚；Settings 测试覆盖
  durable save 前 prepare、成功 commit 与失败 abort。

## Verification

- `task workflow:retention`
- `task webview:smoke:fishing`
- `task windows:build BIN_DIR='.task/v4-production/bin'`
- `task webview:smoke:full`
- 最终 `task check`：12 个受影响 Go 包、Wails 16 服务 / 139 方法契约、format、lint、
  TypeScript、2491 个 i18n key 和 93 个测试文件 / 390 项测试通过。
