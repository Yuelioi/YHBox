<template>
  <UInput
    v-if="scope === 'local'"
    :model-value="modelValue" size="sm" class="w-full"
    @update:model-value="(v: string) => emit('update:modelValue', v)"
  />
  <div v-else class="relative">
    <UInput
      ref="inputRef" :model-value="draft" size="sm" class="w-full"
      :ui="undeclared ? { base: 'ring-1 ring-warning' } : undefined"
      @update:model-value="onType" @focus="open = true" @blur="onBlur"
    />
    <div v-if="open" class="absolute z-50 mt-1 w-full bg-default border border-default rounded shadow-lg max-h-60 overflow-y-auto">
      <button v-for="v in filtered" :key="v.name" type="button"
        class="w-full flex items-center justify-between px-2 py-1 text-xs hover:bg-elevated"
        @mousedown.prevent="pickExisting(v)">
        <span>{{ v.name }}</span><span class="text-[10px] text-dimmed font-mono">·{{ v.type }}</span>
      </button>
      <button v-if="showCreate" type="button"
        class="w-full flex items-center gap-1 px-2 py-1 text-xs text-primary hover:bg-elevated border-t border-default"
        @mousedown.prevent="onCreate">
        <UIcon name="i-tabler-plus" class="size-3.5" />
        {{ t('var.input.create', { name: draft.trim(), type: captureType ?? '?' }) }}
      </button>
    </div>
    <p v-if="conflictMsg" class="text-[11px] text-warning leading-snug mt-0.5">{{ conflictMsg }}</p>
    <button v-if="undeclared" type="button" class="text-[11px] text-warning underline mt-0.5" @click="onCreate">
      {{ t('var.input.declare_now') }}
    </button>
    <NewVarModal v-model:open="modalOpen" :initial-name="draft.trim()"
      :existing-var-names="declaredVars.map(v => v.name)" @confirm="onModalConfirm" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import NewVarModal from '../NewVarModal.vue'
import { validateVarName, zeroDefaultFor, type VarType } from '@/lib/variableRef'
import { filterVars, shouldShowCreate, computeConflict, resolveOnBlur } from './VarNameInput.helpers'

const { t } = useI18n()

const props = withDefaults(defineProps<{
  modelValue: string
  declaredVars: { name: string; type: VarType }[]
  captureType?: string
  scope?: 'auto' | 'global' | 'local'
}>(), { scope: 'auto' })

const emit = defineEmits<{
  'update:modelValue': [name: string]
  'declare-var': [args: { name: string; type: VarType; default: unknown }]
}>()

const inputRef = ref<any>(null)
const open = ref(false)
const modalOpen = ref(false)
const draft = ref(props.modelValue)

watch(() => props.modelValue, (v) => { draft.value = v })

const names = computed(() => props.declaredVars.map(v => v.name))
const filtered = computed(() => filterVars(props.declaredVars, draft.value))
const showCreate = computed(() => shouldShowCreate(draft.value, names.value))
const undeclared = computed(() =>
  props.modelValue.trim() !== '' && !names.value.includes(props.modelValue))
const conflictMsg = computed(() => {
  const c = computeConflict(props.modelValue, props.declaredVars, props.captureType)
  if (!c) return ''
  return t('var.input.type_mismatch', { cap: c.cap, dst: c.dst })
})

function onType(v: string) { draft.value = v; open.value = true }
function pickExisting(v: { name: string; type: VarType }) { open.value = false; emit('update:modelValue', v.name) }
function declare(name: string, type: VarType) {
  emit('declare-var', { name, type, default: zeroDefaultFor(type) })
  emit('update:modelValue', name)
}
function onCreate() {
  open.value = false
  const name = draft.value.trim()
  if (validateVarName(name, names.value) !== null) return // 非法/重名不建
  if (props.captureType) declare(name, props.captureType as VarType)
  else modalOpen.value = true
}
function onModalConfirm(a: { name: string; type: VarType; default: unknown }) {
  emit('declare-var', a)
  emit('update:modelValue', a.name)
}
function onBlur() {
  setTimeout(() => {
    open.value = false
    const resolved = resolveOnBlur(draft.value, props.declaredVars)
    if (resolved !== null) emit('update:modelValue', resolved)
    else draft.value = props.modelValue
  }, 150)
}
</script>
