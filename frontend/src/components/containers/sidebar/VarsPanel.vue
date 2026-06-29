<!-- 容器变量面板. CRUD + reorder. Drag-out 走 useEditorDragDrop. -->
<template>
  <SidebarSection
    :title="t('var.title')"
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
        class="text-primary hover:text-primary/80 px-1 text-base leading-none"
        :title="t('var.add')"
        @click.stop="$emit('add-var')"
      >+</button>
    </template>

    <!-- Search input: only shown when >4 vars -->
    <UInput
      v-if="vars.length > 4"
      v-model="searchQuery"
      size="xs"
      :placeholder="t('var.search_placeholder')"
      icon="i-tabler-search"
      class="mb-2"
    />

    <!-- Empty state -->
    <p v-if="vars.length === 0" class="text-[10px] text-dimmed italic px-1">
      {{ t('var.empty') }}
    </p>

    <!-- Search-active: simple list, no reorder -->
    <div v-else-if="searchQuery.trim()" class="space-y-1 max-h-64 overflow-y-auto pr-1">
      <p v-if="filteredVars.length === 0" class="text-[10px] text-dimmed italic px-1">{{ t('var.no_match') }}</p>
      <VarRow
        v-for="v in filteredVars"
        :key="v.name"
        :decl="v"
        :existing-names="vars.map(x => x.name)"
        @rename="(oldN, newN) => $emit('rename-var', oldN, newN)"
        @update-field="(n, f, val) => $emit('update-var-field', n, f, val)"
        @delete="(n) => $emit('request-delete', n)"
        @insert-incvar="(n) => $emit('insert-incvar', n)"
      />
    </div>

    <!-- No search: full list with reorder -->
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
        @insert-incvar="(n) => $emit('insert-incvar', n)"
      />
    </VueDraggable>
  </SidebarSection>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { VueDraggable } from 'vue-draggable-plus'
import type { VarDecl } from '@/lib/backend'
import SidebarSection from './SidebarSection.vue'
import VarRow from './VarRow.vue'

const { t } = useI18n()

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
  'insert-incvar': [name: string]
}>()

const searchQuery = ref('')

const orderedVars = ref<VarDecl[]>([...props.vars])
watch(() => props.vars, (v) => { orderedVars.value = [...v] }, { deep: true })

const filteredVars = computed(() => {
  const q = searchQuery.value.toLowerCase().trim()
  if (!q) return orderedVars.value
  return orderedVars.value.filter(v => v.name.toLowerCase().includes(q))
})

function onReorderEnd(e: { oldIndex?: number; newIndex?: number }) {
  if (e.oldIndex === undefined || e.newIndex === undefined) return
  if (e.oldIndex === e.newIndex) return
  emit('reorder-vars', e.oldIndex, e.newIndex)
}
</script>
