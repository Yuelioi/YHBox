# Cockpit — YHFish

**Last updated**: 2026-06-13 by 月离 (节点属性面板: header 复制下拉 ID/JSON/脚本调用(带实参)+留住菜单原地闪✓ / 概述+示例收进?icon弹层 / 打印日志 hint 精简; dump 错误码 err[code]=(Throw 自填码可见)。新 incident: 成功反馈一律内联不弹 toast。真机过、全绿。)
**Active focus**: 无活跃开发案 — 本轮: 节点属性面板复制下拉(ID/JSON/脚本)+?说明弹层、dump 错误码 err[code]=,真机过全绿。新 incident: [成功反馈一律内联不弹 toast](incidents/2026-06-13-success-feedback-inline-not-toast.md)。下轮明确留: 临时窗口抓取(EnumWindows 选窗截图)。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

## 进行中

<!-- AUTO:inprogress -->

<!-- /AUTO -->

## 下一步

候选池任挑: **临时窗口抓取(EnumWindows 枚举 + 选窗截图 + 不依赖容器 WindowTarget)— 上轮明确留的下一块,模板截图/重拍现仍只走容器 WindowTarget**; NodeSearchModal / CommandPalette 收进 BaseModal(大复合已全收, 仅剩这俩); 复发#5 promotion 候选(前台容器全局指针 onMounted+onActivated 升 checklist); 脚本 SubgraphID 容错(未拍板); idea 池(cv-perception · editor-footgun · misc-tools); oxlint 预存 18 错; residue 28 处; runtime fixture 缺失红(build.md 在册)。

## 待复核

- 无。(variable-system 路径命中系误报 — 实改仅 Subgraph.Category 一行, 2026-06-12 验收轮确认, 已拍回 active。)

## 待验证

- ⚠未验证: [archive/specs/2026-06-13-editor-rail-resources.md](archive/specs/2026-06-13-editor-rail-resources.md) — 真机 smoke(`task dev`): ① rail 模板/clips 两栏管理(增删改 / 批量改分类标签 / 正逆序 / **输入建新分类标签**);② 主界面只剩本地/在线 tab、在线占位;③ 节点模板字段开新 modal 选取消、详情面板变体重拍/删档/新建截图;④ 子图库排序。过了销此条。
- ⚠未验证: 校验问题面板真机(`task dev`)— 保存失败→自动弹问题面板;点错误行「跳转」定位到出错节点(含跳进子图、选中+居中);LITERAL_TYPE_MISMATCH 干净数字串显「修复」按钮、点了改 number。(SettleMs / dump 日志 took / coerce-at-emit 已随对话真机过, 不在此列。)

## Hanging tasks

- [ ] 无阻塞待办。(原积压路由不变: 编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md); bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md); i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)。)

## 关键上下文

- 常驻技术知识一律在 [docs/](docs/INDEX.md) (表达式 → expression-system; 脚本 → script-system); 历史设计/执行记录在 `archive/specs|plans/` 按日期文件名自查。本节只放**两边都装不下的活上下文**, 现在没有。(2026-06-11 清版: 原"加节点路线图存档指针"等五条已分别沉淀进 docs / 待复核 / build.md, 不再重复。)
- 已知预存失败(非回归): runtime 缺 fish fixture([build.md](checklists/build.md)); i18n residue **39** = 28(misc-tools-backlog) + 11 处 editorTheme.ts 查找面板中文 phrases(**有意的 zh locale 文案映射, 扫描器误报类, 别去翻译**); `pnpm lint` 预存 **18** 错(oxlint 1.64 新规则; 2026-06-12 实测修正, 旧账 16 系漂移, 全部已实证存在于 HEAD 同款代码)。跑全套测试/检查时按此判红。
