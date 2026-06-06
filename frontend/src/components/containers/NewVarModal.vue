<!-- New-Variable modal: used by VarNameInput on "no-intrinsic-type" nodes (GetVar/SetVar/IncVar/VarLastChange). -->
<template>
  <UModal v-model:open="modelOpen" :ui="{ content: 'sm:max-w-md' }">
    <template #content>
      <div class="p-5 space-y-4 bg-default">
        <div class="flex items-center gap-2">
          <UIcon name="i-tabler-variable-plus" class="size-5 text-primary" />
          <h3 class="text-sm font-medium">{{ t('var.new.modal_title') }}</h3>
        </div>

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

        <div class="flex justify-end gap-2 pt-2">
          <UButton variant="ghost" color="neutral" @click="modelOpen = false">{{ t('common.cancel') }}</UButton>
          <UButton color="primary" icon="i-tabler-check" :disabled="!!nameError" @click="confirm">
            {{ t('var.new.confirm') }}
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
