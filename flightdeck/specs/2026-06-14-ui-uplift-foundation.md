---
status: active
summary: "UI 升级第一轮：设计系统地基(v3 克制精致 token + 字体角色) + 共用组件(AppCard/AlertBox/EmptyState/StatusPill/SectionHeader/IconBadge/ListRow) + 主程序门面屏 restyle(容器列表/计划/关于/LogPanel/侧栏) + 工具库常量收敛；编辑器画布与重设计在 Spec B"
last_updated: 2026-06-14
---

# UI 升级第一轮 · 设计系统地基 + 主程序门面

## 背景 / 目标

阶段目标（用户定，2026-06-14）：**样式统一 + 工具库 + 样式升级**。现状底子不差（NuxtUI v4 语义 token 用得很彻底，459 处语义色 vs 32 处散写字面色；已有 BaseModal/HudShell/ConfirmDialog 共用壳、format/color/ids/constants/useInsertPoint 工具），短板集中在：① 卡片/面板/alert/空状态各写各的（散落 8-12 处）；② 管理屏（容器列表/计划）平铺、空状态糙、像 dev 工具不像商业品；③ 散落魔法值常量。

升级方向定为 **v3「克制精致」**：在现有暗色 + 翡翠绿（emerald）品牌上，扁平骨架为底，靠细边框定层次；浮层（阴影+高光）只在「真的悬浮于内容之上」处轻用；渐变全场只留两处。

**关键策略**：样式统一与样式升级**一遍做完**——抽共用组件时一次性把它们按 v3 写定，每个屏迁到共用组件即同时升级，零重复返工。token 驱动，一处改全局变。

## 范围

**本 Spec A（in）**：设计系统地基（token + 字体）· 共用组件层 · 主程序/管理屏门面 restyle（容器列表 / 计划 / 关于 / LogPanel / 主壳轻触）· 工具库常量收敛。

**Spec B（out，第二轮单独 brainstorm + mockup）**：容器编辑器 **壳 + 面板 + 全部 modal**（Toolbar/Breadcrumb/Inspector/左 rail + clip/library/template explorer + 子图 props + 帮助 modal）的 restyle + **布局/UX 重设计**。边界已拍定：**vue-flow 画布、节点框、连线、pin 只继承本 Spec 的 token，不做画布级重设计**（避开 incident 高发的功能核心）。Spec B 复用本 Spec 的设计语言与共用组件，不重造地基。

---

## ① 设计语言 v3（token 规范）

> 表达原则：尽量建立在 NuxtUI 既有 `--ui-*` 语义 token 上，新的浮层/阴影/高光配方派生自 `--ui-bg`（跟现有 `bg-sunken` = `color-mix(in oklab, var(--ui-bg) 45%, black)` 同思路，**禁止散写 zinc-950 字面值**，遵 style.css 既有铁律）。下方 hex 是 mockup 实测值，impl 时换算成派生 token。

### 表面分层（B 的核心「浮层」，但克制）

| 层 | 用途 | 配方（派生自 --ui-bg） |
|---|---|---|
| `sunken` | 日志区 / 时间轴轨道 | 已有 `bg-sunken`（--ui-bg 再深一档）+ inset 顶部内阴影 |
| `base` | app 背景 / 画布 | `--ui-bg` |
| `raised` | 卡片 / 面板 / hover 行 | 比 base 略亮（≈ +3% white）+ **顶部 1px 白高光** `inset 0 1px 0 rgba(255,255,255,.04)` + **极淡顶光渐变**（top 略亮 → bottom，几乎不可见）+ **柔投影** `0 2px 6px rgba(0,0,0,.35)`，上边框比其余边框亮一档 |
| `overlay` | 菜单 / 模态 / popover | 比 raised 再亮一档 + **更明显投影** `0 10px 30px rgba(0,0,0,.5)` + 顶部高光 |

要点：暗底上 drop shadow 本就弱 → 浮起感靠「略亮填充 + 顶部白高光 + 柔投影」三件套，不靠单一重阴影。日常卡片几乎不投影，只有 hover / 菜单 / 模态才真浮起。

### 阴影 / 高光 token

- `--shadow-raised: 0 2px 6px rgba(0,0,0,.35)`
- `--shadow-overlay: 0 10px 30px rgba(0,0,0,.5)`
- 顶部高光统一 `inset 0 1px 0 rgba(255,255,255,.04)`（overlay 可 .05）

### 强调色 & 渐变纪律（翡翠绿 emerald = primary，不变）

- **只点睛**：绿色仅出现在「激活态 / 主按钮 / 状态 / 数值」，别处不滥用。
- 淡底 tint：`rgba(16,185,129,.12)`，文字 `emerald-400 (#34d399)`。
- **渐变全场只两处**：
  1. **主按钮**唯一显眼渐变 `linear-gradient(180deg,#11c08a,#0a9d6f)` + 极柔阴影 `0 1px 3px rgba(16,185,129,.25)` + 顶部高光——做出唯一的「主操作」分量感。
  2. **卡片 raised** 那一丝几乎看不见的顶光渐变。
- 其余（次按钮 / 药丸 / 提示框 / 导航 / 空状态）**全扁平**。**禁止**：渐变文字、角落光晕、辉光、glassmorphism、渐变药丸——渐变用滥画面会乱。
- 激活导航：左侧 `2px` 绿条（`inset 2px 0 0 #10b981`）+ 淡绿底 `rgba(16,185,129,.1)`（扁平无渐变）。

### 圆角三档

| 档 | 值 | 用在 |
|---|---|---|
| 控件 | `rounded-md` (6px) | 按钮 / 输入 / 小徽框 / 菜单项 |
| 卡片·面板·模态 | `rounded-xl` (12px) | AppCard / modal / 大面板 |
| 药丸·头像 | `rounded-full` | StatusPill / 圆形 avatar |

### 间距三档

| 档 | 值 | 用在 |
|---|---|---|
| 控件 | `px-2.5 py-1.5` | 工具条按钮 / 紧凑控件 |
| 面板 | `px-4 py-3` | modal header/body / 列表行 |
| 分区 | `px-6 py-5` | 卡片 / section 容器 |

### 字体角色系统（用户专门要求，立规矩）

app 已打包 `Inter Variable` + `JetBrains Mono Variable`（fontsource 本地打包，**不依赖系统安装**，离线稳 —— 这是有意的，靠系统字体会在别人机器糊掉）。

| 字族 | 用在哪 | 具体 |
|---|---|---|
| **Sans · Inter** | 所有「人读的话」 | UI / 正文 / 标题 / 标签 / 说明 / 按钮文字 |
| **Mono · JetBrains** | 所有「机器读的 / 要精确对齐的」 | 脚本/代码编辑器(CodeInput/ExprInput)、表达式、**坐标/数值/时间戳/耗时**、ID/GUID、日志正文 |

- 会跳动的数值（计时器 / 坐标 / metrics）一律加 `tabular-nums`，防左右抖。
- 脚本 mono = JetBrains Mono（专为代码设计、连字优化、已打包）。**待用户确认**：若用户更想要别款 mono（Cascadia / Fira Code），换打包；无特别偏好按 JetBrains 推进（不阻塞）。

### 字阶

| 角色 | 值 |
|---|---|
| 页标题 | `text-lg` 18px / 600 |
| 分区标题 | `text-xs` 11px / 600 / 大写 / `tracking` |
| 正文 | `text-sm` 14px / 400-500 |
| 元信息 | `text-xs` 12px / muted |
| 数值·代码 | mono 13px / `tabular-nums` |

### 状态色

| 状态 | 色 |
|---|---|
| 在线 / 成功 | emerald |
| 就绪 / 中性 / idle | slate (148,163,184) |
| 暂停 / 警告 | amber (245,158,11) |
| 失败 / 错误 | red (239,68,68) |

药丸：`.13` 底 + `.24` 边框 + 亮文字；提示框：`.07` 底 + `.2` 边框。

---

## ② 共用组件清单

**原则**：只抽已被现状摸查证明**重复 ≥3 处**的；每个组件按 v3 写一遍；放 `components/common/`。impl 时每个组件落地后，逐一迁移消费方（迁移即升级），并删掉被替换的散写样式（二号铁律：一次切干净，不留旧样式）。

| 组件 | 用途 | props / API（初稿，plan 阶段细化） | 替换现状（重复处） |
|---|---|---|---|
| **`AppCard`** | 卡片外壳：raised 表面（顶光+高光+柔投影），圆角 12，padding 档可选 | `padding?: 'panel'\|'section'`，默认 section；`hover?: boolean`（hover 浮起）；slot 默认 | StatCard / ContainersTab 卡 / ComingSoonCard / AboutView 5 卡 + 概念子卡 / inspector 面板卡（**8-12 处**） |
| **`AlertBox`** | 提示框 | `type: 'info'\|'success'\|'warning'\|'error'`；`title`；slot 说明；可选 `icon` 覆盖 | NodeInspector / ContainerHelpModal / SwitchInspector 的 info/warning 散盒（**6-8 处**） |
| **`EmptyState`** | 空状态 | `icon`；`title`；`description`；slot `action`（CTA） | ContainersView 在线 tab / ContainersTab / SchedulesListPanel / LibraryDetailPanel（**3-4 处**） |
| **`StatusPill`** | 状态药丸，配色自动 | `status: 'online'\|'ready'\|'paused'\|'failed'` 或 `color` + slot；可选前导 dot | 容器卡 / 计划表 enabled 徽章 / 散落状态标记 |
| **`SectionHeader`** | 分区头：icon + 标题 + 计数 + 动作槽，统一内距/下边框 | `icon?`；`title`；`count?`；slot `actions` | SidebarSection / LogPanel header / inspector 各分区头（**5 处**） |
| **`IconBadge`** | 图标徽框：小 raised 方框，尺寸+配色 | `icon`；`size?`；`color?`；`shape?: 'square'\|'round'` | LibraryDetailPanel / NodeInspector / AboutView avatar / 列表行图标（**3+ 处**） |
| **`ListRow`** | 列表行：hover 轻浮起，图标 + 主 + 副 + 尾槽 | slots `icon`/`default`/`trailing`；`active?` | VarsPanel 行 / 各管理列表（**~7 处**；AppSidebar nav 已精致，按情况套不强求） |

**按钮不另造组件**：用 NuxtUI `UButton`，把「主按钮克制渐变 + 高光」定到主题层（primary solid variant），次/幽灵/危险走 UButton 自带 variant。
> ⚠️ NuxtUI v4 UButton 主题覆盖的**具体写法**（app.config `ui.button` slots / `:ui` class / 全局 CSS 哪条路）**留 plan 阶段验源码再定**，本 spec 只锁视觉效果。

**全局 token 层**（非组件，补进 `frontend/src/style.css`）：`--surface-raised` / `--shadow-raised` / `--shadow-overlay` / 顶部高光，配 `@utility raised-surface` / `@utility overlay-surface`（给不值得包组件的零散场景直接用，跟现有 `@utility bg-sunken` 同模式）；`.font-mono` 用法约定。

**可选 / 低优先**：`CardGrid` 自动网格——现状 3 处用 `grid-cols-1 md:grid-cols-2 lg:grid-cols-3`，桌面 app 固定列更合适。先列着，不一定做。

---

## ③ 逐屏升级方案（按门面冲击力排序）

### P1 · 容器列表（ContainersView + ContainersTab）— 最高优先

- 卡片 `rounded-xl bg-default border p-4`（ContainersTab.vue:57）→ **`AppCard`**（v3 轻浮起 + hover 浮起）。
- 节点数 / 热键（ContainersTab.vue:80-87）→ mono。
- 加 **`StatusPill`**：运行中 = 在线绿，空闲 = 就绪（现状仅靠 run/stop 按钮切换体现运行态，卡片无状态标识）。
- 网格 `grid-cols-1 md:grid-cols-2 lg:grid-cols-3`（ContainersTab.vue:53）→ 固定列（桌面按宽 2-3 列，弃 md/lg 任意断点）。
- 空状态 `rounded-xl bg-default/50 ... border-dashed`（ContainersTab.vue:42-51）→ **`EmptyState`** + 「新建第一个容器」CTA。
- header 间距规范；create 按钮走主按钮克制渐变。
- ContainersView 「在线」tab 占位（ContainersView.vue:9-13）→ **`EmptyState`** ComingSoon 变体。
- 顺手（不强求）：容器删除现走**局部 UModal**（ContainersTab.vue:124-148），可收进全局 `useConfirm`/`ConfirmDialog`。

### P2 · 计划（SchedulesView + ScheduleListPanel）

- 空状态（ScheduleListPanel.vue:3-12）→ **`EmptyState`**。
- 表格 enabled 徽章（ScheduleListPanel.vue:33-42）→ **`StatusPill`**。
- 触发 / 次数 / 上次时间 → mono + `tabular-nums`。
- 行 hover（`hover:bg-elevated/40`，ScheduleListPanel.vue:25）规范化。
- **表格结构保留**（数据密集，表格合适，不强行改卡片）。

### P3 · 关于（AboutView）

- 5 个 section `rounded-xl bg-default border p-5/8` → **`AppCard`**。
- 概念子卡（AboutView.vue:23-34）→ **`AppCard`** 小号变体。
- 主图 avatar `size-20 rounded-full bg-elevated`（AboutView.vue:6-8）→ **`IconBadge`** 大圆变体。
- 版本号 / 技术栈值 → mono。

### P4 · LogPanel + 主壳轻触

- **LogPanel**：只统一 header / filter 按钮的 token、微调行距。**轻改**，body 已 `font-mono` + `tabular-nums`。
- **AppSidebar**：active 态（`activeClass = 'bg-elevated/60 ...'`，AppSidebar.vue:102）对齐 v3——左 2px 绿条已是 `bg-primary` ✓；active 底色 `bg-elevated/60` → 淡绿 tint `rgba(16,185,129,.1)`。**轻改**。
- **AppStatusBar**：对齐 token，基本不动。

### ⚠️ 读源码挖出的硬约束（统一散写字面色时**必须绕开**）

1. **概念分类色**（AboutView.vue:150-153 的 `text-fuchsia-300`/`text-emerald-300`/`text-amber-300`）**＋日志流身份色**（LogPanel.vue:177,186-189 的 cyan/violet SYS/CTR、node/dump/log 色）——源码明确注释「这是**身份识别色、非状态语义色**，配色统一时别动」。统一散写字面色清单时**跳过这些**。
2. **主壳（侧栏/标题栏/状态栏）已较精致**——轻触别推倒重做（避险 + 不过度工程）。
3. **设置屏（SettingsView 系列）**本轮未逐个读源码；方向「用 `AppCard`/`SectionHeader` 给设置项分组加呼吸」，**细节留 plan 阶段读源码再定**，不脑补。

---

## ④ 工具库 / 常量抽取

**现状（已单一来源，不动）**：`format.ts` / `color.ts` / `composables/containerEditor/ids.ts` / `composables/containerEditor/constants.ts` / `useInsertPoint.ts`。本轮只补缺口。

**A. 散落魔法值 → 收进 `constants.ts`**（plan 阶段逐个 grep 全调用点再改 = consumer-audit，对应踩过的坑）：

| 值 | 现状（各写各的） | 收成 |
|---|---|---|
| 子图入口坐标 `80,160` | `elkGraph.ts` + `useContainerDraft.ts` 两处 | `SUBGRAPH_ENTRY_X/Y` |
| 子图出口坐标 `420,160` | 同上两处 | `SUBGRAPH_OUTPUT_X/Y` |
| CommentBox 尺寸 `320×160` | `elkGraph.ts` | `COMMENT_BOX_W/H` |
| save flash `1600ms` | `useEditorSave.ts` | `SAVE_FLASH_MS` |
| toast 时长 `2500ms` | `App.vue` | `TOAST_DURATION_MS` |
| 左 pane 宽 `200–480` | `ContainerEditorView.vue` | `PANE_MIN/MAX_W` |

> 上述 file:line 来自现状摸查，plan/impl 阶段每项**先 grep 确认全部调用点**再动（防 consumer-audit gap）。

**B. 待定是否抽的 helper**（YAGNI 闸门：plan 阶段 grep，**≥3 处真重复才抽，没有不造**）：clamp/round、clipboard 复制、防抖节流（现用 VueUse，多半不自造）、deep clone/compare。

**C. 设计 token 工具**（CSS 侧，配合②，登记归属）：见 ② 末「全局 token 层」。

**原则**：只补已证明重复的，不预造没人用的（YAGNI + 二号铁律）。

---

## 实现策略 / 推进顺序

1. **地基**：style.css 补 v3 表面/阴影/高光 token + `@utility`；定 NuxtUI 主按钮渐变主题（验源码）；`constants.ts` 收散落魔法值（每项 grep consumer-audit）。
2. **共用组件层**：`AppCard` → `AlertBox` → `EmptyState` → `StatusPill` → `SectionHeader` → `IconBadge` → `ListRow`，每个按 v3 写一遍 + 单元/视觉自检。
3. **逐屏迁移**（迁移即升级，每屏迁完删旧散写样式）：P1 容器列表 → P2 计划 → P3 关于。
4. **门面收尾**：P4 LogPanel / AppSidebar / AppStatusBar 轻触。

每步可独立真机验、独立可回滚，贴「先扫面列清单、再按 footgun 优先级动手」节奏。

## 验证方式

- 每个组件 + 每屏：headless Playwright MCP 离屏渲染自检（[headless-ui-verify](../checklists/headless-ui-verify.md)）看渲染对不对。
- 关键屏：真机过（项目铁律）。
- 全套 build/lint/test 按 [build](../checklists/build.md) 已知预存失败判红（runtime 缺 fish fixture / i18n residue 39 含 11 有意误报 / pnpm lint 预存 18）。
- i18n：新增文案走 zh/en 双语，注意 vue-i18n message-compiler 陷阱（[vue-i18n-traps](../checklists/vue-i18n-message-compiler-traps.md)）。

## 不做的 / YAGNI 边界

- 不动 vue-flow 画布 / 节点框 / 连线 / pin（Spec B 才碰，且只继承不重设计）。
- 不推倒重做主壳（轻触）。
- 不动概念分类色 / 日志流身份色（身份识别色，非状态语义）。
- 不预造没有 ≥3 消费方的组件 / helper。
- 不写兼容/fallback/shim（二号铁律：未发布、无外部用户，一次切干净）。

## Spec B 预告（编辑器，下一轮 brainstorm + mockup）

容器编辑器 **壳 + 面板 + 全部 modal** 的 restyle + **布局/UX 重设计**：Toolbar / Breadcrumb / Inspector / 左 rail；clip/library/template explorer + detail panel；子图 props panel；帮助 modal；NodeInspector。复用本 Spec 设计语言 + 共用组件。**画布/节点框/连线/pin 不重设计，只继承 token**。需单独配 mockup（布局/UX 涉及空间编排）。
