---
status: active
last_updated: 2026-05-20
when_to_read: before writing / editing .vue components or Tailwind classes
applies_to: [vue, tailwind, nuxtui, ui, frontend, component, modal, button]
when_to_update: 改前端 UI 基线 (nuxtui 版本/约定 / Tailwind 配置 / 公共组件风格) 时
last_updated: 2026-06-10
---

# UI Playbook (NuxtUI 偏好)

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
- **frameless 独立工具窗 (HUD) 不用这个** → 走 [standalone-window-style](standalone-window-style.md)。

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

style.css 全局把含 input/textarea 的 NuxtUI 包装层默认 `width: 100%` (表单不用处处写 w-full)。这条的 width 走 `:where()` 零优先级 — **组件上直接写 `w-36` / `max-w-*` / `flex-1` 就能压过它**, 不用包 div。数字步进 / 端口号 / 短数值类输入**不要**放任它撑满 (UInputNumber 拉满一行, ± 按钮分居两端很难用), 给个 `w-28`~`w-40`。

### `:ui="{ base: '...' }"` 是替换不是 merge

调 NuxtUI 组件加 width / 样式 → `class="w-24"`, **不是** `:ui="{ base: 'w-24' }"`. 后者会把 base slot 默认那一长串 (`bg-default text-default ring-default ...`) 清空, 只剩 `w-24`, 背景变裸 HTML 白色. 踩过 Step editor 输入框白底的坑.

### 验证脚本

写完新 view 前跑:

```bash
grep -rn "bg-zinc-\|text-zinc-\|border-zinc-\|bg-black\|bg-white\|text-white\|text-black" frontend/src/
```

必须**零行**.
