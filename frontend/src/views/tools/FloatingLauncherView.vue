<template>
  <HudShell
    dense
    icon="i-tabler-rocket"
    :title="t('floatingLauncher.title')"
    :subtitle="t('floatingLauncher.subtitle')"
    :status="headerStatus"
    :close-title="t('floatingLauncher.hide')"
    @close="onHide"
  >
    <template #actions>
      <UButton
        size="xs"
        variant="ghost"
        :color="pinned ? 'primary' : 'neutral'"
        :icon="pinned ? 'i-tabler-pin-filled' : 'i-tabler-pin'"
        :title="pinned ? t('floatingLauncher.unpin') : t('floatingLauncher.pin')"
        :aria-label="pinned ? t('floatingLauncher.unpin') : t('floatingLauncher.pin')"
        @click="togglePin"
      />
    </template>

    <div ref="contentRef" class="launcher-content">
      <div v-if="resolution.groups.length" class="launcher-search">
        <UInput
          v-model="query"
          size="xs"
          icon="i-tabler-search"
          :placeholder="t('floatingLauncher.search_placeholder')"
          :aria-label="t('floatingLauncher.search_aria')"
          autocomplete="off"
          class="w-full"
        />
        <UKbd value="/" class="launcher-search__kbd" />
      </div>

      <div
        v-if="resolution.groups.length === 0"
        class="flex min-h-24 flex-1 flex-col items-center justify-center gap-2 px-4 py-6 text-center"
      >
        <span
          class="inline-flex size-9 items-center justify-center rounded-xl border border-default bg-elevated/30"
        >
          <UIcon name="i-tabler-layout-grid-add" class="size-4 text-dimmed" />
        </span>
        <p class="text-xs text-dimmed">{{ t('floatingLauncher.empty') }}</p>
        <UButton
          size="xs"
          color="primary"
          variant="soft"
          icon="i-tabler-settings"
          @click="openSettings"
        >
          {{ t('floatingLauncher.configure') }}
        </UButton>
      </div>

      <LauncherSurface
        v-else
        :groups="filteredGroups"
        :display="display"
        :size="size"
        :active-id="activeBlockId"
        :statuses="statuses"
        :empty-label="t('floatingLauncher.no_results')"
        :run-label="(name: string) => t('floatingLauncher.run', { name })"
        :cancel-label="(name: string) => t('floatingLauncher.cancel', { name })"
        :status-labels="statusLabels"
        :stale-label="t('floatingLauncher.stale_item')"
        @run="onRun"
        @select="selectItem"
      />

      <div v-if="runIssue" class="launcher-health launcher-health--error" role="status">
        <UIcon name="i-tabler-alert-circle" class="size-3.5 shrink-0" />
        <span>{{ runIssue }}</span>
      </div>

      <div v-if="resolution.staleBlocks.length" class="launcher-health">
        <UIcon name="i-tabler-alert-triangle" class="size-3.5" />
        <span>{{ t('floatingLauncher.stale_hint', { n: resolution.staleBlocks.length }) }}</span>
      </div>
    </div>

    <div
      class="absolute bottom-0 right-0 size-3.5 cursor-nwse-resize text-dimmed/70 hover:text-toned"
      style="--wails-draggable: no-drag"
      :title="t('floatingLauncher.resize')"
      @pointerdown="onGripDown"
    >
      <svg viewBox="0 0 10 10" class="size-full">
        <path d="M9 1v8H1" fill="none" stroke="currentColor" stroke-width="1" opacity="0.5" />
        <path d="M9 5v4H5" fill="none" stroke="currentColor" stroke-width="1.2" />
      </svg>
    </div>
  </HudShell>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@/composables/useAppToast'
import { Events } from '@wailsio/runtime'
import { backend } from '@/lib/backend'
import { errorMessage } from '@/lib/invoke'
import { runReadinessMessage, runStartOutcome } from '@/app/run/runReadiness'
import { pollTerminalRunStatus } from '@/app/run/followRun'
import { useSettingsStore } from '@/stores/settings'
import { onRunChanged, workflowTransport, type SourceView } from '@/app/transport/workflow'
import HudShell from '@/components/tools/HudShell.vue'
import LauncherSurface, {
  type LauncherCommandStatus,
} from '@/components/launcher/LauncherSurface.vue'
import {
  filterLauncherGroups,
  normalizeLauncherDisplay,
  normalizeLauncherSize,
  resolveLauncher,
  type LauncherDisplay,
  type LauncherSize,
} from '@/components/launcher/launcherModel'

const settingsStore = useSettingsStore()
const { t } = useI18n()
const toast = useToast()

const contentRef = ref<HTMLElement | null>(null)
const workflows = ref<SourceView[]>([])
const query = ref('')
const activeBlockId = ref('')
const requestedId = ref('')
const requestedRunId = ref('')
const feedback = ref<{ id: string; status: 'success' | 'error' | 'cancelled' } | null>(null)
const runIssue = ref('')
let feedbackTimer: ReturnType<typeof setTimeout> | undefined

const pinned = ref(true)
function togglePin() {
  pinned.value = !pinned.value
  void backend.tools.setLauncherAlwaysOnTop(pinned.value).catch(showLauncherError)
}

const display = computed<LauncherDisplay>(() =>
  normalizeLauncherDisplay(settingsStore.data?.ui.launcherDisplay),
)
const size = computed<LauncherSize>(() =>
  normalizeLauncherSize(settingsStore.data?.ui.launcherSize),
)
const resolution = computed(() =>
  resolveLauncher(settingsStore.data?.ui.launcherItems ?? [], workflows.value),
)
const filteredGroups = computed(() => filterLauncherGroups(resolution.value.groups, query.value))
const filteredItems = computed(() =>
  filteredGroups.value.flatMap((group) => group.items).filter((item) => !item.stale),
)
const headerStatus = computed(() => {
  const count = t('floatingLauncher.item_count', { n: resolution.value.items.length })
  return resolution.value.staleBlocks.length
    ? `${count} · ${t('floatingLauncher.stale_count', { n: resolution.value.staleBlocks.length })}`
    : count
})
const statusLabels = computed(() => ({
  running: t('floatingLauncher.running'),
  success: t('floatingLauncher.success'),
  error: t('floatingLauncher.failed'),
  cancelled: t('floatingLauncher.cancelled'),
}))
const statuses = computed<Record<string, LauncherCommandStatus>>(() => {
  const result: Record<string, LauncherCommandStatus> = {}
  if (feedback.value) result[feedback.value.id] = feedback.value.status
  if (requestedId.value) result[requestedId.value] = 'running'
  return result
})

watch(
  filteredItems,
  (items) => {
    if (!items.some((item) => item.id === activeBlockId.value)) {
      activeBlockId.value = items[0]?.id ?? ''
    }
  },
  { immediate: true },
)

function selectItem(id: string) {
  activeBlockId.value = id
}

function moveSelection(delta: number) {
  const items = filteredItems.value
  if (!items.length) return
  const current = items.findIndex((item) => item.id === activeBlockId.value)
  const next = current < 0 ? 0 : (current + delta + items.length) % items.length
  activeBlockId.value = items[next]?.id ?? ''
}

function activeWorkflowId(): string {
  return filteredItems.value.find((item) => item.id === activeBlockId.value)?.workflowId ?? ''
}

function settleRequest(status: 'success' | 'error' | 'cancelled', issue = '') {
  if (!requestedId.value) return
  feedback.value = { id: requestedId.value, status }
  runIssue.value = issue
  requestedId.value = ''
  requestedRunId.value = ''
  clearTimeout(feedbackTimer)
  feedbackTimer = setTimeout(() => {
    feedback.value = null
    runIssue.value = ''
  }, 10_000)
}

function settleTerminalStatus(status: string) {
  if (status === 'succeeded') {
    settleRequest('success')
    return true
  }
  if (status === 'cancelled' || status === 'interrupted') {
    settleRequest('cancelled')
    return true
  }
  if (status === 'failed') {
    settleRequest('error')
    return true
  }
  return false
}

async function onRun(id: string) {
  if (!id) return
  if (requestedId.value === id && requestedRunId.value) {
    try {
      const stopped = await workflowTransport.cancelRun(requestedRunId.value)
      settleTerminalStatus(stopped.status || 'cancelled')
    } catch (error) {
      settleRequest('error', errorMessage(error))
    }
    return
  }
  if (requestedId.value) return
  requestedId.value = id
  feedback.value = null
  runIssue.value = ''
  try {
    const started = await workflowTransport.startRun(id)
    const outcome = runStartOutcome(started)
    if (outcome.state !== 'started') {
      settleRequest('error', runReadinessMessage(outcome))
      return
    }
    requestedRunId.value = outcome.runId
    if (!started.run) return
    if (settleTerminalStatus(started.run.status)) return

    // A short Run can finish before StartRun returns and before requestedRunId is known.
    // Poll through a briefly stale timeline snapshot so the event/snapshot hand-off has no gap.
    const terminal = await pollTerminalRunStatus(
      async () => (await workflowTransport.getRunTimeline(outcome.runId)).status,
      () => requestedRunId.value !== outcome.runId,
    )
    if (terminal) settleTerminalStatus(terminal)
  } catch (error) {
    settleRequest('error', errorMessage(error))
  }
}

function onHide() {
  void backend.tools.hideLauncher().catch(showLauncherError)
}

async function openSettings(): Promise<void> {
  try {
    await backend.tools.openLauncherSettings()
    await backend.tools.hideLauncher()
  } catch (error) {
    toast.add({
      title: t('toast.operation_failed'),
      description: errorMessage(error),
      color: 'error',
    })
  }
}

function focusSearch() {
  document.querySelector<HTMLInputElement>('.launcher-search input')?.focus()
}

function onKeyDown(event: KeyboardEvent) {
  const target = event.target as HTMLElement | null
  const editing = target?.matches('input, textarea, [contenteditable="true"]') ?? false

  if (event.key === '/' && !editing && !event.ctrlKey && !event.metaKey && !event.altKey) {
    event.preventDefault()
    focusSearch()
    return
  }
  if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
    event.preventDefault()
    moveSelection(event.key === 'ArrowDown' ? 1 : -1)
    return
  }
  if (event.key === 'Enter' && activeBlockId.value) {
    event.preventDefault()
    void onRun(activeWorkflowId())
    return
  }
  if (event.key === 'Escape') {
    event.preventDefault()
    if (query.value) query.value = ''
    else onHide()
    return
  }
  if (!editing && event.key >= '1' && event.key <= '9') {
    const item = resolution.value.items[Number(event.key) - 1]
    if (!item) return
    event.preventDefault()
    void onRun(item.workflowId)
    return
  }
  if (!editing && event.key.length === 1 && !event.ctrlKey && !event.metaKey && !event.altKey) {
    event.preventDefault()
    query.value += event.key
    void nextTick(focusSearch)
  }
}

const MIN_W = 220
const MIN_H = 120
const CHROME_H = 35
function fitHeight() {
  const element = contentRef.value
  if (!element) return
  const height = Math.min(720, Math.max(MIN_H, Math.ceil(element.scrollHeight) + CHROME_H))
  void backend.tools
    .setLauncherSize(Math.max(MIN_W, Math.round(window.innerWidth)), height)
    .catch(() => undefined)
}
watch([filteredGroups, display, size], () => void nextTick(fitHeight))

async function refreshLauncherData() {
  const [, listed] = await Promise.all([settingsStore.load(), workflowTransport.listSources()])
  workflows.value = listed
  await nextTick()
  fitHeight()
}

let startX = 0
let startY = 0
let startW = 0
let startH = 0
function onGripMove(event: PointerEvent) {
  const width = Math.max(MIN_W, startW + (event.clientX - startX))
  const height = Math.max(MIN_H, startH + (event.clientY - startY))
  void backend.tools.setLauncherSize(Math.round(width), Math.round(height)).catch(() => undefined)
}

function showLauncherError(error: unknown): void {
  toast.add({
    title: t('toast.operation_failed'),
    description: errorMessage(error),
    color: 'error',
  })
}
function onGripUp() {
  window.removeEventListener('pointermove', onGripMove)
  window.removeEventListener('pointerup', onGripUp)
}
function onGripDown(event: PointerEvent) {
  event.preventDefault()
  startX = event.clientX
  startY = event.clientY
  startW = window.innerWidth
  startH = window.innerHeight
  window.addEventListener('pointermove', onGripMove)
  window.addEventListener('pointerup', onGripUp)
}

let offSettings: (() => void) | null = null
let offRun: (() => void) | null = null
onMounted(() => {
  void refreshLauncherData()
  offSettings = Events.On(
    'settings:changed',
    () => void refreshLauncherData(),
  ) as unknown as () => void
  offRun = onRunChanged((event) => {
    if (!requestedRunId.value || event.runId !== requestedRunId.value) return
    settleTerminalStatus(event.status)
  })
  window.addEventListener('keydown', onKeyDown)
})
onUnmounted(() => {
  offSettings?.()
  offRun?.()
  clearTimeout(feedbackTimer)
  window.removeEventListener('keydown', onKeyDown)
  window.removeEventListener('pointermove', onGripMove)
  window.removeEventListener('pointerup', onGripUp)
})
</script>

<style scoped>
.launcher-content {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  gap: 8px;
  overflow: auto;
  padding: 8px;
}

.launcher-search {
  position: relative;
  flex: none;
}

.launcher-search :deep(input) {
  height: 30px;
  padding-right: 30px;
  font-size: 11px;
}

.launcher-search__kbd {
  position: absolute;
  top: 6px;
  right: 7px;
  width: 18px;
  height: 18px;
  padding: 0;
  font-size: 9px;
  pointer-events: none;
  opacity: 0.62;
}

.launcher-health {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 7px;
  border-top: 1px solid var(--ui-border);
  color: var(--ui-warning);
  font-size: 10px;
  line-height: 14px;
}

.launcher-health--error {
  color: var(--ui-error);
}
</style>
