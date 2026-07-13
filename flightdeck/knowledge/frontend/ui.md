---
kind: checklist
summary: "前端 UI 基线 — 写/改 .vue 或 Tailwind class 前的 NuxtUI 用色/组件/反馈约定."
activation: action
read_when: "before writing / editing .vue components or Tailwind classes"
recheck_when: "改前端 UI 基线 (nuxtui 版本/约定 / Tailwind 配置 / 公共组件风格) 时"
---
# UI (NuxtUI 偏好) checklist
写/改 `.vue` 组件 / Tailwind class 时**前置**读这份.

### 配色从哪来 — 新组件用色决策树 (2026-06-11 配色统一后)

色源只有两类: **chrome 色 = NuxtUI semantic token; 功能识别色 = 集中 TS 调色板**. 按顺序问:

1. **灰阶/文字/边框?** → 下面 semantic token 表 (`bg-default` / `text-muted` / `border-default` …)
2. **状态语义 (警告/错误/成功/提示/选中激活)?** → `warning` / `error` / `success` / `info` / `primary` (透明度照常: `bg-warning/10`)
3. **要比主背景更深 (终端/轨道/凹陷面)?** → `bg-sunken` (style.css 自定 utility)
4. **多彩"身份/区分"色 (节点分类/用户可选色板/pin 类型/日志 TAG)?** → 从集中调色板取, **不自己发明色相**:
   - 节点分类 + 用户色板: `visualRegistry.ts` (PALETTE / GROUP_VISUAL; class + hex 双形态)
   - pin 类型色: `nodeRegistry/index.ts` TYPE_COLOR
   - 日志 TAG/流: `logFormat.ts` / LogPanel
5. **Tailwind class 够不着 (CodeMirror JS 主题 / scoped CSS / 内联 style / canvas 绘制)?** → 直接写 `var(--ui-bg)` / `var(--ui-primary)` 等 CSS 变量; 要透明度用 `color-mix(in oklab, var(--ui-primary) 20%, transparent)`. 范例: `editorTheme.ts` (chrome 全走 var, 语法高亮是有意的 VSCode Dark+ 内容色)

**定义位置** (改基调只动这几处): 三个色相钉在 `vite.config.ts` ui.colors (primary=emerald / neutral=zinc / warning=amber) → NuxtUI 生成 `--ui-*` 变量; `bg-sunken` 在 `style.css`; 画布景深色 (背景渐变/网点/minimap mask) 是有意的非 token, 集中在 ContainerEditorView 带注释.

### 表面分层 — 黑底 + 顶光浮起 (v3, 用户定 2026-06-14)

四档表面**全派生自 `--ui-bg`**, 一处 (`style.css`) 改全局变, 禁止散写 zinc 字面值:

1. **base** `bg-default` = `--ui-bg`。暗色基调钉在 `.dark { --ui-bg: var(--ui-color-neutral-950) }` —— 比 NuxtUI 默认 (neutral-900) **深一档**, 锚 v3 稿 `#09090b`; 整套灰阶 (muted/elevated/accented/border-accented) 跟着下沉一档。
2. **sunken** `bg-sunken` = 再往黑混 (终端日志 / 时间轴轨道)。
3. **raised** `raised-surface` (卡片 / 面板 / hover 行) + **overlay** `overlay-surface` (菜单 / 模态): **卡体保持纯黑** (`background-color: var(--ui-bg)`), **只在顶部给一道渐隐高光** (`linear-gradient(180deg, rgba(255,255,255,.05), transparent 30%)`) + 1px 白内高光 + 亮一档上边框 + 柔投影来「浮起」。**不整面提亮** (试过 mix 白提整面 → 发灰、描边读不出, 用户改回黑底顶光) —— 跟下面 BaseModal「纯黑平铺」同一条 philosophy。

### 设置/表单类页面卡片配方 (chrome 页 — 别用 Inspector 的扁平 SectionHeader)

主区的 chrome/设置/表单页 (设置各 tab / 计划编辑 / 关于) 统一用 **bordered card**, **别**用 NodeInspector 那种扁平 `SectionHeader` (那是侧栏窄面板的; 2026-06-15 在计划编辑器上栽过一次, 用户两次返工才纠到位)。配方 (照 `SettingsGeneral.vue`):

- 每组一张卡: `<section class="rounded-xl bg-default border border-default p-5 space-y-4">` (居中页加 `mx-auto max-w-3xl`; 铺满内容区的不加 max-w)
- 卡头: `<div class="flex items-center gap-2"><UIcon :name class="size-4 text-dimmed" /><h2 class="text-sm font-medium text-highlighted">标题</h2></div>`
- 行: label 左 / 控件右 (`flex items-center justify-between gap-6`), 短控件给宽度 (`w-32`/`w-48`); 行间 `<div class="border-t border-default/60" />`
- 关于页 2026-06-15 从 `AppCard` (raised-surface 顶光) 换成这套 border 卡, 跟设置统一。`AppCard` 留给列表项/浮起卡那类场景。

### 组件必须用 NuxtUI

有 NuxtUI 对应组件就用它. **禁止**裸 `<button>` / `<input>` / `<div role="dialog">` 自搭样式. 自研复合组件 (ConfirmDialog 等) 也只能由 NuxtUI 原子组合: `UButton` / `UInput` / `UModal` / `UPopover` / `UCheckbox` / `UTooltip` / `UTabs` / `UIcon` / `UTextarea` / `USelect` / `UDropdownMenu`.

**唯一例外**: 真无对应 NuxtUI 组件, 或其内部 dom 阻碍原生交互 (e.g. NodePalette 拖拽列表项 — `UButton` 包装层吞 drag 事件). 裸 HTML OK, 但视觉 token 仍走 `bg-elevated` / `text-toned` 等 semantic class.

### 通用 modal 用共享外壳 `components/common/BaseModal.vue`

普通 modal (确认 / 表单 / 面板 / 浏览器) **不要各自手写 `UModal` + `#content`**, 用 `BaseModal`:

- props: `open`(v-model) / `title` / `icon` / `iconColor`(primary|error|warning|success|info, 给删除/错误类上严重度色) / `size`(md..5xl) / `showClose`
- slots: 默认 = body、`#footer` = 按钮行、`#header-extra` = 标题右侧 (tabs / 步骤指示)
- 风格 = **纯黑平铺**: `bg-default` + header(border-b) + body 内容平铺(px-5 py-4) + footer(border-t)。层次靠**内容自身元素** (列表/卡片用 `bg-elevated` 自抬升), 外壳不套浅色面板/不上描边 (试过包裹面板/emerald 边, 用户定回纯黑)。
- 开关跟 `useDialogOpen()` 正交。`ConfirmDialog` (useConfirm Promise API) 也基于它。
- **例外不套 BaseModal**: 搜索面板 (CommandPalette / NodeSearch 结构特殊)、大复合 (TemplatePicker 资产浏览 / TemplateManager 预览)。
- **frameless 独立工具窗 (HUD) 不用这个** → 走 [standalone-window-style.md](../wails/standalone-window-style.md)。

### 永远 dark mode — 四件套一个不能少

1. **`<html class="dark">`** (index.html 静态写)
2. **`<div id="app" class="isolate dark">`** (`isolate` 创 stacking context; `dark` 双保险)
3. **`vite.config.ts` 只配 colors, 别动 slots**:

   ```ts
   ui({ ui: { colors: { primary: 'emerald', neutral: 'zinc' } } })
   ```

4. **`main.ts` 强制 `useDark().value = true`** — **最关键**. NuxtUI v4 plugin 内部用 @vueuse `useDark()`, 启动时读 localStorage + 系统 `prefers-color-scheme` 动态改 `<html>` class, 会把 index.html 的 `class="dark"` 覆盖. 不锁就在用户 light mode 下全失效.

   ```ts
   app.use(pinia).use(router).use(ui).use(i18n)
   useDark().value = true  // 锁 dark
   ```

### 只用 semantic token, 禁止裸 zinc/black/white

| 视觉            | 用                                                    |
| --------------- | ----------------------------------------------------- |
| 凹陷面 (比主背景更深: 终端日志区/时间轴轨道) | `bg-sunken` (自定 utility, style.css 从 --ui-bg 往黑混) |
| 主背景 (最深 semantic) | `bg-default`                                   |
| 卡片/弹层       | `bg-elevated`                                         |
| 分隔区/muted    | `bg-muted`                                            |
| 强调背景 (最高) | `bg-accented`                                         |
| 主文字 (最亮)   | `text-highlighted`                                    |
| 普通文字        | `text-default`                                        |
| 强调文字        | `text-toned`                                          |
| 弱化文字        | `text-muted`                                          |
| 提示文字 (最弱) | `text-dimmed`                                         |
| 边框            | `border-default` / `border-muted` / `border-accented` |

状态色: `bg-primary` / `bg-error` / `bg-warning` / `bg-success` / `bg-info`. **状态语义 (警告/错误/成功/提示/选中激活) 一律走这些, 不许直写 `text-amber-300` / `bg-rose-500/10` 这类色相**; 透明度照常加 (`bg-warning/10`)。直写色相只允许"功能识别色"且必须来自集中调色板: 节点分类 (visualRegistry PALETTE) / pin 类型 (TYPE_COLOR) / 日志 TAG·流 (logFormat / LogPanel) / 编辑器语法色 (editorTheme)。编辑器 chrome 色 → editorTheme.ts 只写 `var(--ui-*)`。

**禁用 list** (写新组件前 grep 自己):

- `bg-zinc-*` / `bg-black` / `bg-white`
- `text-zinc-*` / `text-white` / `text-black`
- `border-zinc-*` / `ring-zinc-*` / `divide-zinc-*`

**唯一例外**: 录制叠加层这类"完全黑" scrim. 模态遮罩 **不要** override — Modal / Slideover 自带 overlay slot, 信任默认.

### 输入框宽度: 默认撑满, 短输入直接加宽度类

style.css 全局把含 input/textarea 的 NuxtUI 包装层默认 `width: 100%` (表单不用处处写 w-full), 这条放在 `@layer components` — **组件上直接写 `w-36` / `max-w-*` 就能压过它** (utilities 层更后), 不用包 div。数字步进 / 端口号 / 短数值类输入**不要**放任它撑满 (UInputNumber 拉满一行, ± 按钮分居两端很难用), 给个 `w-28`~`w-40`。

**底层规律 (踩过两次)**: Tailwind v4 工具类全在 `@layer utilities`, 而 style.css 里**不分层的自定义规则永远压过任何工具类 — 跟优先级数值无关, `:where()` 也救不了**。想让工具类能覆盖的全局默认值, 必须写进 `@layer base/components`; 想压过工具类的 (如改 display), 留在层外。

### 自定义 `@utility` 能加 variant 前缀 (v4 实证)

`style.css` 里 `@utility raised-surface { ... }` 这类自定义工具类, **可直接加 `hover:` / `focus:` 等 variant** —— `hover:raised-surface` 真生效 (v4 自动给 `@utility` 生成 variant), 不必为了 hover 态把配方拆成普通 class 再手写 `:hover`。2026-06-14 UI 升级 Part 1 ListRow 实证 (`raised-surface` / `overlay-surface` / `bg-sunken` 同理)。
> 副记: 想用 DOM 验「某 class 到底生没生成 CSS 规则」, 必须**递归** `CSSLayerBlockRule.cssRules` —— v4 把 utilities 全塞进 `@layer`, 平铺遍历 `document.styleSheets[].cssRules` 查不到 (假阴性)。离屏视觉自检时按此判。

### `:ui="{ base: '...' }"` 是替换不是 merge

调 NuxtUI 组件加 width / 样式 → `class="w-24"`, **不是** `:ui="{ base: 'w-24' }"`. 后者会把 base slot 默认那一长串 (`bg-default text-default ring-default ...`) 清空, 只剩 `w-24`, 背景变裸 HTML 白色. 踩过 Step editor 输入框白底的坑.

### `UInputMenu` 让用户创建新项 → `create-item` + `@create`, 不是 `creatable`

`creatable` 是 NuxtUI v2/headless 旧 prop 名, **v4 改名 `create-item` 且静默忽略旧名** → 写 `creatable` 不报错但创建项压根不出现 (下拉只显 "No matching data", 用户输新分类/标签建不出来)。v4 正确写法:

- `:create-item="'always'"` (输入值非现有精确项就显「创建」; 写 `true` 只在候选**全空**时才显, 部分匹配时反而不给创建, 不够)
- **必须配 `@create` 处理器** —— 选「创建」只 `emit('create', searchTerm)`, **不自动写 model-value**。单值: `@create="(v) => setX(v)"`; 多值 (`multiple`): `@create="(v) => model = [...model, v]"`。
- **单值 menu 还要让新值进入 `items`**。`@create` 只给原始文本, Nuxt UI 不会替你把新项 push 进候选列表；如果 `v-model` 设成一个不在 `items` 里的值, 后续可能显示/选择状态不稳定。做法: 维护 `createdX` 本地数组, `items = unique(existing, createdX, [model])`, `@create` 里先 trim/dedupe/push, 再把 model 设成该值。容器分类用 `categoryOptions.ts` 的 `addCreatedCategory()` / `uniqueCategoryOptions()` 锁这个规则。

范例: `SwitchInspector.vue` (`:create-item="'always'"` + `@create`)。2026-06-13 全仓 13 处 `creatable` 全是坏的 (分类/标签建不了新项), 已统一修 (含 ContainerSettings 一处有 create-item 但漏 @create 同样建不出)。

### 反馈方式: 成功内联在触发点, toast 只留错误/后台事件

用户明确不喜欢成功类 toast (顶上弹出干扰视线、和操作点割裂)。**新代码不加成功 toast** —— 按这棵决策树:

1. **操作结果在当前视野立即可见?** (删除→列表项消失 / 自动布局→画布重排 / 折叠/融合/导入→画布变 / 重拍→详情图更新) → **什么都不弹**, 变化本身就是反馈。
2. **复制到剪贴板** (不可见) → **优先原地反馈, 别 toast**:
   - 触发点是**持久按钮** = 按钮内联闪「已复制 ✓」(本地 `copied` ref + ~1500ms 恢复, icon 切 check, 文案 `common.copied`)。
   - 触发点是**下拉菜单项** = `onSelect(e)` 里 `e.preventDefault()` **留住菜单**, 被点项 label/icon 原地切「已复制 ✓」~1500ms (范例 `NodeInspector.vue` 复制下拉)。菜单能留住就别弹 toast。
   - 仅当触发点**真的留不住**(点完必然关闭、无处可显) 才退而用短 toast (`duration: 1500`)。
   - 错误**始终** toast (下同)。
3. **有明确持久触发按钮的操作** (保存) → 按钮内联「已保存 ✓」(参 ContainerEditorToolbar 保存按钮 + useEditorSave.saveFlash, success soft ~1.6s 恢复)。
4. **后台/跨窗口/跨位置事件, 或系统替你做了非显式动作的告知** (热键触发运行入队、导出/分享到库、录制落盘、自动重连了线、校准值变更) → **保留 toast** (这是 toast 的正当用途)。
5. **错误/警告** → 一律保留 toast (invoke 自动错误 toast 也保留)。

2026-06-11 已按此把全仓 ~20 处成功 toast 收口 (删/内联), 并清掉随之死掉的 i18n 文案 key。

写完新 view 前跑:

```bash
grep -rn "bg-zinc-\|text-zinc-\|border-zinc-\|bg-black\|bg-white\|text-white\|text-black" frontend/src/
```

必须**零行**.
