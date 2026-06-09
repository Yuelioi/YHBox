<!-- frontend/src/components/containers/sidebar/DeleteVarConfirmModal.vue -->
<!-- 删除变量 — 2 选项: 保留 invalid 节点 / cascade 删. 默认保留. -->
<template>
  <BaseModal v-model:open="modelOpen" :title="t('var.delete.title', { name: varName })" icon="i-tabler-alert-triangle" icon-color="warning" size="md">
    <div class="space-y-4">
      <p class="text-xs text-muted">
        {{ t('var.delete.refs_prefix') }} <strong class="text-warning">{{ refCount }}</strong> {{ t('var.delete.refs_suffix') }}
      </p>

      <div class="max-h-32 overflow-y-auto bg-elevated/40 rounded p-2 text-[10px] font-mono space-y-0.5">
        <div v-for="id in refIDs.slice(0, 20)" :key="id" class="text-dimmed">{{ id }}</div>
        <div v-if="refIDs.length > 20" class="text-dimmed italic">{{ t('var.delete.more_count', { n: refIDs.length - 20 }) }}</div>
      </div>

      <p class="text-[11px] text-muted">{{ t('var.delete.choose_mode') }}</p>
    </div>

    <template #footer>
      <div class="flex flex-col gap-2 w-full">
        <UButton color="primary" variant="soft" icon="i-tabler-shield" @click="onConfirm(false)">
          {{ t('var.delete.mode_keep') }}
          <template #trailing>
            <span class="text-[10px] text-dimmed ml-1">{{ t('var.delete.mode_keep_desc_inline') }}</span>
          </template>
        </UButton>
        <UButton color="error" variant="soft" icon="i-tabler-trash" @click="onConfirm(true)">
          {{ t('var.delete.mode_remove', { count: refCount }) }}
        </UButton>
        <UButton color="neutral" variant="ghost" @click="modelOpen = false">{{ t('common.cancel') }}</UButton>
      </div>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDialogOpen } from '@/composables/editor/useDialogOpen'
import BaseModal from '@/components/common/BaseModal.vue'

const { t } = useI18n()

const props = defineProps<{
  open: boolean
  varName: string
  refIDs: string[]
}>()

const emit = defineEmits<{
  'update:open': [v: boolean]
  confirm: [cascade: boolean]
}>()

const modelOpen = useDialogOpen(props, emit)

const refCount = computed(() => props.refIDs.length)

function onConfirm(cascade: boolean) {
  emit('confirm', cascade)
  modelOpen.value = false
}
</script>
