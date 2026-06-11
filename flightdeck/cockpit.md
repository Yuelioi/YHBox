# Cockpit — YHFish

**Last updated**: 2026-06-12 by 月离 (fishing-v2 真机验过; 脚本调子图首试撞"子图不存在"=用户把节点 ID 当 SubgraphID, 根因子图 ID 不可见 → 子图属性面板加 ID 显示+复制, 顺手补齐 subgraphProps i18n 缺 key。)
**Active focus**: **Script 增强两阶段全部落地** — Stage1(资产依赖提取) + Stage2(脚本调子图, 含前置的子图多出口路由根治)均已归档, 剩两条真机验证债(见 待验证)。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

<!-- AUTO:inprogress -->

<!-- /AUTO -->

## 下一步

清两条真机验证债(见 待验证: 脚本删模板 referrer 警告; 多出口子图 + 脚本 Subgraph() 调用重试 — 用户脚本里 SubgraphID 改成 sg-a61b5b3d 后再跑)。之后无在飞项目, 候选(无紧迫): 脚本 SubgraphID 容错(运行时报错附现有子图列表 / 编辑期校验字面 SubgraphID 存在性, 2026-06-12 提出未拍板); 搜索/大复合 modal 是否收进 BaseModal; idea 池(cv-perception · editor-footgun · misc-tools); oxlint 预存 16 错清债; residue 28 处 HUD/Launcher 硬编码中文(misc-tools-backlog 在册); StateCycleSmoke 本机预存红排查(build.md 在册, mock 模板没命中)。

## 待复核

- 无。

## 待验证

- ⚠ [archive/specs/2026-06-11-script-template-dep-extraction.md](archive/specs/2026-06-11-script-template-dep-extraction.md) — 库里删一个被某脚本引用的模板,确认弹「被引用」referrer 警告 + gcBlobs 不回收其 blob(单测已覆盖提取+扫描器接线,差集成/真机这一验)。
- ⚠ [archive/specs/2026-06-11-script-call-subgraph.md](archive/specs/2026-06-11-script-call-subgraph.md) — 编辑器造一个多出口子图 + 脚本 Subgraph() 调它,确认返回的 exit 名和入参都对(fishing-v2 状态机流转已验过; 子图 ID 现在能在子图属性面板复制)。

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。本节只放**两边都装不下的活上下文**, 现在没有。(2026-06-11 清版: 原"加节点路线图存档指针"等五条已分别沉淀进 docs / 待复核 / build.md, 不再重复。)
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **39** = 28(misc-tools-backlog) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 16 错(oxlint 1.64 新规则, 散在 10 个未涉编辑器的文件)。跑全套测试/检查时按此判红。
