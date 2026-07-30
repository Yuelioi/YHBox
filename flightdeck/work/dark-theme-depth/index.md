# 暗色主题层次与设置中心视觉收口

## Goal

把 Yotta 现有依赖纯黑填充、顶光和阴影的暗色表面改成有明确明度层级、克制色彩和稳定语义角色的
桌面工具主题；重点完成设置中心、工作流管理和资产管理的视觉收口，并通过视觉验收与本地门禁。

## Status

Finished

## Current

已完成死机现场恢复、官方设计系统调研、暗色主题实现与视觉验收。全局暗色基底不再使用 zinc-950
纯黑，而是使用偏冷近黑的 OKLCH 语义阶梯：canvas L17、surface 约 L21、hover 约 L24、strong
约 L26，border L29.5；overlay 使用最高实体表面和有位移的柔和阴影。设置分类色仅用于导航、标题和
极轻区域导向，设置区块内部保持平铺，工作流与资产管理共用同一套 surface 角色。

CLI Playwright 使用真实 Edge 渲染并通过受控 Wails RPC 数据检查设置 AI、工作流管理和资产管理；
最终一轮无 `pageerror`、无未知 RPC。Impeccable 检测为零问题，前端 quick check 与最终增量
`task check` 均通过。

## Next

None

## Progress

- 2026-07-31 对照实时 Git 状态恢复死机现场，确认暗色主题工作已产生完整源码 diff，但此前未写入
  Flightdeck；原 `v4-followup-stabilization` Work 保留为 Open，本 Work 成为唯一 Focus。
- 2026-07-31 完成第一轮官方来源矩阵，确认主流暗色系统以实体明度层和语义 token 建立层次，阴影只作
  辅助，不让所有容器停留在同一个黑值。
- 2026-07-31 用 CLI Playwright 完成两轮桌面视觉验收；最终设置、工作流和资产截图均呈现稳定实体层级，
  页面无脚本错误或未知 RPC，截图与临时结果保存在 gitignore 的 `.task/`。
- 2026-07-31 Impeccable 检测零问题；`pnpm -C frontend check:quick` 通过 106 个测试文件、463 个用例；
  最终 `task check` 按变更路由到 `check:frontend:quick` 并以退出码 0 完成。
