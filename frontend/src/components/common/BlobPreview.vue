<template>
  <div
    class="relative flex min-h-10 min-w-10 items-center justify-center overflow-hidden rounded-lg border border-default bg-sunken"
    :class="
      expandable && state === 'ready'
        ? 'cursor-zoom-in outline-none transition-colors hover:border-primary/60 focus-visible:ring-2 focus-visible:ring-primary'
        : ''
    "
    :data-preview-state="state"
    :role="expandable && state === 'ready' ? 'button' : undefined"
    :tabindex="expandable && state === 'ready' ? 0 : undefined"
    :aria-label="
      expandable && state === 'ready' ? t('assets.open_preview', { name: alt }) : undefined
    "
    @click.stop="openViewer"
    @keydown.enter.stop.prevent="openViewer"
    @keydown.space.stop.prevent="openViewer"
  >
    <CappedPreviewImage v-if="objectURL" :src="objectURL" :alt="alt" />
    <USkeleton v-else-if="state === 'loading'" class="absolute inset-0 rounded-none" />
    <div v-else class="flex flex-col items-center gap-1 px-2 text-center text-dimmed">
      <UIcon name="i-tabler-photo-off" class="size-4" />
      <span v-if="showFailureLabel" class="text-[10px]">{{ t('assets.preview_unavailable') }}</span>
    </div>

    <BaseModal
      :open="viewerOpen"
      :title="alt || t('assets.preview_title')"
      icon="i-tabler-photo"
      size="7xl"
      tall
      content-class="w-[94vw]"
      @update:open="viewerOpen = $event"
    >
      <template #header-extra>
        <div class="flex items-center gap-1" role="toolbar" :aria-label="t('assets.preview_tools')">
          <UButton
            icon="i-tabler-arrows-maximize"
            color="neutral"
            :variant="fitMode ? 'soft' : 'ghost'"
            size="xs"
            :aria-label="t('assets.preview_fit')"
            :title="t('assets.preview_fit')"
            @click="fitMode = true"
          />
          <UButton
            icon="i-tabler-aspect-ratio"
            color="neutral"
            :variant="!fitMode && zoom === 1 ? 'soft' : 'ghost'"
            size="xs"
            :aria-label="t('assets.preview_actual_size')"
            :title="t('assets.preview_actual_size')"
            @click="setActualSize"
          />
          <UButton
            icon="i-tabler-minus"
            color="neutral"
            variant="ghost"
            size="xs"
            :disabled="zoom <= minZoom"
            :aria-label="t('assets.preview_zoom_out')"
            @click="zoomBy(-zoomStep)"
          />
          <span class="w-12 text-center font-mono text-[10px] text-muted">
            {{ fitMode ? t('assets.preview_fit_short') : `${Math.round(zoom * 100)}%` }}
          </span>
          <UButton
            icon="i-tabler-plus"
            color="neutral"
            variant="ghost"
            size="xs"
            :disabled="zoom >= maxZoom"
            :aria-label="t('assets.preview_zoom_in')"
            @click="zoomBy(zoomStep)"
          />
        </div>
      </template>

      <div
        class="flex min-h-[24rem] items-center justify-center overflow-auto rounded-lg border border-default bg-sunken p-4"
        :class="fitMode ? 'h-[calc(92vh-8.5rem)]' : 'max-h-[calc(92vh-8.5rem)]'"
      >
        <img
          v-if="objectURL"
          :src="objectURL"
          :alt="alt"
          class="block shrink-0"
          :class="fitMode ? 'h-full w-full object-contain' : 'max-w-none object-contain'"
          :style="viewerImageStyle"
          @load="measureViewerImage"
        />
      </div>
    </BaseModal>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend, type BlobRef } from '@/lib/backend'
import BaseModal from '@/components/common/BaseModal.vue'
import CappedPreviewImage from '@/components/common/CappedPreviewImage.vue'

const props = withDefaults(
  defineProps<{
    blob: BlobRef
    alt?: string
    showFailureLabel?: boolean
    expandable?: boolean
  }>(),
  { alt: '', showFailureLabel: false, expandable: false },
)
const emit = defineEmits<{ state: [state: 'loading' | 'ready' | 'unavailable'] }>()
const { t } = useI18n()
const state = ref<'loading' | 'ready' | 'unavailable'>('loading')
const objectURL = ref('')
const viewerOpen = ref(false)
const fitMode = ref(true)
const zoom = ref(1)
const naturalSize = ref({ width: 0, height: 0 })
const minZoom = 0.25
const maxZoom = 4
const zoomStep = 0.25
let requestGeneration = 0
const viewerImageStyle = computed(() =>
  fitMode.value || !naturalSize.value.width || !naturalSize.value.height
    ? undefined
    : {
        width: `${naturalSize.value.width * zoom.value}px`,
        height: `${naturalSize.value.height * zoom.value}px`,
      },
)

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

function openViewer(): void {
  if (!props.expandable || state.value !== 'ready') return
  fitMode.value = true
  zoom.value = 1
  viewerOpen.value = true
}

function setActualSize(): void {
  fitMode.value = false
  zoom.value = 1
}

function zoomBy(delta: number): void {
  fitMode.value = false
  zoom.value = Math.min(maxZoom, Math.max(minZoom, zoom.value + delta))
}

function measureViewerImage(event: Event): void {
  const image = event.currentTarget as HTMLImageElement
  naturalSize.value = { width: image.naturalWidth, height: image.naturalHeight }
}

function setState(value: 'loading' | 'ready' | 'unavailable'): void {
  state.value = value
  emit('state', value)
}
</script>
