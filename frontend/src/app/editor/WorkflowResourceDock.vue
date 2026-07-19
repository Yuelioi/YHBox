<template>
  <section class="flex h-full min-h-0 flex-col bg-default" data-testid="workflow-resource-dock">
    <header class="shrink-0 border-b border-default px-3 py-3">
      <div class="flex items-start gap-2">
        <div class="min-w-0 flex-1">
          <h2 class="text-xs font-semibold text-highlighted">
            {{ t('workflow.resources.title') }}
          </h2>
          <p class="mt-1 text-[10px] leading-4 text-muted">{{ t('workflow.resources.hint') }}</p>
        </div>
        <UButton
          icon="i-tabler-external-link"
          color="neutral"
          variant="ghost"
          size="xs"
          :aria-label="t('workflow.resources.open_library')"
          @click="emit('open-library')"
        />
      </div>

      <div class="mt-3 grid grid-cols-3 gap-1 rounded-lg bg-elevated/50 p-1">
        <UButton
          v-for="item in tabs"
          :key="item.value"
          :data-testid="`workflow-resource-tab-${item.value}`"
          :icon="item.icon"
          :label="item.label"
          color="neutral"
          :variant="kind === item.value ? 'soft' : 'ghost'"
          size="xs"
          class="min-w-0 justify-center"
          :aria-pressed="kind === item.value"
          @click="kind = item.value"
        />
      </div>

      <div class="mt-2 flex items-center gap-2">
        <UButton
          v-if="kind !== 'template'"
          data-testid="workflow-resource-create"
          :icon="kind === 'macro' ? 'i-tabler-list-details' : 'i-tabler-route-alt-left'"
          :label="
            kind === 'macro'
              ? t('assets.recording.record_macro')
              : t('assets.recording.record_precise')
          "
          size="xs"
          class="min-w-0 flex-1 justify-center"
          :disabled="recordingPhase !== 'idle'"
          @click="emit('start-recording', kind === 'macro' ? 'simple' : 'precise')"
        />
        <UButton
          v-else
          data-testid="workflow-resource-create"
          icon="i-tabler-camera-plus"
          :label="t('assets.templates.capture')"
          size="xs"
          class="min-w-0 flex-1 justify-center"
          @click="emit('capture-template')"
        />
        <UButton
          icon="i-tabler-refresh"
          color="neutral"
          variant="ghost"
          size="xs"
          :loading="loading"
          :aria-label="t('common.refresh')"
          @click="load(true)"
        />
      </div>

      <UInput
        v-model="searchInput"
        icon="i-tabler-search"
        size="sm"
        class="mt-2 w-full"
        :placeholder="t('assets.search_placeholder')"
      />
      <div class="mt-2 grid grid-cols-2 gap-2">
        <AdaptiveSelect
          v-model="category"
          :items="categoryItems"
          icon="i-tabler-category"
          size="sm"
          @update:model-value="resetAndLoad"
        />
        <AdaptiveSelect
          v-model="sort"
          :items="sortItems"
          icon="i-tabler-arrows-sort"
          size="sm"
          @update:model-value="resetAndLoad"
        />
      </div>
    </header>

    <div class="min-h-0 flex-1 overflow-y-auto p-2">
      <div v-if="loading" class="space-y-2">
        <USkeleton v-for="index in 8" :key="index" class="h-16 rounded-lg" />
      </div>
      <div v-else-if="items.length" class="space-y-1.5">
        <article
          v-for="asset in items"
          :key="asset.guid"
          class="group flex cursor-pointer items-center gap-2 rounded-lg border border-default bg-elevated/20 p-2 transition-colors hover:border-primary/40 hover:bg-elevated/50"
          tabindex="0"
          @dblclick="useAsset(asset)"
          @keydown.enter.prevent="useAsset(asset)"
        >
          <BlobPreview
            v-if="asset.thumbnail"
            :blob="asset.thumbnail"
            :alt="asset.name"
            class="size-10 shrink-0"
          />
          <span
            v-else
            class="flex size-9 shrink-0 items-center justify-center rounded-md bg-elevated text-primary"
          >
            <UIcon :name="assetIcon(asset)" class="size-4" />
          </span>
          <span class="min-w-0 flex-1">
            <span class="block truncate text-xs font-medium text-highlighted">{{
              asset.name
            }}</span>
            <span class="mt-0.5 block truncate text-[10px] text-dimmed">
              {{ asset.description || assetMeta(asset) }}
            </span>
            <span
              v-if="asset.category || asset.tags?.length"
              class="mt-1 flex gap-1 overflow-hidden"
            >
              <UBadge v-if="asset.category" color="neutral" variant="soft" size="xs">
                {{ asset.category }}
              </UBadge>
              <span class="truncate text-[9px] text-dimmed">{{
                asset.tags?.slice(0, 2).join(' · ')
              }}</span>
            </span>
          </span>
          <UButton
            v-if="asset.kind === 'macro'"
            icon="i-tabler-pencil"
            color="neutral"
            variant="ghost"
            size="xs"
            :aria-label="t('workflow.resources.edit', { name: asset.name })"
            @click.stop="emit('edit', asset)"
          />
          <UButton
            icon="i-tabler-plus"
            color="neutral"
            variant="ghost"
            size="xs"
            :aria-label="t('workflow.resources.use', { name: asset.name })"
            @click.stop="useAsset(asset)"
          />
        </article>
      </div>
      <EmptyState
        v-else
        inset
        :icon="
          kind === 'macro'
            ? 'i-tabler-list-details'
            : kind === 'clip'
              ? 'i-tabler-route-alt-left'
              : 'i-tabler-photo-off'
        "
        :title="t('workflow.resources.empty')"
        :description="t('workflow.resources.empty_hint')"
      />
    </div>

    <footer class="flex h-10 shrink-0 items-center gap-2 border-t border-default px-3">
      <span class="min-w-0 flex-1 truncate text-[10px] text-dimmed">
        {{ t('assets.page_summary', { page, pages: pageCount, total }) }}
      </span>
      <UButton
        icon="i-tabler-chevron-left"
        color="neutral"
        variant="ghost"
        size="xs"
        :disabled="page <= 1 || loading"
        @click="goToPage(page - 1)"
      />
      <UButton
        icon="i-tabler-chevron-right"
        color="neutral"
        variant="ghost"
        size="xs"
        :disabled="page >= pageCount || loading"
        @click="goToPage(page + 1)"
      />
    </footer>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'
import BlobPreview from '@/components/common/BlobPreview.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import type { AssetSummary } from '@/lib/backend'
import { useAssetsStore, type AssetPickerSelection } from '@/stores/assets'

type ResourceKind = 'macro' | 'clip' | 'template'

defineProps<{
  recordingPhase: 'idle' | 'recording' | 'paused' | 'finalizing' | 'pending'
}>()
const kind = defineModel<ResourceKind>('kind', { default: 'macro' })
const emit = defineEmits<{
  'start-recording': [mode: 'simple' | 'precise']
  'capture-template': []
  'open-library': []
  edit: [asset: AssetSummary]
  use: [selection: AssetPickerSelection]
}>()
const { t } = useI18n()
const assets = useAssetsStore()
const allCategoriesValue = '__yotta_all_categories__'
const searchInput = ref('')
const search = ref('')
const category = ref(allCategoriesValue)
const sort = ref('recent_desc')
const page = ref(1)
const pageSize = 20
const total = ref(0)
const items = ref<AssetSummary[]>([])
const categories = ref<Array<{ value: string; count: number }>>([])
const loading = ref(false)
let requestGeneration = 0
let searchTimer: ReturnType<typeof setTimeout> | undefined

const tabs = computed(() => [
  { value: 'macro' as const, label: t('assets.tabs.macros'), icon: 'i-tabler-list-details' },
  { value: 'clip' as const, label: t('assets.tabs.clips'), icon: 'i-tabler-route-alt-left' },
  { value: 'template' as const, label: t('assets.tabs.templates'), icon: 'i-tabler-photo' },
])
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const categoryItems = computed(() => [
  { label: t('assets.all_categories'), value: allCategoriesValue },
  ...categories.value.map((item) => ({
    label: `${item.value} (${item.count})`,
    value: item.value,
  })),
])
const sortItems = computed(() => [
  { label: t('assetPicker.sort_recent'), value: 'recent_desc' },
  { label: t('assets.sort_name_asc'), value: 'name_asc' },
  { label: t('assets.sort_created_desc'), value: 'created_desc' },
])

onMounted(() => void load())
onBeforeUnmount(() => {
  requestGeneration += 1
  if (searchTimer) clearTimeout(searchTimer)
})

watch(kind, () => {
  category.value = allCategoriesValue
  page.value = 1
  void load()
})
watch(searchInput, (value) => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    search.value = value.trim()
    page.value = 1
    void load()
  }, 250)
})
watch(
  () => assets.epoch,
  () => void load(true),
)

async function load(force = false): Promise<void> {
  const generation = ++requestGeneration
  loading.value = true
  try {
    const result = await assets.query(
      {
        search: search.value,
        kind: kind.value,
        category: category.value === allCategoriesValue ? '' : category.value,
        tags: [],
        sort: sort.value,
        page: page.value,
        pageSize,
        thumbnailBudget: kind.value === 'template' ? 12 : 0,
        recentGUIDs: assets.recentGUIDs,
      },
      { force },
    )
    if (generation !== requestGeneration) return
    items.value = result.items
    total.value = result.total
    categories.value = result.categories
    if (page.value > pageCount.value) {
      page.value = pageCount.value
      await load(force)
    }
  } finally {
    if (generation === requestGeneration) loading.value = false
  }
}

function resetAndLoad(): void {
  page.value = 1
  void load()
}

function goToPage(next: number): void {
  if (next < 1 || next > pageCount.value || next === page.value) return
  page.value = next
  void load()
}

function useAsset(asset: AssetSummary): void {
  const blob = asset.kind === 'template' ? asset.variants[0]?.blob : asset.blob
  if (!blob) return
  assets.markUsed(asset.guid)
  emit('use', {
    guid: asset.guid,
    kind: asset.kind,
    name: asset.name,
    resolution: asset.kind === 'template' ? asset.variants[0]?.resolution : undefined,
    blob: { ...blob },
  })
}

function assetIcon(asset: AssetSummary): string {
  if (asset.kind === 'template') return 'i-tabler-photo'
  if (asset.kind === 'clip') return 'i-tabler-route-alt-left'
  return 'i-tabler-list-details'
}

function assetMeta(asset: AssetSummary): string {
  if (asset.kind === 'template') return t('assets.templates.meta', { count: asset.variantCount })
  if (asset.kind === 'clip') return t('assetPicker.clip_size', { size: asset.blob?.size ?? 0 })
  return t('assetPicker.macro_size', { size: asset.blob?.size ?? 0 })
}
</script>
