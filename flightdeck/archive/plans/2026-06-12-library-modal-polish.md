---
status: done
summary: 6 任务全完 — paginate + anchor TDD + i18n 增删 + 详情栏就地编辑 + modal 重写(checkbox/分页/双态工具栏/批量改分类), LibraryBatchPanel/编辑弹窗连根删。代码全绿, 差真机。
last_updated: 2026-06-12
implements: specs/2026-06-12-library-modal-polish.md
verify: 子图库美化 v3 真机验收 — ①行首 hover 出勾选框, 勾选常显, 批量不用按 Ctrl ②底部工具栏双态: 无选中显「共 N 个」+分页, 选中显「已选 N」+ 批量操作下拉(加标签/改分类/删除) + ✕ 取消, 单行不换行 ③每页条数 20/50/100 可调且重开记住, 过滤变化回第 1 页, >1 页出页码 ④批量改分类生效且分组随之变 ⑤右栏名称/描述双击可改(回车/失焦存, Esc 撤), 标签/分类行内直接改, 改完即存 ⑥「编辑信息」按钮/弹窗/右键项全无 ⑦插入引用是右栏唯一大绿钮, 复制/删除缩到底部小钮 ⑧双击行插入+右键其余菜单+过滤器+缺变量自动补全不回归
---

# 子图库 modal 美化 v3 — 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 子图库 modal: hover 多选框 + 底部双态工具栏(计数/分页 ↔ 批量删/加标签/改分类) + 分页(条数记忆) + 右栏就地编辑 + 单主 CTA 层级; 删 LibraryBatchPanel 与编辑弹窗双轨。

**Architecture:** 分页 = `paginate` 纯函数 (libraryFilter.ts, vitest); 详情栏跟**锚点项** — useListSelection 暴露 `anchor` (取消勾选锚点行 → anchor 置 null); 就地编辑照 CommentBoxNode 习语 (editing ref + draft + nextTick focus), 每字段独立 merge patch + rev。

**Tech Stack:** Vue 3 + Nuxt UI v4 (UCheckbox/UPagination/USelect) + @vueuse useLocalStorage。已知 API 风险: UPagination v4 props 名 (`v-model:page` / `total` / `items-per-page`) — typecheck 红就退化为两个 chevron UButton + 「x/y 页」文本。验证: `pnpm typecheck/test/i18n:check/lint` (oxlint 18 预存口径)。**不 push**。

---

### Task 1: paginate 纯函数 (TDD)

**Files:** Modify `frontend/src/lib/libraryFilter.ts` · Test `frontend/src/lib/libraryFilter.test.ts`

- [ ] **Step 1: 追加失败测试** (libraryFilter.test.ts 末尾)

```ts
describe('paginate', () => {
  const items = Array.from({ length: 23 }, (_, i) => i + 1)
  it('切页 + 总数 + 总页数', () => {
    const r = paginate(items, 1, 10)
    expect(r.pageItems).toEqual([1, 2, 3, 4, 5, 6, 7, 8, 9, 10])
    expect(r.total).toBe(23)
    expect(r.totalPages).toBe(3)
  })
  it('末页不足一页只给剩余', () => {
    expect(paginate(items, 3, 10).pageItems).toEqual([21, 22, 23])
  })
  it('页码越界钳制 (上下界)', () => {
    expect(paginate(items, 99, 10).page).toBe(3)
    expect(paginate(items, 0, 10).page).toBe(1)
  })
  it('空列表 = 1 页 0 项', () => {
    const r = paginate([], 5, 20)
    expect(r.pageItems).toEqual([])
    expect(r.totalPages).toBe(1)
    expect(r.page).toBe(1)
  })
})
```

(import 行加 `paginate`。) Run: `pnpm vitest run src/lib/libraryFilter.test.ts` → FAIL。

- [ ] **Step 2: 实现** (libraryFilter.ts 末尾追加)

```ts
export interface PageResult<T> {
  pageItems: T[]
  total: number
  totalPages: number
  // 钳制后的实际页码 (调用方可回写)
  page: number
}

export function paginate<T>(items: T[], page: number, size: number): PageResult<T> {
  const total = items.length
  const totalPages = Math.max(1, Math.ceil(total / size))
  const clamped = Math.min(Math.max(1, page), totalPages)
  const start = (clamped - 1) * size
  return { pageItems: items.slice(start, start + size), total, totalPages, page: clamped }
}
```

Run: 同上 → PASS。

- [ ] **Step 3: Commit** `feat(library): paginate 分页纯函数 (TDD)`

---

### Task 2: useListSelection 暴露 anchor (TDD)

**Files:** `frontend/src/composables/editor/useListSelection.ts` · `useListSelection.spec.ts`

- [ ] **Step 1: 追加失败测试**

```ts
describe('anchor (详情栏锚点)', () => {
  it('单击/加选都把锚点设到该行', () => {
    const { sel } = setup()
    sel.click('a')
    expect(sel.anchor.value).toBe('a')
    sel.click('c', { ctrl: true })
    expect(sel.anchor.value).toBe('c')
  })
  it('取消勾选锚点行 → 锚点清空; 取消非锚点行 → 锚点不动', () => {
    const { sel } = setup()
    sel.click('a')
    sel.click('c', { ctrl: true })
    sel.click('a', { ctrl: true }) // 取消非锚点 a
    expect(sel.anchor.value).toBe('c')
    sel.click('c', { ctrl: true }) // 取消锚点 c
    expect(sel.anchor.value).toBeNull()
  })
  it('锚点行被过滤掉 → 锚点清空', async () => {
    const { visible, sel } = setup()
    sel.click('c')
    visible.value = ['a', 'b']
    await nextTick()
    expect(sel.anchor.value).toBeNull()
  })
})
```

Run → FAIL (anchor undefined)。

- [ ] **Step 2: 实现** — ctrl 分支取消时清锚点; return 加 `anchor: anchorId`:

```ts
    if (mods.ctrl) {
      const next = new Set(selected.value)
      if (next.has(id)) {
        next.delete(id)
        if (anchorId.value === id) anchorId.value = null
      } else {
        next.add(id)
        anchorId.value = id
      }
      selected.value = next
      return
    }
```

```ts
  return { selected, single, anchor: anchorId, click, clear, isSelected }
```

Run → PASS (旧测试不回归)。

- [ ] **Step 3: Commit** `feat(editor): useListSelection 暴露 anchor — 详情栏跟最后操作行`

---

### Task 3: i18n 增删

**Files:** `frontend/src/i18n/zh.ts` / `en.ts`

- [ ] **Step 1: zh** — `library` 下加 `toolbar` 段 (与 batch 平级):

```ts
    toolbar: {
      total: '共 {n} 个',
      per_page: '{n} / 页',
    },
```

`library.batch` 加:

```ts
      change_category: '批量改分类',
      change_category_title: '批量改分类',
      change_category_placeholder: '选择或输入目标分类',
      change_category_apply: '应用',
```

`library.detail` 加:

```ts
      desc_empty: '双击添加描述',
      dblclick_edit: '双击编辑',
```

**删**: `library.explorer.edit_info` / `edit_title` / `tags_hint` 三键。

- [ ] **Step 2: en 镜像** — `toolbar: { total: '{n} total', per_page: '{n} / page' }`; batch 加 `change_category: 'Change category'` / `change_category_title: 'Change category'` / `change_category_placeholder: 'Pick or type target category'` / `change_category_apply: 'Apply'`; detail 加 `desc_empty: 'Double-click to add a description'` / `dblclick_edit: 'Double-click to edit'`; 删同三键。

- [ ] **Step 3: 验证** `pnpm i18n:check` parity OK (residue 39 预存)。Commit: `feat(library): 美化 v3 i18n — 工具栏/批量改分类/就地编辑文案, 删编辑弹窗键`

---

### Task 4: LibraryDetailPanel 就地编辑重做 (整文件)

**Files:** `frontend/src/components/containers/LibraryDetailPanel.vue`

emits 只剩 `insert`; 编辑弹窗按钮/cleared 事件删; 字段独立 patch (rev 乐观锁) + reload; Esc 取消用 `keydown.esc.stop` (尽量不让 modal 的 Esc 关闭抢先; 真机若仍关 modal 再补 capture)。

- [ ] **Step 1: 整文件替换**

```vue
<!-- 子图库右栏详情 (就地编辑): 名称/描述双击改, 标签/分类行内即改即存;
     插入引用 = 唯一主 CTA; 复制为新/删除弱化到底部一行。 -->
<template>
  <aside class="w-80 shrink-0 border-l border-default overflow-y-auto bg-default">
    <div
      v-if="!sg"
      class="h-full flex flex-col items-center justify-center text-center px-6 py-10"
    >
      <UIcon name="i-tabler-pointer" class="size-10 text-dimmed mb-3" />
      <p class="text-sm text-toned">{{ t('library.detail.empty') }}</p>
      <p class="text-[11px] text-dimmed mt-1">{{ t('library.detail.empty_hint') }}</p>
    </div>

    <div v-else class="p-4 space-y-4">
      <header class="flex items-start gap-3">
        <div class="size-10 rounded-lg flex items-center justify-center shrink-0 bg-fuchsia-500/15 border border-fuchsia-500/40">
          <UIcon name="i-tabler-subtask" class="size-5 text-fuchsia-300" />
        </div>
        <div class="min-w-0 flex-1">
          <UInput
            v-if="editingName"
            ref="nameInputRef"
            v-model="draftName"
            size="sm"
            @keyup.enter="saveName"
            @keydown.esc.stop="editingName = false"
            @blur="saveName"
          />
          <h3
            v-else
            class="group flex items-center gap-1 text-sm font-medium text-highlighted leading-tight cursor-text"
            :title="t('library.detail.dblclick_edit')"
            @dblclick="enterEditName"
          >
            <span class="truncate min-w-0">{{ sg.label || sg.id }}</span>
            <UIcon name="i-tabler-pencil" class="size-3 shrink-0 text-dimmed opacity-0 group-hover:opacity-100" />
          </h3>
          <p class="text-[11px] text-dimmed mt-0.5">
            {{ t('library.detail.nodes_and_outputs', { n: sg.graph?.nodes?.length ?? 0, m: sg.outputPins?.length ?? 0 }) }}
          </p>
        </div>
      </header>

      <UButton color="primary" icon="i-tabler-package-import" block @click="emit('insert')">
        {{ t('library.explorer.insert') }}
      </UButton>

      <section class="space-y-1.5">
        <label class="block text-xs text-toned">{{ t('library.detail.description') }}</label>
        <UTextarea
          v-if="editingDesc"
          ref="descInputRef"
          v-model="draftDesc"
          :rows="3"
          size="sm"
          @keydown.esc.stop="editingDesc = false"
          @blur="saveDesc"
        />
        <p
          v-else-if="sg.description"
          class="text-xs text-default whitespace-pre-line cursor-text"
          :title="t('library.detail.dblclick_edit')"
          @dblclick="enterEditDesc"
        >
          {{ sg.description }}
        </p>
        <p v-else class="text-xs text-dimmed italic cursor-text" @dblclick="enterEditDesc">
          {{ t('library.detail.desc_empty') }}
        </p>
      </section>

      <section class="space-y-1.5">
        <label class="block text-xs text-toned">{{ t('common.category') }}</label>
        <UInputMenu
          :model-value="sg.category ?? ''"
          creatable
          :items="allCategories"
          size="sm"
          :placeholder="t('library.explorer.category_placeholder')"
          @update:model-value="(v: string) => patchField({ category: v ?? '' })"
        />
      </section>

      <section class="space-y-1.5">
        <label class="block text-xs text-toned">{{ t('library.detail.tags') }}</label>
        <UInputMenu
          :model-value="sg.tags ?? []"
          multiple
          creatable
          :items="allTags"
          size="sm"
          @update:model-value="(v: string[]) => patchField({ tags: v })"
        />
      </section>

      <section class="space-y-1 text-[11px] text-dimmed">
        <div class="flex justify-between">
          <span>{{ t('library.detail.used_by') }}</span>
          <span>{{ useCount === null ? '…' : t('library.detail.used_by_n', { n: useCount }) }}</span>
        </div>
        <div v-if="sg.createdAt" class="flex justify-between">
          <span>{{ t('library.detail.created_at') }}</span>
          <span>{{ new Date(sg.createdAt).toLocaleString() }}</span>
        </div>
      </section>

      <section class="space-y-1.5">
        <label class="block text-[10px] uppercase tracking-[0.08em] font-semibold text-dimmed">ID</label>
        <button
          type="button"
          class="w-full text-left text-[11px] font-mono bg-elevated/40 rounded px-2 py-1 hover:bg-elevated/60 transition-colors truncate flex items-center gap-1.5"
          :class="copied ? 'text-success' : 'text-dimmed'"
          :title="t('library.detail.click_to_copy') + sg.id"
          @click="onCopyID"
        >
          <UIcon v-if="copied" name="i-tabler-check" class="size-3 shrink-0" />
          <span class="truncate">{{ copied ? t('common.copied') : sg.id }}</span>
        </button>
      </section>

      <div class="pt-3 border-t border-default flex items-center gap-2">
        <UButton size="xs" variant="soft" color="neutral" icon="i-tabler-copy-plus" @click="onDuplicate">
          {{ t('library.card.duplicate') }}
        </UButton>
        <UButton size="xs" variant="soft" color="error" icon="i-tabler-trash" class="ml-auto" @click="onDelete">
          {{ t('library.detail.delete') }}
        </UButton>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend, type Subgraph } from '@/lib/backend'
import { useLibraryStore } from '@/stores/library'
import { useConfirm } from '@/composables/useConfirm'
import { useToast } from '@nuxt/ui/composables'
import { errorMessage, toastError } from '@/lib/invoke'

const { t } = useI18n()

const props = defineProps<{ sgID: string | null }>()
const emit = defineEmits<{ insert: [] }>()

const libraryStore = useLibraryStore()
const { confirm } = useConfirm()
const toast = useToast()

const sg = computed<Subgraph | undefined>(() => (props.sgID ? libraryStore.byId(props.sgID) : undefined))

const allCategories = computed(() => {
  const set = new Set<string>()
  for (const s of libraryStore.subgraphs) if (s.category) set.add(s.category)
  return [...set].sort()
})
const allTags = computed(() => {
  const set = new Set<string>()
  for (const s of libraryStore.subgraphs) for (const tg of s.tags ?? []) set.add(tg)
  return [...set].sort()
})

// ── 字段级保存 (merge patch + rev 乐观锁); 失败 toast, 成败都 reload 对齐磁盘 ──
async function patchField(patch: Record<string, unknown>) {
  if (!sg.value) return
  try {
    await backend.subgraphs.updateSilent(sg.value.id, JSON.stringify(patch), sg.value.rev)
  } catch (e) {
    toastError(errorMessage(e))
  }
  await libraryStore.reload()
}

// ── 名称双击编辑 (CommentBoxNode 习语: editing + draft + nextTick focus) ──
const editingName = ref(false)
const draftName = ref('')
const nameInputRef = ref<any>(null)

async function enterEditName() {
  if (!sg.value) return
  draftName.value = sg.value.label ?? ''
  editingName.value = true
  await nextTick()
  const el: HTMLInputElement | undefined = nameInputRef.value?.inputRef
  el?.focus()
  el?.select()
}

function saveName() {
  if (!editingName.value) return
  editingName.value = false
  const next = draftName.value.trim()
  if (!next || next === sg.value?.label) return
  void patchField({ label: next })
}

// ── 描述双击编辑 ──
const editingDesc = ref(false)
const draftDesc = ref('')
const descInputRef = ref<any>(null)

async function enterEditDesc() {
  if (!sg.value) return
  draftDesc.value = sg.value.description ?? ''
  editingDesc.value = true
  await nextTick()
  nameInputRefNoop()
  const el: HTMLTextAreaElement | undefined = descInputRef.value?.textareaRef
  el?.focus()
}

function saveDesc() {
  if (!editingDesc.value) return
  editingDesc.value = false
  const next = draftDesc.value.trim()
  if (next === (sg.value?.description ?? '')) return
  void patchField({ description: next })
}

// 切换选中项时退出编辑态, 防 draft 串台
watch(() => props.sgID, () => {
  editingName.value = false
  editingDesc.value = false
})

// ── 引用计数 ──
const useCount = ref<number | null>(null)
watch(() => props.sgID, async (id) => {
  useCount.value = null
  if (!id) return
  const refs = await libraryStore.referrersOf(id)
  useCount.value = libraryStore.containerUseCount(refs)
}, { immediate: true })

const copied = ref(false)
let copiedTimer = 0
async function onCopyID() {
  if (!props.sgID) return
  try {
    await navigator.clipboard.writeText(props.sgID)
    copied.value = true
    window.clearTimeout(copiedTimer)
    copiedTimer = window.setTimeout(() => { copied.value = false }, 1500)
  } catch (e: any) {
    toast.add({ title: t('toast.copy_failed'), description: errorMessage(e), color: 'error' })
  }
}

async function onDuplicate() {
  if (!props.sgID) return
  const dup = await libraryStore.duplicateSubgraph(props.sgID)
  if (dup) {
    toast.add({ title: t('library.card.duplicated', { name: dup.label }), color: 'success', icon: 'i-tabler-check' })
  }
}

async function onDelete() {
  if (!props.sgID || !sg.value) return
  const refs = await libraryStore.referrersOf(props.sgID)
  const n = libraryStore.containerUseCount(refs)
  const desc = n > 0
    ? t('library.card.delete_confirm_referenced', { name: sg.value.label || props.sgID, n })
    : t('library.card.delete_confirm_desc', { name: sg.value.label || props.sgID })
  const yes = await confirm({
    title: t('library.card.delete_confirm_title'),
    description: desc,
    color: 'error',
    confirmText: t('common.delete'),
  })
  if (yes !== true) return
  const ok = await libraryStore.deleteSubgraph(props.sgID)
  if (!ok) {
    toast.add({ title: t('toast.delete_failed'), color: 'error' })
  }
  // 列表收缩 → 选中/锚点由 useListSelection 剪枝, 面板自动回空态
}
</script>
```

注意: 上面 `nameInputRefNoop()` 是占位错误 — **不要**写这行 (执行时删掉; 这里标注防照抄)。

- [ ] **Step 2: 验证** `pnpm typecheck` (modal 还引用着已删的 edit 事件会红 — Task 5 一起绿, 此步只确认本文件无语法错, 可延后到 Task 5 末统一跑)。

- [ ] **Step 3: Commit** (与 Task 5 合并提交亦可)

---

### Task 5: LibraryExplorerModal 重写 + 删 LibraryBatchPanel

**Files:**
- Modify: `frontend/src/components/containers/LibraryExplorerModal.vue`
- Delete: `frontend/src/components/containers/LibraryBatchPanel.vue`

要点 (在现文件上改, 非整文件重抄): ①行加 group + hover checkbox (stop 传播, toggle = `selClick(id, {ctrl:true})`); ②`filteredItems → paginate(page,pageSize) → groupByCategory(pageItems)`; pageSize = `useLocalStorage('library.pageSize', 50)`; 过滤/搜索/条数变化回第 1 页 + totalPages 收缩钳制; ③列表下加底部工具栏 (双态); ④右栏 `:sgID="anchor"`, `@insert="anchor && onPick(anchor)"`, 删 batch panel / cleared / edit 绑定; ⑤删编辑弹窗整块 + editOpen/editTarget/editForm/openEdit/onEditSingle/onSaveEdit + ctx 菜单「编辑信息」项; ⑥加批量改分类弹窗 + onBatchChangeCategory (镜像批量加标签)。

- [ ] **Step 1: script 改动**

imports: 删 `LibraryBatchPanel`; `filterSubgraphs, groupByCategory` 行加 `paginate`; 加 `import { useLocalStorage } from '@vueuse/core'`; `watch` 进 vue import。

分页接线 (替换 groupedItems 一段):

```ts
const page = ref(1)
const pageSize = useLocalStorage('library.pageSize', 50)
const pageSizeItems = computed(() => [20, 50, 100].map((n) => ({ label: t('library.toolbar.per_page', { n }), value: n })))

const pageResult = computed(() => paginate(filteredItems.value, page.value, pageSize.value))
const groupedItems = computed(() => groupByCategory(pageResult.value.pageItems, t('library.explorer.uncategorized')))

watch([query, categoryFilter, tagFilter, pageSize], () => { page.value = 1 })
watch(() => pageResult.value.totalPages, (tp) => { if (page.value > tp) page.value = tp })
```

selection 行换成 `const { selected, anchor, click: selClick, clear: selClear, isSelected } = useListSelection(visibleIds)` (single 不再用)。onOpen 重置加 `page.value = 1`。

批量改分类:

```ts
const batchCategoryOpen = ref(false)
const batchCategory = ref('')

async function onBatchChangeCategory() {
  const target = batchCategory.value.trim()
  const ids = [...selected.value]
  let failed = 0
  for (const id of ids) {
    const sg = lib.byId(id)
    if (!sg) {
      failed++
      continue
    }
    try {
      await backend.subgraphs.updateSilent(sg.id, JSON.stringify({ category: target }), sg.rev)
    } catch {
      failed++
    }
  }
  await lib.reload()
  if (failed > 0) {
    toast.add({ title: t('library.batch.partial_failed', { n: failed }), color: 'error' })
  }
  batchCategoryOpen.value = false
  batchCategory.value = ''
}
```

删: editOpen/editTarget/editForm/openEdit/onEditSingle/onSaveEdit 整段; ctxMenuItems 里编辑项。

- [ ] **Step 2: template 改动**

行 (UContextMenu 内) 加 checkbox + group:

```vue
                <div
                  class="group rounded p-3 cursor-pointer"
                  :class="isSelected(item.id) ? 'bg-primary/15 ring-1 ring-inset ring-primary/50' : 'bg-elevated/30 hover:bg-elevated/60'"
                  @click="onRowClick(item.id, $event)"
                  @dblclick="onPick(item.id)"
                  @contextmenu="selClick(item.id)"
                >
                  <div class="flex items-start gap-2">
                    <span
                      class="mt-0.5 shrink-0 transition-opacity"
                      :class="isSelected(item.id) ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'"
                      @click.stop
                      @dblclick.stop
                    >
                      <UCheckbox
                        :model-value="isSelected(item.id)"
                        size="sm"
                        @update:model-value="selClick(item.id, { ctrl: true })"
                      />
                    </span>
                    <UIcon name="i-tabler-package" class="size-4 text-primary mt-0.5 shrink-0" />
                    ...其余行内容不变...
```

列表滚动区下加工具栏 (列表列 space-y-3 内, overflow 区后):

```vue
        <div class="flex items-center justify-between gap-3 pt-2 border-t border-default">
          <div v-if="selected.size === 0" class="text-[11px] text-dimmed">
            {{ t('library.toolbar.total', { n: pageResult.total }) }}
          </div>
          <div v-else class="flex items-center gap-1.5 flex-wrap">
            <span class="text-[11px] text-toned">{{ t('library.batch.selected_n', { n: selected.size }) }}</span>
            <UButton size="xs" variant="soft" color="error" icon="i-tabler-trash" @click="onBatchDelete">{{ t('library.batch.delete') }}</UButton>
            <UButton size="xs" variant="soft" color="primary" icon="i-tabler-tags" @click="batchTagsOpen = true">{{ t('library.batch.add_tags') }}</UButton>
            <UButton size="xs" variant="soft" color="primary" icon="i-tabler-category" @click="batchCategoryOpen = true">{{ t('library.batch.change_category') }}</UButton>
            <UButton size="xs" variant="ghost" color="neutral" @click="selClear()">{{ t('library.batch.clear') }}</UButton>
          </div>
          <div class="flex items-center gap-2 shrink-0">
            <UPagination v-if="pageResult.totalPages > 1" v-model:page="page" :total="pageResult.total" :items-per-page="pageSize" size="xs" :sibling-count="1" />
            <USelect v-model="pageSize" :items="pageSizeItems" size="xs" class="w-24" />
          </div>
        </div>
```

右栏换成:

```vue
      <LibraryDetailPanel
        class="max-h-[65vh]"
        :sgID="anchor"
        @insert="anchor && onPick(anchor)"
      />
```

(LibraryBatchPanel 块整删。) 编辑弹窗 BaseModal 块整删; 批量改分类弹窗 (批量加标签弹窗后):

```vue
  <BaseModal v-model:open="batchCategoryOpen" :title="t('library.batch.change_category_title')" icon="i-tabler-category" size="md">
    <UInputMenu
      v-model="batchCategory"
      creatable
      :items="allCategories"
      size="sm"
      :placeholder="t('library.batch.change_category_placeholder')"
    />
    <template #footer>
      <UButton variant="ghost" color="neutral" @click="batchCategoryOpen = false">{{ t('common.cancel') }}</UButton>
      <UButton color="primary" @click="onBatchChangeCategory">{{ t('library.batch.change_category_apply') }}</UButton>
    </template>
  </BaseModal>
```

- [ ] **Step 3: 删文件** `frontend/src/components/containers/LibraryBatchPanel.vue`; grep `LibraryBatchPanel` 全仓零命中 (components.d.ts 生成物下次构建自落)。

- [ ] **Step 4: 验证** `pnpm typecheck && pnpm test` 全绿; UPagination 红则按 header 风险条退化。

- [ ] **Step 5: Commit** `feat(library): modal 美化 v3 — hover 多选框+分页+底部双态工具栏+批量改分类+右栏就地编辑, 删批量面板/编辑弹窗`

---

### Task 6: 全量验证

- [ ] `pnpm typecheck && pnpm test && pnpm i18n:check && pnpm lint` 全绿/预存口径 (oxlint 18, residue 39)。
- [ ] grep 自检零命中: `LibraryBatchPanel` / `edit_info` / `edit_title` / `tags_hint` / `editOpen`(本 modal 内)。
- [ ] 禁用色扫描 (ui.md): 改动的两个组件零 `bg-zinc-|text-zinc-|border-zinc-|bg-black|bg-white|text-white|text-black`。
- [ ] 散件 commit。
