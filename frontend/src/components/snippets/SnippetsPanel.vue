<!-- SnippetsPanel — 用户资产列表. 按 category 分 collapsible group.
     drag-out → 画布 → 应用 snippet config 生成节点. -->
<template>
  <div class="snippets-panel flex-1 overflow-y-auto">
    <!-- Search + filter -->
    <div class="px-3 py-2 border-b border-default space-y-2">
      <UInput
        v-model="query"
        size="sm"
        icon="i-tabler-search"
        :aria-label="t('editor.snippet.panel.search_placeholder')"
        :placeholder="t('editor.snippet.panel.search_placeholder')"
        class="w-full"
      />
      <div v-if="store.allTags.length > 0" class="flex flex-wrap gap-1">
        <button
          v-for="t in store.allTags"
          :key="t"
          type="button"
          class="px-1.5 py-0.5 text-[11px] rounded transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
          :aria-pressed="activeTags.has(t)"
          :class="
            activeTags.has(t)
              ? 'bg-primary/30 text-primary border border-primary/50'
              : 'bg-elevated/30 text-dimmed hover:text-default border border-transparent'
          "
          @click="toggleTag(t)"
        >
          #{{ t }}
        </button>
      </div>
    </div>

    <!-- Group list -->
    <div
      v-if="filteredByCategory.length === 0"
      class="text-[11px] text-dimmed italic px-3 py-4 text-center"
    >
      <UIcon name="i-tabler-bookmarks" class="size-5 block mx-auto mb-2 opacity-50" />
      {{
        store.snippets.length === 0
          ? t('editor.snippet.panel.empty_none')
          : t('editor.snippet.panel.empty_no_match')
      }}
    </div>

    <div v-else class="px-2 py-1">
      <div v-for="g in filteredByCategory" :key="g.category" class="mb-2">
        <button
          type="button"
          class="w-full flex items-center gap-1.5 px-2 py-1.5 text-xs font-semibold text-default hover:bg-elevated/40 rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
          :aria-expanded="isExpanded(g.category)"
          @click="toggleGroup(g.category)"
        >
          <UIcon
            :name="isExpanded(g.category) ? 'i-tabler-chevron-down' : 'i-tabler-chevron-right'"
            class="size-3.5"
          />
          <span>{{ g.category }}</span>
          <span class="text-xs opacity-60 font-normal">({{ g.list.length }})</span>
        </button>
        <div v-show="isExpanded(g.category)" class="space-y-1 mt-1">
          <div
            v-for="s in g.list"
            :key="s.id"
            class="snippet-item group"
            role="button"
            tabindex="0"
            :aria-label="s.name"
            :style="s.color ? { borderLeftColor: s.color } : {}"
            draggable="true"
            :title="snippetTooltip(s)"
            @dragstart="(e) => onDragStart(s, e)"
            @click="emit('apply', s)"
            @keydown.enter.prevent="emit('apply', s)"
            @keydown.space.prevent="emit('apply', s)"
          >
            <UIcon
              v-if="s.icon"
              :name="s.icon"
              class="size-3.5 shrink-0"
              :style="s.color ? { color: s.color } : {}"
            />
            <div class="flex-1 min-w-0">
              <div class="text-xs truncate font-medium">{{ s.name }}</div>
              <div class="text-[11px] text-dimmed truncate font-mono">
                {{ s.payload.kind }}<span v-if="s.shortcut"> · {{ s.shortcut }}</span>
              </div>
            </div>
            <button
              type="button"
              class="text-dimmed hover:text-primary p-1 opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 focus-visible:opacity-100 rounded transition-opacity focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
              :title="t('editor.snippet.panel.edit_tip')"
              :aria-label="t('editor.snippet.panel.edit_tip')"
              @click.stop="emit('edit', s)"
            >
              <UIcon name="i-tabler-pencil" class="size-3" />
            </button>
            <button
              type="button"
              class="text-dimmed hover:text-error p-1 opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 focus-visible:opacity-100 rounded transition-opacity focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
              :title="t('editor.snippet.panel.delete_tip')"
              :aria-label="t('editor.snippet.panel.delete_tip')"
              @click.stop="onRemove(s)"
            >
              <UIcon name="i-tabler-x" class="size-3" />
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSnippetsStore, type Snippet } from '@/stores/snippets'
import { startEditorDrag } from '@/composables/editor/useEditorDragDrop'
import { useConfirm } from '@/composables/useConfirm'

const { t } = useI18n()
const { confirm: confirmDialog } = useConfirm()

const emit = defineEmits<{
  apply: [s: Snippet] // 单击应用 (在画布中心生成)
  edit: [s: Snippet] // 编辑 (打开 SaveSnippetDrawer)
}>()

const store = useSnippetsStore()
onMounted(() => store.load())

const query = ref('')
const activeTags = ref(new Set<string>())
const collapsed = ref(new Set<string>())

function isExpanded(cat: string) {
  return !collapsed.value.has(cat)
}
function toggleGroup(cat: string) {
  if (collapsed.value.has(cat)) collapsed.value.delete(cat)
  else collapsed.value.add(cat)
  // trigger reactivity
  collapsed.value = new Set(collapsed.value)
}
function toggleTag(t: string) {
  if (activeTags.value.has(t)) activeTags.value.delete(t)
  else activeTags.value.add(t)
  activeTags.value = new Set(activeTags.value)
}

const filteredByCategory = computed(() => {
  const q = query.value.trim().toLowerCase()
  const tags = activeTags.value
  return store.byCategory
    .map((g) => ({
      category: g.category,
      list: g.list.filter((s) => {
        if (q) {
          const hay =
            `${s.name} ${s.description ?? ''} ${s.payload.kind} ${s.tags.join(' ')}`.toLowerCase()
          if (!hay.includes(q)) return false
        }
        if (tags.size > 0) {
          for (const t of tags) if (!s.tags.includes(t)) return false
        }
        return true
      }),
    }))
    .filter((g) => g.list.length > 0)
})

function snippetTooltip(s: Snippet): string {
  const parts = [
    s.description,
    `kind: ${s.payload.kind}`,
    s.shortcut ? t('editor.snippet.panel.shortcut_tip', { hk: s.shortcut }) : '',
  ]
  return parts.filter(Boolean).join('\n')
}

function onDragStart(s: Snippet, e: DragEvent) {
  startEditorDrag({ type: 'snippet', snippetID: s.id }, e)
}

async function onRemove(s: Snippet) {
  const ok = await confirmDialog({
    title: t('editor.snippet.panel.delete_dialog_title'),
    description: t('editor.snippet.panel.delete_dialog_desc', { name: s.name }),
    confirmText: t('editor.snippet.panel.delete_confirm'),
    cancelText: t('editor.snippet.panel.delete_cancel'),
    color: 'error',
  })
  if (!ok) return
  store.remove(s.id)
}
</script>

<style scoped>
.snippet-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid rgba(255, 255, 255, 0.04);
  border-left: 3px solid rgba(255, 255, 255, 0.1);
  cursor: grab;
  transition:
    background 120ms ease,
    border-color 120ms ease;
}
.snippet-item:hover {
  background: rgba(255, 255, 255, 0.06);
  border-color: rgba(255, 255, 255, 0.08);
}
.snippet-item:focus-visible {
  outline: 2px solid var(--ui-primary);
  outline-offset: 1px;
}
.snippet-item:active {
  cursor: grabbing;
}
</style>
