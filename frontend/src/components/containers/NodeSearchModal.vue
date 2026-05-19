<!-- frontend/src/components/containers/NodeSearchModal.vue -->
<!-- Editor v2 polish round 2 — Ctrl+F canvas search.
     Walks main + all subgraphs, matches label/kind/id (substring).
     Click result → emit pick(r) — parent jumps + selects.
     UE Blueprint "Find in Blueprint" equivalent. -->
<template>
  <UModal v-model:open="modelOpen" :ui="{ content: 'sm:max-w-2xl' }">
    <template #content>
      <div class="bg-default" style="min-height: 40vh; max-height: 70vh; display: flex; flex-direction: column;">
        <div class="p-3 border-b border-default flex items-center gap-2 shrink-0">
          <UIcon name="i-tabler-search" class="size-4 text-primary" />
          <UInput
            ref="searchInputRef"
            v-model="query"
            placeholder="搜节点 (label / kind / id substring)..."
            size="md"
            class="flex-1"
            @keydown.escape.stop="close"
            @keydown.enter.stop="pickFirst"
            @keydown.up.prevent="selectPrev"
            @keydown.down.prevent="selectNext"
          />
        </div>
        <div class="flex-1 overflow-y-auto p-2">
          <div v-if="results.length === 0" class="text-center text-xs text-dimmed py-8 italic">
            <span v-if="!query.trim()">输入关键词搜索节点 (跨主图 + 所有子图)</span>
            <span v-else>无匹配</span>
          </div>
          <div v-else class="space-y-1">
            <button
              v-for="(r, idx) in results"
              :key="`${r.location}/${r.id}`"
              type="button"
              class="w-full text-left px-3 py-2 rounded text-[11px] flex items-center gap-2"
              :class="{
                'bg-elevated/60': idx === activeIdx,
                'hover:bg-elevated/40': idx !== activeIdx,
              }"
              @click="pick(r)"
              @mouseenter="activeIdx = idx"
            >
              <UIcon :name="iconFor(r.kind)" class="size-3.5 shrink-0" :class="iconColorFor(r.kind)" />
              <span class="font-medium">{{ r.label || labelZhFor(r.kind) || r.kind }}</span>
              <span v-if="r.label" class="text-[10px] text-dimmed">({{ labelZhFor(r.kind) || r.kind }})</span>
              <span class="text-dimmed font-mono text-[10px]">{{ r.id }}</span>
              <span class="ml-auto text-[10px] text-indigo-400">{{ r.location }}</span>
            </button>
          </div>
        </div>
        <div class="px-3 py-1.5 border-t border-default text-[9px] text-dimmed flex justify-between shrink-0">
          <span>↑↓ 导航 · Enter 跳转 · Esc 关</span>
          <span>{{ results.length }} 个匹配</span>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import { getSpec } from '@/components/containers/nodeRegistry/registry'
import { nodeIconColor } from '@/composables/editor/useNodeGroupColor'

export interface NodeSearchResult {
  id: string
  kind: string
  label?: string
  location: string         // '主图' | '子图: <label>'
  sgID: string | null      // null if in main graph, else subgraph ID
}

const props = defineProps<{
  open: boolean
  results: NodeSearchResult[]   // computed by parent walking all graphs
}>()
const emit = defineEmits<{
  'update:open': [v: boolean]
  'update:query': [q: string]
  pick: [r: NodeSearchResult]
}>()

const modelOpen = ref(props.open)
watch(() => props.open, v => { modelOpen.value = v })
watch(modelOpen, v => emit('update:open', v))

const query = ref('')
watch(query, v => emit('update:query', v))
const activeIdx = ref(0)
const searchInputRef = ref<any>(null)

watch(() => modelOpen.value, async (v) => {
  if (v) {
    query.value = ''
    activeIdx.value = 0
    await nextTick()
    searchInputRef.value?.input?.focus?.()
  }
})
watch(() => props.results, () => { activeIdx.value = 0 })

function labelZhFor(kind: string): string {
  return getSpec(kind)?.labelZh ?? ''
}
function iconFor(kind: string): string {
  return getSpec(kind)?.visual?.icon ?? 'i-tabler-box'
}
function iconColorFor(kind: string): string {
  return nodeIconColor(getSpec(kind))
}

function selectPrev() {
  if (props.results.length === 0) return
  activeIdx.value = (activeIdx.value - 1 + props.results.length) % props.results.length
}
function selectNext() {
  if (props.results.length === 0) return
  activeIdx.value = (activeIdx.value + 1) % props.results.length
}
function pickFirst() {
  const r = props.results[activeIdx.value]
  if (r) pick(r)
}
function pick(r: NodeSearchResult) {
  emit('pick', r)
  close()
}
function close() {
  modelOpen.value = false
}
</script>
