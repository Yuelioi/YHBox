# Cockpit — YHFish

**Last updated**: 2026-06-11 by 月离 (Stage1 资产依赖提取实现+落地(测试绿); 修早先模板迁移弄回归的 10 个 mock-based state 测试; Stage2 开工撞前置缺口(子图多出口路由没接线)→ 卡住待拍板。)
**Active focus**: **Script 增强 — Stage1 done, Stage2 BLOCKED** 等用户拍板。Stage1(脚本资产依赖提取)已落地归档。Stage2(脚本调子图)读源码撞前置缺口:v5 Subgraph 多出口路由没接线 → 需定 A(先补多出口路由)/ B(先上单 Done 出口)。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

<!-- AUTO:inprogress -->
- [2026-06-11-script-call-subgraph.md](specs/2026-06-11-script-call-subgraph.md) — 把 Subgraph 暴露成脚本绑定函数 Subgraph({SubgraphID, ...params}),让脚本当编排层复用子图库(gated:撞前置缺口,需先定多出口路由) — [note: BLOCKED — 开工读源码撞前置缺口:v5 runtime 的 Subgraph 多出口路由根本没接线(见「开工发现」)。需用户拍板:先补多出口路由,还是 Stage 2 先上单 Done 出口语义。]
<!-- /AUTO -->

## 下一步

用户拍板 Stage 2 走法:**A** 先补「子图多出口路由」(同时修好 fishing try_hook_F.failed 等 graph 层多出口子图,再做脚本调子图返正确出口) 还是 **B** Stage 2 先上单 Done 出口语义。详见 [script-call-subgraph spec 的「开工发现」](specs/2026-06-11-script-call-subgraph.md)。

**之后候选**(无紧迫): 搜索/大复合 modal 是否收进 BaseModal; idea 池(cv-perception · editor-footgun · misc-tools); oxlint 预存 16 错清债; residue 28 处 HUD/Launcher 硬编码中文(misc-tools-backlog 在册)。

## 待复核

- 无。

## 待验证

- ⚠ [archive/specs/2026-06-11-script-template-dep-extraction.md](archive/specs/2026-06-11-script-template-dep-extraction.md) — 库里删一个被某脚本引用的模板,确认弹「被引用」referrer 警告 + gcBlobs 不回收其 blob(单测已覆盖提取+扫描器接线,差集成/真机这一验)。

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。本节只放**两边都装不下的活上下文**, 现在没有。(2026-06-11 清版: 原"加节点路线图存档指针"等五条已分别沉淀进 docs / 待复核 / build.md, 不再重复。)
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **39** = 28(misc-tools-backlog) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 16 错(oxlint 1.64 新规则, 散在 10 个未涉编辑器的文件)。跑全套测试/检查时按此判红。
