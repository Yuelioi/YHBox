# Cockpit — YHFish

**Last updated**: 2026-06-11 by 月离 (fishing-v2 真机过 → 补做模板资产化迁移 (18 模板 namespace.name → GUID assets) + bin/data 清 48MB; Script 增强两阶段立项。)
**Active focus**: **Script 增强两阶段立项** — Stage1 资产依赖提取 (active, 近期可做) / Stage2 脚本调子图 (gated 下一阶段)。fishing-v2 迁移已收口 (容器壳 + 模板资产化, 真机过)。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

<!-- AUTO:inprogress -->
- [2026-06-11-script-template-dep-extraction.md](specs/2026-06-11-script-template-dep-extraction.md) — Script 节点扫 Code 里的模板/clip/subgraph GUID 字面量当依赖,堵住"脚本引用的资产被 GC 误删、库里删不警告"的盲区
- [2026-06-11-script-call-subgraph.md](specs/2026-06-11-script-call-subgraph.md) — 把 Subgraph 暴露成脚本绑定函数 Subgraph({SubgraphID, ...params}),让脚本当编排层复用子图库(下一阶段,gated 在 Stage 1 之后) [note: 下一阶段:自包含单脚本已够用,仅当要复用子图库时启动;先做 Stage 1]
<!-- /AUTO -->

## 下一步

启动 Stage 1 (script 资产依赖提取) 实现, 或先拿 fishing 写一个单脚本版 spike 验证 while-loop 主程序端到端能跑 (用户已倾向自包含路线)。

**之后候选**(无紧迫): 搜索/大复合 modal 是否收进 BaseModal; idea 池(cv-perception · editor-footgun · misc-tools); oxlint 预存 16 错清债; residue 28 处 HUD/Launcher 硬编码中文(misc-tools-backlog 在册)。

## 待复核

- 无。

## 待验证

- 无。

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。本节只放**两边都装不下的活上下文**, 现在没有。(2026-06-11 清版: 原"加节点路线图存档指针"等五条已分别沉淀进 docs / 待复核 / build.md, 不再重复。)
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **39** = 28(misc-tools-backlog) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 16 错(oxlint 1.64 新规则, 散在 10 个未涉编辑器的文件)。跑全套测试/检查时按此判红。
