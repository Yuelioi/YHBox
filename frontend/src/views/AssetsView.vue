<template>
  <div
    data-testid="assets-view"
    class="mx-auto flex h-full min-h-0 w-full max-w-7xl flex-col px-6 py-5"
  >
    <header class="flex shrink-0 items-start justify-between gap-6 border-b border-default pb-5">
      <div>
        <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-primary">
          {{ t('assets.eyebrow') }}
        </p>
        <h1 class="mt-1 text-xl font-semibold text-highlighted">{{ t('assets.title') }}</h1>
        <p class="mt-1 max-w-2xl text-sm leading-6 text-muted">{{ t('assets.description') }}</p>
      </div>
      <USelect
        v-model="selectedTargetSlot"
        :items="targetItems"
        value-key="value"
        label-key="label"
        class="w-64 shrink-0"
        :placeholder="t('assets.target_placeholder')"
        :aria-label="t('assets.target_placeholder')"
      />
    </header>

    <section class="mt-5 shrink-0 rounded-xl border border-default bg-default p-4">
      <span data-testid="assets-recording-controls" class="sr-only">{{
        t('assets.recording.title')
      }}</span>
      <div class="flex items-center gap-4">
        <div class="flex size-10 shrink-0 items-center justify-center rounded-lg bg-primary/10">
          <UIcon name="i-tabler-player-record" class="size-5 text-primary" />
        </div>
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-2">
            <h2 class="text-sm font-semibold text-highlighted">
              {{ t('assets.recording.title') }}
            </h2>
            <UBadge :color="recordingBadge.color" variant="soft" size="sm">
              {{ recordingBadge.label }}
            </UBadge>
          </div>
          <p class="mt-1 text-xs leading-5 text-muted">{{ recordingHint }}</p>
        </div>
        <div class="flex shrink-0 items-center gap-2">
          <USelect
            v-if="recording.state.phase === 'idle'"
            v-model="recordingMode"
            :items="recordingModeItems"
            value-key="value"
            label-key="label"
            class="w-44"
            :aria-label="t('assets.recording.mode')"
          />
          <UButton
            v-if="recording.state.phase === 'recording'"
            color="neutral"
            variant="soft"
            icon="i-tabler-player-pause"
            :label="t('recordingHud.pause')"
            @click="pauseRecording"
          />
          <UButton
            v-if="recording.state.phase === 'paused'"
            color="neutral"
            variant="soft"
            icon="i-tabler-player-play"
            :label="t('recordingHud.resume')"
            @click="resumeRecording"
          />
          <UButton
            v-if="recording.state.phase === 'recording' || recording.state.phase === 'paused'"
            color="error"
            variant="soft"
            icon="i-tabler-square"
            :label="t('recordingHud.stop')"
            @click="stopRecording"
          />
          <UButton
            v-else
            icon="i-tabler-player-record"
            :label="t('assets.recording.start')"
            :disabled="
              !selectedTargetSlot ||
              !selectedTargetSupportsRecording ||
              recording.state.phase !== 'idle' ||
              recordingStarting
            "
            :loading="recordingStarting"
            @click="startRecording"
          />
        </div>
      </div>
    </section>

    <div class="mt-5 flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl border border-default">
      <div
        class="flex shrink-0 flex-wrap items-center gap-2 border-b border-default bg-default px-4 py-3"
      >
        <UButton
          :label="t('assets.tabs.clips')"
          icon="i-tabler-movie"
          color="neutral"
          :variant="activeTab === 'clips' ? 'soft' : 'ghost'"
          @click="activeTab = 'clips'"
        />
        <UButton
          :label="t('assets.tabs.templates')"
          icon="i-tabler-photo"
          color="neutral"
          :variant="activeTab === 'templates' ? 'soft' : 'ghost'"
          @click="activeTab = 'templates'"
        />
        <div class="mx-2 h-6 w-px bg-default" />
        <UInput
          v-model="queryInput"
          icon="i-tabler-search"
          class="min-w-48 flex-1"
          :placeholder="t('assets.search_placeholder')"
          :aria-label="t('assets.search_placeholder')"
          @keyup.enter="applyQuery"
        />
        <UInput
          v-model="categoryFilter"
          class="w-36"
          :placeholder="t('assets.category_filter')"
          @keyup.enter="changeQuery"
        />
        <UInput
          v-model="tagsFilter"
          class="w-40"
          :placeholder="t('assets.tags_filter')"
          @keyup.enter="changeQuery"
        />
        <USelect v-model="sort" :items="sortItems" class="w-44" @update:model-value="changeQuery" />
        <UButton
          color="neutral"
          variant="soft"
          icon="i-tabler-search"
          :label="t('assets.search_action')"
          @click="applyQuery"
        />
        <UButton
          v-if="activeTab === 'templates'"
          color="primary"
          variant="soft"
          icon="i-tabler-camera-plus"
          :label="t('assets.templates.capture')"
          :disabled="!selectedTargetSlot"
          :loading="captureBusy"
          @click="captureTemplate"
        />
        <UButton
          color="neutral"
          variant="ghost"
          icon="i-tabler-recycle"
          :label="t('assets.cleanup_action')"
          :loading="cleanupBusy"
          @click="cleanupLibrary"
        />
        <UButton
          color="neutral"
          variant="ghost"
          icon="i-tabler-refresh"
          :aria-label="t('common.refresh')"
          @click="refreshAssets"
        />
      </div>

      <div
        v-if="selectedRows.length"
        class="flex shrink-0 flex-wrap items-center gap-2 border-b border-default bg-primary/5 px-4 py-3"
      >
        <span class="mr-auto text-sm font-medium text-default">
          {{ t('assets.selected_count', { n: selectedRows.length }) }}
        </span>
        <UButton size="sm" color="neutral" variant="ghost" @click="clearSelection">
          {{ t('assets.clear_selection') }}
        </UButton>
        <UButton
          size="sm"
          color="neutral"
          variant="soft"
          icon="i-tabler-tags"
          :disabled="batchBusy"
          @click="openBatchEdit"
        >
          {{ t('assets.batch_edit') }}
        </UButton>
        <UButton
          size="sm"
          color="error"
          variant="soft"
          icon="i-tabler-trash"
          :loading="batchBusy"
          @click="deleteSelected"
        >
          {{ t('assets.batch_delete') }}
        </UButton>
      </div>

      <div
        v-if="libraryFeedback"
        class="shrink-0 border-b px-4 py-2 text-sm"
        :class="
          libraryFeedback.tone === 'success'
            ? 'border-success/30 bg-success/10 text-success'
            : libraryFeedback.tone === 'warning'
              ? 'border-warning/30 bg-warning/10 text-warning'
              : 'border-error/30 bg-error/10 text-error'
        "
        role="status"
      >
        {{ libraryFeedback.message }}
      </div>

      <div class="min-h-0 flex-1 overflow-y-auto bg-elevated/15 p-4">
        <div v-if="loading" class="grid grid-cols-1 gap-3 lg:grid-cols-2 2xl:grid-cols-3">
          <USkeleton v-for="index in 6" :key="index" class="h-32 rounded-xl" />
        </div>
        <div
          v-else-if="visibleItems.length"
          class="grid grid-cols-1 gap-3 lg:grid-cols-2 2xl:grid-cols-3"
        >
          <article
            v-for="item in visibleItems"
            :key="item.id"
            class="flex min-w-0 items-start gap-3 rounded-xl border border-default bg-default p-4 transition-colors hover:border-accented"
          >
            <input
              type="checkbox"
              class="mt-1 size-4 shrink-0 accent-primary"
              :checked="Boolean(selected[item.id])"
              :aria-label="t('assets.select_named', { name: item.name })"
              @change="toggleAsset(item.source, $event)"
            />
            <BlobPreview
              v-if="item.previewBlob"
              :blob="item.previewBlob"
              :alt="item.name"
              class="size-16 shrink-0"
              @state="previewStates[item.id] = $event"
            />
            <div
              v-else
              class="flex size-10 shrink-0 items-center justify-center rounded-lg bg-elevated/70 text-primary"
            >
              <UIcon :name="item.icon" class="size-5" />
            </div>
            <div class="min-w-0 flex-1">
              <div class="flex items-start gap-2">
                <div class="min-w-0 flex-1">
                  <h3 class="truncate text-sm font-medium text-highlighted">{{ item.name }}</h3>
                  <p class="mt-0.5 text-[11px] text-dimmed">{{ item.meta }}</p>
                  <UBadge
                    v-if="previewStates[item.id] === 'unavailable'"
                    color="error"
                    variant="soft"
                    size="sm"
                    class="mt-1"
                  >
                    {{ t('assets.preview_unavailable') }}
                  </UBadge>
                </div>
                <UDropdownMenu :items="assetMenu(item)">
                  <UButton
                    icon="i-tabler-dots"
                    color="neutral"
                    variant="ghost"
                    size="xs"
                    :aria-label="t('assets.asset_actions', { name: item.name })"
                  />
                </UDropdownMenu>
              </div>
              <p v-if="item.description" class="mt-2 line-clamp-2 text-xs leading-5 text-muted">
                {{ item.description }}
              </p>
              <div v-if="item.category || item.tags.length" class="mt-3 flex flex-wrap gap-1.5">
                <UBadge v-if="item.category" color="neutral" variant="soft" size="sm">
                  {{ item.category }}
                </UBadge>
                <UBadge
                  v-for="tag in item.tags"
                  :key="tag"
                  color="primary"
                  variant="subtle"
                  size="sm"
                >
                  {{ tag }}
                </UBadge>
              </div>
            </div>
          </article>
        </div>
        <EmptyState
          v-else
          :icon="
            query
              ? 'i-tabler-search-off'
              : activeTab === 'clips'
                ? 'i-tabler-movie-off'
                : 'i-tabler-photo-off'
          "
          :title="query ? t('assets.no_results') : t(`assets.${activeTab}.empty`)"
          :description="query ? t('assets.no_results_hint') : t(`assets.${activeTab}.empty_hint`)"
        >
          <template v-if="!query && activeTab === 'templates'" #action>
            <UButton
              icon="i-tabler-camera-plus"
              :label="t('assets.templates.capture')"
              :disabled="!selectedTargetSlot"
              @click="captureTemplate"
            />
          </template>
        </EmptyState>
      </div>
      <footer
        v-if="!loading && total > 0"
        class="flex shrink-0 items-center gap-3 border-t border-default bg-default px-4 py-3"
      >
        <input
          type="checkbox"
          class="size-4 accent-primary"
          :checked="allCurrentPageSelected"
          :aria-label="t('assets.select_page')"
          @change="toggleCurrentPage"
        />
        <span class="mr-auto text-xs text-dimmed">
          {{ t('assets.page_summary', { page, pages: pageCount, total }) }}
        </span>
        <UButton
          size="sm"
          color="neutral"
          variant="soft"
          icon="i-tabler-chevron-left"
          :disabled="page <= 1"
          @click="goToPage(page - 1)"
        />
        <UButton
          size="sm"
          color="neutral"
          variant="soft"
          icon="i-tabler-chevron-right"
          :disabled="page >= pageCount"
          @click="goToPage(page + 1)"
        />
      </footer>
    </div>
  </div>

  <BaseModal
    :open="batchEditing"
    :title="t('assets.batch_edit_title', { n: selectedRows.length })"
    icon="i-tabler-tags"
    size="lg"
    @update:open="(open) => (batchEditing = open)"
  >
    <div class="space-y-4">
      <UFormField :label="t('common.category')">
        <UInput v-model="batchDraft.category" />
      </UFormField>
      <UFormField :label="t('common.tags')" :hint="t('assets.tags_hint')">
        <UInput v-model="batchDraft.tags" />
      </UFormField>
    </div>
    <template #footer>
      <UButton color="neutral" variant="ghost" @click="batchEditing = false">
        {{ t('common.cancel') }}
      </UButton>
      <UButton :loading="batchBusy" @click="saveBatchMeta">{{ t('common.save') }}</UButton>
    </template>
  </BaseModal>

  <BaseModal
    :open="!!variantAsset"
    :title="variantAsset?.name ?? t('assets.templates.manage_variants')"
    icon="i-tabler-photo-cog"
    size="lg"
    @update:open="(open) => !open && (variantAsset = null)"
  >
    <div v-if="variantAsset" class="space-y-3">
      <div
        v-for="variant in variantAsset.variants"
        :key="`${variant.resolution[0]}x${variant.resolution[1]}`"
        class="flex items-center gap-3 rounded-lg border border-default p-3"
      >
        <BlobPreview :blob="variant.blob" :alt="variantAsset.name" class="size-14 shrink-0" />
        <div class="min-w-0 flex-1">
          <p class="font-mono text-sm text-highlighted">
            {{ variant.resolution[0] }}×{{ variant.resolution[1] }}
          </p>
          <p class="truncate font-mono text-[10px] text-dimmed">{{ variant.blob.digest }}</p>
        </div>
        <UButton
          color="error"
          variant="ghost"
          icon="i-tabler-trash"
          :disabled="variantAsset.variantCount <= 1 || variantBusy"
          :aria-label="t('assets.templates.remove_variant')"
          @click="removeVariant(variantAsset, variant.resolution)"
        />
      </div>
    </div>
    <template #footer>
      <span v-if="variantAsset?.variantCount === 1" class="mr-auto text-xs text-muted">
        {{ t('assets.templates.last_variant_hint') }}
      </span>
      <UButton color="neutral" variant="ghost" @click="variantAsset = null">
        {{ t('common.close') }}
      </UButton>
      <UButton
        icon="i-tabler-camera-plus"
        :loading="variantBusy"
        :disabled="!selectedTargetSlot"
        @click="recaptureVariant"
      >
        {{ t('assets.templates.recapture') }}
      </UButton>
    </template>
  </BaseModal>

  <BaseModal
    :open="!!editingItem"
    :title="t('assets.edit_title')"
    icon="i-tabler-edit"
    size="lg"
    @update:open="(open) => !open && (editingItem = null)"
  >
    <div class="space-y-4">
      <UFormField :label="t('common.name')" required>
        <UInput v-model="editDraft.name" maxlength="80" />
      </UFormField>
      <UFormField :label="t('common.description')" :hint="t('common.optional')">
        <UTextarea v-model="editDraft.description" :rows="3" />
      </UFormField>
      <UFormField :label="t('common.category')" :hint="t('common.optional')">
        <UInput v-model="editDraft.category" />
      </UFormField>
      <UFormField :label="t('common.tags')" :hint="t('assets.tags_hint')">
        <UInput v-model="editDraft.tags" />
      </UFormField>
    </div>
    <template #footer>
      <UButton color="neutral" variant="ghost" @click="editingItem = null">{{
        t('common.cancel')
      }}</UButton>
      <UButton :disabled="!editDraft.name.trim()" :loading="editBusy" @click="saveAssetMeta">{{
        t('common.save')
      }}</UButton>
    </template>
  </BaseModal>

  <BaseModal
    :open="!!pendingRecording"
    :title="t('recordingSave.title')"
    icon="i-tabler-device-floppy"
    size="lg"
    :show-close="false"
    :dismissible="false"
  >
    <div v-if="pendingRecording" class="space-y-4">
      <div class="rounded-lg border border-default bg-elevated/35 px-4 py-3">
        <div class="flex items-center justify-between gap-3">
          <p class="text-sm font-medium text-highlighted">{{ t('recordingSave.clip_type') }}</p>
          <UBadge color="neutral" variant="soft">
            {{ t(`recordingSave.mode_${pendingRecording.mode}`) }}
          </UBadge>
        </div>
        <p class="mt-1 text-xs text-muted">
          {{
            t('recordingSave.summary', {
              duration: formatDuration(pendingRecording.durationUs),
              count: pendingRecording.eventCount,
            })
          }}
        </p>
      </div>
      <RecordingActionEditor
        v-if="pendingRecording.mode === 'simple' && pendingRecording.actions"
        v-model="recordingActions"
      />
      <p v-else class="rounded-lg border border-default bg-sunken px-3 py-2 text-xs text-muted">
        {{
          t(
            pendingRecording.mode === 'precise'
              ? 'recordingEditor.precise_hint'
              : 'recordingEditor.editing_unavailable',
          )
        }}
      </p>
      <UFormField :label="t('recordingSave.name')" required>
        <UInput v-model="recordingDraft.name" autofocus maxlength="80" />
      </UFormField>
      <UFormField :label="t('common.description')" :hint="t('common.optional')">
        <UTextarea v-model="recordingDraft.description" :rows="2" />
      </UFormField>
      <div class="grid grid-cols-2 gap-3">
        <UFormField :label="t('common.category')" :hint="t('common.optional')">
          <UInput v-model="recordingDraft.category" />
        </UFormField>
        <UFormField :label="t('common.tags')" :hint="t('assets.tags_hint')">
          <UInput v-model="recordingDraft.tags" />
        </UFormField>
      </div>
    </div>
    <template #footer>
      <UButton
        color="error"
        variant="ghost"
        :disabled="recordingSaveBusy"
        @click="discardRecording"
      >
        {{ t('recordingSave.discard') }}
      </UButton>
      <UButton
        :loading="recordingSaveBusy"
        :disabled="!recordingDraft.name.trim()"
        @click="saveRecording"
      >
        {{ t('assets.recording.save_to_library') }}
      </UButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import { backend, type AssetSummary, type BlobRef } from '@/lib/backend'
import { errorMessage } from '@/lib/invoke'
import {
  useRecordingStore,
  type RecordingAction,
  type RecordingMode,
  type RecordingStopPayload,
} from '@/stores/recording'
import { useSettingsStore } from '@/stores/settings'
import { useAssetsStore } from '@/stores/assets'
import { useConfirm } from '@/composables/useConfirm'
import { awaitWailsEvent } from '@/composables/useWailsEvent'
import { useRecordingStart } from '@/composables/useRecordingStart'
import BaseModal from '@/components/common/BaseModal.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import BlobPreview from '@/components/common/BlobPreview.vue'
import RecordingActionEditor from '@/components/recording/RecordingActionEditor.vue'

type AssetTab = 'clips' | 'templates'
type AssetItem = {
  id: string
  kind: AssetTab
  name: string
  description: string
  category: string
  tags: string[]
  meta: string
  icon: string
  previewBlob?: BlobRef
  source: AssetSummary
}

const { t } = useI18n()
const toast = useToast()
const { confirm } = useConfirm()
const settings = useSettingsStore()
const assets = useAssetsStore()
const recording = useRecordingStore()
const { starting: recordingStarting, start: beginRecording } = useRecordingStart()
const activeTab = ref<AssetTab>('clips')
const queryInput = ref('')
const query = ref('')
const categoryFilter = ref('')
const tagsFilter = ref('')
const sort = ref('name_asc')
const page = ref(1)
const pageSize = ref(24)
const total = ref(0)
const assetPage = ref<AssetSummary[]>([])
const loading = ref(false)
const selected = ref<Record<string, AssetSummary>>({})
const batchEditing = ref(false)
const batchBusy = ref(false)
const batchDraft = reactive({ category: '', tags: '' })
const libraryFeedback = ref<{ tone: 'success' | 'warning' | 'error'; message: string } | null>(null)
const variantAsset = ref<AssetSummary | null>(null)
const variantBusy = ref(false)
const cleanupBusy = ref(false)
const selectedTargetSlot = ref('')
const recordingMode = ref<RecordingMode>('simple')
const captureBusy = ref(false)
const editingItem = ref<AssetItem | null>(null)
const editBusy = ref(false)
const pendingRecording = ref<RecordingStopPayload | null>(null)
const recordingSaveBusy = ref(false)
const recordingActions = ref<RecordingAction[]>([])
const editDraft = reactive({ name: '', description: '', category: '', tags: '' })
const recordingDraft = reactive({ name: '', description: '', category: '', tags: '' })
const previewStates = reactive<Record<string, 'loading' | 'ready' | 'unavailable'>>({})
const selectedRows = computed(() => Object.values(selected.value))
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))
const allCurrentPageSelected = computed(
  () => assetPage.value.length > 0 && assetPage.value.every((asset) => selected.value[asset.guid]),
)
const sortItems = computed(() => [
  { label: t('assets.sort_name_asc'), value: 'name_asc' },
  { label: t('assets.sort_name_desc'), value: 'name_desc' },
  { label: t('assets.sort_created_desc'), value: 'created_desc' },
])
const recordingModeItems = computed<Array<{ label: string; value: RecordingMode }>>(() => [
  { label: t('recordingSave.mode_simple'), value: 'simple' },
  { label: t('recordingSave.mode_precise'), value: 'precise' },
])

const targetItems = computed(() =>
  (settings.data?.automation.targets ?? []).map((target) => ({
    label: `${target.label} · ${target.slot}`,
    value: target.slot,
  })),
)
const selectedTargetSupportsRecording = computed(() =>
  (settings.data?.automation.targets ?? []).some(
    (target) => target.slot === selectedTargetSlot.value && target.targetKind === 'desktop-window',
  ),
)
const items = computed<AssetItem[]>(() => {
  return assetPage.value.map((asset) => {
    if (asset.kind === 'clip')
      return {
        id: asset.guid,
        kind: 'clips',
        name: asset.name || asset.guid,
        description: asset.description ?? '',
        category: asset.category ?? '',
        tags: asset.tags ?? [],
        meta: t('assets.clips.library_meta', { bytes: asset.blob?.size ?? 0 }),
        icon: 'i-tabler-movie',
        source: asset,
      }
    return {
      id: asset.guid,
      kind: 'templates',
      name: asset.name || asset.guid,
      description: asset.description ?? '',
      category: asset.category ?? '',
      tags: asset.tags ?? [],
      meta: t('assets.templates.meta', { count: asset.variantCount }),
      icon: 'i-tabler-photo',
      previewBlob: asset.thumbnail,
      source: asset,
    }
  })
})
const visibleItems = computed(() => items.value)
const recordingBadge = computed(() => {
  switch (recording.state.phase) {
    case 'recording':
      return { label: t('recordingHud.recording'), color: 'error' as const }
    case 'paused':
      return { label: t('recordingHud.paused'), color: 'warning' as const }
    case 'finalizing':
      return { label: t('assets.recording.finalizing'), color: 'warning' as const }
    case 'pending':
      return { label: t('recordingSave.pending'), color: 'warning' as const }
    default:
      return { label: t('assets.recording.ready'), color: 'neutral' as const }
  }
})
const recordingHint = computed(() => {
  if (recording.state.phase === 'recording' || recording.state.phase === 'paused')
    return t('assets.recording.active_hint', { target: recording.state.targetSlot })
  return t('assets.recording.hint')
})

onMounted(async () => {
  selectedTargetSlot.value = targetItems.value[0]?.value ?? ''
  await Promise.all([refreshAssets(), recording.reconcile()])
})

watch(activeTab, async () => {
  page.value = 1
  await refreshAssets()
})

watch(
  () => recording.state.pending,
  (pending) => {
    if (!pending || (recording.invocation && recording.invocation !== 'library')) return
    recording.claimInvocation('library')
    openRecordingSave(pending)
  },
  { immediate: true },
)

watch(
  () => recording.completionFailure,
  (failure) => {
    if (failure && recording.invocation === 'library')
      showError(t('recordingSave.save_failed'), failure.message)
  },
)

async function refreshAssets(): Promise<void> {
  loading.value = true
  try {
    const result = await assets.query({
      search: query.value,
      kind: activeTab.value === 'clips' ? 'clip' : 'template',
      category: categoryFilter.value.trim(),
      tags: splitTags(tagsFilter.value),
      sort: sort.value,
      page: page.value,
      pageSize: pageSize.value,
      thumbnailBudget: pageSize.value,
      recentGUIDs: [],
    })
    assetPage.value = result?.items ?? []
    total.value = result?.total ?? 0
    if (page.value > pageCount.value) {
      page.value = pageCount.value
      await refreshAssets()
    }
  } catch (error) {
    showError(t('assets.load_failed'), error)
  } finally {
    loading.value = false
  }
}

async function applyQuery(): Promise<void> {
  query.value = queryInput.value.trim()
  page.value = 1
  await refreshAssets()
}

async function changeQuery(): Promise<void> {
  page.value = 1
  await refreshAssets()
}

async function goToPage(next: number): Promise<void> {
  if (next < 1 || next > pageCount.value || next === page.value) return
  page.value = next
  await refreshAssets()
}

function toggleAsset(asset: AssetSummary, event: Event): void {
  const next = { ...selected.value }
  if ((event.target as HTMLInputElement).checked) next[asset.guid] = asset
  else delete next[asset.guid]
  selected.value = next
}

function toggleCurrentPage(event: Event): void {
  const next = { ...selected.value }
  for (const asset of assetPage.value) {
    if ((event.target as HTMLInputElement).checked) next[asset.guid] = asset
    else delete next[asset.guid]
  }
  selected.value = next
}

function clearSelection(): void {
  selected.value = {}
}

function openBatchEdit(): void {
  batchDraft.category = ''
  batchDraft.tags = ''
  batchEditing.value = true
}

async function saveBatchMeta(): Promise<void> {
  if (!selectedRows.value.length) return
  batchBusy.value = true
  try {
    const results =
      (await backend.assets.batchUpdateMeta(
        selectedRows.value.map((asset) => ({
          guid: asset.guid,
          category: batchDraft.category.trim(),
          tags: splitTags(batchDraft.tags),
        })),
      )) ?? []
    retainFailedSelection(results.filter((result) => !result.updated).map((result) => result.guid))
    libraryFeedback.value = {
      tone: results.some((result) => !result.updated) ? 'warning' : 'success',
      message: t('assets.batch_update_result', {
        updated: results.filter((result) => result.updated).length,
        failed: results.filter((result) => !result.updated).length,
      }),
    }
    batchEditing.value = false
    await refreshAssets()
  } catch (error) {
    showError(t('assets.save_failed'), error)
  } finally {
    batchBusy.value = false
  }
}

async function deleteSelected(): Promise<void> {
  if (!selectedRows.value.length) return
  const accepted = await confirm({
    title: t('assets.batch_delete_title', { n: selectedRows.value.length }),
    description: t('assets.delete_description'),
    confirmText: t('common.delete'),
    color: 'error',
  })
  if (accepted !== true) return
  batchBusy.value = true
  try {
    const results =
      (await backend.assets.batchDelete(selectedRows.value.map((asset) => asset.guid))) ?? []
    retainFailedSelection(results.filter((result) => !result.deleted).map((result) => result.guid))
    libraryFeedback.value = {
      tone: results.some((result) => !result.deleted) ? 'warning' : 'success',
      message: t('assets.batch_delete_result', {
        deleted: results.filter((result) => result.deleted).length,
        failed: results.filter((result) => !result.deleted).length,
      }),
    }
    await refreshAssets()
  } catch (error) {
    showError(t('assets.delete_failed'), error)
  } finally {
    batchBusy.value = false
  }
}

function retainFailedSelection(guids: string[]): void {
  const failed = new Set(guids)
  selected.value = Object.fromEntries(
    selectedRows.value
      .filter((asset) => failed.has(asset.guid))
      .map((asset) => [asset.guid, asset]),
  )
}

async function startRecording(): Promise<void> {
  try {
    await beginRecording(recordingMode.value, selectedTargetSlot.value, 'library')
  } catch (error) {
    showError(t('assets.recording.start_failed'), error)
  }
}

async function pauseRecording(): Promise<void> {
  try {
    await recording.pause()
  } catch (error) {
    showError(t('assets.recording.control_failed'), error)
  }
}

async function resumeRecording(): Promise<void> {
  try {
    await recording.resume()
  } catch (error) {
    showError(t('assets.recording.control_failed'), error)
  }
}

async function stopRecording(): Promise<void> {
  try {
    const payload = await recording.stop()
    if (payload) openRecordingSave(payload)
  } catch (error) {
    showError(t('assets.recording.control_failed'), error)
  }
}

function openRecordingSave(payload: RecordingStopPayload): void {
  if (pendingRecording.value?.pendingID === payload.pendingID) return
  pendingRecording.value = payload
  recordingActions.value = cloneRecordingActions(payload.actions ?? [])
  recordingDraft.name = ''
  recordingDraft.description = ''
  recordingDraft.category = ''
  recordingDraft.tags = ''
}

async function saveRecording(): Promise<void> {
  const pending = pendingRecording.value
  if (!pending) return
  recordingSaveBusy.value = true
  try {
    await recording.finalize({
      pendingID: pending.pendingID,
      label: recordingDraft.name.trim(),
      description: recordingDraft.description.trim(),
      category: recordingDraft.category.trim(),
      tags: splitTags(recordingDraft.tags),
      actions: pending.actions ? cloneRecordingActions(recordingActions.value) : undefined,
    })
    pendingRecording.value = null
    await refreshAssets()
  } catch (error) {
    showError(t('recordingSave.save_failed'), error)
  } finally {
    recordingSaveBusy.value = false
  }
}

async function discardRecording(): Promise<void> {
  const pending = pendingRecording.value
  if (!pending) return
  const accepted = await confirm({
    title: t('recordingSave.discard'),
    description: t('recordingSave.discard_confirm_hint'),
    confirmText: t('common.delete'),
    color: 'error',
  })
  if (accepted !== true) return
  recordingSaveBusy.value = true
  try {
    await recording.discard(pending.pendingID)
    pendingRecording.value = null
  } catch (error) {
    showError(t('recordingSave.discard_failed'), error)
  } finally {
    recordingSaveBusy.value = false
  }
}

function cloneRecordingActions(actions: RecordingAction[]): RecordingAction[] {
  return actions.map((action) => ({
    ...action,
    keys: action.keys ? [...action.keys] : undefined,
    point: action.point ? { ...action.point } : undefined,
  }))
}

async function captureTemplate(): Promise<void> {
  if (!selectedTargetSlot.value) return
  captureBusy.value = true
  const id = `template-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
  try {
    const resultPromise = awaitWailsEvent<{ id: string; payload?: { cancelled?: boolean } }>(
      'tools:picker-result',
      (payload) => payload?.id === id,
    )
    await backend.tools.openScreenPicker('template_save', id, selectedTargetSlot.value)
    const result = await resultPromise
    if (!result.payload?.cancelled) await refreshAssets()
  } catch (error) {
    showError(t('assets.templates.capture_failed'), error)
  } finally {
    captureBusy.value = false
  }
}

function assetMenu(item: AssetItem) {
  const details =
    item.kind === 'templates'
      ? [
          {
            label: t('assets.templates.manage_variants'),
            icon: 'i-tabler-photo-cog',
            onSelect: () => (variantAsset.value = item.source),
          },
        ]
      : []
  return [
    [
      ...details,
      {
        label: t('common.edit'),
        icon: 'i-tabler-edit',
        onSelect: () => openEdit(item),
      },
    ],
    [
      {
        label: t('common.delete'),
        icon: 'i-tabler-trash',
        color: 'error' as const,
        onSelect: () => void deleteAsset(item),
      },
    ],
  ]
}

async function recaptureVariant(): Promise<void> {
  const asset = variantAsset.value
  if (!asset || !selectedTargetSlot.value || variantBusy.value) return
  variantBusy.value = true
  const id = `template-recapture-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
  try {
    const resultPromise = awaitWailsEvent<{ id: string; payload?: { cancelled?: boolean } }>(
      'tools:picker-result',
      (payload) => payload?.id === id,
    )
    await backend.tools.openScreenPicker(
      'template_recapture',
      id,
      selectedTargetSlot.value,
      '',
      asset.guid,
    )
    const result = await resultPromise
    if (!result.payload?.cancelled) {
      await refreshAssets()
      variantAsset.value =
        assetPage.value.find((candidate) => candidate.guid === asset.guid) ?? null
    }
  } catch (error) {
    showError(t('assets.templates.capture_failed'), error)
  } finally {
    variantBusy.value = false
  }
}

async function removeVariant(asset: AssetSummary, resolution: [number, number]): Promise<void> {
  if (asset.variantCount <= 1 || variantBusy.value) return
  const accepted = await confirm({
    title: t('assets.templates.remove_variant_title', {
      resolution: `${resolution[0]}×${resolution[1]}`,
    }),
    description: t('assets.templates.remove_variant_description'),
    confirmText: t('common.delete'),
    color: 'error',
  })
  if (accepted !== true) return
  variantBusy.value = true
  try {
    await backend.assets.removeVariant(asset.guid, resolution[0], resolution[1])
    await refreshAssets()
    variantAsset.value = assetPage.value.find((candidate) => candidate.guid === asset.guid) ?? null
  } catch (error) {
    showError(t('assets.delete_failed'), error)
  } finally {
    variantBusy.value = false
  }
}

async function cleanupLibrary(): Promise<void> {
  if (cleanupBusy.value) return
  cleanupBusy.value = true
  try {
    const preview = await backend.assets.previewCleanup()
    if (!preview) return
    if (preview.candidateCount === 0) {
      libraryFeedback.value = { tone: 'success', message: t('assets.cleanup_none') }
      return
    }
    const accepted = await confirm({
      title: t('assets.cleanup_title'),
      description: t('assets.cleanup_description', {
        count: preview.candidateCount,
        bytes: preview.candidateBytes,
      }),
      confirmText: t('assets.cleanup_action'),
      color: 'warning',
    })
    if (accepted !== true) return
    const result = (await backend.assets.commitCleanup(preview.token)) as
      | { reclaimed?: number }
      | undefined
    libraryFeedback.value = {
      tone: 'success',
      message: t('assets.cleanup_result', { count: result?.reclaimed ?? 0 }),
    }
  } catch (error) {
    showError(t('assets.cleanup_failed'), error)
  } finally {
    cleanupBusy.value = false
  }
}

function openEdit(item: AssetItem): void {
  editingItem.value = item
  editDraft.name = item.name
  editDraft.description = item.description
  editDraft.category = item.category
  editDraft.tags = item.tags.join(', ')
}

async function saveAssetMeta(): Promise<void> {
  const item = editingItem.value
  if (!item || !editDraft.name.trim()) return
  editBusy.value = true
  try {
    const patch = {
      label: editDraft.name.trim(),
      description: editDraft.description.trim(),
      category: editDraft.category.trim(),
      tags: splitTags(editDraft.tags),
    }
    await backend.assets.updateMeta(
      item.id,
      patch.label,
      patch.description,
      patch.category,
      patch.tags,
    )
    editingItem.value = null
    await refreshAssets()
  } catch (error) {
    showError(t('assets.save_failed'), error)
  } finally {
    editBusy.value = false
  }
}

async function deleteAsset(item: AssetItem): Promise<void> {
  const accepted = await confirm({
    title: t('assets.delete_title', { name: item.name }),
    description: t('assets.delete_description'),
    confirmText: t('common.delete'),
    color: 'error',
  })
  if (accepted !== true) return
  try {
    await backend.assets.delete_(item.id)
    const next = { ...selected.value }
    delete next[item.id]
    selected.value = next
    await refreshAssets()
  } catch (error) {
    showError(t('assets.delete_failed'), error)
  }
}

function splitTags(value: string): string[] {
  return [
    ...new Set(
      value
        .split(/[,，]/)
        .map((tag) => tag.trim())
        .filter(Boolean),
    ),
  ]
}

function formatDuration(durationUs: number): string {
  const seconds = Math.max(0, Math.round(durationUs / 1_000_000))
  return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')}`
}

function showError(title: string, error: unknown): void {
  toast.add({
    title,
    description: errorMessage(error),
    color: 'error',
  })
}
</script>
