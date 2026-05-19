<!-- frontend/src/components/containers/sidebar/VarsPanel.vue -->
<!-- Phase 2: skeleton with header + empty list. Phase 3 adds CRUD. -->
<template>
  <SidebarSection
    title="变量"
    icon="i-tabler-variable"
    title-color="emerald"
    :count="vars.length"
    :expanded="expanded"
    @update:expanded="$emit('update:expanded', $event)"
  >
    <template #header-action>
      <button
        type="button"
        class="text-emerald-400 hover:text-emerald-300 px-1 text-sm leading-none"
        title="添加变量"
        @click.stop="onAdd"
      >+</button>
    </template>

    <p v-if="vars.length === 0" class="text-[10px] text-dimmed italic px-1">
      暂无变量. 点 + 添加.
    </p>
    <div v-else class="space-y-1">
      <div
        v-for="v in vars"
        :key="v.name"
        class="px-2 py-1 bg-elevated/30 rounded text-[11px] flex items-center gap-2"
      >
        <UIcon name="i-tabler-grip-vertical" class="size-3 text-dimmed" />
        <span class="font-medium">{{ v.name }}</span>
        <span class="text-dimmed text-[10px]">{{ v.type }}</span>
        <span class="ml-auto text-emerald-400 text-[10px]">= {{ formatDefault(v.default) }}</span>
      </div>
    </div>
  </SidebarSection>
</template>

<script setup lang="ts">
import type { VarDecl } from '@/lib/backend'
import SidebarSection from './SidebarSection.vue'

defineProps<{
  vars: VarDecl[]
  expanded: boolean
}>()

const emit = defineEmits<{
  'update:expanded': [v: boolean]
  'add-var': []
}>()

function onAdd() {
  emit('add-var')
}

function formatDefault(d: unknown): string {
  if (d === null || d === undefined) return '(unset)'
  if (typeof d === 'object') return JSON.stringify(d)
  return String(d)
}
</script>
