---
status: done
summary: 9 任务全完 — Go category 字段 + FE 类型/i18n + libraryFilter/useListSelection TDD + 详情/批量面板 + modal 改版(选中模型+过滤+批量) + 属性面板分类。代码全绿, 差真机。
last_updated: 2026-06-12
implements: specs/2026-06-12-library-modal-interaction.md
verify: 子图库 modal 真机验收(原③④⑤⑥的 编辑信息按钮/批量面板形态 已被 2026-06-12-library-modal-polish 美化 v3 取代, 批量与编辑以 v3 清单为准) — ①单击=选中出详情不插入, 悬停不再变右栏 ②双击插入引用+缺变量自动补不回归 ⑦分组按分类(空=未分类), 分类下拉+标签多选+文本搜索三过滤叠加 ⑧编辑器内子图属性面板能改分类, 与库内互通 ⑨右键菜单(插入/复制为新/复制ID/删除)与在线 tab 占位照旧
---

# 子图库 modal 交互改版 — 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 子图库 modal 从「悬停详情+单击插入」改为「单击选中+双击/按钮插入」, 加 Ctrl/Shift 多选 + 批量删除/加标签 + 独立分类字段 + 分类/标签过滤器。

**Architecture:** 选中逻辑抽 `useListSelection` 组合式 (纯逻辑, vitest 单测); 过滤/分组抽 `lib/libraryFilter.ts` 纯函数 (单测); modal 只做粘合。后端只加一个 `Category` 字段 — Update 是整结构体 RFC7386 merge patch (service_subgraph.go:91, 无白名单), 无需新 RPC。

**Tech Stack:** Go (wails3) + Vue 3 + Nuxt UI v4 + vitest。规范前置: [checklists/ui.md](../checklists/ui.md) (semantic token / BaseModal / 无成功 toast) · [checklists/code-style.md](../checklists/code-style.md) · [checklists/commits.md](../checklists/commits.md) · i18n 写 zh/en 前过 [checklists/vue-i18n-message-compiler-traps.md](../checklists/vue-i18n-message-compiler-traps.md) (本计划文案已避开裸 `{` `}` `|` `@` `$`)。

**验证命令** (frontend/ 下 pnpm; 仓根 go): `pnpm typecheck` · `pnpm test` · `pnpm lint` (预存 18 oxlint 错, 按 cockpit 口径判红) · `pnpm i18n:check` · `go build ./...`。**默认不 push** (内测期铁律)。

---

### Task 1: Go — Subgraph 加 Category 字段

**Files:**
- Modify: `internal/services/container/model.go:144` (Tags 行后)

无 Go 单测: 纯数据字段, merge patch 路径是泛化代码 (service_subgraph.go Update 对整结构体操作); 为序列化写 marshal/unmarshal 测试是空泛测试 (参 incidents/2026-05-26-vacuous-defensive-cleanup-test)。行为由 Task 9 全栈验证 + 真机清单兜底。

- [ ] **Step 1: 加字段**

在 `Tags` 字段行后加:

```go
	Category         string               `json:"category,omitempty"`         // 库分组键 (空 = 未分类)
```

(对齐方式照该 struct 现有字段列, gofmt 会归位。)

- [ ] **Step 2: 编译验证**

Run: `go build ./...`（仓根）
Expected: 零输出零错。

- [ ] **Step 3: Commit**

```bash
git add internal/services/container/model.go
git commit -m "feat(library): Subgraph 加 category 字段 (库分组键)"
```

---

### Task 2: 前端类型 + i18n 文案

**Files:**
- Modify: `frontend/src/lib/backend.ts:118` (Subgraph 接口 tags 行后)
- Modify: `frontend/src/i18n/zh.ts` (common ~1608 / library ~1969)
- Modify: `frontend/src/i18n/en.ts` (镜像)

- [ ] **Step 1: backend.ts 加字段**

`interface Subgraph` 的 `tags?: string[]` 行后加:

```ts
  category?: string
```

- [ ] **Step 2: zh.ts 文案**

`common` 段 (`tags: '标签',` 后) 加:

```ts
    category: '分类',
```

`library.explorer` 段: `tags_hint` 改为 `'逗号分隔'` (废"第一个标签作为分组"), 并加:

```ts
      filter_category_all: '全部分类',
      filter_tags: '按标签过滤…',
      category_placeholder: '选择或输入新分类',
```

`library.detail.empty_hint` 改为 `'单击查看详情 · 双击插入引用'`。

`library` 下新增 `batch` 段 (与 `detail` 平级):

```ts
    batch: {
      selected_n: '已选 {n} 个子图',
      add_tags: '批量加标签',
      delete: '批量删除',
      clear: '取消选择',
      delete_confirm_title: '批量删除子图',
      delete_confirm_desc: '确认删除选中的 {n} 个子图? 此操作不可恢复.',
      delete_confirm_referenced: '选中的 {n} 个子图中有 {m} 个正被容器使用: {names}。删除后这些容器会报「子图缺失」。确认删除? 此操作不可恢复.',
      add_tags_title: '批量加标签',
      add_tags_placeholder: '选择或输入要追加的标签…',
      add_tags_apply: '添加',
      partial_failed: '{n} 个子图未能完成操作 (可能已被修改), 列表已刷新',
    },
```

- [ ] **Step 3: en.ts 镜像**

`common`: `category: 'Category',`。
`library.explorer`: `tags_hint: 'Comma separated',` + 

```ts
      filter_category_all: 'All categories',
      filter_tags: 'Filter by tags…',
      category_placeholder: 'Pick or type a new category',
```

`library.detail.empty_hint: 'Click to view details · double-click to insert',`。

`library.batch`:

```ts
    batch: {
      selected_n: '{n} subgraphs selected',
      add_tags: 'Add tags',
      delete: 'Delete selected',
      clear: 'Clear selection',
      delete_confirm_title: 'Delete subgraphs',
      delete_confirm_desc: 'Delete the {n} selected subgraphs? This cannot be undone.',
      delete_confirm_referenced: '{m} of the {n} selected subgraphs are used by containers: {names}. Deleting will break them with "subgraph missing". Delete anyway? This cannot be undone.',
      add_tags_title: 'Add tags to selection',
      add_tags_placeholder: 'Pick or type tags to append…',
      add_tags_apply: 'Add',
      partial_failed: '{n} subgraph(s) failed (possibly modified elsewhere); list refreshed',
    },
```

- [ ] **Step 4: 验证**

Run: `pnpm i18n:check && pnpm typecheck`（frontend/）
Expected: i18n key 对齐零缺; typecheck 零错。

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/backend.ts frontend/src/i18n/zh.ts frontend/src/i18n/en.ts
git commit -m "feat(library): category 类型字段 + 库改版 i18n 文案"
```

---

### Task 3: lib/libraryFilter.ts — 过滤/分组纯函数 (TDD)

**Files:**
- Create: `frontend/src/lib/libraryFilter.ts`
- Test: `frontend/src/lib/libraryFilter.test.ts`

- [ ] **Step 1: 写失败测试**

```ts
import { describe, expect, it } from 'vitest'
import { filterSubgraphs, groupByCategory } from './libraryFilter'

const items = [
  { label: '钓鱼主循环', description: '抛竿收竿', tags: ['钓鱼', '主流程'], category: '钓鱼' },
  { label: '上钩检测', tags: ['钓鱼'], category: '钓鱼' },
  { label: '通用点击', description: 'click helper', tags: ['工具'] },
  { label: '空白', tags: [] },
]

describe('filterSubgraphs', () => {
  it('无过滤条件时原样返回', () => {
    expect(filterSubgraphs(items, { query: '', category: null, tags: [] })).toHaveLength(4)
  })
  it('query 匹配 label/description/tags/category, 大小写不敏感', () => {
    expect(filterSubgraphs(items, { query: 'CLICK', category: null, tags: [] })).toHaveLength(1)
    expect(filterSubgraphs(items, { query: '钓鱼', category: null, tags: [] })).toHaveLength(2)
  })
  it('category 精确匹配; 空串 = 未分类', () => {
    expect(filterSubgraphs(items, { query: '', category: '钓鱼', tags: [] })).toHaveLength(2)
    expect(filterSubgraphs(items, { query: '', category: '', tags: [] })).toHaveLength(2)
  })
  it('tags 为 AND 语义', () => {
    expect(filterSubgraphs(items, { query: '', category: null, tags: ['钓鱼'] })).toHaveLength(2)
    expect(filterSubgraphs(items, { query: '', category: null, tags: ['钓鱼', '主流程'] })).toHaveLength(1)
  })
  it('三条件叠加', () => {
    expect(filterSubgraphs(items, { query: '上钩', category: '钓鱼', tags: ['钓鱼'] })).toHaveLength(1)
  })
})

describe('groupByCategory', () => {
  it('按 category 分组, 空归未分类, 组序 = 首现序', () => {
    const groups = groupByCategory(items, '未分类')
    expect(groups.map((g) => g.category)).toEqual(['钓鱼', '未分类'])
    expect(groups[0].items).toHaveLength(2)
    expect(groups[1].items).toHaveLength(2)
  })
})
```

- [ ] **Step 2: 跑测试确认失败**

Run: `pnpm vitest run src/lib/libraryFilter.test.ts`
Expected: FAIL — 模块不存在。

- [ ] **Step 3: 实现**

```ts
// 子图库列表过滤/分组纯函数 — 抽出便于单测, LibraryExplorerModal 消费。
export interface LibraryFilterInput {
  query: string
  // null = 全部; '' = 未分类; 其余 = 精确匹配该分类
  category: string | null
  // AND: 必须含全部所选标签
  tags: string[]
}

interface FilterableSubgraph {
  label: string
  description?: string
  tags?: string[]
  category?: string
}

export function filterSubgraphs<T extends FilterableSubgraph>(items: T[], f: LibraryFilterInput): T[] {
  const q = f.query.toLowerCase().trim()
  return items.filter((item) => {
    if (f.category !== null && (item.category ?? '') !== f.category) return false
    if (f.tags.length > 0 && !f.tags.every((tg) => (item.tags ?? []).includes(tg))) return false
    if (!q) return true
    const hay = `${item.label} ${item.description ?? ''} ${(item.tags ?? []).join(' ')} ${item.category ?? ''}`.toLowerCase()
    return hay.includes(q)
  })
}

export interface CategoryGroup<T> {
  category: string
  items: T[]
}

export function groupByCategory<T extends { category?: string }>(items: T[], uncategorized: string): CategoryGroup<T>[] {
  const map = new Map<string, T[]>()
  for (const item of items) {
    const key = item.category || uncategorized
    if (!map.has(key)) map.set(key, [])
    map.get(key)!.push(item)
  }
  return Array.from(map.entries()).map(([category, items]) => ({ category, items }))
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `pnpm vitest run src/lib/libraryFilter.test.ts`
Expected: PASS 全绿。

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/libraryFilter.ts frontend/src/lib/libraryFilter.test.ts
git commit -m "feat(library): 过滤/分组纯函数 libraryFilter (TDD)"
```

---

### Task 4: useListSelection — 文件管理器式多选 (TDD)

**Files:**
- Create: `frontend/src/composables/editor/useListSelection.ts`
- Test: `frontend/src/composables/editor/useListSelection.spec.ts`

- [ ] **Step 1: 写失败测试**

```ts
import { describe, expect, it } from 'vitest'
import { nextTick, ref } from 'vue'
import { useListSelection } from './useListSelection'

function setup(ids: string[] = ['a', 'b', 'c', 'd', 'e']) {
  const visible = ref(ids)
  return { visible, sel: useListSelection(visible) }
}

describe('useListSelection', () => {
  it('单击 = 替换选中并设锚点', () => {
    const { sel } = setup()
    sel.click('b')
    sel.click('d')
    expect([...sel.selected.value]).toEqual(['d'])
  })
  it('ctrl 单击 = toggle 加选/减选', () => {
    const { sel } = setup()
    sel.click('a')
    sel.click('c', { ctrl: true })
    expect([...sel.selected.value].sort()).toEqual(['a', 'c'])
    sel.click('a', { ctrl: true })
    expect([...sel.selected.value]).toEqual(['c'])
  })
  it('shift 单击 = 锚点到当前的可见范围 (正反向)', () => {
    const { sel } = setup()
    sel.click('b')
    sel.click('d', { shift: true })
    expect([...sel.selected.value].sort()).toEqual(['b', 'c', 'd'])
    sel.click('a', { shift: true }) // 锚点仍是 b, 反向
    expect([...sel.selected.value].sort()).toEqual(['a', 'b'])
  })
  it('无锚点时 shift 退化为单选', () => {
    const { sel } = setup()
    sel.click('c', { shift: true })
    expect([...sel.selected.value]).toEqual(['c'])
  })
  it('single: 恰好 1 个时给 id, 否则 null', () => {
    const { sel } = setup()
    expect(sel.single.value).toBeNull()
    sel.click('a')
    expect(sel.single.value).toBe('a')
    sel.click('b', { ctrl: true })
    expect(sel.single.value).toBeNull()
  })
  it('可见列表收缩时剔除不可见选中项', async () => {
    const { visible, sel } = setup()
    sel.click('b')
    sel.click('d', { shift: true })
    visible.value = ['b', 'e'] // c、d 被过滤掉
    await nextTick()
    expect([...sel.selected.value]).toEqual(['b'])
  })
  it('clear 清空选中与锚点', () => {
    const { sel } = setup()
    sel.click('a')
    sel.clear()
    expect(sel.selected.value.size).toBe(0)
    sel.click('c', { shift: true }) // 锚点已清, shift 退化单选
    expect([...sel.selected.value]).toEqual(['c'])
  })
})
```

- [ ] **Step 2: 跑测试确认失败**

Run: `pnpm vitest run src/composables/editor/useListSelection.spec.ts`
Expected: FAIL — 模块不存在。

- [ ] **Step 3: 实现**

```ts
import { computed, ref, watch, type Ref } from 'vue'

// 文件管理器式列表多选: 单击替换 / Ctrl toggle / Shift 锚点范围 (按可见顺序)。
// 可见列表变化 (过滤/删除) 自动剔除不可见项。Set 一律整体替换保证响应性。
export function useListSelection(visibleIds: Ref<string[]>) {
  const selected = ref<Set<string>>(new Set())
  const anchorId = ref<string | null>(null)

  function plainSelect(id: string) {
    selected.value = new Set([id])
    anchorId.value = id
  }

  function click(id: string, mods: { ctrl?: boolean; shift?: boolean } = {}) {
    if (mods.shift && anchorId.value !== null) {
      const ids = visibleIds.value
      const a = ids.indexOf(anchorId.value)
      const b = ids.indexOf(id)
      if (a === -1 || b === -1) {
        plainSelect(id)
        return
      }
      selected.value = new Set(ids.slice(Math.min(a, b), Math.max(a, b) + 1))
      return
    }
    if (mods.ctrl) {
      const next = new Set(selected.value)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      selected.value = next
      anchorId.value = id
      return
    }
    plainSelect(id)
  }

  function clear() {
    selected.value = new Set()
    anchorId.value = null
  }

  watch(visibleIds, (ids) => {
    const vis = new Set(ids)
    if ([...selected.value].some((id) => !vis.has(id))) {
      selected.value = new Set([...selected.value].filter((id) => vis.has(id)))
    }
    if (anchorId.value !== null && !vis.has(anchorId.value)) anchorId.value = null
  })

  const single = computed(() => (selected.value.size === 1 ? [...selected.value][0]! : null))
  const isSelected = (id: string) => selected.value.has(id)

  return { selected, single, click, clear, isSelected }
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `pnpm vitest run src/composables/editor/useListSelection.spec.ts`
Expected: PASS 全绿。

- [ ] **Step 5: Commit**

```bash
git add frontend/src/composables/editor/useListSelection.ts frontend/src/composables/editor/useListSelection.spec.ts
git commit -m "feat(editor): useListSelection 文件管理器式多选组合式 (TDD)"
```

---

### Task 5: LibraryDetailPanel — 插入/编辑按钮 + 分类行

**Files:**
- Modify: `frontend/src/components/containers/LibraryDetailPanel.vue`

- [ ] **Step 1: emits 加 insert / edit**

```ts
const emit = defineEmits<{
  cleared: []
  insert: []
  edit: []
}>()
```

- [ ] **Step 2: 元信息区加分类行**

「被使用」那个 `<section>` 里, `createdAt` 行前加:

```vue
        <div v-if="sg.category" class="flex justify-between">
          <span>{{ t('common.category') }}</span>
          <span>{{ sg.category }}</span>
        </div>
```

- [ ] **Step 3: 按钮区头部加两钮**

底部按钮区 (`pt-3 border-t` 那个 div) 的「复制为新子图」前插:

```vue
        <UButton size="sm" color="primary" icon="i-tabler-package-import" @click="emit('insert')">
          {{ t('library.explorer.insert') }}
        </UButton>
        <UButton size="sm" variant="soft" color="neutral" icon="i-tabler-pencil" @click="emit('edit')">
          {{ t('library.explorer.edit_info') }}
        </UButton>
```

(插入是唯一 primary 实心钮 — 选中后的主动作; 其余保持 soft。)

- [ ] **Step 4: 验证 + Commit**

Run: `pnpm typecheck`
Expected: 零错。

```bash
git add frontend/src/components/containers/LibraryDetailPanel.vue
git commit -m "feat(library): 详情面板加插入引用/编辑信息按钮 + 分类行"
```

---

### Task 6: LibraryBatchPanel — 批量操作面板 (新组件)

**Files:**
- Create: `frontend/src/components/containers/LibraryBatchPanel.vue`

- [ ] **Step 1: 写组件 (完整文件)**

```vue
<!-- 子图库右栏批量态 (选中 ≥2 时替换详情面板)。 -->
<template>
  <aside class="w-80 shrink-0 border-l border-default overflow-y-auto bg-default">
    <div class="h-full flex flex-col items-center justify-center text-center px-6 py-10">
      <UIcon name="i-tabler-stack-2" class="size-10 text-dimmed mb-3" />
      <p class="text-sm text-toned">{{ t('library.batch.selected_n', { n: count }) }}</p>
      <div class="flex flex-col gap-2 w-full mt-5">
        <UButton size="sm" variant="soft" color="primary" icon="i-tabler-tags" block @click="$emit('batch-add-tags')">
          {{ t('library.batch.add_tags') }}
        </UButton>
        <UButton size="sm" variant="soft" color="error" icon="i-tabler-trash" block @click="$emit('batch-delete')">
          {{ t('library.batch.delete') }}
        </UButton>
        <UButton size="sm" variant="ghost" color="neutral" block @click="$emit('clear')">
          {{ t('library.batch.clear') }}
        </UButton>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

defineProps<{ count: number }>()
defineEmits<{ 'batch-delete': []; 'batch-add-tags': []; clear: [] }>()
</script>
```

- [ ] **Step 2: 验证 + Commit**

Run: `pnpm typecheck`
Expected: 零错。

```bash
git add frontend/src/components/containers/LibraryBatchPanel.vue
git commit -m "feat(library): 批量操作面板组件"
```

---

### Task 7: LibraryExplorerModal — 选中模型 + 过滤器 + 批量流 (主改版)

**Files:**
- Modify: `frontend/src/components/containers/LibraryExplorerModal.vue` (整文件重写)

依赖 Task 2-6 全部就位。改动要点: 删悬停详情 (onHoverRow/去抖 timer/detailID), 删单击插入; 接 useListSelection (单击选中/Ctrl/Shift/右键单选), 双击插入; 分组改 groupByCategory; 过滤行 (分类 USelectMenu + 标签 UInputMenu); 右栏 detail/batch 二态; 编辑 modal 加分类; 批量删除/加标签流。

- [ ] **Step 1: 整文件替换为以下内容**

```vue
<!-- 子图库 modal (编辑器内唯一库管理入口). 入口: toolbar 📚 / 左 rail.
     本地 tab: 单击选中出详情; 双击/详情按钮插引用; Ctrl/Shift 多选 + 批量删/加标签;
     右键单项菜单。在线 tab: 占位 (跨机分享留口)。 -->
<template>
  <BaseModal
    v-model:open="modelOpen"
    :title="t('library.explorer.title')"
    icon="i-tabler-books"
    size="5xl"
  >
    <template #header-extra>
      <UTabs
        v-model="activeTab"
        :items="tabItems"
        size="xs"
        :content="false"
        class="mr-2"
      />
    </template>

    <!-- 在线: 占位 -->
    <div v-if="activeTab === 'online'" class="flex flex-col items-center justify-center text-center py-16">
      <UIcon name="i-tabler-cloud" class="size-12 text-dimmed mb-3" />
      <h3 class="text-sm text-toned font-medium">{{ t('library.online.title') }}</h3>
      <p class="text-xs text-dimmed mt-2 max-w-xs">{{ t('library.online.desc') }}</p>
    </div>

    <!-- 本地: 列表 + 右栏 (详情/批量) 双栏 -->
    <div v-else class="flex gap-4">
      <div class="flex-1 min-w-0 space-y-3">
        <div class="flex items-center gap-3">
          <UInput
            ref="searchInputRef"
            v-model="query"
            :placeholder="t('library.explorer.search')"
            icon="i-tabler-search"
            size="sm"
            class="flex-1"
            @keydown.escape="modelOpen = false"
          />
          <span class="text-[10px] text-dimmed">{{ t('library.explorer.esc_close') }}</span>
        </div>

        <div class="flex items-center gap-2">
          <USelectMenu
            v-model="categoryFilter"
            :items="categoryFilterItems"
            value-key="id"
            size="xs"
            class="w-40"
          />
          <UInputMenu
            v-model="tagFilter"
            multiple
            :items="allTags"
            size="xs"
            :placeholder="t('library.explorer.filter_tags')"
            class="flex-1"
          />
        </div>

        <div class="max-h-[56vh] overflow-y-auto select-none">
          <div
            v-if="filteredItems.length === 0"
            class="text-center text-xs text-dimmed py-8 italic"
          >
            <span v-if="lib.loading">{{ t('library.loading') }}</span>
            <span v-else-if="lib.subgraphs.length === 0"
              >{{ t('library.explorer.empty') }}</span
            >
            <span v-else>{{ t('library.explorer.no_match') }}</span>
          </div>

          <div v-else class="space-y-2">
            <template v-for="group in groupedItems" :key="group.category">
              <div class="text-[10px] font-semibold text-dimmed uppercase tracking-wider px-1 pt-2 pb-0.5">
                {{ group.category }}
              </div>
              <UContextMenu v-for="item in group.items" :key="item.id" :items="ctxMenuItems(item)">
                <div
                  class="rounded p-3 cursor-pointer"
                  :class="isSelected(item.id) ? 'bg-elevated/60' : 'bg-elevated/30 hover:bg-elevated/60'"
                  @click="onRowClick(item.id, $event)"
                  @dblclick="onPick(item.id)"
                  @contextmenu="selClick(item.id)"
                >
                  <div class="flex items-start gap-2">
                    <UIcon name="i-tabler-package" class="size-4 text-primary mt-0.5 shrink-0" />
                    <div class="flex-1 min-w-0">
                      <div class="text-sm font-medium">{{ item.label }}</div>
                      <div
                        v-if="item.description"
                        class="text-[11px] text-dimmed mt-0.5 line-clamp-2"
                      >
                        {{ item.description }}
                      </div>
                      <div
                        v-if="item.tags && item.tags.length > 0"
                        class="flex flex-wrap gap-1 mt-1"
                      >
                        <span
                          v-for="tg in item.tags"
                          :key="tg"
                          class="px-1.5 py-0 bg-elevated/60 text-[9px] rounded text-dimmed"
                        >
                          {{ tg }}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              </UContextMenu>
            </template>
          </div>
        </div>
      </div>

      <LibraryBatchPanel
        v-if="selected.size >= 2"
        class="max-h-[65vh]"
        :count="selected.size"
        @batch-delete="onBatchDelete"
        @batch-add-tags="batchTagsOpen = true"
        @clear="selClear()"
      />
      <LibraryDetailPanel
        v-else
        class="max-h-[65vh]"
        :sgID="single"
        @cleared="selClear()"
        @insert="single && onPick(single)"
        @edit="onEditSingle"
      />
    </div>
  </BaseModal>

  <!-- 编辑信息 (名称/描述/分类/标签) — merge patch + rev 乐观锁 -->
  <BaseModal v-model:open="editOpen" :title="t('library.explorer.edit_title')" icon="i-tabler-pencil" size="md">
    <div class="space-y-3">
      <div class="space-y-1.5">
        <label class="block text-xs text-toned">{{ t('common.name') }}</label>
        <UInput v-model="editForm.label" size="sm" />
      </div>
      <div class="space-y-1.5">
        <label class="block text-xs text-toned">{{ t('common.description') }}</label>
        <UTextarea v-model="editForm.description" :rows="3" size="sm" />
      </div>
      <div class="space-y-1.5">
        <label class="block text-xs text-toned">{{ t('common.category') }}</label>
        <UInputMenu
          v-model="editForm.category"
          creatable
          :items="allCategories"
          size="sm"
          :placeholder="t('library.explorer.category_placeholder')"
        />
      </div>
      <div class="space-y-1.5">
        <label class="block text-xs text-toned">{{ t('common.tags') }}</label>
        <UInput v-model="editForm.tags" size="sm" :placeholder="t('library.explorer.tags_hint')" />
      </div>
    </div>
    <template #footer>
      <UButton variant="ghost" color="neutral" @click="editOpen = false">{{ t('common.cancel') }}</UButton>
      <UButton color="primary" :disabled="!editForm.label.trim()" @click="onSaveEdit">{{ t('common.save') }}</UButton>
    </template>
  </BaseModal>

  <!-- 批量加标签 -->
  <BaseModal v-model:open="batchTagsOpen" :title="t('library.batch.add_tags_title')" icon="i-tabler-tags" size="md">
    <UInputMenu
      v-model="batchTags"
      multiple
      creatable
      :items="allTags"
      size="sm"
      :placeholder="t('library.batch.add_tags_placeholder')"
    />
    <template #footer>
      <UButton variant="ghost" color="neutral" @click="batchTagsOpen = false">{{ t('common.cancel') }}</UButton>
      <UButton color="primary" :disabled="batchTags.length === 0" @click="onBatchAddTags">{{ t('library.batch.add_tags_apply') }}</UButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import { useLibraryStore } from '@/stores/library'
import { useConfirm } from '@/composables/useConfirm'
import { useDialogOpen } from '@/composables/editor/useDialogOpen'
import { useAutoFocusOnOpen } from '@/composables/editor/useAutoFocusOnOpen'
import { useListSelection } from '@/composables/editor/useListSelection'
import { filterSubgraphs, groupByCategory } from '@/lib/libraryFilter'
import BaseModal from '@/components/common/BaseModal.vue'
import LibraryDetailPanel from '@/components/containers/LibraryDetailPanel.vue'
import LibraryBatchPanel from '@/components/containers/LibraryBatchPanel.vue'
import { backend, type Subgraph } from '@/lib/backend'
import { errorMessage, toastError } from '@/lib/invoke'

const { t } = useI18n()

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{
  'update:open': [v: boolean]
  'pick-subgraph': [libraryID: string]
}>()

const modelOpen = useDialogOpen(props, emit)

const query = ref('')
const searchInputRef = ref<any>(null)

const lib = useLibraryStore()
const toast = useToast()
const { confirm } = useConfirm()

const activeTab = ref<'local' | 'online'>('local')
const tabItems = computed(() => [
  { label: t('library.explorer.tab_local'), value: 'local' },
  { label: t('library.explorer.tab_online'), value: 'online' },
])

// Hydrate on mount; refresh when modal opens (cheap — backend caches).
async function refreshLibrary() {
  await lib.reload()
}

// ── 过滤 + 分组 ──
// categoryFilter 用 'all' / 'none' / 'c:<名>' 前缀编码, 防用户分类名撞保留值。
const categoryFilter = ref<string>('all')
const tagFilter = ref<string[]>([])

const allCategories = computed(() => {
  const set = new Set<string>()
  for (const sg of lib.subgraphs) if (sg.category) set.add(sg.category)
  return [...set].sort()
})
const allTags = computed(() => {
  const set = new Set<string>()
  for (const sg of lib.subgraphs) for (const tg of sg.tags ?? []) set.add(tg)
  return [...set].sort()
})

const categoryFilterItems = computed(() => [
  { label: t('library.explorer.filter_category_all'), id: 'all' },
  ...allCategories.value.map((c) => ({ label: c, id: `c:${c}` })),
  { label: t('library.explorer.uncategorized'), id: 'none' },
])

const filteredItems = computed<Subgraph[]>(() =>
  filterSubgraphs(lib.subgraphs, {
    query: query.value,
    category:
      categoryFilter.value === 'all' ? null : categoryFilter.value === 'none' ? '' : categoryFilter.value.slice(2),
    tags: tagFilter.value,
  }),
)

const groupedItems = computed(() => groupByCategory(filteredItems.value, t('library.explorer.uncategorized')))

// ── 选中 (单击/Ctrl/Shift; 右键收敛单选) ──
const visibleIds = computed(() => groupedItems.value.flatMap((g) => g.items.map((i) => i.id)))
const { selected, single, click: selClick, clear: selClear, isSelected } = useListSelection(visibleIds)

onMounted(() => refreshLibrary())
useAutoFocusOnOpen(modelOpen, searchInputRef, {
  onOpen: () => {
    void refreshLibrary()
    query.value = ''
    categoryFilter.value = 'all'
    tagFilter.value = []
    selClear()
  },
})

function onRowClick(id: string, e: MouseEvent) {
  selClick(id, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey })
}

function onPick(libraryID: string) {
  emit('pick-subgraph', libraryID)
  modelOpen.value = false
}

function ctxMenuItems(item: Subgraph) {
  return [
    [
      { label: t('library.explorer.insert'), icon: 'i-tabler-package-import', onSelect: () => onPick(item.id) },
      { label: t('library.card.duplicate'), icon: 'i-tabler-copy-plus', onSelect: () => onDuplicate(item) },
      { label: t('library.explorer.edit_info'), icon: 'i-tabler-pencil', onSelect: () => openEdit(item) },
    ],
    [
      { label: t('library.card.copy_id'), icon: 'i-tabler-copy', onSelect: () => onCopyID(item) },
    ],
    [
      { label: t('library.card.delete'), icon: 'i-tabler-trash', color: 'error' as const, onSelect: () => onDelete(item) },
    ],
  ]
}

async function onCopyID(item: Subgraph) {
  try {
    await navigator.clipboard.writeText(item.id)
    toast.add({ title: t('toast.copy_id_success'), color: 'success', icon: 'i-tabler-check', duration: 1500 })
  } catch (e: any) {
    toast.add({ title: t('toast.copy_failed'), description: errorMessage(e), color: 'error' })
  }
}

// 复制为新子图 (fork, ≈Blender Make Local): 想独立改不影响引用方时用。
async function onDuplicate(item: Subgraph) {
  const dup = await lib.duplicateSubgraph(item.id)
  if (dup) {
    toast.add({ title: t('library.card.duplicated', { name: dup.label }), color: 'success', icon: 'i-tabler-check' })
  }
}

// 删除安全: 先扫引用 — 被容器使用时警告里带"被 N 个容器使用", 确认才真删。
async function onDelete(item: Subgraph) {
  const refs = await lib.referrersOf(item.id)
  const useCount = lib.containerUseCount(refs)
  const name = item.label || item.id
  const desc = useCount > 0
    ? t('library.card.delete_confirm_referenced', { name, n: useCount })
    : t('library.card.delete_confirm_desc', { name })
  const yes = await confirm({
    title: t('library.card.delete_confirm_title'),
    description: desc,
    color: 'error',
    confirmText: t('common.delete'),
  })
  if (yes !== true) return
  const ok = await lib.deleteSubgraph(item.id)
  if (!ok) {
    toast.add({ title: t('toast.delete_failed'), color: 'error' })
  }
  // 选中项随 visibleIds 收缩自动剔除, 无需手动清。
}

// ── 编辑信息 (改名/描述/分类/标签) ──
const editOpen = ref(false)
const editTarget = ref<Subgraph | null>(null)
const editForm = ref({ label: '', description: '', category: '', tags: '' })

function openEdit(item: Subgraph) {
  editTarget.value = item
  editForm.value = {
    label: item.label ?? '',
    description: item.description ?? '',
    category: item.category ?? '',
    tags: (item.tags ?? []).join(', '),
  }
  editOpen.value = true
}

function onEditSingle() {
  const sg = single.value ? lib.byId(single.value) : undefined
  if (sg) openEdit(sg)
}

async function onSaveEdit() {
  const sg = editTarget.value
  if (!sg) return
  const tags = editForm.value.tags.split(',').map((s) => s.trim()).filter(Boolean)
  const patch = {
    label: editForm.value.label.trim(),
    description: editForm.value.description.trim(),
    category: (editForm.value.category ?? '').trim(),
    tags,
  }
  // 裸版本 + try/catch: error-only RPC 经 invoke 包装后成败同为 undefined, 辨不出结果。
  try {
    await backend.subgraphs.updateSilent(sg.id, JSON.stringify(patch), sg.rev)
  } catch (e) {
    toastError(errorMessage(e))
    return
  }
  await lib.reload()
  editOpen.value = false
}

// ── 批量删除: 逐项扫引用汇总警告, 确认后逐项删 (各带 rev), 失败聚合一条 toast ──
async function onBatchDelete() {
  const ids = [...selected.value]
  const referenced: string[] = []
  for (const id of ids) {
    const refs = await lib.referrersOf(id)
    if (lib.containerUseCount(refs) > 0) referenced.push(lib.byId(id)?.label || id)
  }
  const desc = referenced.length > 0
    ? t('library.batch.delete_confirm_referenced', { n: ids.length, m: referenced.length, names: referenced.join('、') })
    : t('library.batch.delete_confirm_desc', { n: ids.length })
  const yes = await confirm({
    title: t('library.batch.delete_confirm_title'),
    description: desc,
    color: 'error',
    confirmText: t('common.delete'),
  })
  if (yes !== true) return
  let failed = 0
  for (const id of ids) {
    if (!(await lib.deleteSubgraph(id))) failed++
  }
  if (failed > 0) {
    toast.add({ title: t('library.batch.partial_failed', { n: failed }), color: 'error' })
  }
  selClear()
}

// ── 批量加标签: 客户端算并集发全量数组 (RFC7386 数组整体替换) ──
const batchTagsOpen = ref(false)
const batchTags = ref<string[]>([])

async function onBatchAddTags() {
  const add = batchTags.value.map((s) => s.trim()).filter(Boolean)
  if (add.length === 0) {
    batchTagsOpen.value = false
    return
  }
  let failed = 0
  for (const id of [...selected.value]) {
    const sg = lib.byId(id)
    if (!sg) {
      failed++
      continue
    }
    const tags = [...new Set([...(sg.tags ?? []), ...add])]
    try {
      await backend.subgraphs.updateSilent(sg.id, JSON.stringify({ tags }), sg.rev)
    } catch {
      failed++
    }
  }
  await lib.reload()
  if (failed > 0) {
    toast.add({ title: t('library.batch.partial_failed', { n: failed }), color: 'error' })
  }
  batchTagsOpen.value = false
  batchTags.value = []
}
</script>
```

- [ ] **Step 2: 自检清单**

- 悬停相关代码 (onHoverRow / hoverTimer / onBeforeUnmount / detailID) 已不存在: `grep -n "hover\|detailID" frontend/src/components/containers/LibraryExplorerModal.vue` 仅 hover:bg 样式类命中。
- 列表容器带 `select-none` (防 Shift 单击拖出文本选区)。

- [ ] **Step 3: 验证**

Run: `pnpm typecheck && pnpm test`
Expected: 零类型错; vitest 全绿 (含 Task 3/4 新测试)。

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/containers/LibraryExplorerModal.vue
git commit -m "feat(library): modal 改版 — 单击选中+双击插入 + 多选批量 + 分类/标签过滤"
```

---

### Task 8: SubgraphPropsPanel 分类编辑 (编辑器内属性面板)

**Files:**
- Modify: `frontend/src/components/containers/SubgraphPropsPanel.vue`
- Modify: `frontend/src/components/containers/ContainerEditorInspector.vue` (传 prop)
- Modify: `frontend/src/views/ContainerEditorView.vue` (~1254 allSubgraphTags 旁)

数据通路已验证: `onSubgraphPropsUpdate` 是通用 `Object.assign(currentSubgraph, patch)` (ContainerEditorView.vue:1262), category 自动流通到保存。

- [ ] **Step 1: SubgraphPropsPanel 加分类 section**

`SubgraphLike` 接口加 `category?: string`; props 加 `allCategories?: string[]`:

```ts
const props = defineProps<{
  subgraph: SubgraphLike | null
  allTags?: string[]
  allCategories?: string[]
}>()
```

```ts
const allCategoriesList = computed(() => props.allCategories ?? [])
```

tags section 前加:

```vue
    <section class="space-y-2">
      <label class="text-xs text-toned">{{ t('common.category') }}</label>
      <UInputMenu
        :model-value="subgraph.category ?? ''"
        creatable
        :items="allCategoriesList"
        size="sm"
        :placeholder="t('library.explorer.category_placeholder')"
        @update:model-value="(v: string) => $emit('update', { category: v ?? '' })"
      />
    </section>
```

- [ ] **Step 2: ContainerEditorInspector 透传**

props 加 `allSubgraphCategories: string[]`; `<SubgraphPropsPanel>` 加 `:all-categories="allSubgraphCategories"`。

- [ ] **Step 3: ContainerEditorView 供数**

`allSubgraphTags` computed (~1254) 旁加:

```ts
const allSubgraphCategories = computed(() => {
  const set = new Set<string>()
  for (const sg of editorStore.visibleSubgraphs) {
    if (sg.category) set.add(sg.category)
  }
  return [...set].sort()
})
```

`<ContainerEditorInspector>` 调用处 (~226) 加 `:all-subgraph-categories="allSubgraphCategories"`。

- [ ] **Step 4: 验证 + Commit**

Run: `pnpm typecheck`
Expected: 零错。

```bash
git add frontend/src/components/containers/SubgraphPropsPanel.vue frontend/src/components/containers/ContainerEditorInspector.vue frontend/src/views/ContainerEditorView.vue
git commit -m "feat(editor): 子图属性面板加分类编辑"
```

---

### Task 9: 全量验证收口

- [ ] **Step 1: 全套检查**

Run（frontend/）: `pnpm typecheck && pnpm test && pnpm i18n:check && pnpm lint`
Run（仓根）: `go build ./...`
Expected: typecheck/test/i18n 全绿; lint 只剩 cockpit 在册预存 18 oxlint 错 (新代码零新增); go build 零错。

- [ ] **Step 2: 禁用色扫描** (ui.md 收尾要求)

Run: `grep -rn "bg-zinc-\|text-zinc-\|border-zinc-\|bg-black\|bg-white\|text-white\|text-black" frontend/src/components/containers/LibraryExplorerModal.vue frontend/src/components/containers/LibraryBatchPanel.vue frontend/src/components/containers/LibraryDetailPanel.vue frontend/src/components/containers/SubgraphPropsPanel.vue`
Expected: 零行。

- [ ] **Step 3: 收尾 commit (如有散件)**

确认工作区干净; 真机验收清单走 spec「验收」节, 由 landing 挂 cockpit 待验证。
