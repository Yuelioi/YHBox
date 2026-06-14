# Cockpit — YHFish

**Last updated**: 2026-06-14 by 月离 (录制产物/子图节点改进 done、真机过: 精准录制→裸 PlayClip 节点(不包子图)·右键「彻底删除」连删底层子图/clip·子图节点显示子图名+字段「标签」→「名称」·简易录制子图间距 440(>maxWidth 360)不重叠)。
**Active focus**: 无活跃开发案 — 本轮: 录制产物/子图节点改进案 done 真机过(精准→裸PlayClip、右键彻底删除、子图节点显名/改名、简易间距440)。新 incident: [vue-flow 删除键无视修饰键](incidents/vue-flow-delete-key-code-ignores-modifiers.md)(带修饰键/需确认的删除别跟 vue-flow 抢键盘, 走右键菜单)。教训入 CLAUDE.md 工作风格: 先定根因、一处修干净、别打补丁兜底。下轮明确留: 临时窗口抓取(EnumWindows 选窗截图)。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

<!-- AUTO:inprogress -->

<!-- /AUTO -->

## 下一步

候选池任挑: **临时窗口抓取(EnumWindows 枚举 + 选窗截图 + 不依赖容器 WindowTarget)— 上轮明确留的下一块,模板截图/重拍现仍只走容器 WindowTarget**; NodeSearchModal / CommandPalette 收进 BaseModal(大复合已全收, 仅剩这俩); 复发#5 promotion 候选(前台容器全局指针 onMounted+onActivated 升 checklist); 脚本 SubgraphID 容错(未拍板); idea 池(cv-perception · editor-footgun · misc-tools); oxlint 预存 18 错; residue 28 处; runtime fixture 缺失红(build.md 在册)。

## 待复核

- 无。

## 待验证

- 无。(editor-rail / 校验问题面板 2026-06-12 真机过, 已销; 详见各 archive spec verified 字段。)

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。本节只放**两边都装不下的活上下文**, 现在没有。(2026-06-11 清版: 原"加节点路线图存档指针"等五条已分别沉淀进 docs / 待复核 / build.md, 不再重复。)
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **39** = 28(misc-tools-backlog) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 **18** 错(oxlint 1.64 新规则; 2026-06-12 实测修正, 旧账 16 系漂移, 全部已实证存在于 HEAD 同款代码)。跑全套测试/检查时按此判红。
