# Cockpit — YHFish

**Last updated**: 2026-06-12 by 月离 (编辑器外壳全面收进左 VS Code 活动栏式 rail: 变量/Snippets 停靠 drawer + 节点库/子图库 modal 启动器 + 底部录制三态; toolbar 中间清空、撤销重做移面包屑后; 面包屑去冗余根可点。参考 UE/Figma/VS Code。前置: 跨容器串根治 + 未保存切容器三态守卫。)
**Active focus**: **编辑器外壳重排 + 跨容器状态串根治**。左栏改 rail+drawer(参考 UE/Figma, 非 modal 以保拖出画布)、面包屑去冗余; 跨容器修复(tplStore 前台指针 + 未保存切容器三态守卫)。子图转脚本用户真机已过。本批 UI/跨容器改动待真机复验。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

<!-- AUTO:inprogress -->

<!-- /AUTO -->

## 下一步

清真机验证债(见 待验证: 跨容器修复三步走一遍; 库里删被脚本引用的模板 → 弹 referrer 警告)。子图转脚本真机用户已过, 销。之后无在飞项目, 候选(无紧迫): WaitTemplate「先连边再建节点」失败留孤儿边的原子性硬化(本次根因修了暂不触发, 真机再现再修); 复发#5 promotion 候选(前台容器全局指针 onMounted+onActivated 规则升 checklist); 脚本 SubgraphID 容错(运行时报错附现有子图列表 / 编辑期校验字面 SubgraphID 存在性, 2026-06-12 提出未拍板); 搜索/大复合 modal 是否收进 BaseModal; idea 池(cv-perception · editor-footgun · misc-tools); oxlint 预存 16 错清债; residue 28 处 HUD/Launcher 硬编码中文(misc-tools-backlog 在册); StateCycleSmoke 本机预存红排查(build.md 在册, mock 模板没命中)。

## 待复核

- 无。(vars.\* 删漏 2026-06-12 用户拍板补删, 已销。)

## 待验证

- ⚠ 编辑器外壳重排真机看一眼(纯布局无单测): 左侧细图标栏共 **4 顶 + 1 底**: 变量/Snippets 点开停靠 drawer(再点收回、互斥), 节点库/子图库 点开各自 5xl modal, 底部录制(空闲下拉精准/简易、倒计时点立即、录制中红脉冲点停止); toolbar 中间清空、撤销/重做在面包屑后。默认只剩细栏画布全宽; 变量在左属性在右; Snippets 仍能拖到画布; 面包屑进子图后根「容器名」+每段可点(根回主图)、只剩最左一个返回箭头; 加节点(Tab/右键/rail 节点库)正常; 录制三态(尤其录制中红点/停止/目标 tooltip)对。
- ⚠ 跨容器修复真机走一遍(无 spec 单测能覆盖路由/keep-alive): ① A 容器改点东西**不保存**→ 切到 B 容器, 应弹保存/丢弃/取消三态(不再裸切); 选「丢弃」切走再切回 A, A 应是磁盘干净态。② B 容器新建 WaitTemplate 截图, 定位的应是 **B 自己的 WindowTarget**(不再报"没有异环窗口"/不留孤儿边)。③ 转不了的子图拒转清单, 同类节点应「×N」合并一行不再重复堆。
- ⚠ [archive/specs/2026-06-11-script-template-dep-extraction.md](archive/specs/2026-06-11-script-template-dep-extraction.md) — 库里删一个被某脚本引用的模板,确认弹「被引用」referrer 警告 + gcBlobs 不回收其 blob(单测已覆盖提取+扫描器接线,差集成/真机这一验)。

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。本节只放**两边都装不下的活上下文**, 现在没有。(2026-06-11 清版: 原"加节点路线图存档指针"等五条已分别沉淀进 docs / 待复核 / build.md, 不再重复。)
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **39** = 28(misc-tools-backlog) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 16 错(oxlint 1.64 新规则, 散在 10 个未涉编辑器的文件)。跑全套测试/检查时按此判红。
