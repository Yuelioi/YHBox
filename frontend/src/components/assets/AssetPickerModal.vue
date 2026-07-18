<template>
  <BaseModal
    :open="open"
    :title="t(kind === 'template' ? 'assetPicker.template_title' : 'assetPicker.clip_title')"
    :icon="kind === 'template' ? 'i-tabler-photo-search' : 'i-tabler-movie'"
    size="5xl"
    tall
    @update:open="emit('update:open', $event)"
  >
    <div class="flex h-full min-h-0 flex-col gap-4">
      <div class="grid shrink-0 grid-cols-1 gap-2 md:grid-cols-[minmax(0,1fr)_11rem_11rem]">
        <UInput
          v-model="searchInput"
          icon="i-tabler-search"
          autofocus
          :placeholder="t('assetPicker.search_placeholder')"
          :aria-label="t('assetPicker.search_placeholder')"
        />
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
      </div>

      <div class="flex shrink-0 items-center gap-2">
        <USelect
          v-model="sort"
          :items="sortItems"
          value-key="value"
          label-key="label"
          class="w-48"
          @update:model-value="applyFilters"
        />
        <span class="ml-auto text-xs text-dimmed">
          {{ t('assetPicker.result_count', { count: total }) }}
        </span>
      </div>

      <div
        v-if="failure"
        class="shrink-0 rounded-lg border border-error/30 bg-error/10 px-3 py-2 text-xs text-error"
        role="alert"
      >
        {{ failure }}
      </div>

      <div class="min-h-0 flex-1 overflow-y-auto pr-1">
        <div v-if="loading" class="grid grid-cols-1 gap-3 lg:grid-cols-2">
          <USkeleton v-for="index in 8" :key="index" class="h-36 rounded-xl" />
        </div>
        <div v-else-if="items.length" class="grid grid-cols-1 gap-3 lg:grid-cols-2">
          <article
            v-for="asset in items"
            :key="asset.guid"
            class="flex min-w-0 gap-3 rounded-xl border bg-elevated/25 p-3"
            :class="assetContainsSelection(asset) ? 'border-primary/50' : 'border-default'"
          >
            <BlobPreview
              v-if="asset.thumbnail"
              :blob="asset.thumbnail"
              :alt="asset.name"
              class="size-20 shrink-0"
            />
            <div
              v-else
              class="flex size-14 shrink-0 items-center justify-center rounded-lg bg-elevated text-primary"
            >
              <UIcon
                :name="asset.kind === 'template' ? 'i-tabler-photo' : 'i-tabler-movie'"
                class="size-5"
              />
            </div>

            <div class="min-w-0 flex-1">
              <div class="flex items-start gap-2">
                <div class="min-w-0 flex-1">
                  <h3 class="truncate text-sm font-medium text-highlighted">{{ asset.name }}</h3>
                  <p
                    v-if="asset.description"
                    class="mt-1 line-clamp-2 text-xs leading-5 text-muted"
                  >
                    {{ asset.description }}
                  </p>
                </div>
                <UBadge
                  v-if="assetContainsSelection(asset)"
                  color="primary"
                  variant="soft"
                  size="sm"
                >
                  {{ t('assetPicker.current') }}
                </UBadge>
              </div>

              <div v-if="asset.category || asset.tags?.length" class="mt-2 flex flex-wrap gap-1">
                <UBadge v-if="asset.category" color="neutral" variant="soft" size="sm">
                  {{ asset.category }}
                </UBadge>
                <UBadge
                  v-for="tag in asset.tags ?? []"
                  :key="tag"
                  color="primary"
                  variant="subtle"
                  size="sm"
                >
                  {{ tag }}
                </UBadge>
              </div>

              <div v-if="asset.kind === 'template'" class="mt-3 flex flex-wrap gap-1.5">
                <UButton
                  v-for="variant in asset.variants"
                  :key="`${variant.resolution[0]}x${variant.resolution[1]}:${variant.blob.digest}`"
                  size="xs"
                  color="neutral"
                  :variant="sameBlob(variant.blob, selectedBlob) ? 'solid' : 'soft'"
                  @click="choose(asset, variant.blob, variant.resolution)"
                >
                  {{ variant.resolution[0] }}×{{ variant.resolution[1] }}
                </UButton>
              </div>
              <UButton
                v-else-if="asset.blob"
                class="mt-3"
                size="xs"
                :label="t('assetPicker.select_clip')"
                icon="i-tabler-check"
                @click="choose(asset, asset.blob)"
              />
            </div>
          </article>
        </div>
        <EmptyState
          v-else
          icon="i-tabler-search-off"
          :title="t('assetPicker.empty')"
          :description="t('assetPicker.empty_hint')"
        />
      </div>

      <div v-if="total > 0" class="flex shrink-0 items-center border-t border-default pt-3">
        <span class="mr-auto text-xs text-dimmed">
          {{ t('assets.page_summary', { page, pages: pageCount, total }) }}
        </span>
        <div class="flex gap-2">
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
const page = ref(1)
const pageSize = 20
const total = ref(0)
const items = ref<AssetSummary[]>([])
const loading = ref(false)
const failure = ref('')
let searchTimer: ReturnType<typeof setTimeout> | undefined
let requestGeneration = 0

const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const sortItems = computed(() => [
  { label: t('assetPicker.sort_recent'), value: 'recent_desc' },
  { label: t('assets.sort_name_asc'), value: 'name_asc' },
  { label: t('assets.sort_created_desc'), value: 'created_desc' },
])

watch(
  () => [props.open, props.kind] as const,
  ([open]) => {
    if (!open) return
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

function choose(asset: AssetSummary, blob: BlobRef, resolution?: [number, number]): void {
  assets.markUsed(asset.guid)
  emit('select', {
    guid: asset.guid,
    kind: asset.kind,
    name: asset.name,
    resolution: resolution ? [resolution[0], resolution[1]] : undefined,
    blob: { ...blob },
  })
  emit('update:open', false)
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
