<template>
  <div class="flex min-h-full flex-col bg-default">
    <header
      class="flex flex-col gap-4 border-b border-default px-4 py-5 sm:flex-row sm:items-end sm:gap-6 sm:px-8 sm:py-6"
    >
      <div class="min-w-0 flex-1">
        <h1 class="text-xl font-semibold tracking-tight text-highlighted">
          {{ t('workflow.list.title') }}
        </h1>
        <p class="mt-1 max-w-2xl text-sm text-muted">
          {{ t('workflow.list.description') }}
        </p>
      </div>
      <form
        class="flex w-full flex-wrap items-end gap-2 sm:w-auto"
        @submit.prevent="createWorkflow"
      >
        <UFormField
          :label="t('workflow.list.new_workflow')"
          class="min-w-52 flex-1 sm:w-64 sm:flex-none"
        >
          <UInput
            v-model="newName"
            data-testid="workflow-create-name"
            :placeholder="t('workflow.list.name_placeholder')"
            class="w-full"
          />
        </UFormField>
        <UFormField :label="t('workflow.list.template_label')" class="min-w-40">
          <USelect v-model="newTemplate" :items="templateItems" class="w-full" />
        </UFormField>
        <UButton
          type="submit"
          data-testid="workflow-create-submit"
          :label="t('workflow.list.create')"
          icon="i-tabler-plus"
          :loading="creating"
          :disabled="!newName.trim()"
        />
      </form>
    </header>

    <main class="flex-1 px-4 py-5 sm:px-8 sm:py-6">
      <div class="mb-4 flex flex-col gap-3 lg:flex-row lg:items-end">
        <form class="min-w-0 flex-1" role="search" @submit.prevent="applySearch">
          <UFormField :label="t('workflow.list.search_label')">
            <div class="flex gap-2">
              <UInput
                v-model="searchInput"
                icon="i-tabler-search"
                :placeholder="t('workflow.list.search_placeholder')"
                class="min-w-0 flex-1"
              />
              <UButton type="submit" color="neutral" variant="soft">
                {{ t('workflow.list.search_action') }}
              </UButton>
            </div>
          </UFormField>
        </form>
        <UFormField :label="t('workflow.list.sort_label')" class="lg:w-52">
          <USelect
            v-model="sort"
            :items="sortItems"
            class="w-full"
            @update:model-value="queryChanged"
          />
        </UFormField>
        <UFormField :label="t('workflow.list.page_size_label')" class="lg:w-32">
          <USelect
            v-model="pageSize"
            :items="pageSizeItems"
            class="w-full"
            @update:model-value="queryChanged"
          />
        </UFormField>
        <UButton
          color="neutral"
          variant="ghost"
          icon="i-tabler-refresh"
          :aria-label="t('common.refresh')"
          :loading="loading"
          @click="load"
        />
      </div>

      <div
        v-if="selectedRows.length"
        class="mb-4 flex flex-col gap-3 rounded-lg border border-primary/30 bg-primary/5 px-4 py-3 sm:flex-row sm:items-center"
        role="status"
      >
        <div class="min-w-0 flex-1">
          <p class="text-sm font-medium text-default">
            {{ t('workflow.list.selected_count', { n: selectedRows.length }) }}
          </p>
          <p class="text-xs text-dimmed">{{ t('workflow.list.selection_scope_hint') }}</p>
        </div>
        <div class="flex flex-wrap gap-2">
          <UButton size="sm" color="neutral" variant="ghost" @click="clearSelection">
            {{ t('workflow.list.clear_selection') }}
          </UButton>
          <UButton
            size="sm"
            color="error"
            variant="soft"
            icon="i-tabler-trash"
            :loading="deleting"
            @click="requestDelete(selectedRows)"
          >
            {{ t('workflow.list.delete_selected') }}
          </UButton>
        </div>
      </div>

      <div
        v-if="deleteFeedback"
        class="mb-4 rounded-lg border px-4 py-3 text-sm"
        :class="
          deleteFeedback.tone === 'success'
            ? 'border-success/30 bg-success/10 text-success'
            : deleteFeedback.tone === 'warning'
              ? 'border-warning/30 bg-warning/10 text-warning'
              : 'border-error/30 bg-error/10 text-error'
        "
        :role="deleteFeedback.tone === 'error' ? 'alert' : 'status'"
      >
        <p>{{ deleteFeedback.message }}</p>
        <ul v-if="deleteFeedback.details.length" class="mt-2 list-disc space-y-1 pl-5 text-xs">
          <li v-for="detail in deleteFeedback.details" :key="detail">{{ detail }}</li>
        </ul>
      </div>

      <div v-if="loading" class="space-y-2" :aria-label="t('workflow.list.loading')">
        <USkeleton v-for="index in 4" :key="index" class="h-16 w-full rounded-lg" />
      </div>

      <div
        v-else-if="failure"
        class="flex items-center gap-3 rounded-lg border border-error/35 bg-error/10 px-4 py-3 text-sm text-error"
        role="alert"
      >
        <span class="min-w-0 flex-1">{{ failure }}</span>
        <UButton size="xs" color="error" variant="soft" @click="load">
          {{ t('common.retry') }}
        </UButton>
      </div>

      <div
        v-else-if="sources.length === 0"
        class="flex min-h-72 items-center justify-center rounded-lg border border-dashed border-default bg-elevated/20 px-8 text-center"
      >
        <div class="max-w-sm">
          <UIcon name="i-tabler-route" class="mx-auto mb-4 size-8 text-primary" />
          <h2 class="text-sm font-semibold text-highlighted">
            {{ t(search ? 'workflow.list.no_results_title' : 'workflow.list.empty_title') }}
          </h2>
          <p class="mt-2 text-xs leading-5 text-muted">
            {{
              t(search ? 'workflow.list.no_results_description' : 'workflow.list.empty_description')
            }}
          </p>
          <UButton
            v-if="search"
            class="mt-4"
            size="sm"
            color="neutral"
            variant="soft"
            @click="clearSearch"
          >
            {{ t('workflow.list.clear_search') }}
          </UButton>
        </div>
      </div>

      <div v-else class="overflow-hidden rounded-lg border border-default">
        <div
          class="hidden grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-4 bg-elevated/60 px-4 py-2 text-[11px] font-medium text-muted sm:grid"
        >
          <input
            type="checkbox"
            class="size-4 accent-primary"
            :checked="allCurrentPageSelected"
            :aria-label="t('workflow.list.select_page')"
            @change="toggleCurrentPage"
          />
          <span>{{ t('workflow.list.name') }}</span>
          <span class="text-right">{{ t('workflow.list.actions') }}</span>
        </div>
        <div class="divide-y divide-default">
          <article
            v-for="source in sources"
            :key="source.workflowId"
            class="flex flex-col gap-3 px-4 py-3 transition-colors hover:bg-elevated/35 sm:grid sm:grid-cols-[auto_minmax(0,1fr)_auto] sm:items-center sm:gap-4"
          >
            <input
              type="checkbox"
              class="size-4 accent-primary"
              :checked="Boolean(selected[source.workflowId])"
              :aria-label="t('workflow.list.select_named', { name: source.name })"
              @change="toggleSource(source, $event)"
            />
            <div class="min-w-0">
              <RouterLink
                :to="`/workflows/${source.workflowId}/edit`"
                :aria-label="t('workflow.action.edit_named', { name: source.name })"
                class="block truncate text-sm font-medium text-highlighted underline-offset-4 hover:text-primary hover:underline focus-visible:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
              >
                {{ source.name }}
              </RouterLink>
              <div
                class="mt-1 flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1 font-mono text-[10px] text-dimmed"
              >
                <span class="shrink-0"
                  >{{ t('workflow.list.revision') }} {{ source.revision }}</span
                >
                <span class="min-w-0 truncate">{{ source.sourceHash }}</span>
                <span class="min-w-0 truncate">{{ source.workflowId }}</span>
              </div>
              <div
                v-if="runFeedbackById[source.workflowId]"
                class="mt-2 flex min-w-0 items-center gap-2 text-[11px]"
                aria-live="polite"
              >
                <UBadge :color="runFeedbackById[source.workflowId].tone" variant="soft" size="xs">
                  {{ runFeedbackById[source.workflowId].label }}
                </UBadge>
                <span class="min-w-0 truncate font-mono text-dimmed">
                  {{ runFeedbackById[source.workflowId].detail }}
                </span>
              </div>
            </div>
            <div class="flex flex-wrap justify-end gap-2">
              <UButton
                :label="t('workflow.action.run')"
                :aria-label="t('workflow.action.run_named', { name: source.name })"
                icon="i-tabler-player-play"
                color="neutral"
                variant="ghost"
                size="xs"
                :loading="runStartingId === source.workflowId"
                :disabled="runStartingId !== '' || deleting"
                @click="runWorkflow(source.workflowId)"
              />
              <UButton
                :label="t('workflow.action.edit')"
                :aria-label="t('workflow.action.edit_named', { name: source.name })"
                icon="i-tabler-schema"
                size="xs"
                @click="router.push(`/workflows/${source.workflowId}/edit`)"
              />
              <UButton
                :label="t('common.delete')"
                :aria-label="t('workflow.action.delete_named', { name: source.name })"
                icon="i-tabler-trash"
                color="error"
                variant="ghost"
                size="xs"
                :disabled="deleting"
                @click="requestDelete([source])"
              />
            </div>
          </article>
        </div>
      </div>

      <footer
        v-if="!loading && total > 0"
        class="mt-4 flex flex-col gap-3 sm:flex-row sm:items-center"
      >
        <p class="min-w-0 flex-1 text-xs text-dimmed">
          {{ t('workflow.list.page_summary', { page, pages: pageCount, total }) }}
        </p>
        <div class="flex gap-2">
          <UButton
            size="sm"
            color="neutral"
            variant="soft"
            icon="i-tabler-chevron-left"
            :disabled="page <= 1"
            @click="goToPage(page - 1)"
          >
            {{ t('workflow.list.previous_page') }}
          </UButton>
          <UButton
            size="sm"
            color="neutral"
            variant="soft"
            trailing-icon="i-tabler-chevron-right"
            :disabled="page >= pageCount"
            @click="goToPage(page + 1)"
          >
            {{ t('workflow.list.next_page') }}
          </UButton>
        </div>
      </footer>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from '@nuxt/ui/composables'
import { useI18n } from 'vue-i18n'
import { useConfirm } from '@/composables/useConfirm'
import {
  workflowTransport,
  type DeleteSourcePreview,
  type SourceView,
} from '@/app/transport/workflow'

defineOptions({ name: 'WorkflowsView' })

type SelectedSource = Pick<SourceView, 'workflowId' | 'name' | 'revision' | 'sourceHash'>

const router = useRouter()
const toast = useToast()
const { t } = useI18n()
const { confirm } = useConfirm()
const sources = ref<SourceView[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const sort = ref('name_asc')
const searchInput = ref('')
const search = ref('')
const selected = ref<Record<string, SelectedSource>>({})
const loading = ref(true)
const creating = ref(false)
const deleting = ref(false)
const newName = ref('')
const newTemplate = ref<'generic' | 'windows' | 'android' | 'cross-target'>('generic')
const failure = ref('')
const deleteFeedback = ref<{
  tone: 'success' | 'warning' | 'error'
  message: string
  details: string[]
} | null>(null)
const runStartingId = ref('')
const runFeedbackById = reactive<
  Record<string, { tone: 'success' | 'warning'; label: string; detail: string }>
>({})

const selectedRows = computed(() => Object.values(selected.value))
const allCurrentPageSelected = computed(
  () =>
    sources.value.length > 0 && sources.value.every((source) => selected.value[source.workflowId]),
)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))
const sortItems = computed(() => [
  { label: t('workflow.list.sort_name_asc'), value: 'name_asc' },
  { label: t('workflow.list.sort_name_desc'), value: 'name_desc' },
  { label: t('workflow.list.sort_revision_desc'), value: 'revision_desc' },
])
const pageSizeItems = [
  { label: '10', value: 10 },
  { label: '20', value: 20 },
  { label: '50', value: 50 },
]
const templateItems = computed(() => [
  { label: t('workflow.list.template_generic'), value: 'generic' },
  { label: t('workflow.list.template_windows'), value: 'windows' },
  { label: t('workflow.list.template_android'), value: 'android' },
  { label: t('workflow.list.template_cross_target'), value: 'cross-target' },
])

onMounted(load)

async function load(): Promise<void> {
  loading.value = true
  failure.value = ''
  try {
    const result = await workflowTransport.querySources({
      search: search.value,
      sort: sort.value,
      page: page.value,
      pageSize: pageSize.value,
    })
    sources.value = result.items
    total.value = result.total
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

async function clearSearch(): Promise<void> {
  searchInput.value = ''
  search.value = ''
  await queryChanged()
}

async function goToPage(next: number): Promise<void> {
  if (next < 1 || next > pageCount.value || next === page.value) return
  page.value = next
  await load()
}

function toggleSource(source: SourceView, event: Event): void {
  const checked = (event.target as HTMLInputElement).checked
  const next = { ...selected.value }
  if (checked) {
    next[source.workflowId] = source
  } else {
    delete next[source.workflowId]
  }
  selected.value = next
}

function toggleCurrentPage(event: Event): void {
  const checked = (event.target as HTMLInputElement).checked
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
    deleteFeedback.value = {
      tone: failed.length || blocked.length ? 'warning' : 'success',
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

async function createWorkflow(): Promise<void> {
  const name = newName.value.trim()
  if (!name || creating.value) return
  creating.value = true
  try {
    const created = await workflowTransport.createSource(name)
    newName.value = ''
    const template = newTemplate.value
    await router.push({
      path: `/workflows/${created.workflowId}/edit`,
      query: template === 'generic' ? {} : { template },
    })
  } catch (error) {
    toast.add({
      title: t('workflow.toast.create_failed'),
      description: errorText(error),
      color: 'error',
    })
  } finally {
    creating.value = false
  }
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
  return error instanceof Error ? error.message : String(error)
}
</script>
