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
| 主背景 (最深)   | `bg-default`                                          |
| 卡片/弹层       | `bg-elevated`                                         |
| 分隔区/muted    | `bg-muted`                                            |
| 强调背景 (最高) | `bg-accented`                                         |
| 主文字 (最亮)   | `text-highlighted`                                    |
| 普通文字        | `text-default`                                        |
| 强调文字        | `text-toned`                                          |
| 弱化文字        | `text-muted`                                          |
| 提示文字 (最弱) | `text-dimmed`                                         |
| 边框            | `border-default` / `border-muted` / `border-accented` |

状态色: `bg-primary` / `bg-error` / `bg-warning` / `bg-success` / `bg-info`.

**禁用 list** (写新组件前 grep 自己):

- `bg-zinc-*` / `bg-black` / `bg-white`
- `text-zinc-*` / `text-white` / `text-black`
- `border-zinc-*` / `ring-zinc-*` / `divide-zinc-*`

**唯一例外**: 录制叠加层这类"完全黑" scrim. 模态遮罩 **不要** override — Modal / Slideover 自带 overlay slot, 信任默认.

### `:ui="{ base: '...' }"` 是替换不是 merge

调 NuxtUI 组件加 width / 样式 → `class="w-24"`, **不是** `:ui="{ base: 'w-24' }"`. 后者会把 base slot 默认那一长串 (`bg-default text-default ring-default ...`) 清空, 只剩 `w-24`, 背景变裸 HTML 白色. 踩过 Step editor 输入框白底的坑.

### 验证脚本

写完新 view 前跑:

```bash
grep -rn "bg-zinc-\|text-zinc-\|border-zinc-\|bg-black\|bg-white\|text-white\|text-black" frontend/src/
```

必须**零行**.
