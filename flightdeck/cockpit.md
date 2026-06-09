# Cockpit — YHFish

**Last updated**: 2026-06-10 by 月离 (smoke 反馈 2 修: ① 删分辨率档确认框补说明文字(原 body 空) ② **容器编辑器导入子图自动补缺失全局变量**(editor 拖/点导入原不提示缺 var → 现 import 后把子图 RequiredGlobals 缺的自动补进容器 + toast; 共享 helper `library.importEnsuringGlobals`, 走 NodePalette/useNodeCreation/useFlowInteraction; 库管理页 dialog 维持原提示步不动)。门全绿: vue-tsc **0**、i18n 1751+compile、vitest 198。｜前序: import-strategy 收口(vue-tsc 0 红); 单变体删除; 当前分辨率感知; 翻页。)
**Active focus**: 资产子系统**代码全完 + vue-tsc 0 红 + smoke 基本全过(剩本轮 2 修复待复验)**。复验 ① 删档确认框有说明 ② editor 导入用了全局变量的子图 → 自动补 var + toast。过后 asset plan/spec → done → graduate 进 docs、归档,资产子系统彻底收口。

## 进行中

- [ ] **资产 modal 收尾 + 全局 modal 风格统一(待用户拍范围)**。已修(多轮): 删除提示(无引用不再误报"引用失效")、网格批量删除(footer「删除选中 N」, **批量也汇总显示被引用处数**)、详情页"包裹"框感(缩放区/信息栏圆角描边+留白)、**详情 ✕ 改返回网格(不直接关 modal, 免误触)**、**截图时可设标签**(SaveTemplateCapture 加 tags 参 + ScreenPicker template_save 表单加标签输入)、**网格按标签筛选 + 卡片缩略图显示标签**。门绿(go build/test、vue-tsc 0 新增、i18n 1744、198 单测)。**用户已拍范围(2026-06-09)**: 先 smoke 确认资产 modal "包裹"基准 → 然后**抽共享 modal 外壳 + 扫一遍常规 UModal**(确认框/导入/容器设置/计划等)统一;**frameless HUD(录制/截屏/DPI 校准)单独评估**(HudShell 独立小窗, 套大面板未必合适)。**铺开这步先不写, 等用户确认资产 modal 基准后再启动。**

## 下一步

**真机复验本轮 2 个 smoke 修复 → 然后 graduate**(代码已绿、待 commit): ① **删分辨率档确认框**(详情页 ≥2 档某 chip 点 ✕ → 确认框现在应有说明文字「只删这一档…」, 不再空 body) ② **editor 导入子图自动补变量**(把用了全局变量的库子图拖/点进某容器 → 应**自动把缺的变量补进容器 + 弹 toast「已自动补充 v1」**, 不再静默丢; 用 sg-9d6e9849 进新容器可复现, 它要 v1)。两项过 → asset plan/spec 转 done → spec graduate 进 docs(graduate:true)、归档 spec+plan, 资产子系统彻底收口。已知预存失败(非回归): runtime TestApplyDirection_*/TestWatchdog_*/TestScanSubgraphDependencies_* 缺 fish fixture, 见 [build.md](checklists/build.md)。

## Hanging tasks

- [ ] **子图系统问题（大部分已根治，剩真机 smoke 复验）**。修了三条真 bug，前两条同症状 "(子图未找到)" 不同根因：① 库导入绕过容器 Store 内存缓存 → [incident](incidents/2026-06-09-import-bypasses-container-store-cache.md)（library SetContainerReloader + 回归测试）；② keep-alive 多容器编辑器共享全局单例子图 store 切回污染 → [incident](incidents/2026-06-09-keepalive-singleton-subgraph-store-stale.md)。② 已**根治**（commit 20e25a9）：store 状态按容器隔离（subgraphsByContainer / editorPathByContainer keyed by containerID），activeContainerID 降级成前台指针，对外 API 不变、单测 59 绿；顺带消除未落盘子图编辑/层级切换丢失 + id 碰撞 mergeSubgraphs 取错版本。先前的 onActivated 补丁(9ccccbf)已被根治替换。**待用户真机 smoke 复验**：容器2 折叠子图 → 切容器3 → 切回容器2，子图节点应仍正常 + 分享成功。**预存红（非本次引入，已回退越界改动）**：frontend vue-tsc 2 错 — backend.ts importToContainer stale `strategy` 参（ImportToContainerDialog/useFlowInteraction 等还在传）+ 手写 SubgraphPackage 缺 templates/clips（LibraryCard/LibraryDetailPanel 在用）；这俩属**资产子系统** strategy/类型漂移，归 asset 重构那波清，别单独动（会牵出 import-strategy UI 是否废弃的决策）。
- [ ] 无阻塞待办。（原积压已路由：编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md)；bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md)；删符号全仓 grep → [checklists/code-style.md](checklists/code-style.md)；i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)；诊断探针目录已删。）
