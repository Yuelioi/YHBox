<!-- frontend/src/components/containers/sidebar/VarsPanel.vue -->
<!-- 完整 CRUD + reorder. Drag-out (Phase 4) 走外部 useEditorDragDrop. -->
<template>
  <SidebarSection
    title="变量"
    icon="i-tabler-variable"
    title-color="emerald"
    :count="undefined"
    :expanded="expanded"
    @update:expanded="$emit('update:expanded', $event)"
  >
    <template #header-action>
      <span class="text-[10px] text-dimmed">{{ vars.length }} vars · {{ usageCount }} refs</span>
      <button
        type="button"
        class="text-emerald-400 hover:text-emerald-300 px-1 text-base leading-none"
        title="添加变量"
        @click.stop="$emit('add-var')"
      >+</button>
    </template>

    <p v-if="vars.length === 0" class="text-[10px] text-dimmed italic px-1">
      暂无变量. 点 + 添加.
    </p>

    <VueDraggable
      v-else
      v-model="orderedVars"
      :animation="150"
      handle=".cursor-grab"
      class="space-y-1"
      @end="onReorderEnd"
    >
      <VarRow
        v-for="v in orderedVars"
        :key="v.name"
        :decl="v"
        :existing-names="vars.map(x => x.name)"
        @rename="(oldN, newN) => $emit('rename-var', oldN, newN)"
        @update-field="(n, f, val) => $emit('update-var-field', n, f, val)"
        @delete="(n) => $emit('request-delete', n)"
      />
    </VueDraggable>
  </SidebarSection>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { VueDraggable } from 'vue-draggable-plus'
import type { VarDecl } from '@/lib/backend'
import SidebarSection from './SidebarSection.vue'
import VarRow from './VarRow.vue'

const props = defineProps<{
  vars: VarDecl[]
  expanded: boolean
  usageCount: number
}>()

const emit = defineEmits<{
  'update:expanded': [v: boolean]
  'add-var': []
  'rename-var': [oldName: string, newName: string]
  'update-var-field': [name: string, field: 'type' | 'default', value: unknown]
  'request-delete': [name: string]
  'reorder-vars': [fromIdx: number, toIdx: number]
}>()

const orderedVars = ref<VarDecl[]>([...props.vars])
watch(() => props.vars, (v) => { orderedVars.value = [...v] }, { deep: true })

function onReorderEnd(e: { oldIndex?: number; newIndex?: number }) {
  if (e.oldIndex === undefined || e.newIndex === undefined) return
  if (e.oldIndex === e.newIndex) return
  emit('reorder-vars', e.oldIndex, e.newIndex)
}
</script>
