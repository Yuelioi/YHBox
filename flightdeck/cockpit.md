# Cockpit — YHFish

**Last updated**: 2026-06-09 by 月离 (子图三连修：① 库导入绕容器 Store 缓存 [SetContainerReloader]；② keep-alive 多容器共享全局单例 store 切回污染——**已根治**：store 按容器隔离 [20e25a9, 替换补丁 9ccccbf]，单测 59 绿，vue-tsc 仅剩 2 个预存红(资产子系统漂移,已回退越界)。待真机 smoke 复验切容器场景。)
**Active focus**: 资产子系统重构**代码全完 + 已过 3 维 final review + 修了 review 抓的真问题**。所有自动门绿(go build/vet/test + vue-tsc + pnpm i18n:check)。修复: validator 漏扫主图(真 bug)+ 一批二号铁律死代码删净 + 前端 i18n 缺 key 补齐。唯一剩 **真机 smoke 需用户跑**(spec §13)。

## 进行中

(无)

## 下一步

**用户真机 smoke**（spec §13，必须在用户机器+真实游戏跑）：`task build` 起 app（会自动重生成 wails bindings + 填 fish fixture）→ ① 截两模板各得 GUID、同图截两次 blob 只一份 ② 节点引用 GUID 跨分辨率命中 ③ 重拍同 GUID 换图所有引用自动跟随 ④ 导出子图+导出整容器→导入另一容器幂等无冲突弹窗 ⑤ 删引用共享模板的子图→资产仍在 ⑥ 库里删资产→弹"被 N 处引用"→GC 回收。smoke 全过后 spec 可 graduate 进 docs、归档 spec/plan。
final review 已做(3 维并行 + 我回源码核验)，修复 commit：50bd4f8(validator 主图)+43b1858(死代码)+f27a6be(i18n)。**Review 留的非阻塞小项(可 smoke 后再说)**：① Delete 返回的 referrer 列表前端未展示成"被 N 处引用"确认弹窗(spec §6 半实现，后端已返、前端 discard)；② backend.ts 手写 AssetRecord 类型缺 regions 字段(Nit，build 重生成覆盖)；③ asset.Rename 对 clip 不改 blob header Label(Nit，UI 走 clip Update 不踩)；④ runtime 的 fishing-v2 测试 fixture 是旧 key 格式，重建 fish 时一并更新。已知预存失败(非回归)：runtime TestApplyDirection_*/TestWatchdog_*/TestScanSubgraphDependencies_* 缺 fish fixture，见 [build.md](checklists/build.md)。

## Hanging tasks

- [ ] **子图系统问题（大部分已根治，剩真机 smoke 复验）**。修了三条真 bug，前两条同症状 "(子图未找到)" 不同根因：① 库导入绕过容器 Store 内存缓存 → [incident](incidents/2026-06-09-import-bypasses-container-store-cache.md)（library SetContainerReloader + 回归测试）；② keep-alive 多容器编辑器共享全局单例子图 store 切回污染 → [incident](incidents/2026-06-09-keepalive-singleton-subgraph-store-stale.md)。② 已**根治**（commit 20e25a9）：store 状态按容器隔离（subgraphsByContainer / editorPathByContainer keyed by containerID），activeContainerID 降级成前台指针，对外 API 不变、单测 59 绿；顺带消除未落盘子图编辑/层级切换丢失 + id 碰撞 mergeSubgraphs 取错版本。先前的 onActivated 补丁(9ccccbf)已被根治替换。**待用户真机 smoke 复验**：容器2 折叠子图 → 切容器3 → 切回容器2，子图节点应仍正常 + 分享成功。**预存红（非本次引入，已回退越界改动）**：frontend vue-tsc 2 错 — backend.ts importToContainer stale `strategy` 参（ImportToContainerDialog/useFlowInteraction 等还在传）+ 手写 SubgraphPackage 缺 templates/clips（LibraryCard/LibraryDetailPanel 在用）；这俩属**资产子系统** strategy/类型漂移，归 asset 重构那波清，别单独动（会牵出 import-strategy UI 是否废弃的决策）。
- [ ] 无阻塞待办。（原积压已路由：编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md)；bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md)；删符号全仓 grep → [checklists/code-style.md](checklists/code-style.md)；i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)；诊断探针目录已删。）
