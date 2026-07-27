<template>
  <div
    v-if="items.length"
    ref="listRoot"
    :class="compact ? 'space-y-1.5' : selectable ? 'min-w-[1080px]' : 'min-w-[960px]'"
  >
    <div
      v-if="!compact"
      class="grid h-9 items-center gap-3 border-b border-default bg-elevated/35 px-3 text-[10px] font-semibold uppercase tracking-wide text-dimmed"
      :class="
        selectable
          ? 'grid-cols-[2.25rem_minmax(18rem,2fr)_10rem_minmax(12rem,1.2fr)_9rem_9rem_2.5rem]'
          : 'grid-cols-[minmax(18rem,2fr)_10rem_minmax(12rem,1.2fr)_9rem_9rem_2.5rem]'
      "
    >
      <slot v-if="selectable" name="select-all" />
      <span>{{ t('assets.columns.asset') }}</span>
      <span>{{ t('common.category') }}</span>
      <span>{{ t('common.tags') }}</span>
      <span>{{ t('assets.columns.details') }}</span>
      <span>{{ t('assets.columns.created') }}</span>
      <span />
    </div>

    <article
      v-for="item in items"
      :key="item.id"
      :data-asset-id="item.id"
      :data-focused="focusedId === item.id"
      :draggable="draggable"
      :class="[
        compact
          ? 'group flex cursor-pointer items-center gap-2 rounded-lg border bg-elevated/20 p-2 transition-colors hover:border-primary/40 hover:bg-elevated/50'
          : selectable
            ? 'grid min-h-16 grid-cols-[2.25rem_minmax(18rem,2fr)_10rem_minmax(12rem,1.2fr)_9rem_9rem_2.5rem] items-center gap-3 border-b px-3 hover:bg-elevated/35'
            : 'grid min-h-16 grid-cols-[minmax(18rem,2fr)_10rem_minmax(12rem,1.2fr)_9rem_9rem_2.5rem] items-center gap-3 border-b px-3 hover:bg-elevated/35',
        focusedId === item.id
          ? 'border-primary/70 bg-primary/10 ring-1 ring-inset ring-primary/55'
          : 'border-default/70',
      ]"
      tabindex="0"
      @dragstart="emit('dragstart', $event, item)"
      @dblclick="emit('use', item)"
      @keydown.enter.prevent="emit('use', item)"
    >
      <template v-if="selectable && (!compact || $slots.select)">
        <slot name="select" :item="item" />
      </template>
      <div :class="compact ? 'contents' : 'flex min-w-0 items-center gap-2.5 py-1.5'">
        <BlobPreview
          v-if="item.previewBlob"
          :blob="item.previewBlob"
          :alt="item.name"
          expandable
          class="size-10 shrink-0"
          @state="emit('preview-state', item, $event)"
        />
        <span
          v-else
          class="flex size-9 shrink-0 items-center justify-center rounded-md bg-elevated text-primary"
        >
          <UIcon :name="item.icon" class="size-4" />
        </span>
        <span class="min-w-0 flex-1">
          <span class="block truncate text-xs font-medium text-highlighted">{{ item.name }}</span>
          <span class="mt-0.5 block truncate text-[10px] text-dimmed">
            {{ item.description || item.meta }}
          </span>
          <span
            v-if="compact && (item.category || item.tags.length)"
            class="mt-1 flex gap-1 overflow-hidden"
          >
            <UBadge v-if="item.category" color="neutral" variant="soft" size="xs">
              {{ item.category }}
            </UBadge>
            <span class="truncate text-[9px] text-dimmed">{{
              item.tags.slice(0, 2).join(' · ')
            }}</span>
          </span>
        </span>
      </div>
      <template v-if="!compact">
        <div class="min-w-0">
          <UBadge v-if="item.category" color="neutral" variant="soft" size="sm">{{
            item.category
          }}</UBadge>
          <span v-else class="text-[10px] text-dimmed">{{ t('assets.unclassified') }}</span>
        </div>
        <div class="flex min-w-0 items-center gap-1 overflow-hidden">
          <UBadge
            v-for="tag in item.tags.slice(0, 3)"
            :key="tag"
            color="neutral"
            variant="subtle"
            size="sm"
            >{{ tag }}</UBadge
          >
          <span v-if="!item.tags.length" class="text-[10px] text-dimmed">{{
            t('assets.no_tags')
          }}</span>
          <span v-else-if="item.tags.length > 3" class="text-[10px] text-dimmed"
            >+{{ item.tags.length - 3 }}</span
          >
        </div>
        <span class="truncate text-[10px] text-muted">{{ item.meta }}</span>
        <span class="truncate text-[10px] text-dimmed">{{ item.createdAt }}</span>
      </template>
      <slot name="actions" :item="item">
        <UButton
          v-if="compact"
          icon="i-tabler-plus"
          color="neutral"
          variant="ghost"
          size="xs"
          :aria-label="t('workflow.resources.use', { name: item.name })"
          @click.stop="emit('use', item)"
        />
      </slot>
    </article>
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BlobPreview from '@/components/common/BlobPreview.vue'
import type { BlobRef } from '../../../../contracts/workflow/current/workflow-source'

export interface AssetLibraryListItem {
  id: string
  name: string
  description: string
  category: string
  tags: string[]
  meta: string
  icon: string
  previewBlob?: BlobRef
  createdAt?: string
}

const props = withDefaults(
  defineProps<{
    items: AssetLibraryListItem[]
    compact?: boolean
    draggable?: boolean
    focusedId?: string
    selectable?: boolean
  }>(),
  { compact: false, draggable: false, focusedId: '', selectable: true },
)
const emit = defineEmits<{
  use: [item: AssetLibraryListItem]
  dragstart: [event: DragEvent, item: AssetLibraryListItem]
  'preview-state': [item: AssetLibraryListItem, state: 'loading' | 'ready' | 'unavailable']
}>()
const { t } = useI18n()
const listRoot = ref<HTMLElement | null>(null)

watch(
  () => [props.focusedId, props.items.map((item) => item.id).join('\u0000')] as const,
  async ([focusedId]) => {
    if (!focusedId) return
    await nextTick()
    const target = [
      ...(listRoot.value?.querySelectorAll<HTMLElement>('[data-asset-id]') ?? []),
    ].find((element) => element.dataset.assetId === focusedId)
    target?.scrollIntoView({ block: 'nearest' })
    target?.focus({ preventScroll: true })
  },
  { immediate: true },
)
</script>
