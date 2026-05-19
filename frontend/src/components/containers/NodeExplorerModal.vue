<!-- frontend/src/components/containers/NodeExplorerModal.vue -->
<!-- Editor v2 B — Houdini-style 多列 node browser.
     入口: toolbar 📐 click / Tab 键 (canvas focused).
     Spec: editor-v2-node-discovery-design.md §4. -->
<template>
  <UModal v-model:open="modelOpen" :ui="{ content: 'sm:max-w-5xl' }">
    <template #content>
      <div class="p-5 bg-default space-y-3" style="min-height: 60vh; max-height: 80vh; display: flex; flex-direction: column;">
        <!-- Header: title + search + close hint -->
        <div class="flex items-center gap-3 shrink-0">
          <UIcon name="i-tabler-grid-dots" class="size-5 text-primary" />
          <h3 class="text-sm font-medium">节点 Explorer</h3>
          <UInput
            ref="searchInputRef"
            v-model="query"
            placeholder="搜节点 (substring)..."
            icon="i-tabler-search"
            size="sm"
            class="flex-1"
            @keydown.escape="onEsc"
          />
          <span class="text-[10px] text-dimmed">Esc 关 · Tab 重开</span>
        </div>

        <!-- Favorites + Recent chip rows -->
        <div
          v-if="store.favorites.length > 0 || store.recent.length > 0"
          class="flex items-center gap-2 flex-wrap p-2 bg-elevated/30 rounded shrink-0"
        >
          <template v-if="store.favorites.length > 0">
            <span class="text-[10px] text-amber-400 font-medium">★ 收藏:</span>
            <button
              v-for="kind in store.favorites"
              :key="`fav-${kind}`"
              type="button"
              class="px-2 py-0.5 bg-amber-500/15 text-amber-400 border border-amber-500/30 rounded-full text-[10px] hover:bg-amber-500/25"
              :title="getSpec(kind)?.description ?? kind"
              @click="onSelectKind(kind)"
            >{{ labelFor(kind) }}</button>
          </template>
          <template v-if="store.recent.length > 0">
            <span class="text-[10px] text-indigo-400 font-medium" :class="{'ml-2': store.favorites.length > 0}">🕐 最近:</span>
            <button
              v-for="kind in store.recent"
              :key="`recent-${kind}`"
              type="button"
              class="px-2 py-0.5 bg-indigo-500/15 text-indigo-400 border border-indigo-500/30 rounded-full text-[10px] hover:bg-indigo-500/25"
              :title="getSpec(kind)?.description ?? kind"
              @click="onSelectKind(kind)"
            >{{ labelFor(kind) }}</button>
          </template>
        </div>

        <!-- Grid: categorized columns, scrollable -->
        <div class="flex-1 overflow-y-auto pr-2">
          <div v-if="filteredGroups.length === 0" class="text-center text-xs text-dimmed py-8 italic">
            没匹配的节点
          </div>
          <div v-else class="grid gap-4" style="grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));">
            <div v-for="g in filteredGroups" :key="g.group">
              <div class="text-[10px] font-medium text-primary border-b border-default pb-1 mb-2">
                {{ groupLabel(g.group) }} ({{ g.specs.length }})
              </div>
              <div class="space-y-1">
                <div
                  v-for="spec in g.specs"
                  :key="spec.kind"
                  class="flex items-center gap-1 px-2 py-1 bg-elevated/30 hover:bg-elevated/60 rounded text-[11px] cursor-pointer"
                  :title="spec.description ?? spec.kind"
                  @click="onSelectKind(spec.kind)"
                >
                  <span class="flex-1">{{ spec.labelZh ?? spec.kind }}</span>
                  <button
                    type="button"
                    class="px-0.5"
                    :title="store.favorites.includes(spec.kind) ? '已收藏' : '加入收藏'"
                    @click.stop="store.toggleFavorite(spec.kind)"
                  >
                    <UIcon
                      :name="store.favorites.includes(spec.kind) ? 'i-tabler-star-filled' : 'i-tabler-star'"
                      class="size-3"
                      :class="store.favorites.includes(spec.kind) ? 'text-amber-400' : 'text-dimmed hover:text-amber-400'"
                    />
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { allSpecs, getSpec } from '@/components/containers/nodeRegistry/registry'
import type { NodeKindSpec } from '@/components/containers/nodeRegistry/index'
import { useDiscoveryStore } from '@/stores/discovery'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{
  'update:open': [v: boolean]
  'pick-kind': [kind: string]
}>()

const modelOpen = ref(props.open)
watch(() => props.open, v => { modelOpen.value = v })
watch(modelOpen, v => emit('update:open', v))

const query = ref('')
watch(() => modelOpen.value, async (v) => {
  if (v) {
    query.value = ''
    await nextTick()
    searchInputRef.value?.input?.focus?.()
  }
})

const searchInputRef = ref<any>(null)
const store = useDiscoveryStore()

// 过滤 + 按 group 分组
const filteredGroups = computed(() => {
  const q = query.value.toLowerCase().trim()
  const matched = allSpecs().filter((s: NodeKindSpec) => {
    if (s.isVisualOnly) return false         // CommentBox 等不在 Explorer
    if (s.excludeFromPalette) return false   // SubgraphInput/Output/CollapsedNode
    if (!q) return true
    const hay = `${s.kind} ${s.labelZh ?? ''} ${s.description ?? ''}`.toLowerCase()
    return hay.includes(q)
  })
  // group by group field
  const byGroup = new Map<string, NodeKindSpec[]>()
  for (const s of matched) {
    const g = s.group ?? 'misc'
    if (!byGroup.has(g)) byGroup.set(g, [])
    byGroup.get(g)!.push(s)
  }
  return Array.from(byGroup.entries())
    .map(([group, specs]) => ({
      group,
      specs: specs.sort((a, b) => a.kind.localeCompare(b.kind)),
    }))
    .sort((a, b) => a.group.localeCompare(b.group))
})

const GROUP_LABELS: Record<string, string> = {
  control: '控制流',
  variables: '变量',
  purefunc: '纯函数',
  detect: '检测',
  input: '输入',
  system: '系统/子图',
  debug: '调试',
  state: '状态',
  visualization: '可视化',
  misc: '其它',
}

function groupLabel(g: string): string {
  return GROUP_LABELS[g] ?? g
}

function labelFor(kind: string): string {
  const s = getSpec(kind)
  return s?.labelZh ?? kind
}

function onSelectKind(kind: string) {
  emit('pick-kind', kind)
  modelOpen.value = false
}

function onEsc() {
  modelOpen.value = false
}
</script>
