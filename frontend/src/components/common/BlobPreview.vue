<template>
  <div
    class="relative flex min-h-10 min-w-10 items-center justify-center overflow-hidden rounded-lg border border-default bg-sunken"
    :data-preview-state="state"
  >
    <CappedPreviewImage v-if="objectURL" :src="objectURL" :alt="alt" />
    <USkeleton v-else-if="state === 'loading'" class="absolute inset-0 rounded-none" />
    <div v-else class="flex flex-col items-center gap-1 px-2 text-center text-dimmed">
      <UIcon name="i-tabler-photo-off" class="size-4" />
      <span v-if="showFailureLabel" class="text-[10px]">{{ t('assets.preview_unavailable') }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend, type BlobRef } from '@/lib/backend'
import CappedPreviewImage from '@/components/common/CappedPreviewImage.vue'

const props = withDefaults(
  defineProps<{
    blob: BlobRef
    alt?: string
    showFailureLabel?: boolean
  }>(),
  { alt: '', showFailureLabel: false },
)
const emit = defineEmits<{ state: [state: 'loading' | 'ready' | 'unavailable'] }>()
const { t } = useI18n()
const state = ref<'loading' | 'ready' | 'unavailable'>('loading')
const objectURL = ref('')
let requestGeneration = 0

watch(
  () => [props.blob.mediaType, props.blob.digest, props.blob.size] as const,
  () => void loadPreview(),
  { immediate: true },
)

onBeforeUnmount(() => {
  requestGeneration += 1
  revokeObjectURL()
})

async function loadPreview(): Promise<void> {
  const generation = ++requestGeneration
  revokeObjectURL()
  setState('loading')
  try {
    const preview = await backend.assets.previewBlob(props.blob)
    if (generation !== requestGeneration) return
    if (!preview?.base64 || preview.mediaType !== 'image/png') {
      setState('unavailable')
      return
    }
    const binary = atob(preview.base64)
    const bytes = new Uint8Array(binary.length)
    for (let index = 0; index < binary.length; index += 1) bytes[index] = binary.charCodeAt(index)
    objectURL.value = URL.createObjectURL(new Blob([bytes], { type: preview.mediaType }))
    setState('ready')
  } catch {
    if (generation === requestGeneration) setState('unavailable')
  }
}

function revokeObjectURL(): void {
  if (!objectURL.value) return
  URL.revokeObjectURL(objectURL.value)
  objectURL.value = ''
}

function setState(value: 'loading' | 'ready' | 'unavailable'): void {
  state.value = value
  emit('state', value)
}
</script>
