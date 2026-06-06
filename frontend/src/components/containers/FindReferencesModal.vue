<!-- 列出引用某变量的所有节点, 点击跳转选中. -->
<template>
  <UModal v-model:open="modelOpen" :ui="{ content: 'sm:max-w-md' }">
    <template #content>
      <div class="p-5 space-y-3 bg-default" style="min-height: 30vh; max-height: 70vh; display: flex; flex-direction: column;">
        <div class="flex items-center gap-2 shrink-0">
          <UIcon name="i-tabler-link" class="size-5 text-primary" />
          <h3 class="text-sm font-medium">{{ t('var.refs.title_prefix') }} <code class="text-amber-400">{{ varName }}</code> {{ t('var.refs.title_suffix') }}</h3>
          <span class="text-[10px] text-dimmed ml-auto">{{ t('var.refs.count_label', { n: refs.length }) }}</span>
        </div>

        <div class="flex-1 overflow-y-auto">
          <div v-if="refs.length === 0" class="text-center text-xs text-dimmed py-8 italic">
            {{ t('var.refs.empty') }}
          </div>
          <div v-else class="space-y-1">
            <button
              v-for="ref in refs"
              :key="ref.id"
              type="button"
              class="w-full text-left px-3 py-2 bg-elevated/30 hover:bg-elevated/60 rounded text-[11px] flex items-center gap-2"
              :title="t('var.refs.click_to_select') + ' ' + ref.id"
              @click="onPick(ref.id)"
            >
              <UIcon :name="iconForKind(ref.kind)" class="size-3.5 shrink-0" />
              <span class="font-medium">{{ ref.label || ref.kind }}</span>
              <span v-if="ref.label" class="text-[10px] text-dimmed">({{ ref.kind }})</span>
              <span class="text-dimmed font-mono text-[10px]">{{ ref.id }}</span>
              <span
                v-if="ref.access"
                class="text-[9px] font-semibold px-1 py-0.5 rounded"
                :class="ref.access === 'read'
                  ? 'bg-sky-500/20 text-sky-400'
                  : 'bg-amber-500/20 text-amber-400'"
              >{{ t(ref.access === 'read' ? 'var.refs.read' : 'var.refs.write') }}</span>
              <span v-if="ref.location" class="ml-auto text-[10px] text-indigo-400">{{ ref.location }}</span>
            </button>
          </div>
        </div>

        <div class="text-[10px] text-dimmed text-center shrink-0">
          {{ t('var.refs.click_to_jump_hint') }}
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useDialogOpen } from '@/composables/editor/useDialogOpen'
import { getSpec } from '@/components/containers/nodeRegistry/registry'

const { t } = useI18n()

export interface RefEntry {
  id: string
  kind: string
  label?: string           // user-set display name
  location?: string        // e.g. "主图" or "子图: foo"
  access?: 'read' | 'write'
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

const modelOpen = useDialogOpen(props, emit)

function iconForKind(kind: string): string {
  return getSpec(kind)?.visual?.icon ?? 'i-tabler-box'
}

function onPick(nodeID: string) {
  emit('pick', nodeID)
  modelOpen.value = false
}
</script>
