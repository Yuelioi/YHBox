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
          <span class="text-[10px] text-dimmed">Esc / Tab 关</span>
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

        <!-- Categorized sections: stacked vertically, each with its own tinted background -->
        <div class="flex-1 overflow-y-auto pr-2">
          <div v-if="filteredGroups.length === 0" class="text-center text-xs text-dimmed py-8 italic">
            没匹配的节点
          </div>
          <div v-else class="space-y-3">
            <div v-for="g in filteredGroups" :key="g.group" class="rounded p-2.5" :class="groupColorClass(g.group)">
              <div class="text-[11px] font-medium mb-2 flex items-center gap-2" :class="groupHeaderClass(g.group)">
                <UIcon :name="groupIcon(g.group)" class="size-3.5" />
                <span>{{ groupLabel(g.group) }}</span>
                <span class="text-[10px] opacity-70">({{ g.specs.length }})</span>
              </div>
              <div class="grid gap-1" style="grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));">
                <div
                  v-for="spec in g.specs"
                  :key="spec.kind"
                  class="flex items-center gap-1 px-2 py-1 bg-default/40 hover:bg-default/80 rounded text-[11px] cursor-pointer"
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
  purefunc: '运算',
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

const GROUP_ICONS: Record<string, string> = {
  control: 'i-tabler-arrow-fork',
  variables: 'i-tabler-variable',
  purefunc: 'i-tabler-math-function',
  detect: 'i-tabler-target',
  input: 'i-tabler-mouse',
  system: 'i-tabler-cpu',
  debug: 'i-tabler-bug',
  state: 'i-tabler-database',
  visualization: 'i-tabler-chart-bar',
  misc: 'i-tabler-dots',
}
function groupIcon(g: string): string {
  return GROUP_ICONS[g] ?? 'i-tabler-folder'
}

// per-group background tint (subtle — distinguishable without being loud)
const GROUP_BG: Record<string, string> = {
  control: 'bg-blue-500/10 border border-blue-500/30',
  variables: 'bg-emerald-500/10 border border-emerald-500/30',
  purefunc: 'bg-purple-500/10 border border-purple-500/30',
  detect: 'bg-orange-500/10 border border-orange-500/30',
  input: 'bg-pink-500/10 border border-pink-500/30',
  system: 'bg-cyan-500/10 border border-cyan-500/30',
  debug: 'bg-red-500/10 border border-red-500/30',
  state: 'bg-teal-500/10 border border-teal-500/30',
  visualization: 'bg-indigo-500/10 border border-indigo-500/30',
  misc: 'bg-zinc-500/10 border border-zinc-500/30',
}
function groupColorClass(g: string): string {
  return GROUP_BG[g] ?? GROUP_BG.misc
}

// per-group header text color (matches bg tint)
const GROUP_HEADER: Record<string, string> = {
  control: 'text-blue-400',
  variables: 'text-emerald-400',
  purefunc: 'text-purple-400',
  detect: 'text-orange-400',
  input: 'text-pink-400',
  system: 'text-cyan-400',
  debug: 'text-red-400',
  state: 'text-teal-400',
  visualization: 'text-indigo-400',
  misc: 'text-zinc-400',
}
function groupHeaderClass(g: string): string {
  return GROUP_HEADER[g] ?? GROUP_HEADER.misc
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
