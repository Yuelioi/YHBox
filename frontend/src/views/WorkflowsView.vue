<template>
  <div class="workspace-page">
    <header class="workspace-page__header">
      <div class="min-w-0">
        <div class="flex items-center gap-3">
          <span class="workspace-page__mark">
            <UIcon name="i-tabler-route" class="size-5" />
          </span>
          <div class="min-w-0">
            <p class="workspace-page__eyebrow">{{ t('workflow.list.eyebrow') }}</p>
            <div class="flex min-w-0 items-center gap-2">
              <h1 class="workspace-page__title truncate">{{ t('workflow.list.title') }}</h1>
              <UBadge color="neutral" variant="soft" size="sm">{{ total }}</UBadge>
            </div>
          </div>
        </div>
      </div>
      <div class="flex shrink-0 flex-wrap items-center justify-end gap-2">
        <UButton
          data-testid="workflow-new-button"
          icon="i-tabler-plus"
          :label="t('workflow.list.new_workflow')"
          @click="openCreateModal"
        />
        <UDropdownMenu :items="libraryMenuItems">
          <UButton
            color="neutral"
            variant="ghost"
            icon="i-tabler-dots-vertical"
            :aria-label="t('workflow.list.library_actions')"
          />
        </UDropdownMenu>
      </div>
    </header>

    <main class="flex min-h-0 flex-1 flex-col px-6 py-4">
      <section
        v-if="recoveries.length"
        class="mb-3 shrink-0 rounded-lg border border-warning/35 bg-warning/10 px-3 py-2"
        role="alert"
        data-testid="workflow-recovery-panel"
      >
        <div class="flex items-center gap-2">
          <UIcon name="i-tabler-first-aid-kit" class="size-4 shrink-0 text-warning" />
          <p class="min-w-0 flex-1 text-xs text-default">
            {{ t('workflow.list.recovery_title', { n: recoveries.length }) }}
          </p>
        </div>
        <ul class="mt-2 grid gap-2 xl:grid-cols-2">
          <li
            v-for="recovery in recoveries"
            :key="recovery.recoveryId"
            class="flex items-center gap-2 rounded-md border border-default bg-default/70 px-3 py-2"
          >
            <div class="min-w-0 flex-1">
              <p class="truncate text-xs font-medium text-default">{{ recovery.originalName }}</p>
              <p class="truncate text-[10px] text-dimmed">{{ recovery.reason }}</p>
            </div>
            <UButton
              size="xs"
              color="neutral"
              variant="soft"
              icon="i-tabler-tool"
              :label="t('workflow.list.recovery_repair')"
              @click="openRecoveryRepair(recovery)"
            />
            <UButton
              size="xs"
              color="error"
              variant="ghost"
              icon="i-tabler-trash"
              :aria-label="t('workflow.list.recovery_delete')"
              :loading="recoveryBusyId === recovery.recoveryId"
              @click="deleteRecovery(recovery)"
            />
          </li>
        </ul>
      </section>

      <section class="shrink-0 overflow-hidden rounded-t-lg border border-default bg-elevated/15">
        <form
          class="flex items-center gap-2 border-b border-default p-3"
          role="search"
          @submit.prevent="applySearch"
        >
          <UInput
            v-model="searchInput"
            icon="i-tabler-search"
            :placeholder="t('workflow.list.search_all_placeholder')"
            class="min-w-0 flex-1"
          />
          <UButton type="submit" color="neutral" variant="soft" icon="i-tabler-search">
            {{ t('workflow.list.search_action') }}
          </UButton>
        </form>
        <LibrarySelectionToolbar
          v-if="selectedRows.length"
          :label="t('workflow.list.selected_count', { n: selectedRows.length })"
          :hint="t('batchMetadata.selection_hint')"
          :clear-label="t('workflow.list.clear_selection')"
          @clear="clearSelection"
        >
          <UButton
            data-testid="workflow-batch-metadata"
            size="sm"
            variant="soft"
            icon="i-tabler-category-plus"
            :disabled="batchBusy"
            @click="openBatchEdit"
          >
            {{ t('workflow.list.batch_edit') }}
          </UButton>
          <UButton
            size="sm"
            color="neutral"
            variant="ghost"
            icon="i-tabler-file-export"
            :loading="batchExporting"
            :disabled="portabilityBusy"
            @click="exportSelected"
          >
            {{ t('workflow.list.export_selected') }}
          </UButton>
          <template #destructive>
            <UButton
              size="sm"
              color="error"
              variant="ghost"
              icon="i-tabler-trash"
              :loading="deleting"
              @click="requestDelete(selectedRows)"
            >
              {{ t('workflow.list.delete_selected') }}
            </UButton>
          </template>
        </LibrarySelectionToolbar>
        <div v-else class="flex flex-wrap items-center gap-2 p-3">
          <AdaptiveSelect
            v-model="categoryFilter"
            :items="categoryFilterItems"
            class="shrink-0"
            icon="i-tabler-category"
            @update:model-value="queryChanged"
          />
          <UInputMenu
            v-model="tagFilters"
            :items="tagOptions"
            multiple
            class="min-w-56 max-w-md flex-1"
            icon="i-tabler-tags"
            :placeholder="t('workflow.list.all_tags')"
            @update:model-value="queryChanged"
          />
          <AdaptiveSelect
            v-model="createdRange"
            :items="createdRangeItems"
            class="shrink-0"
            icon="i-tabler-calendar-plus"
            @update:model-value="queryChanged"
          />
          <AdaptiveSelect
            v-model="updatedRange"
            :items="updatedRangeItems"
            class="shrink-0"
            icon="i-tabler-calendar-stats"
            @update:model-value="queryChanged"
          />
          <AdaptiveSelect
            v-model="sort"
            :items="sortItems"
            class="shrink-0"
            icon="i-tabler-arrows-sort"
            @update:model-value="queryChanged"
          />
          <UDropdownMenu :items="columnMenuItems">
            <UButton
              color="neutral"
              variant="soft"
              icon="i-tabler-columns-3"
              trailing-icon="i-tabler-chevron-down"
              :label="t('workflow.list.columns')"
            />
          </UDropdownMenu>
          <UButton
            v-if="hasFilters"
            color="neutral"
            variant="ghost"
            icon="i-tabler-filter-x"
            :label="t('workflow.list.reset_filters')"
            @click="resetFilters"
          />
        </div>
      </section>

      <div
        v-if="portabilityFeedback || deleteFeedback"
        class="shrink-0 border-x border-b px-4 py-2 text-xs"
        :class="feedbackClass"
        :role="activeFeedback?.tone === 'error' ? 'alert' : 'status'"
      >
        <p>{{ activeFeedback?.message }}</p>
        <ul v-if="activeFeedback?.details.length" class="mt-1 list-disc pl-5 text-[11px]">
          <li v-for="detail in activeFeedback.details" :key="detail">{{ detail }}</li>
        </ul>
      </div>

      <div class="min-h-0 flex-1 overflow-auto border-x border-default bg-default">
        <div v-if="loading" class="space-y-px p-2" :aria-label="t('workflow.list.loading')">
          <USkeleton v-for="index in 10" :key="index" class="h-14 rounded-md" />
        </div>
        <div
          v-else-if="failure"
          class="m-3 flex items-center gap-3 rounded-lg border border-error/35 bg-error/10 px-4 py-3 text-sm text-error"
          role="alert"
        >
          <span class="min-w-0 flex-1">{{ failure }}</span>
          <UButton size="xs" color="error" variant="soft" @click="load">{{
            t('common.retry')
          }}</UButton>
        </div>
        <EmptyState
          v-else-if="sources.length === 0"
          inset
          :icon="hasFilters ? 'i-tabler-filter-off' : 'i-tabler-route-off'"
          :title="t(hasFilters ? 'workflow.list.no_results_title' : 'workflow.list.empty_title')"
          :description="
            t(
              hasFilters
                ? 'workflow.list.no_results_description'
                : 'workflow.list.empty_description',
            )
          "
        >
          <template #action>
            <UButton
              v-if="hasFilters"
              color="neutral"
              variant="soft"
              icon="i-tabler-filter-x"
              :label="t('workflow.list.reset_filters')"
              @click="resetFilters"
            />
            <UButton
              v-else
              icon="i-tabler-plus"
              :label="t('workflow.list.new_workflow')"
              @click="openCreateModal"
            />
          </template>
        </EmptyState>
        <div v-else class="min-w-[1100px]">
          <div
            class="grid h-9 items-center gap-3 border-b border-default bg-elevated/40 px-3 text-[10px] font-semibold uppercase tracking-wide text-dimmed"
            :style="{ gridTemplateColumns: workflowGridTemplate }"
          >
            <UCheckbox
              :model-value="allCurrentPageSelected"
              :aria-label="t('workflow.list.select_page')"
              @update:model-value="toggleCurrentPage(Boolean($event))"
            />
            <span>{{ t('workflow.list.name') }}</span>
            <span v-if="isColumnVisible('category')">{{ t('workflow.list.category') }}</span>
            <span v-if="isColumnVisible('tags')">{{ t('workflow.list.tags') }}</span>
            <span v-if="isColumnVisible('nodes')" class="text-right">{{
              t('workflow.list.nodes')
            }}</span>
            <span v-if="isColumnVisible('revision')" class="text-right">{{
              t('workflow.list.revision')
            }}</span>
            <span v-if="isColumnVisible('createdAt')">{{ t('workflow.list.created_at') }}</span>
            <span v-if="isColumnVisible('updatedAt')">{{ t('workflow.list.updated_at') }}</span>
            <span class="text-right">{{ t('workflow.list.actions') }}</span>
          </div>
          <article
            v-for="source in sources"
            :key="source.workflowId"
            class="grid min-h-16 items-center gap-3 border-b border-default/70 px-3 py-2 hover:bg-elevated/30"
            :style="{ gridTemplateColumns: workflowGridTemplate }"
          >
            <UCheckbox
              :model-value="Boolean(selected[source.workflowId])"
              :aria-label="t('workflow.list.select_named', { name: source.name })"
              @update:model-value="toggleSource(source, Boolean($event))"
            />
            <div class="min-w-0">
              <div class="flex min-w-0 items-center gap-2">
                <RouterLink
                  :to="`/workflows/${source.workflowId}/edit`"
                  class="truncate text-sm font-medium text-highlighted underline-offset-4 hover:text-primary hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                >
                  {{ source.name }}
                </RouterLink>
                <UBadge
                  v-if="runFeedbackById[source.workflowId]"
                  :color="runFeedbackById[source.workflowId].tone"
                  variant="soft"
                  size="xs"
                >
                  {{ runFeedbackById[source.workflowId].label }}
                </UBadge>
              </div>
              <p class="mt-0.5 truncate text-[11px] text-muted">
                {{ source.description || t('workflow.list.no_description') }}
              </p>
              <p class="mt-0.5 truncate font-mono text-[9px] text-dimmed">
                {{ source.workflowId }}
              </p>
            </div>
            <div v-if="isColumnVisible('category')" class="min-w-0">
              <UBadge v-if="source.category" color="neutral" variant="soft" size="sm">{{
                source.category
              }}</UBadge>
              <span v-else class="text-[11px] text-dimmed">{{
                t('workflow.list.unclassified')
              }}</span>
            </div>
            <div
              v-if="isColumnVisible('tags')"
              class="flex min-w-0 items-center gap-1 overflow-hidden"
            >
              <UBadge
                v-for="tag in (source.tags ?? []).slice(0, 3)"
                :key="tag"
                color="neutral"
                variant="subtle"
                size="sm"
              >
                {{ tag }}
              </UBadge>
              <span v-if="!(source.tags ?? []).length" class="text-[11px] text-dimmed">{{
                t('workflow.list.no_tags')
              }}</span>
              <span v-else-if="(source.tags ?? []).length > 3" class="text-[10px] text-dimmed"
                >+{{ (source.tags ?? []).length - 3 }}</span
              >
            </div>
            <span v-if="isColumnVisible('nodes')" class="text-right font-mono text-xs text-muted">{{
              source.nodeCount
            }}</span>
            <span
              v-if="isColumnVisible('revision')"
              class="text-right font-mono text-xs text-muted"
              >{{ source.revision }}</span
            >
            <time
              v-if="isColumnVisible('createdAt')"
              :datetime="source.createdAt || undefined"
              class="text-xs text-muted"
              :title="source.createdAt ? formatExactDate(source.createdAt) : undefined"
            >
              {{ formatListDate(source.createdAt) }}
            </time>
            <time
              v-if="isColumnVisible('updatedAt')"
              :datetime="source.updatedAt || undefined"
              class="text-xs text-muted"
              :title="source.updatedAt ? formatExactDate(source.updatedAt) : undefined"
            >
              {{ formatListDate(source.updatedAt) }}
            </time>
            <div class="flex justify-end gap-1">
              <UButton
                icon="i-tabler-player-play"
                color="neutral"
                variant="ghost"
                size="sm"
                :aria-label="t('workflow.action.run_named', { name: source.name })"
                :loading="runStartingId === source.workflowId"
                :disabled="Boolean(runStartingId) || deleting"
                @click="runWorkflow(source.workflowId)"
              />
              <UButton
                icon="i-tabler-schema"
                size="sm"
                :aria-label="t('workflow.action.edit_named', { name: source.name })"
                @click="router.push(`/workflows/${source.workflowId}/edit`)"
              />
              <UDropdownMenu :items="rowMenuItems(source)">
                <UButton
                  icon="i-tabler-dots"
                  color="neutral"
                  variant="ghost"
                  size="sm"
                  :aria-label="t('workflow.list.row_actions', { name: source.name })"
                />
              </UDropdownMenu>
            </div>
          </article>
        </div>
      </div>

      <footer
        v-if="!loading && total > 0"
        class="flex min-h-14 shrink-0 items-center gap-4 rounded-b-lg border border-default bg-elevated/15 px-3"
      >
        <p class="mr-auto text-xs text-dimmed">
          {{ t('workflow.list.result_range', { start: resultStart, end: resultEnd, total }) }}
        </p>
        <UPagination
          :page="page"
          :total="total"
          :items-per-page="pageSize"
          :sibling-count="1"
          show-edges
          @update:page="goToPage"
        />
        <span class="text-xs text-dimmed">{{ t('workflow.list.per_page') }}</span>
        <AdaptiveSelect
          v-model="pageSize"
          :items="pageSizeItems"
          class="w-24"
          width-mode="fixed"
          @update:model-value="queryChanged"
        />
      </footer>
    </main>

    <BaseModal
      v-model:open="batchEditing"
      :title="t('workflow.list.batch_edit_title', { n: selectedRows.length })"
      icon="i-tabler-category-plus"
      size="lg"
      :dismissible="!batchBusy"
    >
      <div class="space-y-5">
        <p class="text-sm text-muted">{{ t('batchMetadata.description') }}</p>
        <UFormField :label="t('common.category')">
          <div class="flex items-center gap-2">
            <AdaptiveSelect
              v-model="batchDraft.categoryMode"
              :items="categoryModeItems"
              class="w-36 shrink-0"
              width-mode="fixed"
            />
            <UInputMenu
              v-if="batchDraft.categoryMode === 'set'"
              v-model="batchDraft.category"
              :items="batchCategoryOptions"
              :create-item="'always'"
              :placeholder="t('workflow.list.category_placeholder')"
              class="min-w-0 flex-1"
              @create="createBatchCategory"
            />
            <span v-else class="text-xs text-dimmed">{{ categoryModeHint }}</span>
          </div>
        </UFormField>
        <UFormField :label="t('common.tags')">
          <div class="flex items-start gap-2">
            <AdaptiveSelect
              v-model="batchDraft.tagMode"
              :items="tagModeItems"
              class="w-36 shrink-0"
              width-mode="fixed"
            />
            <UInputMenu
              v-if="tagModeNeedsValues"
              v-model="batchDraft.tags"
              :items="batchTagOptions"
              :create-item="'always'"
              multiple
              :placeholder="t('workflow.list.tags_placeholder')"
              class="min-w-0 flex-1"
              @create="createBatchTag"
            />
            <span v-else class="pt-2 text-xs text-dimmed">{{ tagModeHint }}</span>
          </div>
        </UFormField>
      </div>
      <template #footer>
        <UButton
          color="neutral"
          variant="ghost"
          :disabled="batchBusy"
          @click="batchEditing = false"
        >
          {{ t('common.cancel') }}
        </UButton>
        <UButton
          icon="i-tabler-check"
          :label="t('batchMetadata.apply')"
          :loading="batchBusy"
          :disabled="!batchDraftValid"
          @click="saveBatchMetadata"
        />
      </template>
    </BaseModal>

    <BaseModal
      v-model:open="metadataModalOpen"
      :title="t(editingSource ? 'workflow.list.edit_metadata_title' : 'workflow.list.create_title')"
      :icon="editingSource ? 'i-tabler-edit' : 'i-tabler-plus'"
      size="2xl"
      :dismissible="!metadataBusy"
    >
      <form class="grid gap-4" @submit.prevent="saveMetadata">
        <UFormField :label="t('workflow.list.name')" required>
          <UInput
            v-model="metadataDraft.name"
            data-testid="workflow-create-name"
            :placeholder="t('workflow.list.name_placeholder')"
            autofocus
          />
        </UFormField>
        <UFormField v-if="!editingSource" :label="t('workflow.list.template_label')">
          <AdaptiveSelect v-model="metadataDraft.template" :items="templateItems" :max-width="32" />
        </UFormField>
        <UFormField :label="t('workflow.list.description_label')">
          <UTextarea
            v-model="metadataDraft.description"
            :rows="3"
            :placeholder="t('workflow.list.description_placeholder')"
          />
        </UFormField>
        <div class="grid gap-4 sm:grid-cols-2">
          <UFormField :label="t('workflow.list.category')">
            <UInputMenu
              v-model="metadataDraft.category"
              :items="metadataCategoryOptions"
              :create-item="'always'"
              :placeholder="t('workflow.list.category_placeholder')"
              @create="createMetadataCategory"
            />
          </UFormField>
          <UFormField :label="t('workflow.list.tags')">
            <UInputMenu
              v-model="metadataDraft.tags"
              :items="metadataTagOptions"
              :create-item="'always'"
              multiple
              :placeholder="t('workflow.list.tags_placeholder')"
              @create="createMetadataTag"
            />
          </UFormField>
        </div>
        <p v-if="metadataFailure" class="text-sm text-error" role="alert">{{ metadataFailure }}</p>
      </form>
      <template #footer>
        <UButton
          color="neutral"
          variant="ghost"
          :disabled="metadataBusy"
          @click="metadataModalOpen = false"
        >
          {{ t('common.cancel') }}
        </UButton>
        <UButton
          data-testid="workflow-create-submit"
          icon="i-tabler-check"
          :label="t(editingSource ? 'common.save' : 'workflow.list.create')"
          :loading="metadataBusy"
          :disabled="!metadataDraft.name.trim()"
          @click="saveMetadata"
        />
      </template>
    </BaseModal>

    <BaseModal
      v-model:open="recoveryRepairOpen"
      :title="t('workflow.list.recovery_repair_title')"
      icon="i-tabler-first-aid-kit"
      icon-color="warning"
      size="3xl"
      :dismissible="!recoveryBusyId"
    >
      <div class="space-y-3">
        <p class="text-sm text-muted">{{ t('workflow.list.recovery_repair_description') }}</p>
        <p v-if="activeRecovery" class="text-xs text-dimmed">{{ activeRecovery.originalName }}</p>
        <UTextarea
          v-model="recoveryDraft"
          :rows="18"
          autoresize
          class="w-full font-mono text-xs"
          :disabled="Boolean(recoveryBusyId)"
          :aria-label="t('workflow.list.recovery_source_json')"
        />
        <p v-if="recoveryFailure" class="text-sm text-error" role="alert">{{ recoveryFailure }}</p>
      </div>
      <template #footer>
        <UButton
          color="neutral"
          variant="ghost"
          :disabled="Boolean(recoveryBusyId)"
          @click="recoveryRepairOpen = false"
        >
          {{ t('common.cancel') }}
        </UButton>
        <UButton
          color="warning"
          icon="i-tabler-tool"
          :label="t('workflow.list.recovery_validate_repair')"
          :loading="Boolean(recoveryBusyId)"
          :disabled="!activeRecovery || !recoveryDraft.trim()"
          @click="repairRecovery"
        />
      </template>
    </BaseModal>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from '@nuxt/ui/composables'
import { useI18n } from 'vue-i18n'
import {
  workflowTransport,
  type BundleInfoView,
  type DeleteSourcePreview,
  type SourceRecoveryView,
  type SourceView,
} from '@/app/transport/workflow'
import { useConfirm } from '@/composables/useConfirm'
import { useAutoDismissFeedback } from '@/composables/useAutoDismissFeedback'
import { errorMessage } from '@/lib/invoke'
import {
  applyBatchMetadata,
  createBatchMetadataDraft,
  hasBatchMetadataChange,
  uniqueMetadataValues,
} from '@/lib/batchMetadata'
import BaseModal from '@/components/common/BaseModal.vue'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LibrarySelectionToolbar from '@/components/library/LibrarySelectionToolbar.vue'

defineOptions({ name: 'WorkflowsView' })

type SelectedSource = Pick<
  SourceView,
  'workflowId' | 'name' | 'revision' | 'sourceHash' | 'category' | 'tags'
>
type WorkflowColumn = 'category' | 'tags' | 'nodes' | 'revision' | 'createdAt' | 'updatedAt'
type DateRange = 'all' | 'today' | '7d' | '30d' | '90d'
type Feedback = { tone: 'success' | 'warning' | 'error'; message: string; details: string[] }

const defaultColumns: WorkflowColumn[] = ['category', 'tags', 'nodes', 'createdAt', 'updatedAt']
const allCategories = '__all__'
const router = useRouter()
const toast = useToast()
const { t, locale } = useI18n()
const { confirm } = useConfirm()
const sources = ref<SourceView[]>([])
const recoveries = ref<SourceRecoveryView[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const sort = ref('updated_desc')
const searchInput = ref('')
const search = ref('')
const categoryFilter = ref(allCategories)
const tagFilters = ref<string[]>([])
const createdRange = ref<DateRange>('all')
const updatedRange = ref<DateRange>('all')
const categories = ref<Array<{ value: string; count: number }>>([])
const tags = ref<Array<{ value: string; count: number }>>([])
const visibleColumns = ref<WorkflowColumn[]>(loadColumns())
const selected = ref<Record<string, SelectedSource>>({})
const loading = ref(true)
const deleting = ref(false)
const importing = ref(false)
const exportingId = ref('')
const replacingId = ref('')
const batchExporting = ref(false)
const batchEditing = ref(false)
const batchBusy = ref(false)
const batchDraft = reactive(createBatchMetadataDraft())
const failure = ref('')
const recoveryRepairOpen = ref(false)
const recoveryDraft = ref('')
const recoveryFailure = ref('')
const recoveryBusyId = ref('')
const activeRecovery = ref<SourceRecoveryView | null>(null)
const deleteFeedback = ref<Feedback | null>(null)
const portabilityFeedback = ref<Feedback | null>(null)
useAutoDismissFeedback(deleteFeedback)
useAutoDismissFeedback(portabilityFeedback)
const runStartingId = ref('')
const runFeedbackById = reactive<
  Record<string, { tone: 'success' | 'warning'; label: string; detail: string }>
>({})
const metadataModalOpen = ref(false)
const metadataBusy = ref(false)
const metadataFailure = ref('')
const editingSource = ref<SourceView | null>(null)
const createdCategories = ref<string[]>([])
const createdTags = ref<string[]>([])
const metadataDraft = reactive({
  name: '',
  description: '',
  category: '',
  tags: [] as string[],
  template: 'generic' as 'generic' | 'windows' | 'android' | 'browser' | 'cross-target',
})

const selectedRows = computed(() => Object.values(selected.value))
const portabilityBusy = computed(
  () =>
    importing.value ||
    Boolean(exportingId.value) ||
    Boolean(replacingId.value) ||
    batchExporting.value ||
    batchBusy.value,
)
const allCurrentPageSelected = computed(
  () =>
    sources.value.length > 0 && sources.value.every((source) => selected.value[source.workflowId]),
)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))
const resultStart = computed(() => (total.value ? (page.value - 1) * pageSize.value + 1 : 0))
const resultEnd = computed(() => Math.min(page.value * pageSize.value, total.value))
const hasFilters = computed(() =>
  Boolean(
    search.value ||
    categoryFilter.value !== allCategories ||
    tagFilters.value.length ||
    createdRange.value !== 'all' ||
    updatedRange.value !== 'all',
  ),
)
const activeFeedback = computed(() => portabilityFeedback.value ?? deleteFeedback.value)
const feedbackClass = computed(() => {
  const tone = activeFeedback.value?.tone
  if (tone === 'error') return 'border-error/30 bg-error/10 text-error'
  if (tone === 'warning') return 'border-warning/30 bg-warning/10 text-warning'
  return 'border-success/30 bg-success/10 text-success'
})
const categoryFilterItems = computed(() => [
  { label: t('workflow.list.all_categories'), value: allCategories },
  ...categories.value.map((item) => ({
    label: `${item.value} (${item.count})`,
    value: item.value,
  })),
])
const tagOptions = computed(() => tags.value.map((item) => item.value))
const createdRangeItems = computed(() => dateRangeItems('created'))
const updatedRangeItems = computed(() => dateRangeItems('updated'))
const metadataCategoryOptions = computed(() =>
  uniqueStrings([
    ...categories.value.map((item) => item.value),
    ...createdCategories.value,
    metadataDraft.category,
  ]),
)
const metadataTagOptions = computed(() =>
  uniqueStrings([
    ...tags.value.map((item) => item.value),
    ...createdTags.value,
    ...metadataDraft.tags,
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
const sortItems = computed(() => [
  { label: t('workflow.list.sort_name_asc'), value: 'name_asc' },
  { label: t('workflow.list.sort_name_desc'), value: 'name_desc' },
  { label: t('workflow.list.sort_nodes_desc'), value: 'nodes_desc' },
  { label: t('workflow.list.sort_revision_desc'), value: 'revision_desc' },
  { label: t('workflow.list.sort_created_desc'), value: 'created_desc' },
  { label: t('workflow.list.sort_updated_desc'), value: 'updated_desc' },
])
const pageSizeItems = [
  { label: '20', value: 20 },
  { label: '50', value: 50 },
  { label: '100', value: 100 },
]
const templateItems = computed(() => [
  { label: t('workflow.list.template_generic'), value: 'generic' },
  { label: t('workflow.list.template_windows'), value: 'windows' },
  { label: t('workflow.list.template_android'), value: 'android' },
  { label: t('workflow.list.template_browser'), value: 'browser' },
  { label: t('workflow.list.template_cross_target'), value: 'cross-target' },
])
const columnOptions = computed<Array<{ key: WorkflowColumn; label: string }>>(() => [
  { key: 'category', label: t('workflow.list.category') },
  { key: 'tags', label: t('workflow.list.tags') },
  { key: 'nodes', label: t('workflow.list.nodes') },
  { key: 'revision', label: t('workflow.list.revision') },
  { key: 'createdAt', label: t('workflow.list.created_at') },
  { key: 'updatedAt', label: t('workflow.list.updated_at') },
])
const visibleColumnSet = computed(() => new Set(visibleColumns.value))
const columnMenuItems = computed(() => [
  columnOptions.value.map((column) => ({
    label: column.label,
    type: 'checkbox' as const,
    checked: visibleColumnSet.value.has(column.key),
    onUpdateChecked: (checked: boolean) => setColumnVisible(column.key, checked),
  })),
  [
    {
      label: t('workflow.list.reset_columns'),
      icon: 'i-tabler-restore',
      onSelect: () => {
        visibleColumns.value = [...defaultColumns]
      },
    },
  ],
])
const workflowGridTemplate = computed(() => {
  const columns = ['2rem', 'minmax(18rem, 2fr)']
  if (isColumnVisible('category')) columns.push('minmax(8rem, 0.8fr)')
  if (isColumnVisible('tags')) columns.push('minmax(12rem, 1.2fr)')
  if (isColumnVisible('nodes')) columns.push('5rem')
  if (isColumnVisible('revision')) columns.push('5rem')
  if (isColumnVisible('createdAt')) columns.push('8.5rem')
  if (isColumnVisible('updatedAt')) columns.push('8.5rem')
  columns.push('8.5rem')
  return columns.join(' ')
})
const libraryMenuItems = computed(() => [
  [
    {
      label: t('workflow.list.import_source'),
      icon: 'i-tabler-file-import',
      disabled: portabilityBusy.value,
      onSelect: () => void importSourceBundle(),
    },
    { label: t('common.refresh'), icon: 'i-tabler-refresh', onSelect: () => void load() },
  ],
])

watch(
  visibleColumns,
  (value) => localStorage.setItem('yotta.workflow.columns', JSON.stringify(value)),
  { deep: true },
)
onMounted(load)

async function load(): Promise<void> {
  loading.value = true
  failure.value = ''
  try {
    const [result, isolated] = await Promise.all([
      workflowTransport.querySources({
        search: search.value,
        category: categoryFilter.value === allCategories ? '' : categoryFilter.value,
        tags: tagFilters.value,
        createdSince: rangeStart(createdRange.value),
        updatedSince: rangeStart(updatedRange.value),
        sort: sort.value,
        page: page.value,
        pageSize: pageSize.value,
      }),
      workflowTransport.listSourceRecoveries(),
    ])
    sources.value = result.items
    recoveries.value = isolated
    total.value = result.total
    categories.value = result.categories ?? []
    tags.value = result.tags ?? []
    if (page.value > pageCount.value) {
      page.value = pageCount.value
      await load()
    }
  } catch (error) {
    failure.value = errorText(error)
  } finally {
    loading.value = false
  }
}

async function queryChanged(): Promise<void> {
  page.value = 1
  await load()
}

async function applySearch(): Promise<void> {
  search.value = searchInput.value.trim()
  await queryChanged()
}

async function resetFilters(): Promise<void> {
  searchInput.value = ''
  search.value = ''
  categoryFilter.value = allCategories
  tagFilters.value = []
  createdRange.value = 'all'
  updatedRange.value = 'all'
  await queryChanged()
}

async function goToPage(next: number): Promise<void> {
  if (next < 1 || next > pageCount.value || next === page.value) return
  page.value = next
  await load()
}

function toggleSource(source: SourceView, checked: boolean): void {
  const next = { ...selected.value }
  if (checked) next[source.workflowId] = source
  else delete next[source.workflowId]
  selected.value = next
}

function toggleCurrentPage(checked: boolean): void {
  const next = { ...selected.value }
  for (const source of sources.value) {
    if (checked) next[source.workflowId] = source
    else delete next[source.workflowId]
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

async function saveBatchMetadata(): Promise<void> {
  if (!selectedRows.value.length || !batchDraftValid.value || batchBusy.value) return
  batchBusy.value = true
  portabilityFeedback.value = null
  try {
    const results = await workflowTransport.batchUpdateSourceMetadata(
      selectedRows.value.map((source) => {
        const metadata = applyBatchMetadata(
          { category: source.category ?? '', tags: source.tags ?? [] },
          batchDraft,
        )
        return {
          workflowId: source.workflowId,
          baseRevision: source.revision,
          category: metadata.category,
          tags: metadata.tags,
        }
      }),
    )
    const failed = results.filter((result) => !result.updated)
    retainFailedWorkflowSelection(failed.map((result) => result.workflowId))
    portabilityFeedback.value = {
      tone: failed.length ? 'warning' : 'success',
      message: t('workflow.list.batch_update_result', {
        updated: results.length - failed.length,
        failed: failed.length,
      }),
      details: failed.map((result) => `${selectedName(result.workflowId)}: ${result.error ?? ''}`),
    }
    batchEditing.value = false
    await load()
  } catch (error) {
    portabilityFeedback.value = { tone: 'error', message: errorText(error), details: [] }
  } finally {
    batchBusy.value = false
  }
}

function retainFailedWorkflowSelection(workflowIDs: string[]): void {
  const failed = new Set(workflowIDs)
  selected.value = Object.fromEntries(
    selectedRows.value
      .filter((source) => failed.has(source.workflowId))
      .map((source) => [source.workflowId, source]),
  )
}

function isColumnVisible(key: WorkflowColumn): boolean {
  return visibleColumnSet.value.has(key)
}

function setColumnVisible(key: WorkflowColumn, checked: boolean): void {
  const current = new Set(visibleColumns.value)
  if (checked) current.add(key)
  else current.delete(key)
  visibleColumns.value = columnOptions.value
    .map((item) => item.key)
    .filter((item) => current.has(item))
}

function loadColumns(): WorkflowColumn[] {
  try {
    const raw = JSON.parse(localStorage.getItem('yotta.workflow.columns') ?? '[]') as unknown
    if (!Array.isArray(raw)) return [...defaultColumns]
    const allowed = new Set<WorkflowColumn>([
      'category',
      'tags',
      'nodes',
      'revision',
      'createdAt',
      'updatedAt',
    ])
    const values = raw.filter(
      (value): value is WorkflowColumn =>
        typeof value === 'string' && allowed.has(value as WorkflowColumn),
    )
    return values.length ? values : [...defaultColumns]
  } catch {
    return [...defaultColumns]
  }
}

function openCreateModal(): void {
  editingSource.value = null
  metadataDraft.name = ''
  metadataDraft.description = ''
  metadataDraft.category = ''
  metadataDraft.tags = []
  metadataDraft.template = 'generic'
  metadataFailure.value = ''
  metadataModalOpen.value = true
}

function openMetadataEditor(source: SourceView): void {
  editingSource.value = source
  metadataDraft.name = source.name
  metadataDraft.description = source.description ?? ''
  metadataDraft.category = source.category ?? ''
  metadataDraft.tags = [...(source.tags ?? [])]
  metadataDraft.template = 'generic'
  metadataFailure.value = ''
  metadataModalOpen.value = true
}

async function saveMetadata(): Promise<void> {
  if (!metadataDraft.name.trim() || metadataBusy.value) return
  metadataBusy.value = true
  metadataFailure.value = ''
  try {
    const request = {
      name: metadataDraft.name.trim(),
      description: metadataDraft.description.trim(),
      category: metadataDraft.category.trim(),
      tags: uniqueStrings(metadataDraft.tags),
    }
    const editing = editingSource.value
    if (editing) {
      await workflowTransport.updateSourceMetadata(editing.workflowId, editing.revision, request)
      metadataModalOpen.value = false
      await load()
      return
    }
    const created = await workflowTransport.createSourceWithMetadata(request)
    metadataModalOpen.value = false
    const template = metadataDraft.template
    await router.push({
      path: `/workflows/${created.workflowId}/edit`,
      query: template === 'generic' ? {} : { template },
    })
  } catch (error) {
    metadataFailure.value = errorText(error)
  } finally {
    metadataBusy.value = false
  }
}

function createMetadataCategory(value: string): void {
  const category = value.trim()
  if (!category) return
  createdCategories.value = uniqueStrings([...createdCategories.value, category])
  metadataDraft.category = category
}

function createMetadataTag(value: string): void {
  const tag = value.trim()
  if (!tag) return
  createdTags.value = uniqueStrings([...createdTags.value, tag])
  metadataDraft.tags = uniqueStrings([...metadataDraft.tags, tag])
}

function createBatchCategory(value: string): void {
  const category = value.trim()
  if (!category) return
  createdCategories.value = uniqueStrings([...createdCategories.value, category])
  batchDraft.category = category
}

function createBatchTag(value: string): void {
  const tag = value.trim()
  if (!tag) return
  createdTags.value = uniqueStrings([...createdTags.value, tag])
  batchDraft.tags = uniqueMetadataValues([...batchDraft.tags, tag])
}

function dateRangeItems(kind: 'created' | 'updated') {
  const prefix = kind === 'created' ? 'created' : 'updated'
  return [
    { label: t(`workflow.list.${prefix}_any`), value: 'all' },
    { label: t(`workflow.list.${prefix}_today`), value: 'today' },
    { label: t(`workflow.list.${prefix}_days`, { n: 7 }), value: '7d' },
    { label: t(`workflow.list.${prefix}_days`, { n: 30 }), value: '30d' },
    { label: t(`workflow.list.${prefix}_days`, { n: 90 }), value: '90d' },
  ]
}

function rangeStart(range: DateRange): string {
  if (range === 'all') return ''
  const start = new Date()
  if (range === 'today') {
    start.setHours(0, 0, 0, 0)
  } else {
    start.setDate(start.getDate() - Number.parseInt(range, 10))
  }
  return start.toISOString()
}

function formatListDate(value?: string): string {
  if (!value) return '—'
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return '—'
  return new Intl.DateTimeFormat(locale.value, { dateStyle: 'medium' }).format(parsed)
}

function formatExactDate(value: string): string {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return value
  return new Intl.DateTimeFormat(locale.value, {
    dateStyle: 'long',
    timeStyle: 'medium',
  }).format(parsed)
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

function rowMenuItems(source: SourceView) {
  return [
    [
      {
        label: t('workflow.list.edit_metadata'),
        icon: 'i-tabler-edit',
        onSelect: () => openMetadataEditor(source),
      },
      {
        label: t('workflow.list.export_source'),
        icon: 'i-tabler-file-export',
        disabled: portabilityBusy.value,
        onSelect: () => void exportSource(source),
      },
      {
        label: t('workflow.list.replace_source'),
        icon: 'i-tabler-file-arrow-left',
        disabled: portabilityBusy.value,
        onSelect: () => void replaceSource(source),
      },
    ],
    [
      {
        label: t('common.delete'),
        icon: 'i-tabler-trash',
        color: 'error' as const,
        onSelect: () => void requestDelete([source]),
      },
    ],
  ]
}

function openRecoveryRepair(recovery: SourceRecoveryView): void {
  activeRecovery.value = recovery
  recoveryDraft.value = recovery.sourceJson
  recoveryFailure.value = ''
  recoveryRepairOpen.value = true
}

async function repairRecovery(): Promise<void> {
  const recovery = activeRecovery.value
  if (!recovery || recoveryBusyId.value || !recoveryDraft.value.trim()) return
  recoveryBusyId.value = recovery.recoveryId
  recoveryFailure.value = ''
  try {
    await workflowTransport.repairSourceRecovery(recovery.recoveryId, recoveryDraft.value)
    recoveryRepairOpen.value = false
    activeRecovery.value = null
    recoveryDraft.value = ''
    await load()
  } catch (error) {
    recoveryFailure.value = errorText(error)
  } finally {
    recoveryBusyId.value = ''
  }
}

async function deleteRecovery(recovery: SourceRecoveryView): Promise<void> {
  if (recoveryBusyId.value) return
  const accepted = await confirm({
    title: t('workflow.list.recovery_delete_title', { name: recovery.originalName }),
    description: t('workflow.list.recovery_delete_description'),
    confirmText: t('common.delete'),
    cancelText: t('common.cancel'),
    color: 'error',
  })
  if (accepted !== true) return
  recoveryBusyId.value = recovery.recoveryId
  try {
    await workflowTransport.deleteSourceRecovery(recovery.recoveryId)
    await load()
  } catch (error) {
    failure.value = errorText(error)
  } finally {
    recoveryBusyId.value = ''
  }
}

async function importSourceBundle(): Promise<void> {
  if (portabilityBusy.value) return
  portabilityFeedback.value = null
  importing.value = true
  try {
    const path = await workflowTransport.chooseSourceBundle()
    if (!path) return
    const info = await workflowTransport.inspectSourceBundle(path)
    const accepted = await confirm({
      title: t('workflow.list.import_title', { name: info.name }),
      description: bundleDescription(info),
      confirmText: t('workflow.list.import_source'),
      cancelText: t('common.cancel'),
    })
    if (accepted !== true) return
    await workflowTransport.importSourceBundle(path)
    await load()
  } catch (error) {
    portabilityFeedback.value = { tone: 'error', message: errorText(error), details: [] }
  } finally {
    importing.value = false
  }
}

async function exportSource(source: SelectedSource): Promise<void> {
  if (portabilityBusy.value) return
  portabilityFeedback.value = null
  exportingId.value = source.workflowId
  try {
    const destination = await workflowTransport.chooseSourceBundleDestination(
      `${source.workflowId}.yotta-workflow`,
    )
    if (!destination) return
    const result = await workflowTransport.exportSourceBundle(source.workflowId, destination)
    portabilityFeedback.value = {
      tone: 'success',
      message: t('workflow.list.export_result', { n: 1 }),
      details: result.path ? [result.path] : [],
    }
  } catch (error) {
    portabilityFeedback.value = { tone: 'error', message: errorText(error), details: [] }
  } finally {
    exportingId.value = ''
  }
}

async function exportSelected(): Promise<void> {
  if (!selectedRows.value.length || portabilityBusy.value) return
  portabilityFeedback.value = null
  batchExporting.value = true
  try {
    const directory = await workflowTransport.chooseSourceBundleDirectory()
    if (!directory) return
    const results = await workflowTransport.exportSourceBundles(
      selectedRows.value.map((source) => source.workflowId),
      directory,
    )
    const failed = results.filter((result) => !result.exported)
    const exported = results.filter((result) => result.exported)
    portabilityFeedback.value = {
      tone: failed.length ? 'warning' : 'success',
      message: t('workflow.list.export_batch_result', {
        exported: exported.length,
        failed: failed.length,
      }),
      details: failed.map((result) => `${selectedName(result.workflowId)}: ${result.error ?? ''}`),
    }
  } catch (error) {
    portabilityFeedback.value = { tone: 'error', message: errorText(error), details: [] }
  } finally {
    batchExporting.value = false
  }
}

async function replaceSource(source: SelectedSource): Promise<void> {
  if (portabilityBusy.value) return
  portabilityFeedback.value = null
  replacingId.value = source.workflowId
  try {
    const path = await workflowTransport.chooseSourceBundle()
    if (!path) return
    const info = await workflowTransport.inspectSourceBundle(path)
    const accepted = await confirm({
      title: t('workflow.list.replace_title', { name: source.name }),
      description: `${bundleDescription(info)}\n${t('workflow.list.replace_description')}`,
      confirmText: t('workflow.list.replace_source'),
      cancelText: t('common.cancel'),
      color: 'warning',
    })
    if (accepted !== true) return
    await workflowTransport.replaceSourceFromBundle(
      path,
      source.workflowId,
      source.revision,
      source.sourceHash,
    )
    await load()
  } catch (error) {
    portabilityFeedback.value = { tone: 'error', message: errorText(error), details: [] }
  } finally {
    replacingId.value = ''
  }
}

function bundleDescription(info: BundleInfoView): string {
  return t('workflow.list.bundle_description', {
    name: info.name,
    revision: info.revision,
    blobs: info.blobCount,
    bytes: info.blobBytes,
  })
}

function selectedName(workflowId: string): string {
  return selected.value[workflowId]?.name ?? workflowId
}

async function requestDelete(rows: SelectedSource[]): Promise<void> {
  if (!rows.length || deleting.value) return
  deleting.value = true
  deleteFeedback.value = null
  try {
    const previews = await workflowTransport.previewDeleteSources(rows.map((row) => row.workflowId))
    const blocked = previews.filter((preview) => preview.references.length > 0)
    const deletableIds = new Set(
      previews
        .filter((preview) => preview.references.length === 0)
        .map((preview) => preview.workflowId),
    )
    const details = blocked.flatMap(referenceDetails)
    if (deletableIds.size === 0) {
      deleteFeedback.value = {
        tone: 'warning',
        message: t('workflow.list.delete_all_blocked'),
        details,
      }
      return
    }
    const accepted = await confirm({
      title: t('workflow.list.delete_title', { n: deletableIds.size }),
      description: [
        t('workflow.list.delete_description', {
          deletable: deletableIds.size,
          blocked: blocked.length,
        }),
        ...details,
      ].join('\n'),
      confirmText: t('common.delete'),
      cancelText: t('common.cancel'),
      color: 'error',
    })
    if (accepted !== true) return
    const requests = rows
      .filter((row) => deletableIds.has(row.workflowId))
      .map((row) => ({
        workflowId: row.workflowId,
        revision: row.revision,
        sourceHash: row.sourceHash,
      }))
    const results = await workflowTransport.deleteSources(requests)
    const deleted = results.filter((result) => result.deleted)
    const failed = results.filter((result) => !result.deleted)
    const next = { ...selected.value }
    for (const result of deleted) {
      delete next[result.workflowId]
      delete runFeedbackById[result.workflowId]
    }
    selected.value = next
    deleteFeedback.value =
      failed.length || blocked.length
        ? {
            tone: 'warning',
            message: t('workflow.list.delete_result', {
              deleted: deleted.length,
              failed: failed.length,
              blocked: blocked.length,
            }),
            details: [
              ...details,
              ...failed.map(
                (result) => `${sourceName(result.workflowId, previews)}: ${result.error ?? ''}`,
              ),
            ],
          }
        : null
    await load()
  } catch (error) {
    deleteFeedback.value = { tone: 'error', message: errorText(error), details: [] }
  } finally {
    deleting.value = false
  }
}

function referenceDetails(preview: DeleteSourcePreview): string[] {
  return preview.references.map(
    (reference) =>
      `${preview.name}: ${t(`workflow.list.reference_${reference.kind}`, { name: reference.label || reference.id })}`,
  )
}

function sourceName(workflowId: string, previews: DeleteSourcePreview[]): string {
  return previews.find((preview) => preview.workflowId === workflowId)?.name ?? workflowId
}

async function runWorkflow(workflowId: string): Promise<void> {
  if (runStartingId.value) return
  runStartingId.value = workflowId
  try {
    const started = await workflowTransport.startRun(workflowId)
    if (!started.run) {
      runFeedbackById[workflowId] = {
        tone: 'warning',
        label: t('workflow.toast.not_started'),
        detail: diagnosticText(started),
      }
      return
    }
    runFeedbackById[workflowId] = {
      tone: 'success',
      label: t('workflow.toast.queued'),
      detail: started.run.runId,
    }
  } catch (error) {
    delete runFeedbackById[workflowId]
    toast.add({
      title: t('workflow.toast.run_failed'),
      description: errorText(error),
      color: 'error',
    })
  } finally {
    runStartingId.value = ''
  }
}

function diagnosticText(value: { diagnostics: Array<{ code: string }> }): string {
  return (
    value.diagnostics.map((diagnostic) => diagnostic.code).join(', ') ||
    t('workflow.toast.no_run_created')
  )
}

function errorText(error: unknown): string {
  return errorMessage(error)
}
</script>
