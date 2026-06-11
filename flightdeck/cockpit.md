# Cockpit — YHFish

**Last updated**: 2026-06-12 by 月离 (子图一键转脚本全量落地归档(转换器纯函数+预览 modal+两入口), 测试全绿差真机; 同日: 脚本调子图真机全验过债销, 子图 ID 面板可复制。)
**Active focus**: **子图一键转脚本已落地** — 右键/属性面板转等价脚本, 预览+复制/插入, 不支持结构整体拒转。剩两条真机验证债(见 待验证)。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

<!-- AUTO:inprogress -->

<!-- /AUTO -->

## 下一步

清两条真机验证债(见 待验证: 子图转脚本真机走一遍; 库里删被脚本引用的模板 → 弹 referrer 警告)。之后无在飞项目, 候选(无紧迫): 脚本 SubgraphID 容错(运行时报错附现有子图列表 / 编辑期校验字面 SubgraphID 存在性, 2026-06-12 提出未拍板); 搜索/大复合 modal 是否收进 BaseModal; idea 池(cv-perception · editor-footgun · misc-tools); oxlint 预存 16 错清债; residue 28 处 HUD/Launcher 硬编码中文(misc-tools-backlog 在册); StateCycleSmoke 本机预存红排查(build.md 在册, mock 模板没命中)。

## 待复核

- `internal/services/script/binding.go` 仍在装 `vars.get/set/inc` 对象, 但 script-system.md 记 2026-06-11 已拍板删 vars.\*(当时只删了前端补全/糖单源)。按二号铁律疑似删漏 — 待用户确认: 删后端这段还是文档改口。(2026-06-12 做转脚本时撞见)

## 待验证

- ⚠ [archive/specs/2026-06-12-subgraph-to-script.md](archive/specs/2026-06-12-subgraph-to-script.md) — 真机: 右键一个录制类线性子图「转为脚本」→ 预览 → 插入为 Script 节点 → 跑一轮行为与原子图一致; 再转一个含 Loop 的子图, 确认弹人话拒转清单。
- ⚠ [archive/specs/2026-06-11-script-template-dep-extraction.md](archive/specs/2026-06-11-script-template-dep-extraction.md) — 库里删一个被某脚本引用的模板,确认弹「被引用」referrer 警告 + gcBlobs 不回收其 blob(单测已覆盖提取+扫描器接线,差集成/真机这一验)。

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。本节只放**两边都装不下的活上下文**, 现在没有。(2026-06-11 清版: 原"加节点路线图存档指针"等五条已分别沉淀进 docs / 待复核 / build.md, 不再重复。)
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **39** = 28(misc-tools-backlog) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 16 错(oxlint 1.64 新规则, 散在 10 个未涉编辑器的文件)。跑全套测试/检查时按此判红。
