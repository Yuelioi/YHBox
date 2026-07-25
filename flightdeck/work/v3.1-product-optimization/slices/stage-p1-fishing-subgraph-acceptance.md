# P1 — fishing-v2 子图重组与验收

## Goal

用真实复杂工作流证明 Source-native 子图不只适用于 smoke fixture，并把自动钓鱼主图整理到用户可读。

## Status

Finished

## Result

- 原 Source revision 4：单一 main graph，60 个节点、88 条边、18 组双分辨率图片资源、1 个 Run 状态。
- 原 Source 在当前 Catalog 下因旧 `State Read` 精确摘要产生 `NODE_CONTRACT_MISMATCH`，并连锁出现
  `UNKNOWN_PORT` 与 `MISSING_INPUT_BINDING`。
- 正式 authoring seam 拒绝未获许可的节点 contract migration；草稿改用新增当前 State Read、恢复配置/
  数据边、删除旧节点的显式替换。
- 折叠拉钩区域时捕获 `selection has multiple execution entries`：Repeat 未连接的 `continue` 被误当成
  callable graph entry。最小 Go 红灯已锁定，修复改为按 compiler Instruction 的 `entryInput` 识别循环/
  Retry 主入口，同时保留真正多执行根的拒绝。
- 当前 revision 8 已生成 7 张图：main 3 nodes + 6 calls，六个子图合计 57 nodes；总边数仍为 88。
  资源、变量和除显式替换 State Read 外的节点合同/config/binding/label 均与原 Source 一致。
- 第一次截图显示准备和买饵子图横向过长，最终将准备/买饵/卖鱼改成紧凑蛇形分层；只移动节点位置，
  不改变边、binding、阈值或 timeout。
- production `Yotta.CLI.exe` 对真实 `bin/data` 返回 source/program hash 且 diagnostics 为 null。
- 增量 `task check` 通过 12 个受影响 Go 包。

## Subgraphs

- `prepare-cast`：准备并等待咬钩，10 nodes / 5 exits。
- `hook-attempt`：尝试拉钩，6 nodes / 1 exit。
- `balance-fish`：溜鱼控制，15 nodes / 1 exit。
- `settle-catch`：处理钓鱼结算，4 nodes / 3 exits。
- `buy-bait`：购买并更换鱼饵，15 nodes / 7 exits。
- `sell-catch`：出售全部鱼获，7 nodes / 4 exits。

## Verification

- `go test ./internal/workflow/authoring -run TestEngineCollapsesLoopWithoutTreatingOptionalControlInputsAsGraphEntries -count=1`
- `task check`
- `bin/Yotta.CLI.exe --data-root bin/data compile fishing-v2`：source
  `sha256:0adeafb8242e1d1a4f69f86ae31f84800980be447533c34b2a15628a17ff5122`，program
  `sha256:1584a19a123d721394ad9395b720870f2d01a788c18ee1a9c97140d1495af539`，0 diagnostics。
- 原 revision 4 的恢复 envelope 已移至不参与 SourceStore 扫描的备份目录：
  `bin/data/backups/workflow-sources/fishing-v2.pre-subgraph-20260724-233814.recovery-envelope.json`。
  先前把同 ID 备份放入 `workspace/workflows/` 会被正确隔离并显示为损坏源；这不是当前工作流损坏。
- Windows WebView：main 显示六个调用；子图管理器显示每个 definition 恰有一个调用且接口健康；逐一双击
  六个调用，业务节点和 1/5、1/1、1/1、1/3、1/7、1/4 入口/出口边界完整，无 alert 或 JS error。
- 截图目录：`.task/fishing-gui/20260724-234450/`；最终布局重点证据为 `main.png`、
  `prepare-cast-compact.png`、`buy-bait-compact.png`、`sell-catch-compact.png`。
- `task build`：production bundle budget 通过，生成 3.1.0 Windows GUI；binary metadata 与隔离启动
  5 秒 smoke 通过。
