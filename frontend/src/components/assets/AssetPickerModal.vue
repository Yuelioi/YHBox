<template>
  <BaseModal
    :open="open"
    :title="t(kind === 'template' ? 'assetPicker.template_title' : 'assetPicker.clip_title')"
    :icon="kind === 'template' ? 'i-tabler-photo-search' : 'i-tabler-movie'"
    size="5xl"
    tall
    @update:open="emit('update:open', $event)"
  >
    <div class="flex h-full min-h-0 flex-col gap-3">
      <div class="flex shrink-0 items-center gap-2">
        <UInput
          v-model="searchInput"
          icon="i-tabler-search"
          autofocus
          class="min-w-0 flex-1"
          :placeholder="t('assetPicker.search_placeholder')"
          :aria-label="t('assetPicker.search_placeholder')"
        />
        <UButton
          color="neutral"
          :variant="filtersOpen || hasFilters ? 'soft' : 'ghost'"
          icon="i-tabler-filter"
          :label="t('assetPicker.filters')"
          @click="filtersOpen = !filtersOpen"
        />
        <AdaptiveSelect
          v-model="sort"
          :items="sortItems"
          value-key="value"
          label-key="label"
          class="shrink-0"
          @update:model-value="applyFilters"
        />
      </div>

      <div
        v-if="filtersOpen"
        class="grid shrink-0 grid-cols-1 gap-2 rounded-lg border border-default bg-elevated/30 p-3 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]"
      >
        <UInput
          v-model="category"
          :placeholder="t('assetPicker.category_placeholder')"
          @keyup.enter="applyFilters"
        />
        <UInput
          v-model="tagsInput"
          :placeholder="t('assetPicker.tags_placeholder')"
          @keyup.enter="applyFilters"
        />
        <UButton
          color="neutral"
          variant="soft"
          :label="t('assets.search_action')"
          @click="applyFilters"
        />
      </div>

      <div class="flex shrink-0 items-center gap-2 text-xs text-dimmed">
        <span>{{ t('assetPicker.result_count', { count: total }) }}</span>
        <span aria-hidden="true">·</span>
        <span>{{ t('assetPicker.selection_instruction') }}</span>
      </div>

      <div
        v-if="failure"
        class="shrink-0 rounded-lg border border-error/30 bg-error/10 px-3 py-2 text-xs text-error"
        role="alert"
      >
        {{ failure }}
      </div>

      <div class="min-h-0 flex-1 overflow-y-auto pr-1">
        <div
          v-if="loading"
          class="grid grid-cols-1 gap-2"
          :class="kind === 'template' ? 'sm:grid-cols-2 xl:grid-cols-3' : ''"
        >
          <USkeleton
            v-for="index in 12"
            :key="index"
            :class="kind === 'template' ? 'h-32 rounded-lg' : 'h-16 rounded-md'"
          />
        </div>
        <div
          v-else-if="items.length"
          role="listbox"
          :aria-label="
            t(kind === 'template' ? 'assetPicker.template_title' : 'assetPicker.clip_title')
          "
          class="grid grid-cols-1 gap-2"
          :class="kind === 'template' ? 'sm:grid-cols-2 xl:grid-cols-3' : ''"
        >
          <article
            v-for="asset in items"
            :key="asset.guid"
            role="option"
            tabindex="0"
            :aria-selected="candidateContainsSelection(asset)"
            class="group min-w-0 cursor-pointer rounded-lg border px-3 py-2.5 outline-none transition-colors focus-visible:ring-2 focus-visible:ring-primary"
            :class="
              candidateContainsSelection(asset)
                ? 'border-primary bg-primary/10'
                : assetContainsSelection(asset)
                  ? 'border-primary/50 bg-primary/5'
                  : 'border-default bg-default hover:border-accented hover:bg-elevated/35'
            "
            @click="activateAsset(asset)"
            @dblclick="activateAsset(asset, true)"
            @keydown.enter.prevent="activateAsset(asset)"
            @keydown.space.prevent="activateAsset(asset)"
          >
            <div class="flex min-w-0 items-center gap-3">
              <BlobPreview
                v-if="asset.thumbnail"
                :blob="asset.thumbnail"
                :alt="asset.name"
                class="size-14 shrink-0"
              />
              <div
                v-else
                class="flex size-10 shrink-0 items-center justify-center rounded-md bg-elevated text-primary"
              >
                <UIcon
                  :name="asset.kind === 'template' ? 'i-tabler-photo' : 'i-tabler-movie'"
                  class="size-5"
                />
              </div>

              <div class="min-w-0 flex-1">
                <div class="flex min-w-0 items-center gap-2">
                  <h3 class="min-w-0 flex-1 truncate text-sm font-medium text-highlighted">
                    {{ asset.name }}
                  </h3>
                  <UIcon
                    v-if="candidateContainsSelection(asset)"
                    name="i-tabler-circle-check-filled"
                    class="size-4 shrink-0 text-primary"
                  />
                </div>
                <p class="mt-0.5 truncate text-[11px] text-dimmed">
                  {{ assetMeta(asset) }}
                </p>
                <p v-if="asset.description" class="mt-1 truncate text-xs text-muted">
                  {{ asset.description }}
                </p>
              </div>
            </div>

            <div class="mt-2 flex min-h-5 items-center gap-1.5">
              <template v-if="asset.kind === 'template'">
                <UButton
                  v-for="variant in asset.variants"
                  :key="`${variant.resolution[0]}x${variant.resolution[1]}:${variant.blob.digest}`"
                  size="xs"
                  color="neutral"
                  :variant="sameBlob(variant.blob, candidate?.blob) ? 'solid' : 'soft'"
                  @click.stop="selectCandidate(asset, variant.blob, variant.resolution)"
                >
                  {{ variant.resolution[0] }}×{{ variant.resolution[1] }}
                </UButton>
              </template>
              <UBadge v-if="asset.category" color="neutral" variant="soft" size="sm">
                {{ asset.category }}
              </UBadge>
              <span v-if="asset.tags?.length" class="truncate text-[10px] text-dimmed">
                {{ asset.tags.slice(0, 3).join(' · ') }}
              </span>
              <UBadge
                v-if="assetContainsSelection(asset) && !candidateContainsSelection(asset)"
                class="ml-auto"
                color="neutral"
                variant="soft"
                size="sm"
              >
                {{ t('assetPicker.current') }}
              </UBadge>
            </div>
          </article>
        </div>
        <EmptyState
          v-else
          inset
          icon="i-tabler-search-off"
          :title="t('assetPicker.empty')"
          :description="t('assetPicker.empty_hint')"
        />
      </div>

      <div class="flex shrink-0 items-center gap-3 border-t border-default pt-3">
        <span class="min-w-0 flex-1 truncate text-xs text-dimmed">
          {{
            candidate
              ? t('assetPicker.selected_asset', { name: candidate.name })
              : total > 0
                ? t('assets.page_summary', { page, pages: pageCount, total })
                : t('assetPicker.select_hint')
          }}
        </span>
        <div v-if="total > 0" class="flex gap-2">
          <UButton
            color="neutral"
            variant="soft"
            size="sm"
            icon="i-tabler-chevron-left"
            :disabled="page <= 1 || loading"
            @click="goToPage(page - 1)"
          />
          <UButton
            color="neutral"
            variant="soft"
            size="sm"
            icon="i-tabler-chevron-right"
            :disabled="page >= pageCount || loading"
            @click="goToPage(page + 1)"
          />
        </div>
        <UButton icon="i-tabler-check" :disabled="!candidate" @click="confirmSelection()">
          {{ t(kind === 'template' ? 'assetPicker.use_template' : 'assetPicker.use_clip') }}
        </UButton>
      </div>
    </div>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { errorMessage } from '@/lib/invoke'
import type { AssetSummary, BlobRef } from '@/lib/backend'
import { useAssetsStore, type AssetPickerSelection } from '@/stores/assets'
import BaseModal from '@/components/common/BaseModal.vue'
import BlobPreview from '@/components/common/BlobPreview.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'

const props = defineProps<{
  open: boolean
  kind: 'template' | 'clip'
  selectedBlob?: BlobRef
}>()
const emit = defineEmits<{
  'update:open': [value: boolean]
  select: [selection: AssetPickerSelection]
}>()
const { t } = useI18n()
const assets = useAssetsStore()
const searchInput = ref('')
const search = ref('')
const category = ref('')
const tagsInput = ref('')
const sort = ref('recent_desc')
const filtersOpen = ref(false)
const page = ref(1)
const pageSize = 40
const total = ref(0)
const items = ref<AssetSummary[]>([])
const loading = ref(false)
const failure = ref('')
const candidate = ref<AssetPickerSelection | null>(null)
let searchTimer: ReturnType<typeof setTimeout> | undefined
let requestGeneration = 0

const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const hasFilters = computed(() => Boolean(category.value.trim() || tagsInput.value.trim()))
const sortItems = computed(() => [
  { label: t('assetPicker.sort_recent'), value: 'recent_desc' },
  { label: t('assets.sort_name_asc'), value: 'name_asc' },
  { label: t('assets.sort_created_desc'), value: 'created_desc' },
])

watch(
  () => [props.open, props.kind] as const,
  ([open]) => {
    if (!open) return
    candidate.value = null
    page.value = 1
    void loadPage()
  },
  { immediate: true },
)

watch(searchInput, (value) => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    search.value = value.trim()
    page.value = 1
    void loadPage()
  }, 250)
})

watch(
  () => assets.epoch,
  () => {
    if (props.open) void loadPage(true)
  },
)

onBeforeUnmount(() => {
  requestGeneration += 1
  if (searchTimer) clearTimeout(searchTimer)
})

async function loadPage(force = false): Promise<void> {
  if (!props.open) return
  const generation = ++requestGeneration
  loading.value = true
  failure.value = ''
  try {
    const result = await assets.query(
      {
        search: search.value,
        kind: props.kind,
        category: category.value,
        tags: splitTags(tagsInput.value),
        sort: sort.value,
        page: page.value,
        pageSize,
        thumbnailBudget: 12,
        recentGUIDs: assets.recentGUIDs,
      },
      { force },
    )
    if (generation !== requestGeneration) return
    items.value = result.items
    total.value = result.total
    if (!candidate.value && props.selectedBlob) {
      const current = items.value.find((asset) => assetContainsSelection(asset))
      if (current) candidate.value = selectionForAsset(current)
    }
    if (page.value > pageCount.value) {
      page.value = pageCount.value
      await loadPage(force)
    }
  } catch (error) {
    if (generation === requestGeneration) failure.value = errorMessage(error)
  } finally {
    if (generation === requestGeneration) loading.value = false
  }
}

function applyFilters(): void {
  search.value = searchInput.value.trim()
  page.value = 1
  void loadPage()
}

function goToPage(next: number): void {
  if (next < 1 || next > pageCount.value || next === page.value) return
  page.value = next
  void loadPage()
}

function selectCandidate(asset: AssetSummary, blob: BlobRef, resolution?: [number, number]): void {
  candidate.value = {
    guid: asset.guid,
    kind: asset.kind,
    name: asset.name,
    resolution: resolution ? [resolution[0], resolution[1]] : undefined,
    blob: { ...blob },
  }
}

function activateAsset(asset: AssetSummary, confirm = false): void {
  const selection = selectionForAsset(asset)
  if (!selection) return
  candidate.value = selection
  if (confirm) confirmSelection(selection)
}

function selectionForAsset(asset: AssetSummary): AssetPickerSelection | null {
  if (asset.kind === 'clip') {
    return asset.blob
      ? {
          guid: asset.guid,
          kind: asset.kind,
          name: asset.name,
          blob: { ...asset.blob },
        }
      : null
  }
  const variant =
    asset.variants.find((item) => sameBlob(item.blob, props.selectedBlob)) ?? asset.variants[0]
  if (!variant) return null
  return {
    guid: asset.guid,
    kind: asset.kind,
    name: asset.name,
    resolution: [variant.resolution[0], variant.resolution[1]],
    blob: { ...variant.blob },
  }
}

function confirmSelection(selection = candidate.value): void {
  if (!selection) return
  candidate.value = null
  assets.markUsed(selection.guid)
  emit('select', selection)
  emit('update:open', false)
}

function assetMeta(asset: AssetSummary): string {
  if (asset.kind === 'template') {
    return t('assets.templates.meta', { count: asset.variantCount })
  }
  return t('assets.clips.library_meta', { bytes: asset.blob?.size ?? 0 })
}

function candidateContainsSelection(asset: AssetSummary): boolean {
  return candidate.value?.guid === asset.guid
}

function assetContainsSelection(asset: AssetSummary): boolean {
  if (!props.selectedBlob) return false
  if (asset.blob && sameBlob(asset.blob, props.selectedBlob)) return true
  return asset.variants.some((variant) => sameBlob(variant.blob, props.selectedBlob))
}

function sameBlob(left: BlobRef | undefined, right: BlobRef | undefined): boolean {
  return Boolean(
    left &&
    right &&
    left.digest === right.digest &&
    left.mediaType === right.mediaType &&
    left.size === right.size,
  )
}

function splitTags(value: string): string[] {
  return value
    .split(/[,，]/)
    .map((tag) => tag.trim())
    .filter(Boolean)
}
</script>
