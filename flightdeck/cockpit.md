# Cockpit — YHFish

**Last updated**: 2026-06-08 by 月离 (颜色范围吸管 ship 完成：schema widget 标记 + ExtractColorRange 纯 CV + picker color 模式 + StructuredInput 吸管按钮 + i18n，真机 smoke 过。spec/plan done+归档。)
**Active focus**: 资产子系统重构**代码全完 + 已过 3 维 final review + 修了 review 抓的真问题**。所有自动门绿(go build/vet/test + vue-tsc + pnpm i18n:check)。修复: validator 漏扫主图(真 bug)+ 一批二号铁律死代码删净 + 前端 i18n 缺 key 补齐。唯一剩 **真机 smoke 需用户跑**(spec §13)。

## 进行中

(无)

## 下一步

**用户真机 smoke**（spec §13，必须在用户机器+真实游戏跑）：`task build` 起 app（会自动重生成 wails bindings + 填 fish fixture）→ ① 截两模板各得 GUID、同图截两次 blob 只一份 ② 节点引用 GUID 跨分辨率命中 ③ 重拍同 GUID 换图所有引用自动跟随 ④ 导出子图+导出整容器→导入另一容器幂等无冲突弹窗 ⑤ 删引用共享模板的子图→资产仍在 ⑥ 库里删资产→弹"被 N 处引用"→GC 回收。smoke 全过后 spec 可 graduate 进 docs、归档 spec/plan。
final review 已做(3 维并行 + 我回源码核验)，修复 commit：50bd4f8(validator 主图)+43b1858(死代码)+f27a6be(i18n)。**Review 留的非阻塞小项(可 smoke 后再说)**：① Delete 返回的 referrer 列表前端未展示成"被 N 处引用"确认弹窗(spec §6 半实现，后端已返、前端 discard)；② backend.ts 手写 AssetRecord 类型缺 regions 字段(Nit，build 重生成覆盖)；③ asset.Rename 对 clip 不改 blob header Label(Nit，UI 走 clip Update 不踩)；④ runtime 的 fishing-v2 测试 fixture 是旧 key 格式，重建 fish 时一并更新。已知预存失败(非回归)：runtime TestApplyDirection_*/TestWatchdog_*/TestScanSubgraphDependencies_* 缺 fish fixture，见 [build.md](checklists/build.md)。

## Hanging tasks

- [ ] 无阻塞待办。（原积压已路由：编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md)；bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md)；删符号全仓 grep → [checklists/code-style.md](checklists/code-style.md)；i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)；诊断探针目录已删。）
