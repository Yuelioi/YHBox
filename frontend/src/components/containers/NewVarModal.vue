<!-- New-Variable modal: used by VarNameInput on "no-intrinsic-type" nodes (GetVar/SetVar/IncVar/VarLastChange). -->
<template>
  <BaseModal
    v-model:open="modelOpen"
    :title="t('var.new.modal_title')"
    icon="i-tabler-variable-plus"
    size="md"
  >
    <div class="space-y-4">
      <UFormField :label="t('var.new.name_label')" required :error="nameError ?? undefined">
        <UInput
          ref="nameInputRef"
          v-model="varName"
          size="sm"
          @keydown.enter="confirm"
        />
      </UFormField>

      <UFormField :label="t('var.new.type_label')">
        <USelect v-model="varType" :items="VAR_TYPE_OPTIONS" size="sm" />
      </UFormField>
    </div>

    <template #footer>
      <UButton variant="ghost" color="neutral" @click="modelOpen = false">{{ t('common.cancel') }}</UButton>
      <UButton color="primary" icon="i-tabler-check" :disabled="!!nameError" @click="confirm">
        {{ t('var.new.confirm') }}
      </UButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { ref, watch, computed, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDialogOpen } from '@/composables/editor/useDialogOpen'
import BaseModal from '@/components/common/BaseModal.vue'
import { validateVarName, VAR_TYPE_OPTIONS, zeroDefaultFor, type VarType } from '@/lib/variableRef'

const { t } = useI18n()

const props = defineProps<{
  open: boolean
  initialName: string
  existingVarNames: string[]
}>()

const emit = defineEmits<{
  'update:open': [v: boolean]
  confirm: [args: { name: string; type: VarType; default: unknown }]
}>()

const modelOpen = useDialogOpen(props, emit)

const varName = ref(props.initialName)
const varType = ref<VarType>('number')
const nameInputRef = ref<any>(null)

watch(() => props.open, async (v) => {
  if (v) {
    varName.value = props.initialName
    varType.value = 'number'
    await nextTick()
    nameInputRef.value?.inputRef?.focus?.()
    nameInputRef.value?.inputRef?.select?.()
  }
})

const nameError = computed<string | null>(() => {
  const key = validateVarName(varName.value, props.existingVarNames)
  return key ? t(key, key === 'var.error.duplicate' ? { name: varName.value.trim() } : {}) : null
})

function confirm() {
  if (nameError.value) return
  emit('confirm', {
    name: varName.value.trim(),
    type: varType.value,
    default: zeroDefaultFor(varType.value),
  })
  modelOpen.value = false
}
</script>
