<!-- Promote-to-Variable modal. -->
<template>
  <UModal v-model:open="modelOpen" :ui="{ content: 'sm:max-w-md' }">
    <template #content>
      <div class="p-5 space-y-4 bg-default">
        <div class="flex items-center gap-2">
          <UIcon name="i-tabler-arrows-up-right" class="size-5 text-primary" />
          <h3 class="text-sm font-medium">提取为变量 (Promote)</h3>
        </div>

        <p class="text-xs text-muted">
          把节点 <code class="text-primary">{{ context?.nodeID }}.{{ context?.pinName }}</code>
          的 literal 值 <code class="text-emerald-400">{{ formatLit(context?.literal) }}</code>
          提取为容器变量 + 自动建 GetVar + 连边.
        </p>

        <UFormField label="变量名" required :error="nameError ?? undefined">
          <UInput
            ref="nameInputRef"
            v-model="varName"
            placeholder="新变量名 (字母数字下划线)"
            size="sm"
            @keydown.enter="confirm"
          />
        </UFormField>

        <UFormField label="类型">
          <USelect v-model="varType" :items="TYPE_OPTIONS" size="sm" />
        </UFormField>

        <UFormField label="默认值">
          <UInput :model-value="formatLit(context?.literal)" disabled size="sm" />
          <template #help>
            <p class="text-[10px] text-muted">默认值 = 当前 literal. var 提取后, 实际运行时 var 可改.</p>
          </template>
        </UFormField>

        <div class="flex justify-end gap-2 pt-2">
          <UButton variant="ghost" color="neutral" @click="modelOpen = false">取消</UButton>
          <UButton color="primary" icon="i-tabler-check" :disabled="!!nameError" @click="confirm">
            提取
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { ref, watch, computed, nextTick } from 'vue'
import { useDialogOpen } from '@/composables/editor/useDialogOpen'
import type { VarType } from '@/lib/variableRef'

export interface PromoteContext {
  nodeID: string
  pinName: string
  pinType: VarType
  literal: unknown  // current literal value
}

const props = defineProps<{
  open: boolean
  context: PromoteContext | null
  existingVarNames: string[]
}>()
const emit = defineEmits<{
  'update:open': [v: boolean]
  confirm: [args: { varName: string; varType: VarType }]
}>()

const TYPE_OPTIONS = [
  { value: 'number' as VarType, label: 'number' },
  { value: 'string' as VarType, label: 'string' },
  { value: 'bool' as VarType, label: 'bool' },
  { value: 'point' as VarType, label: 'point' },
  { value: 'any' as VarType, label: 'any' },
]

const modelOpen = useDialogOpen(props, emit)

const varName = ref('')
const varType = ref<VarType>('any')
const nameInputRef = ref<any>(null)

watch(() => props.context, async (c) => {
  if (c) {
    varName.value = suggestName(c.pinName)
    varType.value = c.pinType
    await nextTick()
    nameInputRef.value?.input?.focus?.()
    nameInputRef.value?.input?.select?.()
  }
}, { immediate: true })

function suggestName(pinName: string): string {
  return pinName.replace(/[^a-zA-Z0-9_]/g, '_').toLowerCase()
}

const nameError = computed<string | null>(() => {
  const n = varName.value.trim()
  if (!n) return '名称不能为空'
  if (!/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(n)) return '需字母/_ 起, 仅字母数字下划线'
  if (props.existingVarNames.includes(n)) return `已存在 var '${n}' (重名)`
  return null
})

function formatLit(v: unknown): string {
  if (v === null || v === undefined) return '(unset)'
  if (typeof v === 'object') return JSON.stringify(v)
  return String(v)
}

function confirm() {
  if (nameError.value) return
  emit('confirm', { varName: varName.value.trim(), varType: varType.value })
  modelOpen.value = false
}
</script>
