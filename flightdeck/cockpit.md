# Cockpit — YHFish

**Last updated**: 2026-06-08 by 月离 (颜色范围吸管 ship 完成：schema widget 标记 + ExtractColorRange 纯 CV + picker color 模式 + StructuredInput 吸管按钮 + i18n，真机 smoke 过。spec/plan done+归档。)
**Active focus**: 资产子系统重构 **代码全部完成、所有自动门绿**(go build/vet/test + 前端 vue-tsc + pnpm build)。subagent 执行 Phase 0-7 + 接线全完。唯一剩 **真机 smoke 需用户跑**(spec §13)。template 包+旧 inputclip 存储+cmd/import-fish-data 已整删(二号铁律)。

## 进行中

(无)

## 下一步

**用户真机 smoke**（spec §13，必须在用户机器+真实游戏跑）：`task build` 起 app（会自动重生成 wails bindings + 填 fish fixture）→ ① 截两模板各得 GUID、同图截两次 blob 只一份 ② 节点引用 GUID 跨分辨率命中 ③ 重拍同 GUID 换图所有引用自动跟随 ④ 导出子图+导出整容器→导入另一容器幂等无冲突弹窗 ⑤ 删引用共享模板的子图→资产仍在 ⑥ 库里删资产→弹"被 N 处引用"→GC 回收。smoke 全过后 spec 可 graduate 进 docs、归档 spec/plan。
可选：smoke 前过一轮 final code review（/code-review 或喂 diff 给 AI reviewer）。
commit 链：1af14f5→792bcaa→f048268→4bf8f82→94c0e0f→50ff19c(后端)→dd12be8(前端)。已知预存失败(非回归)：runtime 的 TestApplyDirection_*/TestWatchdog_*/TestScanSubgraphDependencies_* 缺 fish fixture，见 [build.md](checklists/build.md)。

## Hanging tasks

- [ ] 无阻塞待办。（原积压已路由：编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md)；bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md)；删符号全仓 grep → [checklists/code-style.md](checklists/code-style.md)；i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)；诊断探针目录已删。）
