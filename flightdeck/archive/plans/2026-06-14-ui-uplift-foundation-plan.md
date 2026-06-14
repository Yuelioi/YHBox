---
status: done
summary: "Spec A 实现计划 Part 1：设计系统地基(style.css v3 token/工具 + 主按钮渐变主题 + 子图坐标常量收敛) + 7 个共用组件(AppCard/AlertBox/EmptyState/StatusPill/SectionHeader/IconBadge/ListRow)，有逻辑的抽 helper 单测 + Playwright 视觉验；逐屏迁移在 Part 2"
last_updated: 2026-06-14
implements: specs/2026-06-14-ui-uplift-foundation.md
---

# UI 升级地基 + 共用组件 Implementation Plan (Spec A · Part 1)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 建立 v3「克制精致」设计系统地基（CSS token/工具 + 主按钮渐变 + 散落坐标常量收敛），并产出 7 个按 v3 写定的共用 UI 组件，供 Part 2 逐屏迁移消费。

**Architecture:** 不引新依赖。CSS 层：在 `frontend/src/style.css` 用 Tailwind v4 `@utility` 加 `raised-surface`/`overlay-surface`（派生自 `--ui-bg`，跟现有 `bg-sunken` 同模式），加一条 `.btn-primary-raised` 渐变类；主按钮渐变经 `vite.config.ts` 的 NuxtUI `button.compoundVariants` build-time 注入。组件层：放 `components/common/`，颜色映射一律用**字面 class 映射表**（动态 `text-${x}` 会被 Tailwind purge，全仓惯例见 BaseModal/StatCard）；有决策逻辑的把纯函数抽到 `.helpers.ts` 兄弟文件单测（NuxtUI 组件无法在 vitest mount，全仓惯例见 VarNameInput.helpers.ts）。

**Tech Stack:** Vue 3.5 + NuxtUI v4.7 + Tailwind v4 + Vite 8 + Vitest 4 (happy-dom)；图标 tabler；字体 Inter Variable / JetBrains Mono Variable（已打包）。

---

## Progress

current: done — Part 1 全部 task 落地 (12 commit, 062bdb6..d98e6b5)，验证全绿：vitest 329 passed / typecheck clean / build:dev ok / Playwright 离屏视觉自检过 (v3 表面观感克制、主按钮渐变生效其它扁平、ListRow `hover:raised-surface` 实证生效无需回退)。code review 抓出 1 处 consumer-audit gap (ContainerEditorView startNodeOf 第三处子图入口默认坐标) 已收敛。Part 2 逐屏迁移另起 plan。

---

## File Structure

**新建（components/common/）**：
- `AppCard.vue` — 卡片外壳（raised 表面 + padding 档 + 可选 hover 浮起）。纯展示。
- `AlertBox.vue` + `AlertBox.helpers.ts` — 提示框（info/success/warning/error）。helper = type→class/icon 映射。
- `EmptyState.vue` — 空状态（icon + 标题 + 说明 + action 槽）。纯展示。
- `StatusPill.vue` + `StatusPill.helpers.ts` — 状态药丸。helper = status→class 映射。
- `SectionHeader.vue` — 非折叠分区头（icon + 标题 + 计数 + actions 槽）。纯展示。**不吞** SidebarSection（那是可折叠版，保留）。
- `IconBadge.vue` + `IconBadge.helpers.ts` — 图标徽框。helper = size/color/shape→class 映射。
- `ListRow.vue` — 列表行（icon/默认/trailing 槽 + hover 浮起）。纯展示。

**测试（同目录兄弟文件）**：`AlertBox.helpers.spec.ts` · `StatusPill.helpers.spec.ts` · `IconBadge.helpers.spec.ts`。

**修改**：
- `frontend/src/style.css` — 加 `@utility raised-surface` / `@utility overlay-surface` / `.btn-primary-raised`。
- `frontend/vite.config.ts:43-52` — NuxtUI `ui.button.compoundVariants` 注入主按钮渐变。
- `frontend/src/composables/containerEditor/constants.ts` — 加 `SUBGRAPH_ENTRY_DEFAULT` / `SUBGRAPH_OUTPUT_DEFAULT`。
- `frontend/src/composables/containerEditor/elkGraph.ts:29,36,40` — 引常量替字面 80/160/420。
- `frontend/src/composables/containerEditor/useContainerDraft.ts:115,124` — 同上。

**纯展示组件的验证**：本 Part 末 Task 2.8 用一次性 scratch 路由 + Playwright MCP 离屏渲染做视觉自检（见 [headless-ui-verify](../checklists/headless-ui-verify.md)），验完移除 scratch 路由。

---

## Phase 1 · 设计系统地基

### Task 1.1: style.css 加 v3 表面 token + 工具类 + 主按钮渐变类

**Files:**
- Modify: `frontend/src/style.css`（在现有 `@utility bg-sunken` 块之后追加）

- [ ] **Step 1: 追加 CSS**

在 `frontend/src/style.css` 里 `@utility bg-sunken { ... }` 块之后插入：

```css
/* ── v3 浮层表面: 比 base 略亮 + 顶部高光 + 柔投影, 让卡片/面板「轻浮起」。
   派生自 --ui-bg, 跟 bg-sunken 同思路, 禁止散写 zinc 字面值。 ── */
@utility raised-surface {
  background-image: linear-gradient(
    180deg,
    color-mix(in oklab, var(--ui-bg) 86%, white),
    color-mix(in oklab, var(--ui-bg) 92%, white)
  );
  border: 1px solid var(--ui-border);
  border-top-color: color-mix(in oklab, var(--ui-border) 55%, white);
  box-shadow:
    0 2px 6px rgba(0, 0, 0, 0.35),
    inset 0 1px 0 rgba(255, 255, 255, 0.045);
}

/* ── 浮层 overlay: 菜单 / 模态 / popover, 比 raised 再亮一档 + 更重投影。 ── */
@utility overlay-surface {
  background-color: color-mix(in oklab, var(--ui-bg) 84%, white);
  border: 1px solid var(--ui-border-accented);
  box-shadow:
    0 10px 30px rgba(0, 0, 0, 0.5),
    inset 0 1px 0 rgba(255, 255, 255, 0.05);
}

/* ── 主按钮唯一显眼渐变 (v3 渐变纪律: 全场只两处, 此其一)。
   由 vite.config 的 button.compoundVariants base 引用; 普通 CSS 类(非 @utility),
   始终输出、不被 purge。盖在 NuxtUI 默认 bg-primary 之上 (background-image 覆盖 background-color)。 ── */
.btn-primary-raised {
  background-image: linear-gradient(180deg, #11c08a, #0a9d6f);
  box-shadow:
    0 1px 3px rgba(16, 185, 129, 0.25),
    inset 0 1px 0 rgba(255, 255, 255, 0.12);
}
.btn-primary-raised:hover {
  background-image: linear-gradient(180deg, #15cf96, #0caf7c);
}
```

- [ ] **Step 2: 验证 build 不炸**

Run: `cd frontend; pnpm build:dev`
Expected: 构建成功，无 CSS 解析错误。（color-mix / @utility 是 webview2 chromium 已支持 + Tailwind v4 原生语法。）

- [ ] **Step 3: Commit**

```bash
git add frontend/src/style.css
git commit -m "feat(ui): v3 raised/overlay surface utilities + primary button gradient class"
```

---

### Task 1.2: vite.config 注入主按钮渐变 (NuxtUI button.compoundVariants)

**Files:**
- Modify: `frontend/vite.config.ts:43-52`

- [ ] **Step 1: 改 ui() 配置**

把 `frontend/vite.config.ts` 的 `ui({ ... })`（43-52 行）改成：

```typescript
    ui({
      ui: {
        colors: {
          primary: "emerald",
          neutral: "zinc",
          // 项目警告色一直是 amber 系 (NuxtUI 默认 yellow) — 钉死防漂移
          warning: "amber",
        },
        // 主按钮 (primary + solid) 唯一显眼渐变; 其它 variant (soft/ghost/outline) 零改动。
        // .btn-primary-raised 在 style.css, background-image 覆盖默认 bg-primary。
        button: {
          compoundVariants: [
            {
              color: "primary",
              variant: "solid",
              class: { base: "btn-primary-raised" },
            },
          ],
        },
      },
    }),
```

- [ ] **Step 2: 验证类型 + build**

Run: `cd frontend; pnpm build:dev`
Expected: 构建成功。`compoundVariants` 是 NuxtUI v4 `TVConfig` 支持字段（node_modules/@nuxt/ui dist/runtime/types/tv.d.ts 已验），类型通过。

- [ ] **Step 3: Playwright 视觉验主按钮**

按 [headless-ui-verify](../checklists/headless-ui-verify.md) 起离屏渲染，开任一带 primary UButton 的页（如容器列表「新建」按钮），确认按钮呈竖向翡翠渐变 + 极柔阴影；soft/ghost 按钮**不变**（仍扁平）。
Expected: 主按钮有渐变质感，其它 variant 无变化。

- [ ] **Step 4: Commit**

```bash
git add frontend/vite.config.ts
git commit -m "feat(ui): primary solid button gradient via NuxtUI button.compoundVariants"
```

---

### Task 1.3: 收敛子图入口/出口默认坐标常量

> 现状摸查 + grep 实证：`80,160`(entry) / `420,160`(output) 在 `elkGraph.ts:36,40` **与** `useContainerDraft.ts:115,124` 各写一份，且 `elkGraph.ts:29` 有注释「两处必须一致」、`elkGraph.test.ts` 在测它 —— 真·「各写各的」，收成单一来源。
> ⚠️ **只动这 4 个 `?? 80/160/420` 点**。同文件域内别的 80/160 是不同含义（elkConfig 层间距 '80'、useSnapEngine `+80`、useSubgraphToScript `+80`、测试 fixture 宽度、CommentBox 默认 `320×160`），**不要 blanket 替换**。CommentBox/save-flash/toast 等单点魔法值按 YAGNI 不收。

**Files:**
- Modify: `frontend/src/composables/containerEditor/constants.ts`
- Modify: `frontend/src/composables/containerEditor/elkGraph.ts:29,36,40`
- Modify: `frontend/src/composables/containerEditor/useContainerDraft.ts:115,124`
- Test (已存在，作回归守门): `frontend/src/composables/containerEditor/elkGraph.test.ts:76-79`

- [ ] **Step 1: 先跑既有测试确认绿（基线）**

Run: `cd frontend; pnpm test -- elkGraph`
Expected: PASS，含「缺坐标走默认 (entry 80,160 / output 420,160)」用例。

- [ ] **Step 2: constants.ts 加常量**

在 `constants.ts` 末尾追加：

```typescript
/** 子图入口 marker 缺省坐标 (flow-coord). elkGraph 与 useContainerDraft 两处必须一致 — 单一来源. */
export const SUBGRAPH_ENTRY_DEFAULT = { x: 80, y: 160 }

/** 子图出口 marker 缺省坐标 (flow-coord). 同上, 两处必须一致. */
export const SUBGRAPH_OUTPUT_DEFAULT = { x: 420, y: 160 }
```

- [ ] **Step 3: elkGraph.ts 引常量**

`elkGraph.ts` 顶部 import 区加（与现有 import 合并）：

```typescript
import { SUBGRAPH_ENTRY_DEFAULT, SUBGRAPH_OUTPUT_DEFAULT } from './constants'
```

第 36 行 entry default 改为：

```typescript
    out.push({ id: entry.nodeID, kind: 'SubgraphInput', x: entry.x ?? SUBGRAPH_ENTRY_DEFAULT.x, y: entry.y ?? SUBGRAPH_ENTRY_DEFAULT.y, config: {} })
```

第 40 行 output default 改为：

```typescript
    out.push({ id: p.nodeID, kind: 'SubgraphOutput', x: p.x ?? SUBGRAPH_OUTPUT_DEFAULT.x, y: p.y ?? SUBGRAPH_OUTPUT_DEFAULT.y, config: {} })
```

第 29 行注释更新为（保留「两处一致」语义、指向常量）：

```typescript
// 默认坐标走 SUBGRAPH_ENTRY_DEFAULT / SUBGRAPH_OUTPUT_DEFAULT (constants.ts), 跟 syncFlowFromDraft 一致。
```

- [ ] **Step 4: useContainerDraft.ts 引常量**

`useContainerDraft.ts` import 区加：

```typescript
import { SUBGRAPH_ENTRY_DEFAULT, SUBGRAPH_OUTPUT_DEFAULT } from './constants'
```

第 115 行改为：

```typescript
            position: { x: sg.entry.x ?? SUBGRAPH_ENTRY_DEFAULT.x, y: sg.entry.y ?? SUBGRAPH_ENTRY_DEFAULT.y },
```

第 124 行改为：

```typescript
            position: { x: p.x ?? SUBGRAPH_OUTPUT_DEFAULT.x, y: p.y ?? SUBGRAPH_OUTPUT_DEFAULT.y },
```

- [ ] **Step 5: 跑测试验证不回归**

Run: `cd frontend; pnpm test -- elkGraph; pnpm typecheck`
Expected: PASS（默认值未变，仅来源收敛）；typecheck 无新错。

- [ ] **Step 6: Commit**

```bash
git add frontend/src/composables/containerEditor/constants.ts frontend/src/composables/containerEditor/elkGraph.ts frontend/src/composables/containerEditor/useContainerDraft.ts
git commit -m "refactor(editor): hoist subgraph marker default coords into constants"
```

---

## Phase 2 · 共用组件（每个按 v3 写一遍）

> 通则：组件**不含 i18n**（title/description 等走 props，由消费方传 i18n 文案）；颜色映射用字面 class 表；图标用 tabler。

### Task 2.1: AppCard.vue（卡片外壳）

**Files:**
- Create: `frontend/src/components/common/AppCard.vue`

- [ ] **Step 1: 写组件**

```vue
<!-- 卡片外壳: v3 raised 表面 (顶光 + 高光 + 柔投影)。padding 档可选; hover=true 时悬停加强浮起。
     内容走默认 slot。颜色/文字由内容自身决定, 本组件只给外壳。 -->
<template>
  <div
    class="rounded-xl raised-surface"
    :class="[paddingClass, hover ? 'transition-shadow duration-150 hover:shadow-[0_6px_20px_rgba(0,0,0,0.45)]' : '']"
  >
    <slot />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    /** padding 档: panel=p-4 (紧凑面板) / section=p-6 (卡片) / none=无 */
    padding?: 'panel' | 'section' | 'none'
    /** 悬停加强浮起 (列表卡片用) */
    hover?: boolean
  }>(),
  { padding: 'section' },
)

const PAD: Record<NonNullable<typeof props.padding>, string> = {
  panel: 'p-4',
  section: 'p-6',
  none: '',
}
const paddingClass = computed(() => PAD[props.padding])
</script>
```

- [ ] **Step 2: typecheck**

Run: `cd frontend; pnpm typecheck`
Expected: 无新错（视觉验留 Task 2.8 统一做）。

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/common/AppCard.vue
git commit -m "feat(ui): AppCard shared card shell (v3 raised surface)"
```

---

### Task 2.2: StatusPill.vue（状态药丸 + helper 单测）

**Files:**
- Create: `frontend/src/components/common/StatusPill.helpers.ts`
- Create: `frontend/src/components/common/StatusPill.helpers.spec.ts`
- Create: `frontend/src/components/common/StatusPill.vue`

- [ ] **Step 1: 写 helper**

```typescript
// StatusPill 纯映射: status → 字面 class (动态 text-${x} 会被 Tailwind purge, 故用字面表)。
export type PillStatus = 'online' | 'ready' | 'paused' | 'failed'

const MAP: Record<PillStatus, string> = {
  online: 'bg-primary/15 text-primary',
  ready: 'bg-elevated text-muted',
  paused: 'bg-warning/15 text-warning',
  failed: 'bg-error/15 text-error',
}

export function statusPillClass(status: PillStatus): string {
  return MAP[status]
}
```

- [ ] **Step 2: 写失败测试**

```typescript
import { describe, it, expect } from 'vitest'
import { statusPillClass } from './StatusPill.helpers'

describe('statusPillClass', () => {
  it('online → primary tint', () => expect(statusPillClass('online')).toContain('text-primary'))
  it('ready → muted', () => expect(statusPillClass('ready')).toContain('text-muted'))
  it('paused → warning', () => expect(statusPillClass('paused')).toContain('text-warning'))
  it('failed → error', () => expect(statusPillClass('failed')).toContain('text-error'))
})
```

- [ ] **Step 3: 跑测试验证先失败**

Run: `cd frontend; pnpm test -- StatusPill`
Expected: FAIL（helper 尚未创建 → import 报错）。
> 注: 若按 Step 1→2 顺序已建 helper, 此步改为「确认 helper 缺时会 fail」的认知确认；TDD 严格做法是先写 spec 再写 helper。本任务两文件一并交付，跑一次确认绿即可。

- [ ] **Step 4: 写组件**

```vue
<!-- 状态药丸: status 驱动配色 (字面表)。文字走 slot 或 label; dot=true 加前导圆点。 -->
<template>
  <span
    class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-semibold"
    :class="pillClass"
  >
    <span v-if="dot" class="size-1.5 rounded-full bg-current" />
    <slot>{{ label }}</slot>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { statusPillClass, type PillStatus } from './StatusPill.helpers'

const props = defineProps<{ status: PillStatus; label?: string; dot?: boolean }>()
const pillClass = computed(() => statusPillClass(props.status))
</script>
```

- [ ] **Step 5: 跑测试 + typecheck 验证绿**

Run: `cd frontend; pnpm test -- StatusPill; pnpm typecheck`
Expected: PASS。

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/common/StatusPill.vue frontend/src/components/common/StatusPill.helpers.ts frontend/src/components/common/StatusPill.helpers.spec.ts
git commit -m "feat(ui): StatusPill shared component + helper test"
```

---

### Task 2.3: AlertBox.vue（提示框 + helper 单测）

**Files:**
- Create: `frontend/src/components/common/AlertBox.helpers.ts`
- Create: `frontend/src/components/common/AlertBox.helpers.spec.ts`
- Create: `frontend/src/components/common/AlertBox.vue`

- [ ] **Step 1: 写 helper**

```typescript
// AlertBox 纯映射: type → 盒/图标 class + 默认图标 (字面表)。
export type AlertType = 'info' | 'success' | 'warning' | 'error'

interface AlertStyle {
  box: string
  icon: string
  defaultIcon: string
}

const MAP: Record<AlertType, AlertStyle> = {
  info: { box: 'bg-info/10 ring-info/20', icon: 'text-info', defaultIcon: 'i-tabler-info-circle' },
  success: { box: 'bg-success/10 ring-success/20', icon: 'text-success', defaultIcon: 'i-tabler-circle-check' },
  warning: { box: 'bg-warning/10 ring-warning/20', icon: 'text-warning', defaultIcon: 'i-tabler-alert-triangle' },
  error: { box: 'bg-error/10 ring-error/20', icon: 'text-error', defaultIcon: 'i-tabler-alert-circle' },
}

export function alertStyle(type: AlertType): AlertStyle {
  return MAP[type]
}
```

- [ ] **Step 2: 写测试**

```typescript
import { describe, it, expect } from 'vitest'
import { alertStyle } from './AlertBox.helpers'

describe('alertStyle', () => {
  it('warning → amber box + 三角图标', () => {
    const s = alertStyle('warning')
    expect(s.box).toContain('bg-warning/10')
    expect(s.icon).toBe('text-warning')
    expect(s.defaultIcon).toBe('i-tabler-alert-triangle')
  })
  it('error → red', () => expect(alertStyle('error').icon).toBe('text-error'))
  it('info → info', () => expect(alertStyle('info').icon).toBe('text-info'))
  it('success → success', () => expect(alertStyle('success').icon).toBe('text-success'))
})
```

- [ ] **Step 3: 写组件**

```vue
<!-- 提示框: type 驱动配色 (扁平 tint + ring, 无浮层)。title 必传; 说明走默认 slot; icon 可覆盖默认。 -->
<template>
  <div class="flex gap-2.5 rounded-lg px-3 py-2.5 ring-1" :class="style.box">
    <UIcon :name="icon || style.defaultIcon" class="mt-0.5 size-4 shrink-0" :class="style.icon" />
    <div class="min-w-0 flex-1">
      <p v-if="title" class="text-xs font-medium text-highlighted">{{ title }}</p>
      <p v-if="$slots.default" class="text-[11px] text-muted" :class="title ? 'mt-0.5' : ''">
        <slot />
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { alertStyle, type AlertType } from './AlertBox.helpers'

const props = defineProps<{ type: AlertType; title?: string; icon?: string }>()
const style = computed(() => alertStyle(props.type))
</script>
```

- [ ] **Step 4: 跑测试 + typecheck**

Run: `cd frontend; pnpm test -- AlertBox; pnpm typecheck`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/common/AlertBox.vue frontend/src/components/common/AlertBox.helpers.ts frontend/src/components/common/AlertBox.helpers.spec.ts
git commit -m "feat(ui): AlertBox shared component + helper test"
```

---

### Task 2.4: EmptyState.vue（空状态）

**Files:**
- Create: `frontend/src/components/common/EmptyState.vue`

- [ ] **Step 1: 写组件**

```vue
<!-- 空状态: icon 徽框 (raised) + 标题 + 可选说明 + action 槽 (CTA)。文字走 props (消费方传 i18n)。 -->
<template>
  <div class="rounded-xl border border-dashed border-default bg-default/40 px-6 py-12 text-center">
    <div class="mx-auto mb-3 flex size-12 items-center justify-center rounded-xl raised-surface">
      <UIcon :name="icon" class="size-6 text-muted" />
    </div>
    <p class="text-sm font-medium text-highlighted">{{ title }}</p>
    <p v-if="description" class="mx-auto mt-1 max-w-xs text-xs text-muted">{{ description }}</p>
    <div v-if="$slots.action" class="mt-4 flex justify-center">
      <slot name="action" />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{ icon: string; title: string; description?: string }>()
</script>
```

- [ ] **Step 2: typecheck**

Run: `cd frontend; pnpm typecheck`
Expected: 无新错。

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/common/EmptyState.vue
git commit -m "feat(ui): EmptyState shared component"
```

---

### Task 2.5: SectionHeader.vue（非折叠分区头）

**Files:**
- Create: `frontend/src/components/common/SectionHeader.vue`

- [ ] **Step 1: 写组件**

```vue
<!-- 非折叠分区头: icon + 标题 (xs/600 大写 tracking) + 可选计数 + actions 槽。
     可折叠场景用既有 SidebarSection, 不归此组件。 -->
<template>
  <div class="flex items-center gap-2 border-b border-default px-3 py-2">
    <UIcon v-if="icon" :name="icon" class="size-3.5 text-dimmed" />
    <span class="text-xs font-semibold uppercase tracking-wider text-dimmed">{{ title }}</span>
    <span v-if="count !== undefined" class="text-[10px] text-dimmed">({{ count }})</span>
    <div class="flex-1" />
    <slot name="actions" />
  </div>
</template>

<script setup lang="ts">
defineProps<{ title: string; icon?: string; count?: number }>()
</script>
```

- [ ] **Step 2: typecheck**

Run: `cd frontend; pnpm typecheck`
Expected: 无新错。

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/common/SectionHeader.vue
git commit -m "feat(ui): SectionHeader shared component (non-collapsible)"
```

---

### Task 2.6: IconBadge.vue（图标徽框 + helper 单测）

**Files:**
- Create: `frontend/src/components/common/IconBadge.helpers.ts`
- Create: `frontend/src/components/common/IconBadge.helpers.spec.ts`
- Create: `frontend/src/components/common/IconBadge.vue`

- [ ] **Step 1: 写 helper**

```typescript
// IconBadge 纯映射: size→框/图标尺寸; color→图标色 (字面表)。
export type BadgeSize = 'sm' | 'md' | 'lg'
export type BadgeColor = 'default' | 'primary' | 'violet' | 'amber' | 'sky'

const BOX: Record<BadgeSize, string> = {
  sm: 'size-7 rounded-md',
  md: 'size-10 rounded-lg',
  lg: 'size-14 rounded-xl',
}
const ICON: Record<BadgeSize, string> = {
  sm: 'size-3.5',
  md: 'size-5',
  lg: 'size-7',
}
const COLOR: Record<BadgeColor, string> = {
  default: 'text-muted',
  primary: 'text-primary',
  violet: 'text-violet-400',
  amber: 'text-warning',
  sky: 'text-sky-400',
}

export function badgeBoxClass(size: BadgeSize): string {
  return BOX[size]
}
export function badgeIconSize(size: BadgeSize): string {
  return ICON[size]
}
export function badgeIconColor(color: BadgeColor): string {
  return COLOR[color]
}
```

- [ ] **Step 2: 写测试**

```typescript
import { describe, it, expect } from 'vitest'
import { badgeBoxClass, badgeIconSize, badgeIconColor } from './IconBadge.helpers'

describe('IconBadge helpers', () => {
  it('md 框 size-10', () => expect(badgeBoxClass('md')).toContain('size-10'))
  it('lg 图标 size-7', () => expect(badgeIconSize('lg')).toBe('size-7'))
  it('primary 图标 text-primary', () => expect(badgeIconColor('primary')).toBe('text-primary'))
  it('default 图标 text-muted', () => expect(badgeIconColor('default')).toBe('text-muted'))
})
```

- [ ] **Step 3: 写组件**

```vue
<!-- 图标徽框: raised 小方框 + 居中图标。size/color/shape 可选; shape=round 时圆形。 -->
<template>
  <div class="flex shrink-0 items-center justify-center raised-surface" :class="[boxClass, shape === 'round' ? '!rounded-full' : '']">
    <UIcon :name="icon" :class="[iconSizeClass, iconColorClass]" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { badgeBoxClass, badgeIconSize, badgeIconColor, type BadgeSize, type BadgeColor } from './IconBadge.helpers'

const props = withDefaults(
  defineProps<{ icon: string; size?: BadgeSize; color?: BadgeColor; shape?: 'square' | 'round' }>(),
  { size: 'md', color: 'default', shape: 'square' },
)
const boxClass = computed(() => badgeBoxClass(props.size))
const iconSizeClass = computed(() => badgeIconSize(props.size))
const iconColorClass = computed(() => badgeIconColor(props.color))
</script>
```

- [ ] **Step 4: 跑测试 + typecheck**

Run: `cd frontend; pnpm test -- IconBadge; pnpm typecheck`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/common/IconBadge.vue frontend/src/components/common/IconBadge.helpers.ts frontend/src/components/common/IconBadge.helpers.spec.ts
git commit -m "feat(ui): IconBadge shared component + helper test"
```

---

### Task 2.7: ListRow.vue（列表行 hover 浮起）

**Files:**
- Create: `frontend/src/components/common/ListRow.vue`

- [ ] **Step 1: 写组件**

```vue
<!-- 列表行: icon / 默认 / trailing 三槽。静止扁平; hover 套 raised 轻浮起。active=true 常驻浮起。 -->
<template>
  <div
    class="flex items-center gap-2.5 rounded-lg px-3 py-2 transition-colors duration-150"
    :class="active ? 'raised-surface' : 'border border-transparent hover:raised-surface'"
  >
    <slot name="icon" />
    <div class="min-w-0 flex-1">
      <slot />
    </div>
    <slot name="trailing" />
  </div>
</template>

<script setup lang="ts">
defineProps<{ active?: boolean }>()
</script>
```

- [ ] **Step 2: typecheck**

Run: `cd frontend; pnpm typecheck`
Expected: 无新错。
> ⚠️ `hover:raised-surface`（在自定义 `@utility` 上加 `hover:` variant）需 Task 2.8 视觉验证生效；若 Tailwind v4 不对自定义 utility 应用 variant，回退方案：把 raised 配方提成普通 class `.raised-surface` 并写 `.group:hover` 或在 ListRow 内联 `hover:` 的具体 box-shadow/bg 工具类。

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/common/ListRow.vue
git commit -m "feat(ui): ListRow shared component (hover raise)"
```

---

### Task 2.8: Scratch 路由 + Playwright 视觉自检（全组件一次过）

**Files:**
- Create (临时): `frontend/src/views/_ScratchShowcase.vue`
- Modify (临时): `frontend/src/router.ts`（加一条 `/scratch` 路由指向 showcase）

- [ ] **Step 1: 写 scratch showcase**

`_ScratchShowcase.vue` 里把 7 个组件各摆 2-3 个状态（AppCard 三档 padding；StatusPill 四态；AlertBox 四型；EmptyState 带/不带 CTA；SectionHeader；IconBadge 三 size × 几色；ListRow 静止+active）。纯静态、写死示例数据。

- [ ] **Step 2: 临时挂路由**

`router.ts` 加：

```typescript
{ path: '/scratch', name: 'Scratch', component: () => import('@/views/_ScratchShowcase.vue') },
```

- [ ] **Step 3: Playwright MCP 离屏渲染 `/scratch`**

按 [headless-ui-verify](../checklists/headless-ui-verify.md) 起 vite + Playwright MCP，导航 `/scratch`，截图。逐项核对：
- raised 卡片/徽框有「略亮 + 顶部高光 + 柔投影」的轻浮起感（不浮夸）；
- 主按钮渐变生效、其它按钮扁平；
- StatusPill 四态配色对（在线绿/就绪灰/暂停琥珀/失败红）；
- AlertBox 四型扁平 tint + ring；
- **ListRow hover 真的浮起**（验证 `hover:raised-surface`；不生效则走 Step 2 回退方案修 ListRow）；
- EmptyState 居中、徽框轻浮起、CTA 在位。

Expected: 各项符合 v3「克制精致」；不符就调 style.css 参数（color-mix 百分比 / 阴影强度）或组件 class，重渲直到对。

- [ ] **Step 4: 撤掉 scratch（验完即删，不留垃圾路由）**

删 `_ScratchShowcase.vue` + router.ts 里的 `/scratch` 行。
> 二号铁律：临时脚手架不留。组件正式视觉归宿是 Part 2 各屏。

- [ ] **Step 5: 全量门禁 + Commit**

Run: `cd frontend; pnpm test; pnpm typecheck; pnpm build:dev`
Expected: 测试/类型/构建绿（预存失败按 [build](../checklists/build.md) 判：runtime fixture / i18n residue 39 / lint 18，非本轮回归）。

```bash
git add -A
git commit -m "chore(ui): visual self-check shared components (scratch route removed)"
```

---

## Self-Review（写计划后自检）

- **Spec 覆盖**：① token/字体/渐变 → Task 1.1-1.2 ✓；② 7 组件 → Task 2.1-2.7 ✓（按钮走主题非组件 → Task 1.2 ✓）；④ 常量收敛 → Task 1.3 ✓（已按 grep 实证缩到真·重复的子图坐标，单点魔法值按 YAGNI 不收，spec ④B helper 待 Part 2/后续 grep 决定）。③ 逐屏迁移 = **Part 2**（见下），本 Part 不含。字体 mono 选型（JetBrains vs 其它）= 用户待确认项，不阻塞本 Part（组件不碰字体族，沿用全局）。
- **占位符扫描**：无 TBD/TODO；每个 code step 有完整代码；ListRow 的 `hover:raised-surface` 风险已标回退方案（非占位）。
- **类型一致**：`PillStatus`/`AlertType`/`BadgeSize`/`BadgeColor` helper 与组件 import 同名；`statusPillClass`/`alertStyle`/`badge*` 函数名跨 step 一致。

## Part 2 预告（逐屏迁移，待 Part 1 落地后单独写计划）

待本 Part 组件 API 定稿，再写 `2026-06-14-ui-uplift-migration-plan.md`：
- **P1 容器列表**（ContainersView/ContainersTab）：卡片→AppCard、加 StatusPill、空状态→EmptyState、网格固定列、节点数/热键→mono、create 按钮渐变（已由 1.2 全局生效）、在线 tab→EmptyState。
- **P2 计划**（ScheduleListPanel）：空状态→EmptyState、enabled 徽章→StatusPill、时间/次数→mono+tabular-nums。
- **P3 关于**（AboutView）：5 section→AppCard、概念子卡→AppCard、avatar→IconBadge、版本/技术栈→mono。
- **P4 轻触**：LogPanel header/filter token、AppSidebar active 态→淡绿 tint、AppStatusBar 对齐。
- **硬约束**（已在 spec）：概念分类色 / 日志流身份色不动；主壳轻触；画布留 Spec B。
