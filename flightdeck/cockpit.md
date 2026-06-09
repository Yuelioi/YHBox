# Cockpit — YHFish

**Last updated**: 2026-06-10 by 月离 (import-strategy 收口 + 单变体删除。删 strategy/conflict 整套: ImportToContainerDialog 简化成单次幂等导入、SubgraphPackage 类型 templates/clips→assets、LibraryCard/DetailPanel 改 assets 计数、3 调用点去 strategy 参 → **vue-tsc 2 预存红清零(现 0 红)**。单变体删除: 详情页 >1 档时 chip 加 ✕ + 后端 `RemoveVariant`(守卫最后一档不可删)。门全绿: vue-tsc **0**、i18n 1749+compile、go asset+library test、vitest 198。i18n residue 28 = 未碰文件旧 backlog 非回归。｜前序: 当前分辨率感知; 翻页; 钻入式 modal。)
**Active focus**: 资产子系统**代码全完 + vue-tsc 全清(0 红)+ smoke 主体全过**。剩 2 项待真机快验: ① 导入子图到容器 dialog(已简化成单次幂等导入)② 单变体删除。两项过 → asset plan/spec → done → graduate 进 docs、归档,资产子系统彻底收口。

## 进行中

- [ ] **资产 modal 收尾 + 全局 modal 风格统一(待用户拍范围)**。已修(多轮): 删除提示(无引用不再误报"引用失效")、网格批量删除(footer「删除选中 N」, **批量也汇总显示被引用处数**)、详情页"包裹"框感(缩放区/信息栏圆角描边+留白)、**详情 ✕ 改返回网格(不直接关 modal, 免误触)**、**截图时可设标签**(SaveTemplateCapture 加 tags 参 + ScreenPicker template_save 表单加标签输入)、**网格按标签筛选 + 卡片缩略图显示标签**。门绿(go build/test、vue-tsc 0 新增、i18n 1744、198 单测)。**用户已拍范围(2026-06-09)**: 先 smoke 确认资产 modal "包裹"基准 → 然后**抽共享 modal 外壳 + 扫一遍常规 UModal**(确认框/导入/容器设置/计划等)统一;**frameless HUD(录制/截屏/DPI 校准)单独评估**(HudShell 独立小窗, 套大面板未必合适)。**铺开这步先不写, 等用户确认资产 modal 基准后再启动。**

## 下一步

**真机快验 2 项 → 然后 graduate**(代码已绿、待 commit): ① **导入子图到容器**(库面板某子图 → 导入 → 选目标容器 → 「下一步」: 无缺失变量直接 done / 有则提示补变量再 done; **已无冲突步、单次导入即落盘**) ② **单变体删除**(详情页某素材有 ≥2 分辨率档时每个 chip 右侧有 ✕ → 删一档, 另一档还在、引用节点用剩下的档; 仅 1 档时无 ✕, 删它走整删按钮)。两项过 → asset plan/spec 转 done → spec graduate 进 docs(graduate:true)、归档 spec+plan, 资产子系统彻底收口。已知预存失败(非回归): runtime TestApplyDirection_*/TestWatchdog_*/TestScanSubgraphDependencies_* 缺 fish fixture, 见 [build.md](checklists/build.md)。

## Hanging tasks

- [ ] **子图系统问题（大部分已根治，剩真机 smoke 复验）**。修了三条真 bug，前两条同症状 "(子图未找到)" 不同根因：① 库导入绕过容器 Store 内存缓存 → [incident](incidents/2026-06-09-import-bypasses-container-store-cache.md)（library SetContainerReloader + 回归测试）；② keep-alive 多容器编辑器共享全局单例子图 store 切回污染 → [incident](incidents/2026-06-09-keepalive-singleton-subgraph-store-stale.md)。② 已**根治**（commit 20e25a9）：store 状态按容器隔离（subgraphsByContainer / editorPathByContainer keyed by containerID），activeContainerID 降级成前台指针，对外 API 不变、单测 59 绿；顺带消除未落盘子图编辑/层级切换丢失 + id 碰撞 mergeSubgraphs 取错版本。先前的 onActivated 补丁(9ccccbf)已被根治替换。**待用户真机 smoke 复验**：容器2 折叠子图 → 切容器3 → 切回容器2，子图节点应仍正常 + 分享成功。**预存红（非本次引入，已回退越界改动）**：frontend vue-tsc 2 错 — backend.ts importToContainer stale `strategy` 参（ImportToContainerDialog/useFlowInteraction 等还在传）+ 手写 SubgraphPackage 缺 templates/clips（LibraryCard/LibraryDetailPanel 在用）；这俩属**资产子系统** strategy/类型漂移，归 asset 重构那波清，别单独动（会牵出 import-strategy UI 是否废弃的决策）。
- [ ] 无阻塞待办。（原积压已路由：编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md)；bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md)；删符号全仓 grep → [checklists/code-style.md](checklists/code-style.md)；i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)；诊断探针目录已删。）
