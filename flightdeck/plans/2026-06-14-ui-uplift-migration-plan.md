---
status: active
summary: "Spec A 实现计划 Part 2：逐屏迁移到 Part 1 共用组件 + v3 token —— P1 容器列表(AppCard/StatusPill/EmptyState/mono/固定列 + 删除 modal 收 useConfirm) · P2 计划(EmptyState/StatusPill/mono) · P3 关于(AppCard×5/IconBadge avatar/mono) · P4 主壳轻触(AppSidebar 绿 active tint / LogPanel filter token / AppStatusBar 验不动)。迁移即升级、删旧散写样式; 离屏视觉自检 + 真机过"
last_updated: 2026-06-14
implements: specs/2026-06-14-ui-uplift-foundation.md
---

# UI 升级逐屏迁移 Implementation Plan (Spec A · Part 2)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把主程序门面四块屏 (容器列表 / 计划 / 关于 / 主壳) 迁到 Part 1 的共用组件 + v3 token，迁移即升级，删掉被替换的散写样式。

**Architecture:** 纯前端 `.vue` 模板迁移，零新依赖、零新组件 (组件都在 Part 1 已落地)。每屏把散写的卡片/空状态/状态徽章换成 `AppCard`/`EmptyState`/`StatusPill`/`IconBadge`，数值/代码/时间戳加 `font-mono` (+ `tabular-nums`)，删掉旧 class。Part 2 实际消费 4 个组件 (AppCard·StatusPill·EmptyState·IconBadge)；AlertBox·SectionHeader·ListRow 等 Spec B 编辑器屏再用，本 Part 不强凑。主按钮渐变 Part 1 已全局生效 (primary+solid)，本 Part 的「新建」按钮自动带渐变，无需改。

**Tech Stack:** Vue 3.5 + NuxtUI v4.7 + Tailwind v4 + Vite 8；i18n = `src/i18n/{zh,en}.ts` 纯 TS object (注意 [vue-i18n message-compiler 陷阱](../checklists/vue-i18n-message-compiler-traps.md))；图标 tabler；字体 Inter / JetBrains Mono (已打包)。

---

## Progress

current: Task 1 — i18n 新键 (containers.empty_cta / status.running / status.idle)

---

## File Structure

**修改 (前端屏)**：
- `frontend/src/i18n/zh.ts` + `frontend/src/i18n/en.ts` — 加 `containers.empty_cta` / `containers.status.{running,idle}`（schedule 状态徽章复用既有 `schedule.enable`/`disable`，无新键）。
- `frontend/src/components/tasks/ContainersTab.vue` — 卡片→`AppCard`+`StatusPill`、节点数/热键 mono、网格固定列、空状态→`EmptyState`+CTA、删除局部 `UModal`→`useConfirm`。
- `frontend/src/views/ContainersView.vue` — 「在线」tab 占位→`EmptyState`。
- `frontend/src/components/schedules/ScheduleListPanel.vue` — 空状态→`EmptyState`、enabled 徽章→`StatusPill`、次数/时间→mono。
- `frontend/src/views/AboutView.vue` — 5 section→`AppCard`、概念子卡→`AppCard`、avatar→`IconBadge`、版本/技术栈值→mono。
- `frontend/src/components/AppSidebar.vue` — active 底色→淡绿 tint。
- `frontend/src/components/LogPanel.vue` — filter 激活态 tint 对齐 v3 标准 (轻改)。

**只读验证（不改）**：`frontend/src/components/AppStatusBar.vue` —— 已对齐 v3 token (semantic 色 + tabular-nums)，本 Part **不动**，Task 8 只眼验确认。

**消费的 Part 1 组件 API（已落地，照用）**：
- `AppCard` — `padding?: 'panel'(p-4)|'section'(p-6)|'none'`（默认 section）；`hover?: boolean`；默认 slot。根是单 div，`@click`/`class` 走 fallthrough 合并。
- `StatusPill` — `status: 'online'|'ready'|'paused'|'failed'`；`label?`；`dot?`；默认 slot 覆盖 label。
- `EmptyState` — `icon: string`；`title: string`；`description?: string`；具名 slot `action`（CTA）。
- `IconBadge` — `icon: string`；`size?: 'sm'|'md'|'lg'`（默认 md）；`color?: 'default'|'primary'|'violet'|'amber'|'sky'`（默认 default）；`shape?: 'square'|'round'`（默认 square）。

**验证**：每屏改完 `pnpm typecheck`；逐屏离屏渲染视觉自检集中在 Task 8（注入 mock store 数据，见 [headless-ui-verify](../checklists/headless-ui-verify.md)）；关键屏真机过（项目铁律）。

---

## Phase 0 · i18n

### Task 1: 加 i18n 新键

**Files:**
- Modify: `frontend/src/i18n/zh.ts`（`containers` 块，`empty_desc: ...` 之后）
- Modify: `frontend/src/i18n/en.ts`（同位置）

> ⚠️ [vue-i18n 陷阱](../checklists/vue-i18n-message-compiler-traps.md)：这几条是纯静态中英文，无 `{` `}` `|` `@` `$`，安全。`zh.ts` 全 plain string。

- [ ] **Step 1: zh.ts 加键**

在 `frontend/src/i18n/zh.ts` 的 `containers:` 块里、`empty_desc: '...'` 那行之后插入：

```typescript
    empty_cta: '新建第一个容器',
    status: {
      running: '运行中',
      idle: '空闲',
    },
```

- [ ] **Step 2: en.ts 加同结构键**

在 `frontend/src/i18n/en.ts` 的 `containers:` 块对应位置（`empty_desc` 后）插入：

```typescript
    empty_cta: 'Create your first container',
    status: {
      running: 'Running',
      idle: 'Idle',
    },
```

- [ ] **Step 3: 验证 i18n 对齐 + 类型**

Run: `cd frontend; pnpm i18n:check; pnpm typecheck`
Expected: i18n:check 不报「键缺失/多出」(zh/en 对齐)；typecheck 干净。
> 注：i18n residue 预存 39 (含 11 有意误报) 是已知账，本步只看**新增键**没引入 zh/en 不对齐。

- [ ] **Step 4: Commit**

```bash
git add frontend/src/i18n/zh.ts frontend/src/i18n/en.ts
git commit -m "i18n(containers): empty CTA + run/idle status pill labels"
```

---

## Phase 1 · P1 容器列表（最高优先）

### Task 2: ContainersTab 卡片 → AppCard + StatusPill + mono + 固定列

**Files:**
- Modify: `frontend/src/components/tasks/ContainersTab.vue`（template 卡片网格 53-122；script import）

- [ ] **Step 1: import 共用组件**

`<script setup>` 顶部（与现有 import 合并，单引号无分号）加：

```typescript
import AppCard from '@/components/common/AppCard.vue'
import StatusPill from '@/components/common/StatusPill.vue'
```

- [ ] **Step 2: 网格改固定列**

第 53 行 `<div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">` 改为：

```html
    <div v-else class="grid grid-cols-2 2xl:grid-cols-3 gap-3">
```

（桌面 app 默认 2 列、超宽 3 列；弃 `grid-cols-1`/`md`/`lg` 任意断点。）

- [ ] **Step 3: 卡片外壳换 AppCard**

把第 54-63 行的卡片根 `<div>`（含 `rounded-xl bg-default border p-4 ...` + `:class` 选中态 + `@click`）整段替换为 `AppCard`。原结构：

```html
      <div
        v-for="c in filtered"
        :key="c.id"
        class="rounded-xl bg-default border p-4 flex flex-col gap-3 transition-colors relative"
        :class="[
          'hover:border-accented',
          batch.isSelected(c.id) ? 'border-primary ring-2 ring-primary/40' : 'border-default',
        ]"
        @click="batch.enabled.value ? batch.toggle(c.id) : undefined"
      >
```

改为（`AppCard` 给 raised 表面 + hover 浮起；选中态用 `!border-primary` 盖掉 raised 的边框色 + ring；旧 `hover:border-accented` 删掉，hover 浮起由 AppCard 接管）：

```html
      <AppCard
        v-for="c in filtered"
        :key="c.id"
        padding="panel"
        hover
        class="flex flex-col gap-3 relative"
        :class="batch.isSelected(c.id) ? '!border-primary ring-2 ring-primary/40' : ''"
        @click="batch.enabled.value ? batch.toggle(c.id) : undefined"
      >
```

对应的卡片闭合 `</div>`（第 121 行）改为 `</AppCard>`。

> ⚠️ 卡片内部结构（checkbox / title / meta / footer 按钮行）**全部保留不动**，只换最外层壳。checkbox 仍 `absolute top-2 left-2`（AppCard 有 `relative`）。

- [ ] **Step 4: 标题行加 StatusPill（运行中/空闲）**

第 72-78 行的标题块：

```html
        <div class="min-w-0">
          <h3 class="text-sm font-medium text-highlighted truncate">
            {{ c.name || t('common.untitled') }}
          </h3>
          <p v-if="c.description" class="text-xs text-dimmed truncate mt-0.5">
            {{ c.description }}
          </p>
```

把 `<h3>` 那行包进一个 flex 行并加 StatusPill：

```html
        <div class="min-w-0">
          <div class="flex items-center justify-between gap-2">
            <h3 class="text-sm font-medium text-highlighted truncate">
              {{ c.name || t('common.untitled') }}
            </h3>
            <StatusPill
              :status="isRunning(c.id) ? 'online' : 'ready'"
              :label="isRunning(c.id) ? t('containers.status.running') : t('containers.status.idle')"
              :dot="isRunning(c.id)"
              class="shrink-0"
            />
          </div>
          <p v-if="c.description" class="text-xs text-dimmed truncate mt-0.5">
            {{ c.description }}
          </p>
```

- [ ] **Step 5: 节点数 / 热键 → mono**

第 80-87 行的两个 meta span，各加 `font-mono tabular-nums`（节点数 span）和 `font-mono`（热键 `<code>`）。节点数 span：

```html
            <span class="text-[11px] text-dimmed inline-flex items-center gap-1 font-mono tabular-nums">
              <UIcon name="i-tabler-cpu" class="size-3" />
              {{ t('containers.node_count', { n: c.graph.nodes.length }) }}
            </span>
```

热键 `<code>`（第 86 行）：

```html
              <code class="text-toned bg-elevated/60 px-1 rounded font-mono">{{ c.hotkey }}</code>
```

- [ ] **Step 6: typecheck**

Run: `cd frontend; pnpm typecheck`
Expected: 干净（视觉验留 Task 8）。

- [ ] **Step 7: Commit**

```bash
git add frontend/src/components/tasks/ContainersTab.vue
git commit -m "feat(ui): P1 container cards → AppCard + StatusPill + mono + fixed grid"
```

---

### Task 3: 容器列表空状态 → EmptyState（ContainersTab + ContainersView 在线 tab）

**Files:**
- Modify: `frontend/src/components/tasks/ContainersTab.vue`（空状态 42-51）
- Modify: `frontend/src/views/ContainersView.vue`（在线 tab 占位 9-13）

- [ ] **Step 1: ContainersTab 空状态换 EmptyState + CTA**

第 42-51 行的空状态散写 `<div>`：

```html
    <div
      v-if="filtered.length === 0"
      class="rounded-xl bg-default/50 border border-default/60 border-dashed py-12 px-6 text-center"
    >
      <UIcon name="i-tabler-schema" class="size-8 text-dimmed mx-auto mb-3" />
      <p class="text-sm text-muted">{{ t('containers.empty_title') }}</p>
      <p class="text-xs text-dimmed mt-1">
        {{ t('containers.empty_desc') }}
      </p>
    </div>
```

替换为：

```html
    <EmptyState
      v-if="filtered.length === 0"
      icon="i-tabler-schema"
      :title="t('containers.empty_title')"
      :description="t('containers.empty_desc')"
    >
      <template #action>
        <UButton color="primary" icon="i-tabler-plus" @click="onCreate">
          {{ t('containers.empty_cta') }}
        </UButton>
      </template>
    </EmptyState>
```

并在 `<script setup>` import 区加：

```typescript
import EmptyState from '@/components/common/EmptyState.vue'
```

- [ ] **Step 2: ContainersView 在线 tab 占位换 EmptyState**

`ContainersView.vue` 第 9-13 行：

```html
    <!-- 在线容器: 占位 (整包容器分享/下载留口, 未实现 — 见 specs/2026-06-13-editor-rail-resources.md ⑥) -->
    <div v-else class="flex flex-col items-center justify-center text-center py-16">
      <UIcon name="i-tabler-cloud" class="size-12 text-dimmed mb-3" />
      <h3 class="text-sm text-toned font-medium">{{ t('containers.online.title') }}</h3>
      <p class="text-xs text-dimmed mt-2 max-w-xs">{{ t('containers.online.desc') }}</p>
    </div>
```

替换为：

```html
    <!-- 在线容器: 占位 (整包容器分享/下载留口, 未实现 — 见 specs/2026-06-13-editor-rail-resources.md ⑥) -->
    <EmptyState
      v-else
      icon="i-tabler-cloud"
      :title="t('containers.online.title')"
      :description="t('containers.online.desc')"
    />
```

`<script setup>` import 区加：

```typescript
import EmptyState from '@/components/common/EmptyState.vue'
```

- [ ] **Step 3: typecheck**

Run: `cd frontend; pnpm typecheck`
Expected: 干净。

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/tasks/ContainersTab.vue frontend/src/views/ContainersView.vue
git commit -m "feat(ui): P1 empty states → EmptyState (container list + online tab)"
```

---

### Task 4: 容器单删局部 UModal → useConfirm（去掉手写 modal）

> spec P1「顺手(不强求)」。`ContainersTab` 已 import `useConfirm` 且批量删用它；单删却另写一个局部 `UModal`(124-148) + `pendingDelete` ref。收口成 `useConfirm`，删手写 modal（二号铁律：一处删干净）。

**Files:**
- Modify: `frontend/src/components/tasks/ContainersTab.vue`

- [ ] **Step 1: 删模板里的局部 UModal**

删第 124-148 行整段 `<UModal :open="!!pendingDelete" ...> ... </UModal>`。

- [ ] **Step 2: onAskDelete 改走 useConfirm**

把 `onAskDelete`（第 268-274 行）+ `onConfirmDelete`（276-281）+ `pendingDelete` ref（196）合并成一个 async 流。`pendingDelete` ref（第 196 行 `const pendingDelete = ref<Container | null>(null)`）删掉。`onAskDelete` 改为：

```typescript
async function onAskDelete(c: Container) {
  if (store.isRecordingLocked(c.id)) {
    toast.add({ title: t('containers.toast.recording_locked'), color: 'warning' })
    return
  }
  const yes = await confirm({
    title: t('containers.delete.title'),
    description: `${t('containers.delete.desc_prefix')}${c.name}${t('containers.delete.desc_suffix')}`,
    color: 'error',
    confirmText: t('containers.delete.confirm'),
  })
  if (yes !== true) return
  await store.remove(c.id)
}
```

并删掉 `onConfirmDelete` 函数（不再被引用）。

> 模板里删除按钮（第 113-119 行）`@click="onAskDelete(c)"` 不变（onAskDelete 现自带确认）。

- [ ] **Step 3: typecheck（确认无悬空引用）**

Run: `cd frontend; pnpm typecheck`
Expected: 干净（`pendingDelete` / `onConfirmDelete` 全删干净，无引用残留）。

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/tasks/ContainersTab.vue
git commit -m "refactor(ui): P1 container single-delete local UModal → useConfirm"
```

---

## Phase 2 · P2 计划

### Task 5: ScheduleListPanel 空状态 + enabled 徽章 + mono

**Files:**
- Modify: `frontend/src/components/schedules/ScheduleListPanel.vue`

- [ ] **Step 1: import EmptyState + StatusPill**

`<script setup>`（第 66-68 行附近，与现有 import 合并）加：

```typescript
import EmptyState from '@/components/common/EmptyState.vue'
import StatusPill from '@/components/common/StatusPill.vue'
```

- [ ] **Step 2: 空状态换 EmptyState**

第 3-12 行：

```html
    <div
      v-if="list.length === 0"
      class="rounded-xl bg-default/50 border border-default/60 border-dashed py-12 px-6 text-center"
    >
      <UIcon name="i-tabler-clock" class="size-8 text-dimmed mx-auto mb-3" />
      <p class="text-sm text-muted">{{ t('schedule.empty') }}</p>
      <p class="text-xs text-dimmed mt-1">
        {{ t('schedule.empty_desc') }}
      </p>
    </div>
```

替换为：

```html
    <EmptyState
      v-if="list.length === 0"
      icon="i-tabler-clock"
      :title="t('schedule.empty')"
      :description="t('schedule.empty_desc')"
    />
```

- [ ] **Step 3: enabled 徽章换 StatusPill**

第 32-43 行的 `<td>`：

```html
          <td class="p-2">
            <span
              class="text-[10px] px-1.5 py-0.5 rounded"
              :class="
                s.enabled
                  ? 'bg-primary/10 text-primary border border-primary/20'
                  : 'bg-elevated text-dimmed'
              "
            >
              {{ s.enabled ? t('schedule.enable') : t('schedule.disable') }}
            </span>
          </td>
```

替换为（enabled→online 绿，disabled→ready 灰；复用既有 `schedule.enable`/`disable` 文案）：

```html
          <td class="p-2">
            <StatusPill
              :status="s.enabled ? 'online' : 'ready'"
              :label="s.enabled ? t('schedule.enable') : t('schedule.disable')"
              :dot="s.enabled"
            />
          </td>
```

- [ ] **Step 4: 次数 / 上次时间 → mono + tabular-nums**

第 28-31 行（次数、上次时间 td）加 `font-mono tabular-nums`：

```html
          <td class="p-2 text-dimmed font-mono tabular-nums">{{ s.targets.length }}</td>
          <td class="p-2 text-dimmed font-mono tabular-nums">
            {{ s.lastFiredAt?.slice(0, 16).replace('T', ' ') ?? '—' }}
          </td>
```

（触发 td 第 27 行是 cron/hotkey 文案，含 CJK，**不加 mono** —— 避免 CJK 落 fallback 字体；spec「触发→mono」此处按可读性优先跳过，只数值/时间戳上 mono。）

- [ ] **Step 5: typecheck**

Run: `cd frontend; pnpm typecheck`
Expected: 干净。

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/schedules/ScheduleListPanel.vue
git commit -m "feat(ui): P2 schedule list → EmptyState + StatusPill + mono metrics"
```

---

## Phase 3 · P3 关于

### Task 6: AboutView 5 section → AppCard + 概念子卡 + avatar IconBadge + mono

**Files:**
- Modify: `frontend/src/views/AboutView.vue`

- [ ] **Step 1: import AppCard + IconBadge**

`<script setup>`（第 129-133 行附近）加：

```typescript
import AppCard from '@/components/common/AppCard.vue'
import IconBadge from '@/components/common/IconBadge.vue'
```

- [ ] **Step 2: 主介绍卡 section → AppCard，avatar → IconBadge**

第 3-17 行：

```html
    <!-- 主介绍卡 -->
    <section class="rounded-xl bg-default border border-default p-8">
      <div class="flex flex-col items-center text-center">
        <div class="size-20 rounded-full bg-elevated flex items-center justify-center mb-5">
          <UIcon name="i-tabler-info-circle" class="size-9 text-muted" />
        </div>
        <h2 class="text-lg font-semibold text-highlighted mb-2">
          {{ info?.name ?? 'Yotta' }}
          <span class="text-muted font-normal ml-1">v{{ info?.version ?? '...' }}</span>
        </h2>
        <p class="text-sm text-muted leading-relaxed max-w-sm">
          {{ t('about.tagline') }}
        </p>
      </div>
    </section>
```

替换为（section→AppCard；avatar→IconBadge 大圆；version 数值→mono tabular-nums）：

```html
    <!-- 主介绍卡 -->
    <AppCard>
      <div class="flex flex-col items-center text-center">
        <IconBadge icon="i-tabler-info-circle" size="lg" shape="round" color="primary" class="mb-5" />
        <h2 class="text-lg font-semibold text-highlighted mb-2">
          {{ info?.name ?? 'Yotta' }}
          <span class="text-muted font-normal ml-1 font-mono tabular-nums">v{{ info?.version ?? '...' }}</span>
        </h2>
        <p class="text-sm text-muted leading-relaxed max-w-sm">
          {{ t('about.tagline') }}
        </p>
      </div>
    </AppCard>
```

> ⚠️ IconBadge `lg` = `size-14`(56px)，比原 `size-20`(80px) 小一档（v3 克制取向，可接受）。Task 8 眼验若觉太小，回退：直接 `<div class="size-20 rounded-full raised-surface flex items-center justify-center mb-5"><UIcon name="i-tabler-info-circle" class="size-9 text-muted" /></div>`（用 raised-surface utility 保 v3 表面、不强用 IconBadge）。

- [ ] **Step 3: 核心概念 section → AppCard，概念子卡 → AppCard**

第 19-35 行。外层 section→AppCard，内层概念子卡 (`rounded-lg bg-default/50 border p-4`) → AppCard `padding="panel"`。**概念 icon 色 (`c.iconClass` = fuchsia/emerald/amber) 是身份识别色，保留不动**（spec 硬约束）：

```html
    <!-- 核心概念 -->
    <AppCard>
      <h3 class="text-xs font-semibold uppercase tracking-wider text-dimmed mb-3">{{ t('about.concepts.title') }}</h3>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
        <AppCard
          v-for="c in concepts"
          :key="c.key"
          padding="panel"
        >
          <div class="flex items-center gap-2 mb-2">
            <UIcon :name="c.icon" class="size-4" :class="c.iconClass" />
            <span class="text-sm font-medium text-highlighted">{{ t(`about.concepts.${c.key}.name`) }}</span>
          </div>
          <p class="text-xs text-muted leading-relaxed">{{ t(`about.concepts.${c.key}.desc`) }}</p>
        </AppCard>
      </div>
    </AppCard>
```

- [ ] **Step 4: 作者 / 技术栈 / 致谢 三 section → AppCard**

第 37-39 行 `<section class="rounded-xl bg-default border border-default p-5">`（作者）→ `<AppCard>`，对应闭合 `</section>`（第 74 行）→ `</AppCard>`。
第 77 行（技术栈 section 同款 class）→ `<AppCard>`，闭合（第 105 行）→ `</AppCard>`。
第 108 行（致谢 section）→ `<AppCard>`，闭合（第 125 行）→ `</AppCard>`。
**section 内部结构全部不动**（h3 标题 + 内容行）。

- [ ] **Step 5: 技术栈值 → mono**

第 80-104 行技术栈 6 个值 span（`<span class="text-default font-medium">Wails 3</span>` 等 6 处）各加 `font-mono`：

```html
          <span class="text-default font-medium font-mono">Wails 3</span>
```

（6 个值依次：`Wails 3` / `Vue 3 + TS` / `NuxtUI v4` / `Go 1.25` / `zerolog` / `pkg/vision + Win32`，每个的 `text-default font-medium` → `text-default font-medium font-mono`。）

- [ ] **Step 6: typecheck**

Run: `cd frontend; pnpm typecheck`
Expected: 干净。

- [ ] **Step 7: Commit**

```bash
git add frontend/src/views/AboutView.vue
git commit -m "feat(ui): P3 About → AppCard sections + IconBadge avatar + mono values"
```

---

## Phase 4 · P4 主壳轻触

### Task 7: AppSidebar active 绿 tint + LogPanel filter token 对齐

**Files:**
- Modify: `frontend/src/components/AppSidebar.vue`（active class 102）
- Modify: `frontend/src/components/LogPanel.vue`（filter 按钮 46）

- [ ] **Step 1: AppSidebar active 底色 → 淡绿 tint**

第 102 行：

```typescript
const activeClass = 'bg-elevated/60 text-highlighted font-medium'
```

改为（左 2px 绿条已是 `bg-primary` ✓；active 底色 → 淡绿 tint，对齐 v3 激活态规范）：

```typescript
const activeClass = 'bg-primary/10 text-highlighted font-medium'
```

- [ ] **Step 2: LogPanel filter 激活态 tint 对齐**

第 46 行 filter 按钮 `:class`：

```html
          :class="filter === opt ? 'bg-primary/20 text-primary' : 'text-dimmed hover:text-toned'"
```

改为（`/20` → `/15` 对齐 StatusPill·active 全局淡绿底标准；其余不动）：

```html
          :class="filter === opt ? 'bg-primary/15 text-primary' : 'text-dimmed hover:text-toned'"
```

> ⚠️ LogPanel body 的日志流身份色 (`sourceClass` cyan/violet SYS/CTR) 是身份识别色，**不动**（spec 硬约束）；本步只碰 filter 按钮 tint。

- [ ] **Step 3: typecheck**

Run: `cd frontend; pnpm typecheck`
Expected: 干净。

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/AppSidebar.vue frontend/src/components/LogPanel.vue
git commit -m "feat(ui): P4 sidebar active green tint + LogPanel filter token align"
```

---

## Phase 5 · 验证收尾

### Task 8: 离屏视觉自检（真实屏 + mock 数据）+ 全量门禁 + 真机清单

**Files:**
- 临时改 `frontend/src/main.ts`（try/catch 包后端 await 让无后端也挂载，验完还原，**不提交**）

- [ ] **Step 1: AppStatusBar 只读眼验确认不动**

打开 `frontend/src/components/AppStatusBar.vue` 通读：确认它已全用 semantic token (`bg-default`/`text-muted`/`text-primary`/`bg-error/15`) + `tabular-nums`，无散写字面色 → **本 Part 不改**（避免过度工程）。若发现散写 zinc/black/white 字面色才动，否则跳过。

- [ ] **Step 2: 起离屏渲染 + 临时让无后端挂载**

按 [headless-ui-verify](../checklists/headless-ui-verify.md)：
- 临时把 `main.ts` 末尾 `await useSettingsStore().load()` 与 `await populateRegistryFromBackend()` 各包 `try {} catch {}`（验完还原，不提交）。
- 起 vite：`cd frontend; pnpm exec vite --port 9246 --strictPort`（后台）。
- 无 Playwright MCP 时用 puppeteer-core + 系统 Edge（`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`）；临时 `pnpm add -D puppeteer-core`，验完 `pnpm remove`。

- [ ] **Step 3: 逐屏注入 mock 数据 + 截图核对**

三块屏走 hash 路由 `/#/containers`、`/#/schedules`、`/#/about`（都经主壳，需 mock store）。用 `page.evaluate` 拿 pinia 灌假数据（见 checklist 第 4 步取 pinia 法），每屏截图 Read 核对：

- **容器列表 `/#/containers`**：灌 `containers.list` = 2-3 个假容器（带 name/description/graph.nodes/hotkey/tags）+ 1 个空 list 验空状态。核对：卡片 v3 轻浮起 + hover 浮起；StatusPill 运行中绿/空闲灰；节点数+热键 mono 对齐；网格 2 列；空 list → EmptyState + 「新建第一个容器」CTA；「在线」tab → EmptyState；create 按钮翡翠渐变。
- **计划 `/#/schedules`**：灌 `schedules.list` = 2 个假计划（一 enabled 一 disabled，带 trigger/targets/lastFiredAt）+ 空 list。核对：空 → EmptyState；enabled 徽章 → StatusPill（绿/灰）；次数+上次时间 mono tabular-nums 列对齐；表格结构保留。
- **关于 `/#/about`**：灌 `appInfo`（或它走 backend.appInfo.info() RPC 拿不到 → 显默认 `Yotta`/`v...`，可接受）。核对：5 卡片 v3 表面；概念子卡 raised；avatar IconBadge 大圆（太小则走 Task 6 Step 2 回退）；版本/技术栈值 mono；概念 icon 身份色（fuchsia/emerald/amber）**仍在**。

Expected: 各屏符合 v3「克制精致」；不符就调对应 .vue class 或回退方案，重渲直到对。

- [ ] **Step 4: 主壳轻触眼验**

任一屏截图核对：AppSidebar 当前项 = 淡绿底 + 左绿条（非旧灰底）；LogPanel filter 激活 = 淡绿 tint；AppStatusBar 不变。

- [ ] **Step 5: 还原 + 清理**

还原 `main.ts`（去掉 try/catch）；`pnpm remove puppeteer-core`；删所有截图 PNG / 临时脚本 / `.playwright-mcp/`。`git status` 确认无临时残留。

- [ ] **Step 6: 全量门禁**

Run: `cd frontend; pnpm test; pnpm typecheck; pnpm build:dev`
Expected: 测试/类型/构建绿（预存失败按 [build](../checklists/build.md) 判：runtime fixture / i18n residue 39 / lint 18，非本轮回归）。
> build:dev 会重生 `components.d.ts`（本 Part 没新增组件，应无新增组件行；若 NuxtUI U* 集或 pnpm hash 漂移，按 Part 1 经验 `git checkout` 后 `pnpm install --frozen-lockfile` 再 build 收敛）。

- [ ] **Step 7: Commit（若 Step 6 有合法生成物变更）**

```bash
git add -A
git commit -m "chore(ui): Part 2 migration visual self-check + full gate green"
```
> 若 Step 5 清理后工作区无残留、Step 6 无生成物变更 → **无可提交，跳过本步**（不造空 commit；二号铁律）。

- [ ] **Step 8: 真机 smoke（项目铁律，交用户或自测）**

`task dev` 起完整 app，肉眼过：① 容器列表卡片/状态药丸/空状态/新建渐变按钮；② 计划列表空态/徽章/对齐；③ 关于页五卡/头像/版本号；④ 侧栏 active 绿、LogPanel filter。一句话验收：四块屏长得像「一个商业品」、无错位/无白底/无丢色。

---

## Self-Review（写计划后自检）

- **Spec 覆盖**：spec ③ P1（容器列表：AppCard✓/StatusPill✓/EmptyState✓/mono✓/固定列✓/在线 tab✓/删除 modal 收口✓）→ Task 2-4；P2（计划：EmptyState✓/StatusPill✓/mono✓/表格保留✓）→ Task 5；P3（关于：AppCard×5✓/概念子卡✓/avatar IconBadge✓/mono✓）→ Task 6；P4（LogPanel filter✓/AppSidebar 绿 active✓/AppStatusBar 不动✓）→ Task 7+Task8.S1。硬约束（概念分类色/日志流身份色不动、主壳轻触）→ Task 6 S3 + Task 7 S2 显式保留。spec ④（常量收敛）= Part 1 已做（子图坐标）+ 单点魔法值按 YAGNI 不收，本 Part 不含。
- **占位符扫描**：无 TBD/TODO；每个 code step 给完整 before→after；IconBadge 尺寸风险标了回退方案（非占位）。
- **类型/命名一致**：组件 prop（`padding`/`hover`/`status`/`label`/`dot`/`icon`/`title`/`description`/`size`/`shape`/`color`）与 Part 1 组件定义一致；i18n 新键 `containers.empty_cta`/`containers.status.{running,idle}` 在 Task 1 定义、Task 2-3 消费，键名一致；复用键 `schedule.enable`/`disable`/`containers.online.*`/`containers.empty_*` 均已存在于现 zh.ts。
- **未消费组件说明**：本 Part 只用 AppCard/StatusPill/EmptyState/IconBadge；AlertBox/SectionHeader/ListRow 留 Spec B 编辑器屏（NodeInspector/VarsPanel/inspector 分区），非遗漏。

## 下一轮（Part 2 落地后）

Spec A 全部完成 → 把 spec flip done + 归档 spec & 两个 plan。然后 Spec B（容器编辑器 壳+面板+全 modal restyle + 布局/UX）第二轮 brainstorm + mockup。
