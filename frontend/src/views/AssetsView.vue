<template>
  <div data-testid="assets-view" class="workspace-page">
    <header class="workspace-page__header">
      <div class="min-w-0">
        <div class="flex items-center gap-3">
          <span class="workspace-page__mark">
            <UIcon name="i-tabler-library" class="size-5" />
          </span>
          <div class="min-w-0">
            <p class="workspace-page__eyebrow">{{ t('assets.eyebrow') }}</p>
            <div class="flex min-w-0 items-center gap-2">
              <h1 class="workspace-page__title truncate">{{ t('assets.title') }}</h1>
              <UBadge color="neutral" variant="soft" size="sm">{{ total }}</UBadge>
            </div>
          </div>
        </div>
      </div>
      <div class="flex shrink-0 flex-wrap items-center justify-end gap-2">
        <AdaptiveSelect
          v-model="selectedTargetSlot"
          :items="targetItems"
          value-key="value"
          label-key="label"
          class="shrink-0"
          :placeholder="t('assets.target_placeholder')"
          :aria-label="t('assets.target_placeholder')"
        />
        <template v-if="activeTab === 'clips' && recording.state.phase === 'idle'">
          <AdaptiveSelect
            v-model="recordingMode"
            :items="recordingModeItems"
            value-key="value"
            label-key="label"
            class="shrink-0"
            :aria-label="t('assets.recording.mode')"
          />
          <UButton
            data-testid="assets-recording-start"
            icon="i-tabler-player-record"
            :label="t('assets.recording.start')"
            :disabled="!selectedTargetSlot || !selectedTargetSupportsRecording || recordingStarting"
            :loading="recordingStarting"
            @click="startRecording"
          />
        </template>
        <UButton
          v-else-if="activeTab === 'templates'"
          icon="i-tabler-camera-plus"
          :label="t('assets.templates.capture')"
          :disabled="!selectedTargetSlot"
          :loading="captureBusy"
          @click="captureTemplate"
        />
        <UDropdownMenu :items="libraryMenuItems">
          <UButton
            icon="i-tabler-dots-vertical"
            color="neutral"
            variant="ghost"
            :aria-label="t('assets.library_actions')"
          />
        </UDropdownMenu>
      </div>
    </header>

    <div class="flex min-h-0 flex-1">
      <aside class="flex w-52 shrink-0 flex-col border-r border-default bg-elevated/15 p-2">
        <p class="px-2 pb-2 pt-1 text-[10px] font-semibold uppercase tracking-wide text-dimmed">
          {{ t('assets.asset_types') }}
        </p>
        <UButton
          color="neutral"
          :variant="activeTab === 'clips' ? 'soft' : 'ghost'"
          icon="i-tabler-movie"
          class="h-auto w-full justify-start px-2.5 py-2 text-left"
          @click="activeTab = 'clips'"
        >
          <span class="min-w-0 flex-1">
            <span class="block text-xs font-medium">{{ t('assets.tabs.clips') }}</span>
            <span class="mt-0.5 block truncate text-[10px] text-dimmed">{{
              t('assets.clips.nav_hint')
            }}</span>
          </span>
          <UBadge v-if="activeTab === 'clips'" color="neutral" variant="soft" size="xs">{{
            total
          }}</UBadge>
        </UButton>
        <UButton
          color="neutral"
          :variant="activeTab === 'templates' ? 'soft' : 'ghost'"
          icon="i-tabler-photo"
          class="mt-1 h-auto w-full justify-start px-2.5 py-2 text-left"
          @click="activeTab = 'templates'"
        >
          <span class="min-w-0 flex-1">
            <span class="block text-xs font-medium">{{ t('assets.tabs.templates') }}</span>
            <span class="mt-0.5 block truncate text-[10px] text-dimmed">{{
              t('assets.templates.nav_hint')
            }}</span>
          </span>
          <UBadge v-if="activeTab === 'templates'" color="neutral" variant="soft" size="xs">{{
            total
          }}</UBadge>
        </UButton>
      </aside>

      <main class="flex min-h-0 min-w-0 flex-1 flex-col">
        <div class="shrink-0 border-b border-default bg-elevated/10">
          <form
            class="flex items-center gap-2 border-b border-default p-3"
            role="search"
            @submit.prevent="applyQuery"
          >
            <UInput
              v-model="queryInput"
              icon="i-tabler-search"
              class="min-w-56 flex-1"
              :placeholder="t('assets.search_all_placeholder')"
              :aria-label="t('assets.search_all_placeholder')"
            />
            <UButton
              type="submit"
              color="neutral"
              variant="soft"
              icon="i-tabler-search"
              :label="t('assets.search_action')"
            />
          </form>
          <LibrarySelectionToolbar
            v-if="selectedRows.length"
            :label="t('assets.selected_count', { n: selectedRows.length })"
            :hint="t('batchMetadata.selection_hint')"
            :clear-label="t('assets.clear_selection')"
            @clear="clearSelection"
          >
            <UButton
              data-testid="asset-batch-metadata"
              size="sm"
              variant="soft"
              icon="i-tabler-category-plus"
              :disabled="batchBusy"
              @click="openBatchEdit"
            >
              {{ t('assets.batch_edit') }}
            </UButton>
            <template #destructive>
              <UButton
                size="sm"
                color="error"
                variant="ghost"
                icon="i-tabler-trash"
                :loading="batchBusy"
                @click="deleteSelected"
              >
                {{ t('assets.batch_delete') }}
              </UButton>
            </template>
          </LibrarySelectionToolbar>
          <div v-else class="flex flex-wrap items-center gap-2 p-3">
            <AdaptiveSelect
              v-model="categoryFilter"
              :items="categoryFilterItems"
              icon="i-tabler-category"
              @update:model-value="changeQuery"
            />
            <UInputMenu
              v-model="tagFilters"
              :items="tagOptions"
              multiple
              icon="i-tabler-tags"
              class="min-w-56 max-w-md flex-1"
              :placeholder="t('assets.all_tags')"
              @update:model-value="changeQuery"
            />
            <AdaptiveSelect
              v-model="sort"
              :items="sortItems"
              icon="i-tabler-arrows-sort"
              @update:model-value="changeQuery"
            />
            <UButton
              v-if="hasLibraryFilters"
              color="neutral"
              variant="ghost"
              icon="i-tabler-filter-x"
              :label="t('assets.reset_filters')"
              @click="resetLibraryFilters"
            />
          </div>
        </div>

        <section
          v-if="activeTab === 'clips' && recording.state.phase !== 'idle'"
          data-testid="assets-recording-controls"
          class="flex shrink-0 items-center gap-3 border-b border-default bg-primary/5 px-3 py-2"
        >
          <span class="size-2 rounded-full bg-error" aria-hidden="true" />
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <p class="text-xs font-medium text-highlighted">{{ t('assets.recording.title') }}</p>
              <UBadge :color="recordingBadge.color" variant="soft" size="xs">{{
                recordingBadge.label
              }}</UBadge>
            </div>
            <p class="truncate text-[10px] text-dimmed">{{ recordingHint }}</p>
          </div>
          <UButton
            v-if="recording.state.phase === 'recording'"
            size="xs"
            color="neutral"
            variant="soft"
            icon="i-tabler-player-pause"
            :label="t('recordingHud.pause')"
            @click="pauseRecording"
          />
          <UButton
            v-if="recording.state.phase === 'paused'"
            size="xs"
            color="neutral"
            variant="soft"
            icon="i-tabler-player-play"
            :label="t('recordingHud.resume')"
            @click="resumeRecording"
          />
          <UButton
            v-if="recording.state.phase === 'recording' || recording.state.phase === 'paused'"
            size="xs"
            color="error"
            variant="soft"
            icon="i-tabler-square"
            :label="t('recordingHud.stop')"
            @click="stopRecording"
          />
        </section>

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

        <div class="min-h-0 flex-1 overflow-y-auto bg-elevated/10">
          <div v-if="loading" class="space-y-px p-2">
            <USkeleton v-for="index in 10" :key="index" class="h-14 rounded-md" />
          </div>
          <div v-else-if="visibleItems.length" class="min-w-[1080px]">
            <div
              class="grid h-9 grid-cols-[2.25rem_minmax(18rem,2fr)_10rem_minmax(12rem,1.2fr)_9rem_9rem_2.5rem] items-center gap-3 border-b border-default bg-elevated/35 px-3 text-[10px] font-semibold uppercase tracking-wide text-dimmed"
            >
              <UCheckbox
                :model-value="allCurrentPageSelected"
                :aria-label="t('assets.select_page')"
                @update:model-value="toggleCurrentPage(Boolean($event))"
              />
              <span>{{ t('assets.columns.asset') }}</span>
              <span>{{ t('common.category') }}</span>
              <span>{{ t('common.tags') }}</span>
              <span>{{ t('assets.columns.details') }}</span>
              <span>{{ t('assets.columns.created') }}</span>
              <span />
            </div>
            <article
              v-for="item in visibleItems"
              :key="item.id"
              class="grid min-h-16 grid-cols-[2.25rem_minmax(18rem,2fr)_10rem_minmax(12rem,1.2fr)_9rem_9rem_2.5rem] items-center gap-3 border-b border-default/70 px-3 hover:bg-elevated/35"
            >
              <UCheckbox
                :model-value="Boolean(selected[item.id])"
                :aria-label="t('assets.select_named', { name: item.name })"
                @update:model-value="toggleAsset(item.source, Boolean($event))"
              />
              <div class="flex min-w-0 items-center gap-2.5 py-1.5">
                <BlobPreview
                  v-if="item.previewBlob"
                  :blob="item.previewBlob"
                  :alt="item.name"
                  class="size-10 shrink-0"
                  @state="previewStates[item.id] = $event"
                />
                <div
                  v-else
                  class="flex size-9 shrink-0 items-center justify-center rounded-md bg-elevated text-primary"
                >
                  <UIcon :name="item.icon" class="size-4" />
                </div>
                <div class="min-w-0">
                  <h3 class="truncate text-xs font-medium text-highlighted">{{ item.name }}</h3>
                  <p class="mt-0.5 truncate text-[10px] text-dimmed">
                    {{ item.description || item.meta }}
                  </p>
                </div>
              </div>
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
                >
                  {{ tag }}
                </UBadge>
                <span v-if="!item.tags.length" class="text-[10px] text-dimmed">{{
                  t('assets.no_tags')
                }}</span>
                <span v-else-if="item.tags.length > 3" class="text-[10px] text-dimmed"
                  >+{{ item.tags.length - 3 }}</span
                >
              </div>
              <span class="truncate text-[10px] text-muted">{{ item.meta }}</span>
              <span class="truncate text-[10px] text-dimmed">{{
                formatAssetDate(item.source.createdAt)
              }}</span>
              <UDropdownMenu :items="assetMenu(item)">
                <UButton
                  icon="i-tabler-dots"
                  color="neutral"
                  variant="ghost"
                  size="xs"
                  :aria-label="t('assets.asset_actions', { name: item.name })"
                />
              </UDropdownMenu>
            </article>
          </div>
          <EmptyState
            v-else
            inset
            :icon="
              hasLibraryFilters
                ? 'i-tabler-search-off'
                : activeTab === 'clips'
                  ? 'i-tabler-movie-off'
                  : 'i-tabler-photo-off'
            "
            :title="hasLibraryFilters ? t('assets.no_results') : t(`assets.${activeTab}.empty`)"
            :description="
              hasLibraryFilters ? t('assets.no_results_hint') : t(`assets.${activeTab}.empty_hint`)
            "
          >
            <template #action>
              <UButton
                v-if="hasLibraryFilters"
                color="neutral"
                variant="soft"
                icon="i-tabler-filter-x"
                :label="t('assets.reset_filters')"
                @click="resetLibraryFilters"
              />
              <UButton
                v-else-if="activeTab === 'templates'"
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
          class="flex h-11 shrink-0 items-center gap-3 border-t border-default bg-default px-3"
        >
          <span class="mr-auto text-xs text-dimmed">
            {{ t('assets.result_range', { start: resultStart, end: resultEnd, total }) }}
          </span>
          <UPagination
            :page="page"
            :total="total"
            :items-per-page="pageSize"
            :sibling-count="1"
            show-edges
            @update:page="goToPage"
          />
          <span class="text-xs text-dimmed">{{ t('assets.per_page') }}</span>
          <AdaptiveSelect
            v-model="pageSize"
            :items="pageSizeItems"
            class="w-24"
            width-mode="fixed"
            @update:model-value="changeQuery"
          />
        </footer>
      </main>
    </div>
  </div>

  <BaseModal
    :open="batchEditing"
    :title="t('assets.batch_edit_title', { n: selectedRows.length })"
    icon="i-tabler-tags"
    size="lg"
    @update:open="(open) => (batchEditing = open)"
  >
    <div class="space-y-5">
      <p class="text-sm text-muted">{{ t('batchMetadata.description') }}</p>
      <UFormField :label="t('common.category')">
        <div class="flex items-center gap-2">
          <AdaptiveSelect
            v-model="batchDraft.categoryMode"
            :items="categoryModeItems"
            class="shrink-0"
          />
          <UInputMenu
            v-if="batchDraft.categoryMode === 'set'"
            v-model="batchDraft.category"
            :items="batchCategoryOptions"
            :create-item="'always'"
            class="min-w-0 flex-1"
            @create="createBatchCategory"
          />
          <span v-else class="text-xs text-dimmed">{{ categoryModeHint }}</span>
        </div>
      </UFormField>
      <UFormField :label="t('common.tags')">
        <div class="flex items-start gap-2">
          <AdaptiveSelect v-model="batchDraft.tagMode" :items="tagModeItems" class="shrink-0" />
          <UInputMenu
            v-if="tagModeNeedsValues"
            v-model="batchDraft.tags"
            :items="batchTagOptions"
            :create-item="'always'"
            multiple
            class="min-w-0 flex-1"
            @create="createBatchTag"
          />
          <span v-else class="pt-2 text-xs text-dimmed">{{ tagModeHint }}</span>
        </div>
      </UFormField>
    </div>
    <template #footer>
      <UButton color="neutral" variant="ghost" @click="batchEditing = false">
        {{ t('common.cancel') }}
      </UButton>
      <UButton
        icon="i-tabler-check"
        :label="t('batchMetadata.apply')"
        :loading="batchBusy"
        :disabled="!batchDraftValid"
        @click="saveBatchMeta"
      />
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
        <UInputMenu
          v-model="editDraft.category"
          :items="metadataCategoryOptions"
          :create-item="'always'"
          @create="createEditCategory"
        />
      </UFormField>
      <UFormField :label="t('common.tags')" :hint="t('common.optional')">
        <UInputMenu
          v-model="editDraft.tags"
          :items="metadataTagOptions"
          :create-item="'always'"
          multiple
          @create="createEditTag"
        />
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
          <UInputMenu
            v-model="recordingDraft.category"
            :items="metadataCategoryOptions"
            :create-item="'always'"
            @create="createRecordingCategory"
          />
        </UFormField>
        <UFormField :label="t('common.tags')" :hint="t('common.optional')">
          <UInputMenu
            v-model="recordingDraft.tags"
            :items="metadataTagOptions"
            :create-item="'always'"
            multiple
            @create="createRecordingTag"
          />
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
  applyBatchMetadata,
  createBatchMetadataDraft,
  hasBatchMetadataChange,
} from '@/lib/batchMetadata'
import {
  useRecordingStore,
  type RecordingAction,
  type RecordingMode,
  type RecordingStopPayload,
} from '@/stores/recording'
import { useSettingsStore } from '@/stores/settings'
import { useAssetsStore } from '@/stores/assets'
import { useConfirm } from '@/composables/useConfirm'
import { useAutoDismissFeedback } from '@/composables/useAutoDismissFeedback'
import { awaitWailsEvent } from '@/composables/useWailsEvent'
import { useRecordingStart } from '@/composables/useRecordingStart'
import BaseModal from '@/components/common/BaseModal.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import BlobPreview from '@/components/common/BlobPreview.vue'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'
import RecordingActionEditor from '@/components/recording/RecordingActionEditor.vue'
import LibrarySelectionToolbar from '@/components/library/LibrarySelectionToolbar.vue'

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
type AssetMetadataDraft = { category: string; tags: string[] }
const allCategories = '__all__'

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
const categoryFilter = ref(allCategories)
const tagFilters = ref<string[]>([])
const categories = ref<Array<{ value: string; count: number }>>([])
const tags = ref<Array<{ value: string; count: number }>>([])
const sort = ref('name_asc')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const assetPage = ref<AssetSummary[]>([])
const loading = ref(false)
const selected = ref<Record<string, AssetSummary>>({})
const batchEditing = ref(false)
const batchBusy = ref(false)
const batchDraft = reactive(createBatchMetadataDraft())
const libraryFeedback = ref<{ tone: 'success' | 'warning' | 'error'; message: string } | null>(null)
useAutoDismissFeedback(libraryFeedback)
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
const editDraft = reactive({ name: '', description: '', category: '', tags: [] as string[] })
const recordingDraft = reactive({ name: '', description: '', category: '', tags: [] as string[] })
const createdCategories = ref<string[]>([])
const createdTags = ref<string[]>([])
const previewStates = reactive<Record<string, 'loading' | 'ready' | 'unavailable'>>({})
const selectedRows = computed(() => Object.values(selected.value))
const hasLibraryFilters = computed(() =>
  Boolean(query.value || categoryFilter.value !== allCategories || tagFilters.value.length),
)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))
const resultStart = computed(() => (total.value ? (page.value - 1) * pageSize.value + 1 : 0))
const resultEnd = computed(() => Math.min(page.value * pageSize.value, total.value))
const allCurrentPageSelected = computed(
  () => assetPage.value.length > 0 && assetPage.value.every((asset) => selected.value[asset.guid]),
)
const sortItems = computed(() => [
  { label: t('assets.sort_name_asc'), value: 'name_asc' },
  { label: t('assets.sort_name_desc'), value: 'name_desc' },
  { label: t('assets.sort_created_desc'), value: 'created_desc' },
])
const pageSizeItems = [
  { label: '20', value: 20 },
  { label: '50', value: 50 },
  { label: '100', value: 100 },
]
const categoryFilterItems = computed(() => [
  { label: t('assets.all_categories'), value: allCategories },
  ...categories.value.map((item) => ({
    label: `${item.value} (${item.count})`,
    value: item.value,
  })),
])
const tagOptions = computed(() => tags.value.map((item) => item.value))
const metadataCategoryOptions = computed(() =>
  uniqueStrings([
    ...categories.value.map((item) => item.value),
    ...createdCategories.value,
    editDraft.category,
    recordingDraft.category,
  ]),
)
const metadataTagOptions = computed(() =>
  uniqueStrings([
    ...tags.value.map((item) => item.value),
    ...createdTags.value,
    ...editDraft.tags,
    ...recordingDraft.tags,
  ]),
)
const batchCategoryOptions = computed(() =>
  uniqueStrings([
    ...categories.value.map((item) => item.value),
    ...createdCategories.value,
    batchDraft.category,
  ]),
)
const batchTagOptions = computed(() =>
  uniqueStrings([
    ...tags.value.map((item) => item.value),
    ...createdTags.value,
    ...batchDraft.tags,
  ]),
)
const categoryModeItems = computed(() => [
  { label: t('batchMetadata.keep'), value: 'keep' },
  { label: t('batchMetadata.set'), value: 'set' },
  { label: t('batchMetadata.clear'), value: 'clear' },
])
const tagModeItems = computed(() => [
  { label: t('batchMetadata.keep'), value: 'keep' },
  { label: t('batchMetadata.add'), value: 'add' },
  { label: t('batchMetadata.remove'), value: 'remove' },
  { label: t('batchMetadata.replace'), value: 'replace' },
  { label: t('batchMetadata.clear'), value: 'clear' },
])
const tagModeNeedsValues = computed(() => ['add', 'remove', 'replace'].includes(batchDraft.tagMode))
const batchDraftValid = computed(() => hasBatchMetadataChange(batchDraft))
const categoryModeHint = computed(() =>
  t(
    batchDraft.categoryMode === 'clear'
      ? 'batchMetadata.category_clear_hint'
      : 'batchMetadata.keep_hint',
  ),
)
const tagModeHint = computed(() =>
  t(batchDraft.tagMode === 'clear' ? 'batchMetadata.tags_clear_hint' : 'batchMetadata.keep_hint'),
)
const recordingModeItems = computed<Array<{ label: string; value: RecordingMode }>>(() => [
  { label: t('recordingSave.mode_simple'), value: 'simple' },
  { label: t('recordingSave.mode_precise'), value: 'precise' },
])
const libraryMenuItems = computed(() => [
  [
    {
      label: t('common.refresh'),
      icon: 'i-tabler-refresh',
      onSelect: () => void refreshAssets(),
    },
    {
      label: t('assets.cleanup_action'),
      icon: 'i-tabler-recycle',
      disabled: cleanupBusy.value,
      onSelect: () => void cleanupLibrary(),
    },
  ],
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
  clearSelection()
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
      category: categoryFilter.value === allCategories ? '' : categoryFilter.value.trim(),
      tags: tagFilters.value,
      sort: sort.value,
      page: page.value,
      pageSize: pageSize.value,
      thumbnailBudget: pageSize.value,
      recentGUIDs: [],
    })
    assetPage.value = result?.items ?? []
    total.value = result?.total ?? 0
    categories.value = result?.categories ?? []
    tags.value = result?.tags ?? []
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

async function resetLibraryFilters(): Promise<void> {
  queryInput.value = ''
  query.value = ''
  categoryFilter.value = allCategories
  tagFilters.value = []
  await changeQuery()
}

async function goToPage(next: number): Promise<void> {
  if (next < 1 || next > pageCount.value || next === page.value) return
  page.value = next
  await refreshAssets()
}

function toggleAsset(asset: AssetSummary, checked: boolean): void {
  const next = { ...selected.value }
  if (checked) next[asset.guid] = asset
  else delete next[asset.guid]
  selected.value = next
}

function toggleCurrentPage(checked: boolean): void {
  const next = { ...selected.value }
  for (const asset of assetPage.value) {
    if (checked) next[asset.guid] = asset
    else delete next[asset.guid]
  }
  selected.value = next
}

function clearSelection(): void {
  selected.value = {}
}

function openBatchEdit(): void {
  Object.assign(batchDraft, createBatchMetadataDraft())
  batchEditing.value = true
}

async function saveBatchMeta(): Promise<void> {
  if (!selectedRows.value.length || !batchDraftValid.value) return
  batchBusy.value = true
  try {
    const results =
      (await backend.assets.batchUpdateMeta(
        selectedRows.value.map((asset) => {
          const metadata = applyBatchMetadata(
            { category: asset.category ?? '', tags: asset.tags ?? [] },
            batchDraft,
          )
          return { guid: asset.guid, category: metadata.category, tags: metadata.tags }
        }),
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
  recordingDraft.tags = []
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
      tags: uniqueStrings(recordingDraft.tags),
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
  editDraft.tags = [...item.tags]
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
      tags: uniqueStrings(editDraft.tags),
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

function createAssetCategory(value: string, draft: AssetMetadataDraft): void {
  const category = value.trim()
  if (!category) return
  createdCategories.value = uniqueStrings([...createdCategories.value, category])
  draft.category = category
}

function createBatchCategory(value: string): void {
  createAssetCategory(value, batchDraft)
}

function createEditCategory(value: string): void {
  createAssetCategory(value, editDraft)
}

function createRecordingCategory(value: string): void {
  createAssetCategory(value, recordingDraft)
}

function createAssetTag(value: string, draft: AssetMetadataDraft): void {
  const tag = value.trim()
  if (!tag) return
  createdTags.value = uniqueStrings([...createdTags.value, tag])
  draft.tags = uniqueStrings([...draft.tags, tag])
}

function createBatchTag(value: string): void {
  createAssetTag(value, batchDraft)
}

function createEditTag(value: string): void {
  createAssetTag(value, editDraft)
}

function createRecordingTag(value: string): void {
  createAssetTag(value, recordingDraft)
}

function uniqueStrings(values: string[]): string[] {
  const seen = new Set<string>()
  return values
    .map((value) => value.trim())
    .filter((value) => {
      const key = value.toLocaleLowerCase()
      if (!key || seen.has(key)) return false
      seen.add(key)
      return true
    })
}

function formatDuration(durationUs: number): string {
  const seconds = Math.max(0, Math.round(durationUs / 1_000_000))
  return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')}`
}

function formatAssetDate(value?: string): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return new Intl.DateTimeFormat(undefined, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(date)
}

function showError(title: string, error: unknown): void {
  toast.add({
    title,
    description: errorMessage(error),
    color: 'error',
  })
}
</script>
