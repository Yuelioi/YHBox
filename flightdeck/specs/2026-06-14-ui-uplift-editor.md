---
status: active
summary: "UI 升级第二轮 (Spec B): 容器编辑器 restyle + 布局/UX 重设计。方向 = A 双栏停靠 + Inspector 选中才出 (全量重做, 解 4 大痛点: modal 盖画布 / 三层边栏挤画布 / Inspector 扁平 / Toolbar 主次乱)。① 面板 IA: 左侧自适应宽度停靠区收纳 节点库·变量·Snippets·资产浏览器 (窄列表↔宽网格自适应, 始终挤画布不盖画布), 小弹窗 restyle 留 modal; ② Toolbar 三区分层 (导航 / 主操作 hero / 文档+⋯收纳) + 底部问题条; ③ Inspector 三态收起规则 + SectionHeader 分组。复用 Spec A 设计语言 + 共用组件 (含 AlertBox/SectionHeader/ListRow)。边界: vue-flow 画布·节点框·连线·pin·画布右键 只继承 token, 不重设计。"
last_updated: 2026-06-14
---

# UI 升级第二轮 · 容器编辑器 restyle + 布局/UX 重设计 (Spec B)

> brainstorm 可视化稿: `.superpowers/brainstorm/956-1781446969/content/`(editor-scope / layout-directions / ia / toolbar / inspector, gitignored)。承接 Spec A(已 done 归档: `archive/specs/2026-06-14-ui-uplift-foundation.md`)。

## 背景 / 目标

Spec A 把设计系统地基 + 7 共用组件 + **主程序门面 4 屏** restyle 落地了, 设计语言常驻 [ui.md](../checklists/ui.md) §表面分层。剩下编辑器这块没动 —— 它是全 app 最复杂的屏, 也是用户每天搭流程待的地方, 现在是「能用但散」。

用户拍定野心 = **全量重做 (restyle + 布局/UX 重设计一遍做完)**, 要解的 4 个痛点 (用户全选):

1. **大 modal 太多盖画布** — 节点库 / 模板 / 库 / Clip 浏览器都是近全屏 5xl modal, 打开盖画布、丢上下文, 来回切烦。
2. **三层边栏挤画布** — 左 rail + 变量/Snippets 抽屉 + 右 Inspector 同时占地, 画布被挤窄。
3. **Inspector 扁平** — 节点属性是一连串等权重 flat section, 没有「基础/输入/输出」分组层级。
4. **Toolbar 主次乱** — 约 16 个动作平铺一排, 录制/运行等主操作和 设置/吸附 等杂项不分。

**统一根问题**: 编辑器 chrome 太散 (3 条边栏 + ~12 个 modal + 密 toolbar)。**统一解法**: 一套更克制、面板化的信息架构, 把画布让出来、层级理清, 全部沿用 Spec A 的「黑底 + 顶光浮起」克制语言。

## 范围

**本 Spec B (in)**:

- **编辑器壳布局重排** (`ContainerEditorView.vue` 的 chrome 框架部分)
- **左侧停靠区** (新建; 收纳 节点库 / 变量 / Snippets / 资产浏览器)
- **顶部 Toolbar 重组** (`ContainerEditorToolbar.vue`)
- **右侧 Inspector** 三态 + 分组 (`ContainerEditorInspector.vue` / `NodeInspector.vue` / `SubgraphPropsPanel.vue`)
- **底部「问题」条** (`ValidationErrorPanel.vue` 浮层 → 停靠底条)
- **全部浮层 modal restyle** 成纯黑 BaseModal / 克制浮层 (清单见 §5)

**边界 (out, 已拍定)**: **vue-flow 画布、节点框 (`ContainerFlowNode.vue`)、连线、pin、画布/节点/边/pin 右键菜单 (`menus/*`)、inline pin 输入件 (`inline/*`)** —— **只继承 Spec A 的 token, 不做画布级重设计**。避开 incident 高发的功能核心 (节点渲染/连线/pin 接线/keep-alive/前台容器指针那一堆), 降风险。这些组件该跟着 token 变深变克制就行, 不动结构和交互。

---

## 设计决策 (brainstorm 2026-06-14 逐节拍定)

### 0. 大方向 — A 双栏停靠 (Dock) + Inspector 选中才出

三方向 (Dock / Canvas-first / Command-first) 里选 **A**: 大部分浮层收进左侧停靠面板, 左 rail+面板、右 Inspector, 最多两侧。理由: 这是给人搭自动化流程的工具, **可发现性比极简更重要**。借 B 的一个点: **Inspector 没选中节点时自动收起**给画布让位 (细化规则见 §3)。

### 1. 面板信息架构 — 左侧"自适应宽度"停靠区

核心: **一个停靠区, 宽度随内容自适应, 但始终"挤画布"而非"盖画布"**。这一招同时解痛点 1 (modal 盖画布) + 2 (边栏挤画布)。

| 内容 | 模式 | 来源 |
|---|---|---|
| 节点库 (加节点) | 窄 ~300px (列表+搜索) | 原 `NodeExplorerModal` 5xl → 停靠 |
| 变量 | 窄 | 原 `VarsPanel` 抽屉 |
| Snippets | 窄 | 原 `SnippetsPanel` 抽屉 |
| 资产 (模板 / 库 / Clip 三合一带 tab) | **宽 ~600px (缩略图网格)** | 原 `TemplateExplorerModal` / `LibraryExplorerModal` / `ClipExplorerModal` 3 个 modal → 停靠 |

- rail 切换 (节点库/变量/Snippets/资产 4 个图标), 选中项滑出停靠面板, 再点同图标收起。停靠区可拖宽。
- 资产 tab 时停靠区**自动加宽**到网格够用 (~600px), 但仍是 docked = 挤画布、不盖画布。
- **资产浏览器从节点字段唤起** (如 TemplatePickerField「选模板」) 时, 也定位到这个停靠区的资产 tab (而非另弹大 modal), 保持「就近」。

### 2. 顶部 Toolbar — 三区分层 + ⋯ 收纳

约 16 个平铺动作 → 四层归位:

- **左 · 导航上下文**: 返回 · 面包屑 (主图›子图) · 撤销/重做
- **中 · 主操作 (突出)**: **录制 · 运行** (运行 = 绿渐变 hero, 运行时变停止 + 状态指示)
- **右 · 文档 + 收纳**: 保存 (dirty 黄点) · 校验 · 自动布局 · **⋯ 菜单** (= 重载 / 吸附开关 / 连线样式 / 设置 / 帮助)
- 折叠 Inspector 的 ⊟ **挪出 toolbar** → 右边缘 / Inspector 自己的头部。

### 3. 右侧 Inspector — 三态收起 + 分组

**三态收起规则** (它有三态, 不能一刀切收):

1. **选中节点** → 展开节点属性面板 (主用途, 见下分组)。
2. **子图内 · 空选** → **保留** `SubgraphPropsPanel` (这正是"你在编辑的东西", 不收)。
3. **根图 · 空选** → **自动收起** → 画布全宽。容器概览 (节点/变量/子图数 · 热键) 挪到面包屑旁的小信息位; 「快捷开始」提示只在 **画布空** 时作为画布空态显示, 不再常占一栏。

**节点属性面板 — 扁平 → 分组**: 用 Spec A 已有但门面屏没消费的 **SectionHeader** 组件分 **「基础 / 输入 / 输出」** 三组 (基础 = 标签 + 打印日志; 输入 = 各 pin 配置件; 输出 = 出口 pin 速览)。header 保留 (图标 + 名 + kind·id + ?说明 / 复制菜单 / 删除), 整体密度收紧、对齐统一。

### 4. 底部「问题」条

`ValidationErrorPanel` 从浮层 modal → **底部可折叠"问题"条** (像 VS Code Problems): 平时收起只显错误数徽章; 展开列出校验错误; 点条目跳到对应节点。不盖画布。

### 5. 浮层 modal restyle 清单 (留 modal, 不进停靠区)

全部 restyle 成 ui.md 定义的 **纯黑平铺 BaseModal** (或快捷浮层):

- **小弹窗 → BaseModal**: 容器设置 (`ContainerSettingsModal`) · 帮助 (`ContainerHelpModal`) · 引用查找 (`FindReferencesModal`) · 子图脚本预览 (`SubgraphScriptPreviewModal`) · 新建变量 (`NewVarModal`) · 提升为变量 (`PromoteToVarModal`)。其中结构特殊的搜索/大复合按 ui.md 例外评估 (`NodeSearchModal` / `CommandPalette` 结构特殊, 维持自有壳, 只对齐 token)。
- **快捷浮层维持**: 命令面板 (`CommandPalette`) · Tab 搜节点 (`NodeSearchModal`) —— 行为不变, 只继承新 token。
- 详情面板 (`TemplateDetailPanel` / `LibraryDetailPanel` / `ClipDetailPanel`) 跟随资产停靠区一起重排。

---

## 复用 Spec A (不重造地基)

- **设计语言**: 黑底 + 顶部渐隐高光浮起 (`raised-surface` / `overlay-surface` / `bg-sunken`)、base neutral-950、语义 token —— 全照 [ui.md](../checklists/ui.md) §表面分层。
- **共用组件**: `AppCard` / `StatusPill` / `EmptyState` / `IconBadge` 继续用; **本轮首次消费 `SectionHeader` (Inspector 分组) · `AlertBox` (校验/警告类提示) · `ListRow` (停靠面板列表项)** —— Spec A 留给编辑器屏的这 3 个正好用上。
- **modal 壳**: 走 `components/common/BaseModal.vue` 纯黑平铺。

## 边界与风险

- **不碰画布功能核心**: vue-flow 节点/连线/pin/右键/inline 输入件只跟 token, 不动结构。改 chrome 时注意别误伤画布层的 keep-alive / 前台容器全局指针 / viewport 相机 (见 incidents: keepalive-singleton / vueflow-single-camera / deep-watch-swapping)。
- **停靠区取代 modal 是结构性改动**: 节点库/资产浏览器从 modal 调用改成停靠面板, 要理清原 modal 的 open/close 状态、从节点字段唤起的路径 (TemplatePickerField)、以及 import/拖入容器 的数据流 (见 incidents: import-bypasses-container-store / draft-subgraphs-phantom)。
- **Inspector 自动收起** 不能误判三态 (尤其子图内空选要保留), 别把正在编辑的子图属性收没了。

## 验证

- 每屏改完跑离屏视觉自检 ([headless-ui-verify](../checklists/headless-ui-verify.md)): 停靠区窄/宽模式、Toolbar 三区、Inspector 三态、底部问题条、各 restyle modal。
- 全套: vitest / typecheck / build 绿 (预存失败按 cockpit「已知预存失败」判)。
- 收尾 **真机 smoke**: task dev 起 app, 进编辑器把 加节点 / 切资产 / 选节点编属性 / 折叠 inspector / 跑校验 / 开各 modal 走一遍, 像商业品、无错位/盖画布/丢色, 且 **画布交互 (拖节点/连线/缩放/子图进出) 不回归**。

## 实现分期 (plan 切分建议, 写 plan 时定)

体量大, 建议分 Part 拆 plan (仿 Spec A 的 Part 1/2):

- **Part 1 · 左侧停靠区** (新壳 + 节点库/变量/Snippets/资产 自适应宽度收纳, 替掉对应 modal/抽屉)
- **Part 2 · Toolbar 重组 + 底部问题条 + Inspector 三态/分组**
- **Part 3 · 剩余浮层 modal restyle 清单**

(顺序可调; Part 1 是结构大头、风险集中, 先做先验。)
