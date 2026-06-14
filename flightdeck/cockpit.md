# Cockpit — YHFish

**Last updated**: 2026-06-14 by 月离 (clip 库可插入容器→裸 PlayClip 节点(镜像子图库)、节点插入落点统一进 useInsertPoint:库/picker/录制落当前视口中心、拖放/快捷键/pin拖空白落指针处(原落世界原点附近),真机过)。
**Active focus**: 无活跃开发案 — 本轮: ① clip 库加「插入引用」→ 裸 PlayClip 节点(镜像子图库 onPick:详情按钮 + 双击行)。② 节点插入落点全部收进 [useInsertPoint](../../e:/projects/tools/YHFish/frontend/src/composables/containerEditor/useInsertPoint.ts):视口中心类 viewportCenterForNode(库/picker/录制/变量+/snippet单击,原散落世界原点)+ 指针位置类 screenPointToFlow(拖放/snippet快捷键/Tab picker/pin拖空白菜单);删掉 view 内重复的 recordingDropPoint 与散落 screenToFlowCoordinate。真机过、全绿一次过、无新 incident。下一步候选: 临时窗口抓取(EnumWindows 选窗截图)。**本仓内测期: 默认不 push, 用户说推才推**(commits.md 铁律)。

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
