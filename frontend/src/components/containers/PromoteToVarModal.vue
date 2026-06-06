<!-- Promote-to-Variable modal. -->
<template>
  <UModal v-model:open="modelOpen" :ui="{ content: 'sm:max-w-md' }">
    <template #content>
      <div class="p-5 space-y-4 bg-default">
        <div class="flex items-center gap-2">
          <UIcon name="i-tabler-arrows-up-right" class="size-5 text-primary" />
          <h3 class="text-sm font-medium">{{ t('var.promote.modal_title') }}</h3>
        </div>

        <p class="text-xs text-muted">
          {{ t('var.promote.desc_prefix') }} <code class="text-primary">{{ context?.nodeID }}.{{ context?.pinName }}</code>
          {{ t('var.promote.desc_literal_label') }} <code class="text-emerald-400">{{ formatLit(context?.literal) }}</code>
          {{ t('var.promote.desc_suffix') }}
        </p>

        <UFormField :label="t('var.promote.name_label')" required :error="nameError ?? undefined">
          <UInput
            ref="nameInputRef"
            v-model="varName"
            :placeholder="t('var.promote.name_input_placeholder')"
            size="sm"
            @keydown.enter="confirm"
          />
        </UFormField>

        <UFormField :label="t('var.promote.type_label')">
          <USelect v-model="varType" :items="VAR_TYPE_OPTIONS" size="sm" />
        </UFormField>

        <UFormField :label="t('var.promote.default_label')">
          <UInput :model-value="formatLit(context?.literal)" disabled size="sm" />
          <template #help>
            <p class="text-[10px] text-muted">{{ t('var.promote.default_help') }}</p>
          </template>
        </UFormField>

        <div class="flex justify-end gap-2 pt-2">
          <UButton variant="ghost" color="neutral" @click="modelOpen = false">{{ t('common.cancel') }}</UButton>
          <UButton color="primary" icon="i-tabler-check" :disabled="!!nameError" @click="confirm">
            {{ t('var.promote.confirm') }}
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { ref, watch, computed, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDialogOpen } from '@/composables/editor/useDialogOpen'
import { validateVarName, VAR_TYPE_OPTIONS, type VarType } from '@/lib/variableRef'

const { t } = useI18n()

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

const modelOpen = useDialogOpen(props, emit)

const varName = ref('')
const varType = ref<VarType>('any')
const nameInputRef = ref<any>(null)

watch(() => props.context, async (c) => {
  if (c) {
    varName.value = suggestName(c.pinName)
    varType.value = c.pinType
    await nextTick()
    nameInputRef.value?.inputRef?.focus?.()
    nameInputRef.value?.inputRef?.select?.()
  }
}, { immediate: true })

function suggestName(pinName: string): string {
  return pinName.replace(/[^a-zA-Z0-9_]/g, '_').toLowerCase()
}

const nameError = computed<string | null>(() => {
  const key = validateVarName(varName.value, props.existingVarNames)
  return key ? t(key, key === 'var.error.duplicate' ? { name: varName.value.trim() } : {}) : null
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
