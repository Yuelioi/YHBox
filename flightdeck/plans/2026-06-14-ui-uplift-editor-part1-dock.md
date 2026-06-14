---
status: active
summary: "Spec B 实现计划 Part 1: 左侧停靠区 (Editor Left Dock)。把 4 个近全屏 explorer modal (节点库/模板/库/Clip) + 2 个抽屉 (变量/Snippets) 收进一个左侧停靠区 — 窄列表(~300px)↔宽资产网格(~600px)自适应, 始终挤画布不盖画布。新建 7 文件 (ContainerEditorDock + NodeLibraryPanel + AssetDockPanel + Template/Library/ClipAssetPanel + useAssetPicker), 删 4 modal, 改状态模型(useSidebarPrefs 加 'nodes'/'assets'+assetTab) + rail + Tab 热键 + 命令面板 + TemplatePickerField 字段唤起路由。复用 Spec A 黑底浮起语言。边界: vue-flow 画布/节点/连线/pin 不碰。"
last_updated: 2026-06-14
implements: specs/2026-06-14-ui-uplift-editor.md
---

# 编辑器左侧停靠区 Implementation Plan (Spec B · Part 1)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把容器编辑器里 4 个近全屏 explorer modal(节点库 / 模板 / 子图库 / Clip)和 2 个抽屉(变量 / Snippets)统一收进**一个左侧停靠区**:窄模式列表(~300px)、资产模式缩略图网格(~600px),宽度自适应且**始终挤画布、不盖画布**。一招同解 Spec B 痛点 1(modal 盖画布)+ 2(边栏挤画布)。

**Architecture:** 纯前端 Vue 重构,零新依赖。停靠区沿用现有 `<aside>` + `SplitHandle` pattern,抽成 `ContainerEditorDock.vue` 外壳(多根 = aside + 拖宽手柄,内含窄/宽两套持久化宽度)。每个 modal 的**内容主体**(去掉 BaseModal 外壳的那段)抽成停靠面板组件;资产三类(模板/库/Clip)由 `AssetDockPanel.vue` 用 `UTabs` 收成一个带 tab 的宽面板。面板切换由 `useSidebarPrefs.leftDrawer`(扩 `'nodes' | 'assets'`)驱动,4 个 rail 图标(节点库/变量/Snippets/资产)切。节点字段唤起选模板(`TemplatePickerField`)改成走 `useAssetPicker` 通道路由到停靠区资产 tab 的 pick 模式,写回路径不变。表面全用 Spec A 的 `raised-surface` / `bg-default`。**边界**: vue-flow 画布、节点框、连线、pin、右键菜单、inline 输入件一律不碰,只随 token 走。

**Tech Stack:** Vue 3.5 `<script setup>` + TS + NuxtUI v4.7(`UTabs`/`UInput`/`UButton`)+ Tailwind v4 + Vite 8 + Pinia + vitest;i18n = `src/i18n/{zh,en}.ts` 纯 TS object;图标 tabler。stores: `useTemplatesStore`(`@/stores/templates`)/ `useLibraryStore`(`@/stores/library`)/ `useClipsStore`(`@/stores/clips`)。

---

## Progress

current: not-started

> 每个 Task 落地后在此追加一行:`T<n> done — <commit>`。全部完成后跑 Task 9 总验证 + 真机 smoke。

---

## File Structure

**新建(7 文件)**:

- `frontend/src/composables/editor/useAssetPicker.ts` — 节点字段→停靠区资产 tab 的 pick 通道(模块单例 reactive request,仿 useSidebarPrefs)。
- `frontend/src/composables/editor/useAssetPicker.spec.ts` — 单测(同目录,仿 useSidebarPrefs.spec)。
- `frontend/src/components/containers/dock/ContainerEditorDock.vue` — 停靠区外壳:`<aside>` + `<SplitHandle>`,窄/宽两套持久化宽度,默认 slot 装当前面板。
- `frontend/src/components/containers/dock/NodeLibraryPanel.vue` — 节点库停靠面板(从 `NodeExplorerModal` body 抽)。
- `frontend/src/components/containers/dock/AssetDockPanel.vue` — 资产停靠面板:`UTabs`(模板/库/Clip)宿主。
- `frontend/src/components/containers/dock/TemplateAssetPanel.vue` — 模板面板(从 `TemplateExplorerModal` body 抽,保留 pickMode/modelValue)。
- `frontend/src/components/containers/dock/LibraryAssetPanel.vue` — 子图库面板(从 `LibraryExplorerModal` body 抽,emit `pick-subgraph`)。
- `frontend/src/components/containers/dock/ClipAssetPanel.vue` — Clip 面板(从 `ClipExplorerModal` body 抽,emit `pick-clip`)。

**修改**:

- `frontend/src/composables/editor/useSidebarPrefs.ts` — `leftDrawer` 联合加 `'nodes' | 'assets'`;新增 `assetTab: 'templates' | 'library' | 'clips'`(默认 `'templates'`)。
- `frontend/src/composables/editor/useSidebarPrefs.spec.ts` — 加 assetTab 默认 + leftDrawer 新值持久化断言。
- `frontend/src/views/ContainerEditorView.vue` — `<aside>`+独立 `SplitHandle` → `<ContainerEditorDock>`;rail 改 4 图标全 dock 化;删 4 modal 挂载 + 4 open ref + `onOpenNodeExplorer`/`onOpenLibraryExplorer` + `leftPane` splitpane;接 `useAssetPicker`;spawn-pos / Tab / 命令面板 / 热键 改连停靠区。
- `frontend/src/components/containers/TemplatePickerField.vue` — 删内嵌 `TemplateExplorerModal`;加 `pin` prop;按钮点击 → `requestTemplatePick`;watch request 镜像回 `update:modelValue`。
- `frontend/src/components/containers/NodeInspector.vue` — 给 `TemplatePickerField` 传 `:pin="lit.name"`(L522-526)。
- `frontend/src/composables/containerEditor/useEditorHotkeys.ts` — opts 去 `nodeExplorerOpen`,加 `isNodeLibraryOpen` / `toggleNodeLibrary`;Tab 分支改连这俩。
- `frontend/src/composables/containerEditor/useCommandPalette.ts` — opts 去 `nodeExplorerOpen` / `libraryExplorerOpen`;`navigate.node-explorer` / `navigate.library` exec 改 mutate `sidebarPrefs`。
- `frontend/src/i18n/zh.ts` + `frontend/src/i18n/en.ts` — 加 `editor.dock.assets`(资产 / Assets)。

**删除(4 文件)**:

- `frontend/src/components/containers/NodeExplorerModal.vue`(body 进 NodeLibraryPanel;Task 4)
- `frontend/src/components/containers/LibraryExplorerModal.vue`(body 进 LibraryAssetPanel;Task 6)
- `frontend/src/components/containers/ClipExplorerModal.vue`(body 进 ClipAssetPanel;Task 6)
- `frontend/src/components/containers/TemplateExplorerModal.vue`(body 进 TemplateAssetPanel;**最后**删 — Task 7 解掉 TemplatePickerField 的内嵌引用后)

**不动(随 token 走,边界外)**: `*DetailPanel.vue`(模板/库/Clip 右栏详情,继续 `w-80`,原样被各 AssetPanel 复用)、`BaseModal.vue`、`SidebarSection.vue`、所有 vue-flow 画布/节点/menus/inline 组件。

**消费的现成 API(已读源码,照用,不重造)**:

- `useSidebarPrefs()` → `{ prefs }`;`prefs` 是模块单例 `ref<SidebarPrefs>`,直接 mutate 单字段(`prefs.value.leftDrawer = 'nodes'`),localStorage key `yotta.editor.sidebar`。
- `useSplitpane(key, { default, min, max })` → `{ width: Ref<number>, setWidth(v) }`;`setWidth` 自 clamp + 落盘。
- `SplitHandle` props: `modelValue:number`(必)/ `min:number`(必)/ `max:number`(必)/ `reverse?`/ `vertical?`;emit `update:modelValue`;自身 clamp,不落盘(靠父接 useSplitpane)。停靠区手柄在 dock 右侧,**不用 reverse**(向右拖=变宽=正向)。
- `ListRow` props `{ active?: boolean }`,slots `icon`/默认/`trailing`;`active`→常驻 `raised-surface`,否则 `hover:raised-surface`。
- `SectionHeader` props `{ title:string; icon?:string; count?:number }`,slot `actions`。
- 表面 utility(`style.css`):`raised-surface`(卡片/面板/ListRow active)、`bg-sunken`(凹槽)、`bg-default`(=`--ui-bg` 黑底)。
- NuxtUI `UTabs` 用法照 `LibraryExplorerModal.vue` L12-18 既有写法(`:items` 带 `{value,label,icon}`、`v-model` 受控、`:content="false"` 关内置面板只当切换器)。

---

## Task 1: 扩展侧栏状态模型(leftDrawer + assetTab)

**Files:**
- Modify: `frontend/src/composables/editor/useSidebarPrefs.ts`
- Test: `frontend/src/composables/editor/useSidebarPrefs.spec.ts`

纯逻辑、模块单例 — TDD。测试用 `vi.resetModules()` + 动态 `await import()` 拿"重新初始化"实例(仿现有 spec)。

- [ ] **Step 1: 先加失败测试**

在 `useSidebarPrefs.spec.ts` 末尾追加(沿用文件顶部既有 `beforeEach(() => { localStorage.clear(); vi.resetModules() })` 与 `KEY` 常量;若文件用的是 import 的 `SIDEBAR_PREFS_KEY`,照它来):

```ts
it('assetTab 默认是 templates', async () => {
  const { useSidebarPrefs } = await import('./useSidebarPrefs')
  const { prefs } = useSidebarPrefs()
  expect(prefs.value.assetTab).toBe('templates')
})

it('leftDrawer 可设为 nodes / assets 并持久化', async () => {
  const { useSidebarPrefs, SIDEBAR_PREFS_KEY } = await import('./useSidebarPrefs')
  const { prefs } = useSidebarPrefs()
  prefs.value.leftDrawer = 'assets'
  prefs.value.assetTab = 'clips'
  await nextTick()
  const saved = JSON.parse(localStorage.getItem(SIDEBAR_PREFS_KEY)!)
  expect(saved.leftDrawer).toBe('assets')
  expect(saved.assetTab).toBe('clips')
})
```

(若 `nextTick` 未 import,在文件顶部 `import { nextTick } from 'vue'`。)

- [ ] **Step 2: 跑测试确认失败**

Run: `pnpm --prefix frontend exec vitest run src/composables/editor/useSidebarPrefs.spec.ts`
Expected: FAIL — `assetTab` undefined / 类型不含 `'assets'`。

- [ ] **Step 3: 改 interface + DEFAULTS**

`useSidebarPrefs.ts` 的 `SidebarPrefs` 接口与 `DEFAULTS`:

```ts
export interface SidebarPrefs {
  leftDrawer: 'vars' | 'snippets' | 'nodes' | 'assets' | null
  assetTab: 'templates' | 'library' | 'clips'
  inspectorCollapsed: boolean
  varsExpanded: boolean
  snapEnabled: boolean
  edgeStyle: 'default' | 'smoothstep' | 'step'
}

const DEFAULTS: SidebarPrefs = {
  leftDrawer: null,
  assetTab: 'templates',
  inspectorCollapsed: false,
  varsExpanded: true,
  snapEnabled: true,
  edgeStyle: 'default',
}
```

(`loadInitial()` 用 `{ ...DEFAULTS }` 打底再浅合并 localStorage — 旧的没有 `assetTab` 的持久值会自动拿到默认 `'templates'`,无需迁移分支。)

- [ ] **Step 4: 跑测试确认通过**

Run: `pnpm --prefix frontend exec vitest run src/composables/editor/useSidebarPrefs.spec.ts`
Expected: PASS。

- [ ] **Step 5: typecheck + commit**

Run: `pnpm --prefix frontend typecheck`(预存 lint 18 错与此无关;typecheck 应 clean)
```bash
git add frontend/src/composables/editor/useSidebarPrefs.ts frontend/src/composables/editor/useSidebarPrefs.spec.ts
git commit -m "feat(editor): extend sidebar prefs with nodes/assets dock states"
```

---

## Task 2: 资产唤起通道 composable(useAssetPicker)

**Files:**
- Create: `frontend/src/composables/editor/useAssetPicker.ts`
- Test: `frontend/src/composables/editor/useAssetPicker.spec.ts`

节点字段(TemplatePickerField)与停靠区资产 tab 跨组件通信用。模块单例 reactive request:字段发起 `requestTemplatePick(pin, selected)` → view 监听打开停靠区资产 tab 的 pick 模式;停靠区切换指派 → `updateSelection` → 字段镜像回写 `config.literal`。写回路径不变(仍 字段 → NodeInspector.setLiteral → view),通道只搬"选模板的 UI"。

- [ ] **Step 1: 先加失败测试**

`useAssetPicker.spec.ts`(同目录,仿 useSidebarPrefs.spec 的 resetModules + 动态 import):

```ts
import { describe, it, expect, beforeEach, vi } from 'vitest'

describe('useAssetPicker', () => {
  beforeEach(() => {
    vi.resetModules()
  })

  it('初始 request 为 null', async () => {
    const { useAssetPicker } = await import('./useAssetPicker')
    expect(useAssetPicker().request.value).toBeNull()
  })

  it('requestTemplatePick 写入 pin + selected 副本', async () => {
    const { useAssetPicker } = await import('./useAssetPicker')
    const { request, requestTemplatePick } = useAssetPicker()
    const src = ['a', 'b']
    requestTemplatePick('Templates', src)
    expect(request.value).toEqual({ pin: 'Templates', selected: ['a', 'b'] })
    src.push('c') // 副本: 不被外部 mutate 影响
    expect(request.value!.selected).toEqual(['a', 'b'])
  })

  it('updateSelection 仅在有 request 时改 selected', async () => {
    const { useAssetPicker } = await import('./useAssetPicker')
    const { request, requestTemplatePick, updateSelection } = useAssetPicker()
    updateSelection(['x']) // 无 request → no-op
    expect(request.value).toBeNull()
    requestTemplatePick('Templates', [])
    updateSelection(['x', 'y'])
    expect(request.value).toEqual({ pin: 'Templates', selected: ['x', 'y'] })
  })

  it('cancel 清空 request', async () => {
    const { useAssetPicker } = await import('./useAssetPicker')
    const { request, requestTemplatePick, cancel } = useAssetPicker()
    requestTemplatePick('Templates', ['a'])
    cancel()
    expect(request.value).toBeNull()
  })
})
```

- [ ] **Step 2: 跑测试确认失败**

Run: `pnpm --prefix frontend exec vitest run src/composables/editor/useAssetPicker.spec.ts`
Expected: FAIL — 模块不存在。

- [ ] **Step 3: 写实现**

`useAssetPicker.ts`:

```ts
import { ref } from 'vue'

export interface AssetPickRequest {
  /** 发起选择的节点 literal pin 名 (如 'Templates') */
  pin: string
  /** 当前已指派的模板 GUID 列表 */
  selected: string[]
}

// 模块单例: 字段与停靠区共享同一份 request (仿 useSidebarPrefs).
const request = ref<AssetPickRequest | null>(null)

/** 节点字段点"选模板" → 请求把停靠区资产 tab 切到 pick 模式. */
function requestTemplatePick(pin: string, selected: string[]) {
  request.value = { pin, selected: [...selected] }
}

/** 停靠区里勾选/取消 → 更新当前选择 (字段会镜像回写). */
function updateSelection(selected: string[]) {
  if (!request.value) return
  request.value = { ...request.value, selected: [...selected] }
}

/** 取消 pick 上下文 (关停靠区 / 切节点). */
function cancel() {
  request.value = null
}

export function useAssetPicker() {
  return { request, requestTemplatePick, updateSelection, cancel }
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `pnpm --prefix frontend exec vitest run src/composables/editor/useAssetPicker.spec.ts`
Expected: PASS(4 项)。

- [ ] **Step 5: commit**

```bash
git add frontend/src/composables/editor/useAssetPicker.ts frontend/src/composables/editor/useAssetPicker.spec.ts
git commit -m "feat(editor): add useAssetPicker channel for field-to-dock template pick"
```

---

## Task 3: 停靠区外壳组件(ContainerEditorDock)

**Files:**
- Create: `frontend/src/components/containers/dock/ContainerEditorDock.vue`

> UI 组件,项目无组件级测试基建(现有 spec 全是 composable)。验证走 typecheck + build + 后续 Task 真机/离屏视觉。本 Task 只建组件、暂不接入,接入在 Task 4。

外壳 = 多根(`<aside>` + `<SplitHandle>`),内含窄/宽两套持久化宽度(资产宽态拖宽不缩窄列表态,反之亦然),默认 slot 装当前面板。面板各自带标题/搜索,外壳**不加 header**(沿用现 `<aside>` L103 约定)。

- [ ] **Step 1: 写组件**

`ContainerEditorDock.vue`:

```vue
<template>
  <!-- 停靠区: 黑底面板挤画布 (不盖). 多根 = aside + 右侧拖宽手柄, 与画布同处编辑器 flex 行. -->
  <aside
    :style="{ width: width + 'px' }"
    class="shrink-0 border-r border-default overflow-hidden flex flex-col bg-default"
  >
    <slot />
  </aside>
  <SplitHandle
    :model-value="width"
    :min="min"
    :max="max"
    @update:model-value="setWidth"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import SplitHandle from '@/components/common/SplitHandle.vue'
import { useSplitpane } from '@/composables/useSplitpane'

const props = defineProps<{ wide: boolean }>()

// 窄模式 (节点库/变量/Snippets 列表) 与 宽模式 (资产缩略图网格) 各自一套宽度,
// 各自持久化、互不挤压.
const narrow = useSplitpane('editor.dock.narrow', { default: 300, min: 240, max: 480 })
const widePane = useSplitpane('editor.dock.wide', { default: 600, min: 420, max: 820 })

const active = computed(() => (props.wide ? widePane : narrow))
const width = computed(() => active.value.width.value)
const min = computed(() => (props.wide ? 420 : 240))
const max = computed(() => (props.wide ? 820 : 480))
function setWidth(v: number) {
  active.value.setWidth(v)
}
</script>
```

- [ ] **Step 2: typecheck**

Run: `pnpm --prefix frontend typecheck`
Expected: clean(组件未被引用也应通过类型检查)。

- [ ] **Step 3: commit**

```bash
git add frontend/src/components/containers/dock/ContainerEditorDock.vue
git commit -m "feat(editor): add ContainerEditorDock shell with narrow/wide widths"
```

---

## Task 4: 节点库停靠面板(NodeLibraryPanel)+ 接入停靠区 + rail 全 dock 化

**Files:**
- Create: `frontend/src/components/containers/dock/NodeLibraryPanel.vue`
- Delete: `frontend/src/components/containers/NodeExplorerModal.vue`
- Modify: `frontend/src/views/ContainerEditorView.vue`
- Modify: `frontend/src/composables/containerEditor/useEditorHotkeys.ts`
- Modify: `frontend/src/composables/containerEditor/useCommandPalette.ts`
- Modify: `frontend/src/i18n/zh.ts` + `frontend/src/i18n/en.ts`

本 Task 之后:节点库 / 变量 / Snippets 三类全在停靠区;资产三类(模板/库/Clip)暂仍是 modal(Task 6 收)。app 全程可用。

### 4a. 抽 NodeLibraryPanel

- [ ] **Step 1: 建 NodeLibraryPanel.vue,搬 NodeExplorerModal body**

把 `NodeExplorerModal.vue` 的**内容主体**(L10-56 那段 `<div class="space-y-3">`,含搜索行 + 树体网格)整段作为新组件模板根。`<script setup>` 几乎原样搬:`query` / `expandedGroups`(localStorage `yotta.explorer.expanded`)/ `filteredGroups` computed / `allSpecs` / `useNodeGroupColor` / `useAutoFocusOnOpen` 全保留。**改动点**:

1. 模板根从 `<BaseModal ...>` 改为内容主体那段 `<div>`,**最外层再包一层 `<div class="flex flex-col h-full min-h-0">`** 让它填满停靠区高度(原 modal 靠 BaseModal 给高度,停靠区里要自管):
   ```vue
   <template>
     <div class="flex flex-col h-full min-h-0">
       <!-- 标题栏 (停靠区无外层 header, 面板自带) -->
       <div class="flex items-center gap-2 border-b border-default px-3 py-2">
         <UIcon name="i-tabler-grid-dots" class="size-4 text-dimmed" />
         <span class="text-sm font-medium">{{ t('nodeExplorer.title') }}</span>
       </div>
       <!-- 原 NodeExplorerModal body (L10-56): 搜索行 + 树体网格, 套进可滚动区 -->
       <div class="flex-1 min-h-0 overflow-y-auto px-3 py-3 space-y-3">
         ...（原 L12-55 内容原样搬入）
       </div>
     </div>
   </template>
   ```
2. 去掉 props.open / emits `update:open` / `useDialogOpen` / `modelOpen`。新 props/emits:
   ```ts
   const emit = defineEmits<{ 'pick-kind': [kind: string] }>()
   ```
   原 modal 内点节点条目处 emit `pick-kind` 的调用不变(只是不再先关 modal)。
3. `useAutoFocusOnOpen`:原是"modal open 时聚焦搜索框 + 清 query"。停靠区改成 `onMounted` 聚焦(面板 mount 即出现);`searchInputRef` 保留。把 `useAutoFocusOnOpen(modelOpen, ...)` 换成 `onMounted(() => { query.value = ''; searchInputRef.value?.$el?.querySelector('input')?.focus() })`(或保留 composable 但传一个 `ref(true)`)。**注意**: 面板靠 `v-if` 挂载/卸载,每次切到 'nodes' 都重新 mount → onMounted 重跑,等价原 open 行为。
4. i18n key 前缀 `nodeExplorer.*` 不变。无 store 依赖(`allSpecs` 静态 import)。

读源码核对:`NodeExplorerModal.vue` 完整读一遍确认 L10-56 边界、`searchInputRef` 用法、pick-kind 调用点,再原样搬。

- [ ] **Step 2: typecheck**

Run: `pnpm --prefix frontend typecheck`
Expected: NodeLibraryPanel clean(此时尚未接入 view,NodeExplorerModal 仍在)。

### 4b. 接入 view + rail 全 dock 化

- [ ] **Step 3: view 顶部 import 调整**

`ContainerEditorView.vue`:
- 加 `import ContainerEditorDock from '@/components/containers/dock/ContainerEditorDock.vue'`
- 加 `import NodeLibraryPanel from '@/components/containers/dock/NodeLibraryPanel.vue'`
- 加 `import { useAssetPicker } from '@/composables/editor/useAssetPicker'`
- 删 `import NodeExplorerModal from '@/components/containers/NodeExplorerModal.vue'`

- [ ] **Step 4: 替换 rail 定义 + 切换逻辑(L635-662)**

把 `leftRail` / `toggleLeftDrawer` / `onRailClick` / `railActive` / `leftPane` 整块换成:

```ts
type DockPanel = 'nodes' | 'vars' | 'snippets' | 'assets'

const leftRail = [
  { key: 'nodes', icon: 'i-tabler-grid-dots', title: t('editor.toolbar.node_explorer') },
  { key: 'vars', icon: 'i-tabler-variable', title: t('var.title') },
  { key: 'snippets', icon: 'i-tabler-bookmarks', title: 'Snippets' },
  { key: 'assets', icon: 'i-tabler-stack-2', title: t('editor.dock.assets') },
] as const

const { request: assetPickRequest, updateSelection: updateAssetPick, cancel: cancelAssetPick } = useAssetPicker()

function toggleDock(key: DockPanel) {
  const next = sidebarPrefs.value.leftDrawer === key ? null : key
  // 离开资产 tab → 取消可能挂着的字段 pick 上下文
  if (sidebarPrefs.value.leftDrawer === 'assets' && next !== 'assets') cancelAssetPick()
  sidebarPrefs.value.leftDrawer = next
}
function onRailClick(item: (typeof leftRail)[number]) {
  toggleDock(item.key)
}
function railActive(item: (typeof leftRail)[number]): boolean {
  return sidebarPrefs.value.leftDrawer === item.key
}
```

删掉 `const leftPane = useSplitpane('editor.splitpane.left', ...)`(宽度移进 dock 组件)。`rightPane` 保留不动。

- [ ] **Step 5: 替换停靠区模板(L104-133)**

把原 `<aside v-if="sidebarPrefs.leftDrawer">...VarsPanel/SnippetsPanel...</aside>` + 紧随的独立 `<SplitHandle>` 整块换成:

```vue
<ContainerEditorDock
  v-if="sidebarPrefs.leftDrawer"
  :wide="sidebarPrefs.leftDrawer === 'assets'"
>
  <NodeLibraryPanel
    v-if="sidebarPrefs.leftDrawer === 'nodes'"
    @pick-kind="(k: string) => onPickKind(k, nodeExplorerSpawnPos)"
  />
  <VarsPanel
    v-else-if="sidebarPrefs.leftDrawer === 'vars'"
    :vars="draft?.vars ?? []"
    :usage-count="totalVarUsageCount"
    v-model:expanded="sidebarPrefs.varsExpanded"
    @add-var="onAddVar"
    @rename-var="onRenameVar"
    @update-var-field="onUpdateVarField"
    @request-delete="onRequestDeleteVar"
    @reorder-vars="onReorderVars"
    @insert-incvar="onInsertIncVar"
  />
  <SnippetsPanel
    v-else-if="sidebarPrefs.leftDrawer === 'snippets'"
    @apply="onApplySnippet"
    @edit="onEditSnippet"
  />
  <!-- AssetDockPanel 在 Task 6 接入 -->
</ContainerEditorDock>
```

(VarsPanel / SnippetsPanel 的 props/emits 原样不变,只是父容器从 `<aside>` 换成 `<ContainerEditorDock>` 的 slot。)

- [ ] **Step 6: 删 NodeExplorerModal 挂载(L279-282)+ open ref + spawn-pos watch**

- 删模板里 `<NodeExplorerModal v-model:open="nodeExplorerOpen" @pick-kind=... />`(L279-282)。
- 删 `const nodeExplorerOpen = ref(false)`(L664)。
- 删 `function onOpenNodeExplorer() {...}`(L851-853)。
- spawn-pos 快照:把 `watch(nodeExplorerOpen, (open) => { if (open) nodeExplorerSpawnPos.value = {...lastMousePos.value} })`(L785-787)改成:
  ```ts
  watch(() => sidebarPrefs.value.leftDrawer, (d) => {
    if (d === 'nodes') nodeExplorerSpawnPos.value = { ...lastMousePos.value }
  })
  ```
  (`nodeExplorerSpawnPos` ref 与 `onPickKind` 调用保持。)

- [ ] **Step 7: 改 useEditorHotkeys 接线(view L1036-1041)**

view 里把 `nodeExplorerOpen` 从 opts 去掉,加节点库 toggle:
```ts
useEditorHotkeys({
  commandPaletteOpen, nodeSearchOpen, settingsOpen,
  dirty, onSave, undo, redo,
  togglePalette: () => toggleDock('vars'),
  toggleInspector: () => { sidebarPrefs.value.inspectorCollapsed = !sidebarPrefs.value.inspectorCollapsed },
  isNodeLibraryOpen: () => sidebarPrefs.value.leftDrawer === 'nodes',
  toggleNodeLibrary: () => toggleDock('nodes'),
})
```
(原 `togglePalette: () => toggleLeftDrawer('vars')` → `toggleDock('vars')`,Alt+1 仍切变量抽屉,行为不变。)

`useEditorHotkeys.ts`:`UseEditorHotkeysOpts` 去 `nodeExplorerOpen: Ref<boolean>`,加:
```ts
  isNodeLibraryOpen: () => boolean
  toggleNodeLibrary: () => void
```
Tab 分支(L96-105)改:
```ts
    if (e.key === 'Tab') {
      if (opts.isNodeLibraryOpen()) {
        e.preventDefault()
        opts.toggleNodeLibrary() // 已开 → 收起
        return
      }
      if (isTypingTarget(e)) return
      e.preventDefault()
      opts.toggleNodeLibrary() // 未开 → 展开
    }
```

- [ ] **Step 8: 改 useCommandPalette 接线(view + composable)**

view 调用 `useCommandPalette` 处去掉传 `nodeExplorerOpen`(`libraryExplorerOpen` 在 Task 6 一并去,本 Task 先把 node-explorer 改掉、library 暂留)。
`useCommandPalette.ts`:
- `navigate.node-explorer` exec(L155)改:`exec: () => { opts.sidebarPrefs.value.leftDrawer = opts.sidebarPrefs.value.leftDrawer === 'nodes' ? null : 'nodes' }`
- opts 接口去 `nodeExplorerOpen: Ref<boolean>`(L25)。

(`navigate.library` 与 `libraryExplorerOpen` 留到 Task 6 改。)

- [ ] **Step 9: 加 i18n key**

`zh.ts`:`editor.dock.assets` 区段加 `assets: '资产'`(放进 `editor.dock = { assets: '资产' }`,若 `editor.dock` 不存在则新建该对象)。
`en.ts`:对应 `assets: 'Assets'`。

- [ ] **Step 10: typecheck + i18n check**

Run: `pnpm --prefix frontend typecheck`
Expected: clean(若仍引用已删的 `nodeExplorerOpen` 会在此暴露,逐处清掉)。
Run: `pnpm --prefix frontend run i18n:check`
Expected: 无新增缺键(`editor.dock.assets` zh/en 都在)。

- [ ] **Step 11: 删文件 + 离屏视觉自检 + commit**

```bash
git rm frontend/src/components/containers/NodeExplorerModal.vue
```
离屏视觉自检(参 `flightdeck/checklists/headless-ui-verify.md`):起 vite,进编辑器,点 rail 节点库/变量/Snippets 三图标,确认停靠区滑出、挤画布不盖、Tab 开收节点库、宽度可拖。截图亲眼核黑底浮起对、无错位。
```bash
git add -A
git commit -m "feat(editor): node library + vars + snippets into left dock, rail dock-ified"
```

---

## Task 5: 三个资产停靠面板(Template / Library / Clip AssetPanel)

**Files:**
- Create: `frontend/src/components/containers/dock/TemplateAssetPanel.vue`(从 `TemplateExplorerModal.vue` body 抽)
- Create: `frontend/src/components/containers/dock/LibraryAssetPanel.vue`(从 `LibraryExplorerModal.vue` body 抽)
- Create: `frontend/src/components/containers/dock/ClipAssetPanel.vue`(从 `ClipExplorerModal.vue` body 抽)

> 本 Task 只建 3 个面板(未接入,modal 仍在)。三者都是"把 BaseModal 外壳剥掉、body 当根、自管高度"的机械搬运;内部 filter / 批量操作 / 各自 `*DetailPanel`(`w-80`)/ store / RPC **全保留不动**。

### 5a. TemplateAssetPanel

- [ ] **Step 1: 搬 body**

读 `TemplateExplorerModal.vue` 全文。搬法:
- 模板根从 `<BaseModal v-model:open="modelOpen" size="5xl" ...>` 改成包一层 `<div class="flex flex-col h-full min-h-0">`;内放:
  - 顶部工具行:把原 `#header-extra` 的"新建截图"按钮(L6-10)移到这里一行。
  - body 主体 = 原 L12-130 那段 `<div class="flex gap-4">`(左列表 + 右 `<TemplateDetailPanel>`),外面套 `<div class="flex-1 min-h-0 overflow-hidden">`。左列表原 `max-h-[56vh]`、detail 原 `max-h-[65vh]` 改成 `h-full overflow-y-auto`(停靠区自管高度,不再用 vh 写死)。
- **保留** props/emits 契约(pick 模式要):
  ```ts
  defineProps<{ pickMode?: boolean; modelValue?: string[] }>()
  const emit = defineEmits<{ 'update:modelValue': [v: string[]] }>()
  ```
  去掉 `open` / `update:open` / `useDialogOpen` / `modelOpen`。原模板里 `<BaseModal v-model:open>` 删掉;`toggleAssign` 等用 `props.modelValue` / `emit('update:modelValue', ...)` 的逻辑不变。
- 平级的两个批量操作 BaseModal(加标签 L134-140 / 改分类 L143-149)**保留**,继续作为面板内嵌小 modal(它们是真 modal,合理)。
- store(`useTemplatesStore`)、`backend.assets.*` RPC、`tplStore.containerId` 全保留。`TemplateDetailPanel` import 保留。

- [ ] **Step 2: typecheck**

Run: `pnpm --prefix frontend typecheck`
Expected: TemplateAssetPanel clean。

### 5b. LibraryAssetPanel

- [ ] **Step 3: 搬 body**

读 `LibraryExplorerModal.vue` 全文。搬法同上:
- 根 → `<div class="flex flex-col h-full min-h-0">`。
- 原 `#header-extra` 的本地/在线 `UTabs`(L12-18)移到面板顶部一行(保留 local/online 切换 + online 占位逻辑 L22-26)。
- 本地 body = 原 L29-178 那段 `<div v-else class="flex gap-4">`(左列表含右键 `UContextMenu` + 右 `<LibraryDetailPanel>`),套 `<div class="flex-1 min-h-0 overflow-hidden">`,`max-h-[56vh]`/`max-h-[65vh]` → `h-full overflow-y-auto`。
- props/emits:
  ```ts
  defineProps<{ }>() // 无 props
  const emit = defineEmits<{ 'pick-subgraph': [libraryID: string] }>()
  ```
  去 `open`/`update:open`/`useDialogOpen`。原 `onPick(id)` 里"emit 后关 modal"的关 modal 那句删掉,只留 `emit('pick-subgraph', id)`(关停靠区由父决定;插完节点是否收停靠区见 Task 6 接线)。
- 批量 modal(L182-196 / L199-211)保留。store(`useLibraryStore`)+ `backend.subgraphs.updateSilent` 保留。

- [ ] **Step 4: typecheck**

Run: `pnpm --prefix frontend typecheck`
Expected: clean。

### 5c. ClipAssetPanel

- [ ] **Step 5: 搬 body**

读 `ClipExplorerModal.vue` 全文。搬法:
- 根 → `<div class="flex flex-col h-full min-h-0">`。无 header-extra。
- body = 原 L7-116 那段 `<div class="flex gap-4">`(左列表 + 右 `<ClipDetailPanel>`),`max-h-*` → `h-full overflow-y-auto`。
- props/emits:
  ```ts
  const emit = defineEmits<{ 'pick-clip': [clipID: string] }>()
  ```
  去 `open`/`update:open`/`useDialogOpen`;`onPick(id)` 去掉关 modal 句,留 `emit('pick-clip', id)`。
- 批量 modal(L120-126 / L129-135)保留。store(`useClipsStore`)+ `backend.clipsContainer.*` 保留。

- [ ] **Step 6: typecheck + commit**

Run: `pnpm --prefix frontend typecheck`
Expected: clean(3 面板都未接入,modal 仍并存,应全过)。
```bash
git add frontend/src/components/containers/dock/TemplateAssetPanel.vue frontend/src/components/containers/dock/LibraryAssetPanel.vue frontend/src/components/containers/dock/ClipAssetPanel.vue
git commit -m "feat(editor): extract template/library/clip explorer bodies into dock panels"
```

---

## Task 6: 资产停靠面板宿主(AssetDockPanel)+ 接入 + 删 Library/Clip modal

**Files:**
- Create: `frontend/src/components/containers/dock/AssetDockPanel.vue`
- Delete: `frontend/src/components/containers/LibraryExplorerModal.vue`
- Delete: `frontend/src/components/containers/ClipExplorerModal.vue`
- Modify: `frontend/src/views/ContainerEditorView.vue`
- Modify: `frontend/src/composables/containerEditor/useCommandPalette.ts`

本 Task 之后:资产三类进停靠区资产 tab;Library/Clip modal 删除;**TemplateExplorerModal.vue 暂留**(TemplatePickerField 仍内嵌它,Task 7 解掉后再删)。

- [ ] **Step 1: 写 AssetDockPanel**

```vue
<template>
  <div class="flex flex-col h-full min-h-0">
    <!-- 资产三类切换: 模板(缩略图) / 子图库 / Clip -->
    <UTabs
      :model-value="tab"
      :items="tabItems"
      :content="false"
      size="sm"
      class="px-2 pt-2 shrink-0"
      @update:model-value="(v: string | number) => emit('update:tab', v as AssetTab)"
    />
    <div class="flex-1 min-h-0 overflow-hidden">
      <TemplateAssetPanel
        v-if="tab === 'templates'"
        :pick-mode="templatePickMode"
        :model-value="templateSelected"
        @update:model-value="(v: string[]) => emit('update:template-selected', v)"
      />
      <LibraryAssetPanel
        v-else-if="tab === 'library'"
        @pick-subgraph="(id: string) => emit('pick-subgraph', id)"
      />
      <ClipAssetPanel
        v-else-if="tab === 'clips'"
        @pick-clip="(id: string) => emit('pick-clip', id)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import TemplateAssetPanel from './TemplateAssetPanel.vue'
import LibraryAssetPanel from './LibraryAssetPanel.vue'
import ClipAssetPanel from './ClipAssetPanel.vue'

type AssetTab = 'templates' | 'library' | 'clips'

defineProps<{
  tab: AssetTab
  templatePickMode: boolean
  templateSelected: string[]
}>()
const emit = defineEmits<{
  'update:tab': [v: AssetTab]
  'update:template-selected': [v: string[]]
  'pick-subgraph': [id: string]
  'pick-clip': [id: string]
}>()

const { t } = useI18n()
const tabItems = computed(() => [
  { value: 'templates', label: t('template.manager.title'), icon: 'i-tabler-photo' },
  { value: 'library', label: t('editor.toolbar.library_explorer'), icon: 'i-tabler-books' },
  { value: 'clips', label: t('clip.manager.title'), icon: 'i-tabler-movie' },
])
</script>
```

> `UTabs` 的确切 prop(`:items` 项形状 / `:content` / `v-model` 值类型)以 `LibraryExplorerModal.vue` 原 L12-18 既有写法为准 — 实现时对照那段,签名不符就照它改(别脑补 NuxtUI API)。

- [ ] **Step 2: 接入 view 停靠区**

`ContainerEditorView.vue`:
- import `AssetDockPanel`。
- 在 Task 4 留的 `<!-- AssetDockPanel 在 Task 6 接入 -->` 处补:
  ```vue
  <AssetDockPanel
    v-else-if="sidebarPrefs.leftDrawer === 'assets'"
    v-model:tab="sidebarPrefs.assetTab"
    :template-pick-mode="!!assetPickRequest"
    :template-selected="assetPickRequest?.selected ?? []"
    @update:template-selected="updateAssetPick"
    @pick-subgraph="onPickLibrarySubgraph"
    @pick-clip="onPickLibraryClip"
  />
  ```
  (`onPickLibrarySubgraph` / `onPickLibraryClip` 来自 useNodeCreation,保留;它们插完节点不再关 modal — 停靠区维持打开,符合"就近浏览继续"。)

- [ ] **Step 3: 删 Library/Clip modal 挂载 + open ref**

- 删模板里 `<LibraryExplorerModal v-model:open="libraryExplorerOpen" @pick-subgraph=.../>`(L294-297)、`<ClipExplorerModal v-model:open="clipsExplorerOpen" @pick-clip=.../>`(L301)。
- 删 import `LibraryExplorerModal` / `ClipExplorerModal`。
- 删 `const libraryExplorerOpen = ref(false)`、`const clipsExplorerOpen = ref(false)`(L667-668)。
- 删 `function onOpenLibraryExplorer() {...}`(L855-857)。
- **TemplateExplorerModal 挂载(L299,管理入口那个无 pickMode 实例)删掉**;`const templatesExplorerOpen = ref(false)`(L667)删掉。但 **import TemplateExplorerModal 暂留**(TemplatePickerField 还引;实际上 view 不再直接用,可删 view 里的 import —— TemplatePickerField 自己 import 自己的)。核对:view 删 `import TemplateExplorerModal`。

- [ ] **Step 4: 改 useCommandPalette library 命令**

`useCommandPalette.ts`:
- `navigate.library` exec(L161)改:`exec: () => { opts.sidebarPrefs.value.leftDrawer = 'assets'; opts.sidebarPrefs.value.assetTab = 'library' }`
- opts 接口去 `libraryExplorerOpen: Ref<boolean>`(L26)。
view 调用处去掉传 `libraryExplorerOpen`。

- [ ] **Step 5: typecheck**

Run: `pnpm --prefix frontend typecheck`
Expected: clean(残留对 `libraryExplorerOpen`/`clipsExplorerOpen`/`templatesExplorerOpen` 的引用会在此暴露,清掉)。

- [ ] **Step 6: 删 modal + 视觉自检 + commit**

```bash
git rm frontend/src/components/containers/LibraryExplorerModal.vue frontend/src/components/containers/ClipExplorerModal.vue
```
离屏视觉自检:点 rail 资产图标 → 停靠区切宽(~600px)、模板/库/Clip tab 切换、双击库项/Clip 插节点、命令面板 navigate.library 跳资产库 tab。截图核 tab 对齐、缩略图网格、detail 栏不挤崩。
```bash
git add -A
git commit -m "feat(editor): asset explorers into tabbed dock panel, drop library/clip modals"
```

---

## Task 7: 节点字段唤起路由到停靠区(TemplatePickerField)+ 删 TemplateExplorerModal

**Files:**
- Modify: `frontend/src/components/containers/TemplatePickerField.vue`
- Modify: `frontend/src/components/containers/NodeInspector.vue`
- Modify: `frontend/src/views/ContainerEditorView.vue`
- Delete: `frontend/src/components/containers/TemplateExplorerModal.vue`

把"点字段→弹大 modal 选模板"改成"点字段→停靠区资产 tab 切模板 + pick 模式";写回路径(字段 `update:modelValue` → NodeInspector.setLiteral → view)**不变**。

- [ ] **Step 1: TemplatePickerField 改唤起**

读 `TemplatePickerField.vue` 全文(确认 L13 按钮 `@click="open=true"`、L29 内嵌 modal、L62 `open` ref、L70-72 `onUpdate`、L64-69 modelValue 归一)。改:
- 加 `pin` prop:
  ```ts
  const props = defineProps<{ modelValue?: string[] | string; pin: string }>()
  ```
- 删内嵌 `<TemplateExplorerModal v-model:open="open" pick-mode .../>`(L29)+ `import TemplateExplorerModal`(L55)+ `const open = ref(false)`(L62)。
- 按钮 `@click` 改:
  ```ts
  import { useAssetPicker } from '@/composables/editor/useAssetPicker'
  const { request: pickRequest, requestTemplatePick } = useAssetPicker()
  function openPicker() {
    requestTemplatePick(props.pin, selected.value)
  }
  ```
  按钮 `@click="open = true"` → `@click="openPicker"`。
- 镜像回写(停靠区改选择 → 回写字段 → NodeInspector):
  ```ts
  import { watch } from 'vue'
  watch(pickRequest, (req) => {
    if (req && req.pin === props.pin) emit('update:modelValue', req.selected)
  })
  ```
  (`emit('update:modelValue', ...)` 与原 `onUpdate` 同;`onUpdate` 可删或保留无引用即删。`selected` computed / `firstThumb` / chips / 缺失标红 全保留不变。)

> 无环验证:停靠区 toggle → AssetDockPanel emit `update:template-selected` → view `updateAssetPick` → `request.selected` 变 → 字段 watch → `emit update:modelValue` → NodeInspector.setLiteral → node config 变 → 字段 `modelValue` prop 变 → `selected` 重算(=同值)。字段只读 request、不反写,无回环。

- [ ] **Step 2: NodeInspector 传 pin**

`NodeInspector.vue` L522-526 给 `<TemplatePickerField>` 加 `:pin`:
```vue
          <TemplatePickerField
            v-else-if="fieldFor(lit.name)?.widgetKind === 'template-picker'"
            :model-value="asTemplateList(getLiteral(lit.name))"
            :pin="lit.name"
            @update:model-value="(v: string[]) => setLiteral(lit.name, v)"
          />
```

- [ ] **Step 3: view 接 pick 上下文取消**

`ContainerEditorView.vue`:加 watch,字段发起 pick → 自动开停靠区资产 tab;切节点 → 取消 pick:
```ts
watch(assetPickRequest, (req) => {
  if (req) {
    sidebarPrefs.value.leftDrawer = 'assets'
    sidebarPrefs.value.assetTab = 'templates'
  }
})
watch(selectedID, () => { if (assetPickRequest.value) cancelAssetPick() })
```
(`assetPickRequest` / `cancelAssetPick` 已在 Task 4 解构。)

- [ ] **Step 4: typecheck**

Run: `pnpm --prefix frontend typecheck`
Expected: clean。若 TemplatePickerField 仍有 `TemplateExplorerModal` 残留引用 → 清。

- [ ] **Step 5: 删 modal + 视觉自检 + commit**

```bash
git rm frontend/src/components/containers/TemplateExplorerModal.vue
```
离屏视觉自检 / 真机:选一个含模板字段的节点(如 WaitTemplate)→ 点字段"选模板"→ 停靠区自动切到资产·模板 tab + pick 模式 → 勾选模板 → 字段 chips 同步更新 → 切到别的节点 pick 上下文自动取消。
```bash
git add -A
git commit -m "feat(editor): route template picker field to dock asset tab, drop template modal"
```

---

## Task 8: 死状态清扫 + 全局一致性核对

**Files:**
- Modify: `frontend/src/views/ContainerEditorView.vue`(及任何残留)

把前面各 Task 顺手没清干净的死代码一次性扫掉(二号铁律:不留 shim / 死 ref)。

- [ ] **Step 1: grep 残留引用**

Run(逐个确认已无引用):
- `nodeExplorerOpen` / `libraryExplorerOpen` / `templatesExplorerOpen` / `clipsExplorerOpen`
- `onOpenNodeExplorer` / `onOpenLibraryExplorer`
- `leftPane`(原 `editor.splitpane.left`)
- `toggleLeftDrawer`(应已全换成 `toggleDock`)
- `NodeExplorerModal` / `TemplateExplorerModal` / `LibraryExplorerModal` / `ClipExplorerModal`(import 与挂载应全删)

用 Grep 工具搜 `frontend/src`,每个都应只在"已删除"语境无命中(或仅注释)。有残留就删。

- [ ] **Step 2: localStorage 旧 key 说明**

旧 `editor.splitpane.left` 持久值成孤儿(无害,不再读)。**不写迁移/清理代码**(二号铁律);停靠区用新 key `editor.dock.narrow` / `editor.dock.wide`。

- [ ] **Step 3: typecheck + 全量 vitest**

Run: `pnpm --prefix frontend typecheck`
Expected: clean。
Run: `pnpm --prefix frontend test`
Expected: 全绿(新增 useSidebarPrefs / useAssetPicker 用例 + 既有用例;按 cockpit「已知预存失败」判 — runtime fish fixture 缺失类非回归)。

- [ ] **Step 4: commit**

```bash
git add -A
git commit -m "chore(editor): sweep dead explorer-modal state after dock migration"
```

---

## Task 9: 总验证 + 真机 smoke

**Files:** 无(纯验证)。

- [ ] **Step 1: 静态全绿**

Run: `pnpm --prefix frontend typecheck` → clean
Run: `pnpm --prefix frontend test` → 全绿
Run: `pnpm --prefix frontend run build:dev` → 成功出包
Run: `pnpm --prefix frontend run i18n:check` → 无新增缺键
(`pnpm lint` 预存 18 错为既有漂移,不在本 Part 范围 — 不因它判红,但别新增。)

- [ ] **Step 2: 离屏视觉自检(全停靠区矩阵)**

参 `flightdeck/checklists/headless-ui-verify.md`,逐项截图核:
- rail 4 图标(节点库/变量/Snippets/资产)active 高亮(绿 tint)对。
- 节点库 / 变量 / Snippets 窄态(~300px);资产态自动加宽(~600px),拖宽各自独立记忆。
- 资产三 tab 切换:模板缩略图、库列表 + detail、Clip 列表 + detail,detail `w-80` 不挤崩。
- 黑底 + 顶光浮起一致(`raised-surface`/`bg-default`),无整面提亮、无错位、无丢身份色。
- 停靠区**挤画布不盖画布**:画布随停靠区开合变窄/变宽,canvas 内容不被遮。

- [ ] **Step 3: 真机 smoke(task dev)**

起 app 进编辑器走一遍,像商业品、无错位/盖画布/丢色,且**画布交互不回归**:
- rail 点开/同图标收起每个面板;Tab 开收节点库;Alt+1 切变量。
- 节点库搜索 + 点节点落画布(spawn-pos 对:落在视口/指针位)。
- 资产:切 tab、双击库项/Clip 插节点、命令面板 navigate.library 跳库 tab。
- 字段唤起:选 WaitTemplate 节点 → 字段"选模板"→ 停靠区资产·模板 pick 模式 → 勾选同步 chips → 切节点 pick 自动取消。
- **画布不回归**:拖节点、连线、缩放、子图进出、右键菜单、inline pin 编辑 全照旧(边界外组件未碰,应零影响)。

- [ ] **Step 4: 落地**

全绿后走 `/flightdeck:landing`:plan flip `done`、归档、cockpit 同步、Part 2 排上 `## 下一步`。真机 smoke 若留待用户跑,在 frontmatter 标 `verify:`。

---

## Self-Review(写完核对 spec 覆盖)

对照 `specs/2026-06-14-ui-uplift-editor.md` §1 面板信息架构:

- ✅ 节点库 modal → 停靠(窄)— Task 4。
- ✅ 变量 / Snippets 抽屉 → 停靠(窄)— Task 4(本就在 aside,改挂 dock)。
- ✅ 资产(模板/库/Clip 三合一带 tab)→ 停靠(宽 ~600px 网格)— Task 5+6。
- ✅ rail 4 图标切换 + 同图标收起 + 可拖宽 — Task 4(rail)+ Task 3(SplitHandle)。
- ✅ 资产 tab 自动加宽、仍 docked 挤画布 — Task 3(narrow/wide 双宽度)。
- ✅ 资产浏览器从节点字段唤起也定位到停靠区资产 tab(非另弹大 modal)— Task 7。
- ✅ 替掉对应 modal/抽屉(删 4 modal)— Task 4/6/7。

**边界守住**: 全程不碰 vue-flow 画布/节点/连线/pin/右键/inline(Task 9 Step 3 明确回归核对)。**复用 Spec A**: 表面 `raised-surface`/`bg-default`,后续 Inspector 分组(SectionHeader)/校验提示(AlertBox)留 Part 2。

**未纳入本 Part(留 Part 2/3,符合 spec 分期)**: Toolbar 三区重组、底部问题条、Inspector 三态/分组、剩余小弹窗 restyle。
