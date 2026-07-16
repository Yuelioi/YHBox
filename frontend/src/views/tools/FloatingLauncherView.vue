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
      </div>

      <LauncherSurface
        v-else
        :groups="filteredGroups"
        :display="display"
        :selected-id="selectedId"
        :statuses="statuses"
        :empty-label="t('floatingLauncher.no_results')"
        :run-label="(name: string) => t('floatingLauncher.run', { name })"
        :status-labels="statusLabels"
        :stale-label="t('floatingLauncher.stale_item')"
        @run="onRun"
        @select="selectItem"
      />

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
import { Events } from '@wailsio/runtime'
import { backend } from '@/lib/backend'
import { useSettingsStore } from '@/stores/settings'
import { onRunChanged, workflowTransport, type SourceView } from '@/app/transport/workflow31'
import HudShell from '@/components/tools/HudShell.vue'
import LauncherSurface, {
  type LauncherCommandStatus,
} from '@/components/launcher/LauncherSurface.vue'
import {
  filterLauncherGroups,
  normalizeLauncherDisplay,
  resolveLauncher,
  type LauncherDisplay,
} from '@/components/launcher/launcherModel'

const settingsStore = useSettingsStore()
const { t } = useI18n()

const contentRef = ref<HTMLElement | null>(null)
const workflows = ref<SourceView[]>([])
const query = ref('')
const selectedId = ref('')
const requestedId = ref('')
const requestedRunId = ref('')
const feedback = ref<{ id: string; status: 'success' | 'error' } | null>(null)
let feedbackTimer: ReturnType<typeof setTimeout> | undefined

const pinned = ref(true)
function togglePin() {
  pinned.value = !pinned.value
  void backend.tools.setLauncherAlwaysOnTop(pinned.value)
}

const display = computed<LauncherDisplay>(() =>
  normalizeLauncherDisplay(settingsStore.data?.ui.launcherDisplay),
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
    if (!items.some((item) => item.workflowId === selectedId.value)) {
      selectedId.value = items[0]?.workflowId ?? ''
    }
  },
  { immediate: true },
)

function selectItem(id: string) {
  selectedId.value = id
}

function moveSelection(delta: number) {
  const items = filteredItems.value
  if (!items.length) return
  const current = items.findIndex((item) => item.workflowId === selectedId.value)
  const next = current < 0 ? 0 : (current + delta + items.length) % items.length
  selectedId.value = items[next]?.workflowId ?? ''
}

function settleRequest(status: 'success' | 'error') {
  if (!requestedId.value) return
  feedback.value = { id: requestedId.value, status }
  requestedId.value = ''
  requestedRunId.value = ''
  clearTimeout(feedbackTimer)
  feedbackTimer = setTimeout(() => (feedback.value = null), 1800)
}

function settleTerminalStatus(status: string) {
  if (status === 'succeeded') {
    settleRequest('success')
    return true
  }
  if (status === 'failed' || status === 'cancelled' || status === 'interrupted') {
    settleRequest('error')
    return true
  }
  return false
}

async function onRun(id: string) {
  if (!id || requestedId.value) return
  requestedId.value = id
  feedback.value = null
  try {
    const started = await workflowTransport.startRun(id)
    if (!started.run) {
      settleRequest('error')
      return
    }
    requestedRunId.value = started.run.runId
    if (settleTerminalStatus(started.run.status)) return

    // A short Run can finish before StartRun returns and before requestedRunId is known.
    // Re-read the authoritative record once so the event/snapshot hand-off has no gap.
    const current = await workflowTransport.getRunTimeline(started.run.runId)
    settleTerminalStatus(current.status)
  } catch {
    settleRequest('error')
  }
}

function onHide() {
  void backend.tools.hideLauncher()
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
  if (event.key === 'Enter' && selectedId.value) {
    event.preventDefault()
    void onRun(selectedId.value)
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
  void backend.tools.setLauncherSize(Math.max(MIN_W, Math.round(window.innerWidth)), height)
}
watch([filteredGroups, display], () => void nextTick(fitHeight))

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
  void backend.tools.setLauncherSize(Math.round(width), Math.round(height))
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
</style>
