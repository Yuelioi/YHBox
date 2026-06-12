# Cockpit — YHFish

**Last updated**: 2026-06-12 by 月离 (数据层大整理 spec 三轮外审收口(第三轮: schemaVersion 读取契约/引用只走 ID/rev 仅单机/RequiredGlobals 只存名字), P1/P2 两个 plan 落盘, 待用户过目后开干。)
**Active focus**: **数据层大整理**(目录平铺 + 子图全局化) — spec 三轮外审定稿 + [P1 存储平铺](plans/2026-06-12-p1-storage-flatten.md) / [P2 子图全局化](plans/2026-06-12-p2-subgraph-globalize.md) 两计划就绪, 说 go 即开 P1。迁移一次性脚本在 P2 末跑+真机。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

<!-- AUTO:inprogress -->
- [2026-06-12-data-layout-flatten-subgraph-globalize.md](specs/2026-06-12-data-layout-flatten-subgraph-globalize.md) — 数据层大整理 — 目录平铺(删 assets/library 中间层) + 子图全局化(容器只引用不复制) + 架构耐久性根基
- [2026-06-12-p1-storage-flatten.md](plans/2026-06-12-p1-storage-flatten.md) — Phase 1 存储平铺 — assets 按类拆 templates//clips/ + blobs 上提 + schedules 拍平 + 死目录清除 + 启动防呆闸 + schemaVersion 字段
- [2026-06-12-p2-subgraph-globalize.md](plans/2026-06-12-p2-subgraph-globalize.md) — Phase 2 子图全局化 — 全局 subgraph.Store + rev 乐观锁 + 闭包咽喉 ClosureResult + referrer 删除安全 + 匿名 GC + library 整删 + 前端池化 + 一次性迁移脚本
<!-- /AUTO -->

## 下一步

用户过目 P1/P2 计划 → go 即按 [P1](plans/2026-06-12-p1-storage-flatten.md) 任务 1-7 开干(asset 按类拆 + blobs 上提 + schedules 拍平 + 死目录清除 + 防呆闸), 接 [P2](plans/2026-06-12-p2-subgraph-globalize.md) 1-17(全局 store → 咽喉 → 运行时 → RPC/GC → library 删 → 前端池化 → 迁移脚本+真机)。真机验证与数据迁移统一压 P2 末(P1 期间旧数据不可启动是预期, 防呆闸兜着)。真机债一条不变(待验证: 删被引用模板 referrer 警告)。其余候选(无紧迫): WaitTemplate 孤儿边原子性硬化(真机再现再修); 复发#5 promotion 候选(前台容器全局指针 onMounted+onActivated 升 checklist); 脚本 SubgraphID 容错(未拍板, Phase 2 validator 全局校验会顺带覆盖大半); 搜索/大复合 modal 收进 BaseModal; idea 池(cv-perception · editor-footgun · misc-tools); oxlint 预存 16 错; residue 28 处; StateCycleSmoke 预存红(build.md 在册)。

## 待复核

- 无。(vars.\* 删漏 2026-06-12 用户拍板补删, 已销。)

## 待验证

- ⚠ [archive/specs/2026-06-11-script-template-dep-extraction.md](archive/specs/2026-06-11-script-template-dep-extraction.md) — 库里删一个被某脚本引用的模板,确认弹「被引用」referrer 警告 + gcBlobs 不回收其 blob(单测已覆盖提取+扫描器接线,差集成/真机这一验)。

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。本节只放**两边都装不下的活上下文**, 现在没有。(2026-06-11 清版: 原"加节点路线图存档指针"等五条已分别沉淀进 docs / 待复核 / build.md, 不再重复。)
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **39** = 28(misc-tools-backlog) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 16 错(oxlint 1.64 新规则, 散在 10 个未涉编辑器的文件)。跑全套测试/检查时按此判红。
