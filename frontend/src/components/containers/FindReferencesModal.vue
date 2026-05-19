<!-- frontend/src/components/containers/FindReferencesModal.vue -->
<!-- Editor v2 C polish — list引用变量的所有节点, 可点跳转选中.
     Replaces toast UX from C2 (commit 9bdf110). -->
<template>
  <UModal v-model:open="modelOpen" :ui="{ content: 'sm:max-w-md' }">
    <template #content>
      <div class="p-5 space-y-3 bg-default" style="min-height: 30vh; max-height: 70vh; display: flex; flex-direction: column;">
        <div class="flex items-center gap-2 shrink-0">
          <UIcon name="i-tabler-link" class="size-5 text-primary" />
          <h3 class="text-sm font-medium">变量 <code class="text-amber-400">{{ varName }}</code> 的引用</h3>
          <span class="text-[10px] text-dimmed ml-auto">{{ refs.length }} 个节点</span>
        </div>

        <div class="flex-1 overflow-y-auto">
          <div v-if="refs.length === 0" class="text-center text-xs text-dimmed py-8 italic">
            无引用节点
          </div>
          <div v-else class="space-y-1">
            <button
              v-for="ref in refs"
              :key="ref.id"
              type="button"
              class="w-full text-left px-3 py-2 bg-elevated/30 hover:bg-elevated/60 rounded text-[11px] flex items-center gap-2"
              :title="`点击选中 + 跳到画布 ${ref.id}`"
              @click="onPick(ref.id)"
            >
              <UIcon :name="iconForKind(ref.kind)" class="size-3.5 shrink-0" />
              <span class="font-medium">{{ ref.kind }}</span>
              <span class="text-dimmed font-mono text-[10px]">{{ ref.id }}</span>
              <span v-if="ref.location" class="ml-auto text-[10px] text-indigo-400">{{ ref.location }}</span>
            </button>
          </div>
        </div>

        <div class="text-[10px] text-dimmed text-center shrink-0">
          点行 → 跳到节点 (关闭 modal + 选中节点)
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { getSpec } from '@/components/containers/nodeRegistry/registry'

export interface RefEntry {
  id: string
  kind: string
  location?: string  // e.g. "主图" or "子图: foo"
}

const props = defineProps<{
  open: boolean
  varName: string
  refs: RefEntry[]
}>()
const emit = defineEmits<{
  'update:open': [v: boolean]
  pick: [nodeID: string]
}>()

const modelOpen = ref(props.open)
watch(() => props.open, v => { modelOpen.value = v })
watch(modelOpen, v => emit('update:open', v))

function iconForKind(kind: string): string {
  return getSpec(kind)?.visual?.icon ?? 'i-tabler-box'
}

function onPick(nodeID: string) {
  emit('pick', nodeID)
  modelOpen.value = false
}
</script>
