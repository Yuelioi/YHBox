# Cockpit — YHFish

**Last updated**: 2026-06-18 by 月离 (用户验收「待验证全算过」—— CV 借鉴批整批收尾(三节点 + 颜色签名 Signature 双段结构化输入 + 出口/Data i18n); 教训沉淀 [add-node.md](checklists/add-node.md): 出口/Data 字段 i18n 别漏 · 结构化/变长输入配 Schema 别裸 JSON; 看板已清。**默认不 push**)。
**Active focus**: **无进行中 spec**。上一批 (CV 低依赖借鉴批: 颜色签名 / 二维码解码 / 模板全部命中 + 颜色签名 Signature 双段输入, 全纯 Go) 已**用户验收收尾**。下一步候选池见 ## 下一步。**默认不 push**。

## 进行中

<!-- AUTO:inprogress -->

<!-- /AUTO -->

## 下一步

- 无进行中 spec。候选池: 临时窗口抓取(EnumWindows 选窗截图); 复发#5 promotion(前台容器全局指针升 checklist); **cv-perception 池剩余** (ONNX/YOLO/OCR/blob 等大依赖路线, 本批只挑了低依赖三件, [cv-perception](specs/cv-perception-pool.md)); idea 池([editor-footgun](specs/editor-footgun-backlog.md) · [misc-tools](specs/misc-tools-backlog.md))。

## 待复核

- 无。

## 待验证

- 无。(2026-06-18 用户拍板「待验证全算过」: CV 借鉴批三节点真机 smoke · 颜色签名 Signature 双段输入 · 三节点出口/Data i18n · ClickTemplate 验证重试 —— 全部标记已过; 后续真机若冒小问题单独开 task。归档计划 verify 字段已标 verified。)

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- **知识在哪**: 常驻技术知识/系统架构 → [docs/](docs/INDEX.md); 加节点/写代码/UI 的规范与收尾清单 → [checklists/](checklists/INDEX.md)(按 when_to_read 路由); 反复踩的坑 → [incidents/](incidents/INDEX.md); 历史设计/执行记录 → `archive/specs|plans/` 按日期。
- **已知预存失败(非回归, 跑测试/检查时按此判红)**: runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **42**(misc-tools-backlog 未翻译 UI: SettingsLauncher/FloatingLauncherView/HudShell/IconPicker + 1 处 console.log; 另 11 处 editorTheme.ts 查找面板中文 = 有意 zh 映射, 别翻); `pnpm lint` 预存 **18** 错(oxlint 1.64 新规则, 已实证全在 HEAD)。
