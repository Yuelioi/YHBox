<template>
  <BaseModal
    :open="open"
    :title="title"
    icon="i-tabler-photo-cog"
    size="lg"
    @update:open="emit('update:open', $event)"
  >
    <div class="space-y-2">
      <div
        v-for="variant in variants"
        :key="variantKey(variant)"
        class="flex items-center gap-3 rounded-lg border border-default px-3 py-2.5"
      >
        <BlobPreview :blob="variant.blob" :alt="title" expandable class="size-14 shrink-0" />
        <div class="min-w-0 flex-1">
          <p class="text-sm font-medium text-highlighted">{{ resolution(variant) }}</p>
          <p class="truncate font-mono text-[10px] text-dimmed">{{ variant.blob.digest }}</p>
        </div>
        <div class="flex shrink-0 items-center gap-1">
          <UButton
            color="neutral"
            variant="ghost"
            icon="i-tabler-camera-rotate"
            :disabled="busy"
            :aria-label="
              t('assets.templates.recapture_variant', { resolution: resolution(variant) })
            "
            :title="t('assets.templates.recapture_variant', { resolution: resolution(variant) })"
            @click="startRecapture(variant)"
          />
          <UButton
            color="error"
            variant="ghost"
            icon="i-tabler-trash"
            :disabled="busy"
            :aria-label="t('assets.templates.remove_variant')"
            @click="requestRemove(variant)"
          />
        </div>
      </div>
    </div>
    <template #footer>
      <UButton color="neutral" variant="ghost" @click="emit('update:open', false)">
        {{ t('common.close') }}
      </UButton>
      <UButton
        icon="i-tabler-camera-plus"
        :loading="busy"
        :disabled="addDisabled"
        @click="startAdd"
      >
        {{ t('assets.templates.add_current_resolution') }}
      </UButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useConfirm } from '@/composables/useConfirm'
import type { BlobRef } from '@/lib/backend'
import BaseModal from '@/components/common/BaseModal.vue'
import BlobPreview from '@/components/common/BlobPreview.vue'

export interface ImageVariantItem {
  id?: string
  resolution: [number, number]
  blob: BlobRef
}

const props = withDefaults(
  defineProps<{
    open: boolean
    title: string
    variants: ImageVariantItem[]
    busy?: boolean
    addDisabled?: boolean
  }>(),
  { busy: false, addDisabled: false },
)
const emit = defineEmits<{
  'update:open': [open: boolean]
  add: []
  recapture: [variant: ImageVariantItem]
  remove: [variant: ImageVariantItem]
  'remove-blocked': []
}>()
const { t } = useI18n()
const { confirm } = useConfirm()

function resolution(variant: ImageVariantItem): string {
  return `${variant.resolution[0]}×${variant.resolution[1]}`
}

function variantKey(variant: ImageVariantItem): string {
  return variant.id ?? `${resolution(variant)}:${variant.blob.digest}`
}

function startAdd(): void {
  emit('add')
  emit('update:open', false)
}

function startRecapture(variant: ImageVariantItem): void {
  emit('recapture', variant)
  emit('update:open', false)
}

async function requestRemove(variant: ImageVariantItem): Promise<void> {
  if (props.variants.length <= 1) {
    emit('remove-blocked')
    return
  }
  const accepted = await confirm({
    title: t('assets.templates.remove_variant_title', { resolution: resolution(variant) }),
    description: t('assets.templates.remove_variant_description'),
    confirmText: t('common.delete'),
    color: 'error',
  })
  if (accepted === true) emit('remove', variant)
}
</script>
