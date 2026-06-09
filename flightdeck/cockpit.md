# Cockpit — YHFish

**Last updated**: 2026-06-10 by 月离 (资产子系统 **graduate 归档** —— spec → [docs/asset-subsystem.md](docs/asset-subsystem.md), plan → archive。全套真机 smoke 过、vue-tsc 0、UX 反馈全收口(变体删确认/editor 导入自动补 var/分享改 NuxtUI)。资产子系统彻底收口。)
**Active focus**: 资产子系统**已收口归档**(→ [docs/asset-subsystem.md](docs/asset-subsystem.md))。无 active spec/plan。下一步候选见下(modal 风格统一已拍范围 / 子图切换真机 smoke / idea 池)。

## 进行中

- [ ] **资产 modal 收尾 + 全局 modal 风格统一(待用户拍范围)**。已修(多轮): 删除提示(无引用不再误报"引用失效")、网格批量删除(footer「删除选中 N」, **批量也汇总显示被引用处数**)、详情页"包裹"框感(缩放区/信息栏圆角描边+留白)、**详情 ✕ 改返回网格(不直接关 modal, 免误触)**、**截图时可设标签**(SaveTemplateCapture 加 tags 参 + ScreenPicker template_save 表单加标签输入)、**网格按标签筛选 + 卡片缩略图显示标签**。门绿(go build/test、vue-tsc 0 新增、i18n 1744、198 单测)。**用户已拍范围(2026-06-09)**: 先 smoke 确认资产 modal "包裹"基准 → 然后**抽共享 modal 外壳 + 扫一遍常规 UModal**(确认框/导入/容器设置/计划等)统一;**frameless HUD(录制/截屏/DPI 校准)单独评估**(HudShell 独立小窗, 套大面板未必合适)。**资产 modal 基准已 smoke 确认(2026-06-10),可启动统一这步。**

## 下一步

资产子系统已收口。候选下一步(用户拍): ① **全局 modal 风格统一**(资产 modal 基准已立、范围已拍 2026-06-09): 抽共享 modal 外壳 → 扫常规 UModal(确认框/导入/容器设置/计划)统一; frameless HUD 单独评估。② **子图切换真机 smoke 复验**(Hanging, 见下: 容器2 折叠子图→切容器3→切回容器2 应正常+分享成功)。③ idea 池(cv-perception / editor-footgun / misc-tools)。已知预存失败(非回归): runtime TestApplyDirection_*/TestWatchdog_*/TestScanSubgraphDependencies_* 缺 fish fixture, 见 [build.md](checklists/build.md)。

## Hanging tasks

- [ ] **子图系统问题（大部分已根治，剩真机 smoke 复验）**。修了三条真 bug，前两条同症状 "(子图未找到)" 不同根因：① 库导入绕过容器 Store 内存缓存 → [incident](incidents/2026-06-09-import-bypasses-container-store-cache.md)（library SetContainerReloader + 回归测试）；② keep-alive 多容器编辑器共享全局单例子图 store 切回污染 → [incident](incidents/2026-06-09-keepalive-singleton-subgraph-store-stale.md)。② 已**根治**（commit 20e25a9）：store 状态按容器隔离（subgraphsByContainer / editorPathByContainer keyed by containerID），activeContainerID 降级成前台指针，对外 API 不变、单测 59 绿；顺带消除未落盘子图编辑/层级切换丢失 + id 碰撞 mergeSubgraphs 取错版本。先前的 onActivated 补丁(9ccccbf)已被根治替换。**待用户真机 smoke 复验**：容器2 折叠子图 → 切容器3 → 切回容器2，子图节点应仍正常 + 分享成功。（原记的 2 个预存 vue-tsc 红已在 2026-06-10 import-strategy 收口随资产子系统清零。）
- [ ] 无阻塞待办。（原积压已路由：编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md)；bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md)；删符号全仓 grep → [checklists/code-style.md](checklists/code-style.md)；i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)；诊断探针目录已删。）
