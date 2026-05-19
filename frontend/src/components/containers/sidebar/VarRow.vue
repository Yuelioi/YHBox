<!-- frontend/src/components/containers/sidebar/VarRow.vue -->
<!-- Var 行: 折叠摘要 + ✎ 切换展开行内编辑. Phase 4: drag-start + IncVar hover button. -->
<template>
  <div class="rounded text-[11px] group" :class="expanded ? 'bg-elevated/60 border border-default' : 'bg-elevated/30'">
    <!-- Collapsed -->
    <div
      v-if="!expanded"
      class="px-2 py-1 flex items-center gap-2"
      draggable="true"
      @dragstart="onDragStart"
      @dblclick.stop="expanded = true"
    >
      <UIcon name="i-tabler-grip-vertical" class="size-3 text-dimmed cursor-grab" />
      <span class="font-medium">{{ decl.name }}</span>
      <span class="text-dimmed text-[10px]">{{ shortType(decl.type) }}</span>
      <span class="ml-auto text-emerald-400 text-[10px] truncate max-w-20">
        = {{ formatDefault(decl.default) }}
      </span>
      <button
        v-if="decl.type === 'number'"
        type="button"
        class="text-dimmed hover:text-default px-0.5 opacity-0 group-hover:opacity-100 transition-opacity"
        title="插入 IncVar (delta=1)"
        @click="$emit('insert-incvar', decl.name)"
      >
        <UIcon name="i-tabler-circle-plus" class="size-3" />
      </button>
      <button type="button" class="text-dimmed hover:text-default px-1" title="编辑" @click="expanded = true">
        <UIcon name="i-tabler-edit" class="size-3" />
      </button>
    </div>

    <!-- Expanded -->
    <div v-else class="p-1.5 space-y-1.5 min-w-0">
      <div class="flex items-center gap-1 min-w-0">
        <UIcon name="i-tabler-grip-vertical" class="size-3 text-dimmed cursor-grab shrink-0" />
        <input
          ref="nameInputRef"
          v-model="editName"
          class="flex-1 min-w-0 bg-elevated/80 border rounded px-1.5 py-0.5 text-[11px] font-medium focus:outline-none"
          :class="nameError ? 'border-red-500' : 'border-default focus:border-emerald-500'"
          :title="nameError || ''"
          @blur="commitName"
          @keydown.enter="commitName"
          @keydown.esc="cancelName"
        >
        <button type="button" class="text-red-500 hover:text-red-400 px-1 shrink-0" title="删除" @click="$emit('delete', decl.name)">
          <UIcon name="i-tabler-trash" class="size-3" />
        </button>
      </div>

      <div class="flex items-center gap-1.5 min-w-0">
        <USelect
          v-model="editType"
          :items="TYPE_OPTIONS_OBJ"
          size="xs"
          class="w-20 shrink-0"
          @update:model-value="(v: VarType) => commitField('type', v)"
        />
        <span class="text-[10px] text-dimmed shrink-0">=</span>
        <!-- Default value editor by type -->
        <VarPointInput
          v-if="editType === 'point'"
          :model-value="defaultAsPoint"
          class="min-w-0"
          @update:model-value="(v) => commitField('default', v)"
        />
        <UCheckbox
          v-else-if="editType === 'bool'"
          :model-value="!!editDefault"
          @update:model-value="(v: boolean) => commitField('default', v)"
        />
        <input
          v-else
          :type="editType === 'number' ? 'number' : 'text'"
          :step="editType === 'number' ? 'any' : undefined"
          :value="editDefault ?? ''"
          class="flex-1 min-w-0 w-0 bg-elevated/80 border border-default rounded px-1 py-0.5 text-[10px] focus:border-emerald-500 focus:outline-none"
          :placeholder="editType === 'any' ? 'any (自负)' : ''"
          @input="commitField('default', parseDefault($event, editType))"
        >
        <button
          type="button"
          class="text-dimmed hover:text-default px-1 shrink-0"
          title="收起"
          @click="expanded = false"
        >
          <UIcon name="i-tabler-x" class="size-3" />
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import type { VarDecl } from '@/lib/backend'
import VarPointInput from './VarPointInput.vue'
import { startEditorDrag } from '@/composables/editor/useEditorDragDrop'

const props = defineProps<{
  decl: VarDecl
  existingNames: string[]   // for uniqueness validation
}>()

const emit = defineEmits<{
  rename: [oldName: string, newName: string]
  'update-field': [name: string, field: 'type' | 'default', value: unknown]
  delete: [name: string]
  'insert-incvar': [name: string]
}>()

const TYPE_OPTIONS = ['number', 'string', 'bool', 'point', 'any'] as const
type VarType = typeof TYPE_OPTIONS[number]

const TYPE_OPTIONS_OBJ = TYPE_OPTIONS.map(v => ({ value: v, label: v }))

const expanded = ref(false)
const nameInputRef = ref<HTMLInputElement | null>(null)
const editName = ref(props.decl.name)
const editType = ref<VarType>(props.decl.type as VarType)
const editDefault = ref(props.decl.default)

watch(() => props.decl, (d) => {
  editName.value = d.name
  editType.value = d.type
  editDefault.value = d.default
}, { deep: true })

watch(expanded, async (v) => {
  if (v) {
    await nextTick()
    nameInputRef.value?.focus()
    nameInputRef.value?.select()
  }
})

const nameError = computed<string | null>(() => {
  const n = editName.value.trim()
  if (!n) return '名称不能为空'
  if (!/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(n)) return '需以字母/_ 起, 仅含字母数字下划线'
  if (n !== props.decl.name && props.existingNames.includes(n)) return `已存在 var '${n}'`
  return null
})

const defaultAsPoint = computed(() => {
  const d = editDefault.value as { x?: number; y?: number } | null | undefined
  if (!d || typeof d !== 'object') return { x: 0, y: 0 }
  return { x: d.x ?? 0, y: d.y ?? 0 }
})

function commitName() {
  if (nameError.value) return
  const n = editName.value.trim()
  if (n !== props.decl.name) emit('rename', props.decl.name, n)
}

function cancelName() {
  editName.value = props.decl.name
  expanded.value = false
}

function commitField(field: 'type' | 'default', value: unknown) {
  emit('update-field', props.decl.name, field, value)
}

function parseDefault(e: Event, type: string): unknown {
  const raw = (e.target as HTMLInputElement).value
  if (type === 'number') {
    const n = parseFloat(raw)
    return Number.isNaN(n) ? null : n
  }
  return raw
}

function shortType(t: string): string {
  return t === 'number' ? 'num' : t
}

function formatDefault(d: unknown): string {
  if (d === null || d === undefined) return '(unset)'
  if (typeof d === 'object') return JSON.stringify(d)
  return String(d)
}

function onDragStart(e: DragEvent) {
  startEditorDrag(
    { type: 'var', ref: { name: props.decl.name, type: props.decl.type as VarType }, modifier: 'none' },
    e,
  )
}
</script>
