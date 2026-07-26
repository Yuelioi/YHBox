<template>
  <UButton
    v-if="!bound"
    class="h-auto w-full justify-start rounded-lg border border-dashed border-default px-3 py-3 text-left"
    color="neutral"
    variant="ghost"
    :icon="
      kind === 'template'
        ? 'i-tabler-photo-search'
        : kind === 'macro'
          ? 'i-tabler-list-details'
          : 'i-tabler-route-alt-left'
    "
    trailing-icon="i-tabler-chevron-right"
    @click="emit('change')"
  >
    <span class="min-w-0">
      <span class="block text-xs font-medium text-toned">{{ placeholder }}</span>
      <span class="mt-0.5 block text-[10px] text-dimmed">{{ t('assetPicker.open_library') }}</span>
    </span>
  </UButton>

  <div
    v-else
    class="flex min-w-0 items-center gap-2 rounded-lg border border-default bg-default px-2.5 py-2"
    :class="unavailable ? 'border-error/40' : ''"
  >
    <BlobPreview
      v-if="kind === 'template' && blob"
      :blob="blob"
      :alt="label"
      expandable
      class="size-10 shrink-0"
      @state="previewState = $event"
    />
    <div
      v-else
      class="flex size-9 shrink-0 items-center justify-center rounded-md bg-elevated text-primary"
    >
      <UIcon
        :name="
          kind === 'template'
            ? 'i-tabler-photo'
            : kind === 'macro'
              ? 'i-tabler-list-details'
              : 'i-tabler-route-alt-left'
        "
        class="size-4"
      />
    </div>

    <button
      type="button"
      class="group min-w-0 flex-1 rounded text-left outline-none"
      :class="
        locatable
          ? 'cursor-pointer hover:text-primary focus-visible:ring-2 focus-visible:ring-primary'
          : 'cursor-default'
      "
      :disabled="!locatable"
      :title="locatable ? t('workflow.inspector.locate_resource') : undefined"
      @click="locatable && emit('locate')"
    >
      <span class="flex min-w-0 items-center gap-1">
        <span class="min-w-0 flex-1 truncate text-xs font-medium text-toned">{{ label }}</span>
        <UIcon
          v-if="locatable"
          name="i-tabler-current-location"
          class="size-3.5 shrink-0 text-dimmed transition-colors group-hover:text-primary"
          aria-hidden="true"
        />
      </span>
      <span class="mt-0.5 block truncate text-[10px] text-dimmed">{{ metadata }}</span>
    </button>

    <UBadge v-if="unavailable" color="error" variant="soft" size="sm">
      {{ t('workflow.inspector.resource_stale') }}
    </UBadge>
    <UButton
      v-if="clearable"
      color="neutral"
      variant="ghost"
      size="xs"
      icon="i-tabler-edit"
      :label="t('assetPicker.change')"
      @click="emit('change')"
    />
    <UButton
      color="neutral"
      variant="ghost"
      size="xs"
      icon="i-tabler-x"
      :aria-label="t('workflow.inspector.clear')"
      @click="emit('clear')"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BlobRef } from '@/lib/backend'
import BlobPreview from '@/components/common/BlobPreview.vue'

const props = defineProps<{
  kind: 'template' | 'macro' | 'clip'
  bound: boolean
  blob?: BlobRef
  label: string
  placeholder: string
  stale?: boolean
  clearable?: boolean
  locatable?: boolean
  identity?: string
}>()
const emit = defineEmits<{
  change: []
  clear: []
  locate: []
}>()
const { t } = useI18n()
const previewState = ref<'loading' | 'ready' | 'unavailable'>('loading')
const unavailable = computed(() =>
  Boolean(props.stale || (props.kind === 'template' && previewState.value === 'unavailable')),
)
const metadata = computed(() => {
  if (!props.blob) return t('workflow.inspector.resource_missing')
  if (props.kind === 'template') {
    const digest = props.blob.digest.slice(0, 18)
    return props.identity ? `${props.identity} · ${digest}` : digest
  }
  if (props.kind === 'macro') return t('assetPicker.macro_size', { size: props.blob.size })
  return t('assetPicker.clip_size', { size: props.blob.size })
})
</script>
