<template>
  <section
    class="flex h-full min-h-0 flex-col overflow-hidden rounded-xl border border-default bg-default"
  >
    <header
      class="flex shrink-0 flex-wrap items-center gap-2 border-b border-default bg-elevated/25 p-3"
    >
      <div class="min-w-48 flex-1">
        <h3 class="text-sm font-semibold text-highlighted">{{ t('macroEditor.title') }}</h3>
        <p class="mt-0.5 text-xs text-muted">{{ summaryLabel }}</p>
      </div>
      <div
        class="flex items-center rounded-lg border border-default bg-default p-0.5"
        role="group"
        :aria-label="t('macroEditor.view_mode')"
      >
        <UButton
          size="xs"
          color="neutral"
          :variant="viewMode === 'simple' ? 'soft' : 'ghost'"
          :aria-pressed="viewMode === 'simple'"
          :label="t('macroEditor.simple_view')"
          @click="viewMode = 'simple'"
        />
        <UButton
          size="xs"
          color="neutral"
          :variant="viewMode === 'atomic' ? 'soft' : 'ghost'"
          :aria-pressed="viewMode === 'atomic'"
          :label="t('macroEditor.atomic_view')"
          @click="viewMode = 'atomic'"
        />
      </div>
      <UInput
        v-model="search"
        icon="i-tabler-search"
        size="sm"
        class="w-52"
        :placeholder="t('macroEditor.search')"
      />
      <UDropdownMenu :items="addMenuItems">
        <UButton
          size="sm"
          color="primary"
          variant="soft"
          icon="i-tabler-plus"
          :label="addButtonLabel"
        />
      </UDropdownMenu>
    </header>

    <div
      v-if="analysis.issues.length"
      class="flex shrink-0 items-start gap-2 border-b border-error/30 bg-error/10 px-3 py-2 text-xs text-error"
      role="alert"
    >
      <UIcon name="i-tabler-alert-triangle" class="mt-0.5 size-4 shrink-0" />
      <span>{{ issueMessage }}</span>
    </div>

    <div v-if="visibleRows.length" class="min-h-0 flex-1 overflow-auto">
      <div
        class="sticky top-0 z-10 grid min-w-[940px] grid-cols-[2.25rem_3rem_8rem_minmax(18rem,1fr)_8rem_5rem] items-center gap-2 border-b border-default bg-elevated px-3 py-2 text-[10px] font-semibold uppercase tracking-wide text-dimmed"
      >
        <UCheckbox
          :model-value="allVisibleSelected"
          :aria-label="t('macroEditor.select_visible')"
          @update:model-value="toggleVisible(Boolean($event))"
        />
        <span>#</span>
        <span>{{ t('macroEditor.action') }}</span>
        <span>{{ t('macroEditor.parameters') }}</span>
        <span>{{ t('macroEditor.state_after') }}</span>
        <span />
      </div>
      <article
        v-for="entry in visibleRows"
        :key="entry.row.id"
        class="grid min-h-12 min-w-[940px] grid-cols-[2.25rem_3rem_8rem_minmax(18rem,1fr)_8rem_5rem] items-center gap-2 border-b border-default/70 px-3 py-2 transition-colors hover:bg-elevated/35"
        :class="[
          selected.has(entry.row.id) ? 'bg-primary/5' : '',
          dragTarget?.rowId === entry.row.id ? 'ring-1 ring-inset ring-primary/70' : '',
        ]"
        @dragover.prevent="continueDrag(entry.row, $event)"
        @drop.prevent="dropRows"
      >
        <UCheckbox
          :model-value="selected.has(entry.row.id)"
          :aria-label="t('macroEditor.select_action', { n: entry.position + 1 })"
          @update:model-value="toggle(entry.row.id, Boolean($event))"
        />
        <span class="flex items-center gap-1 font-mono text-[10px] text-dimmed">
          <span
            draggable="true"
            class="cursor-grab rounded p-0.5 text-dimmed hover:bg-elevated hover:text-toned active:cursor-grabbing"
            :aria-label="t('macroEditor.drag_action', { n: entry.position + 1 })"
            :title="t('macroEditor.drag_hint')"
            @dragstart.stop="beginDrag(entry.row, $event)"
            @dragend="endDrag"
          >
            <UIcon name="i-tabler-grip-vertical" class="size-3.5" />
          </span>
          {{ entry.position + 1 }}
        </span>
        <UBadge :color="actionTone(entry.row.kind)" variant="soft" size="sm">
          {{ actionLabel(entry.row.kind) }}
        </UBadge>

        <div class="flex min-w-0 items-center gap-2">
          <template v-if="isKeyRow(entry.row)">
            <UInput
              :model-value="entry.row.key ?? ''"
              icon="i-tabler-keyboard"
              class="w-44"
              :placeholder="t('macroEditor.press_key')"
              readonly
              @keydown.prevent.stop="captureKey(entry.row, $event)"
            />
            <template v-if="entry.row.kind === 'key-press'">
              <UInputNumber
                :model-value="microsecondsToMilliseconds(entry.row.durationUs)"
                :min="1"
                :max="3_600_000"
                :step="10"
                class="w-32"
                @update:model-value="updateDuration(entry.row, $event)"
              />
              <span class="text-xs text-muted">ms</span>
            </template>
            <span v-else class="text-xs text-muted">{{ t('macroEditor.atomic_key_hint') }}</span>
          </template>
          <template v-else-if="entry.row.kind === 'sleep'">
            <UInputNumber
              :model-value="microsecondsToMilliseconds(entry.row.durationUs)"
              :min="1"
              :max="3_600_000"
              :step="10"
              class="w-36"
              @update:model-value="updateDuration(entry.row, $event)"
            />
            <span class="text-xs text-muted">ms</span>
          </template>
          <template v-else>
            <AdaptiveSelect
              v-if="entry.row.kind !== 'scroll'"
              :model-value="entry.row.button ?? 'left'"
              :items="buttonItems"
              class="w-28"
              width-mode="fixed"
              @update:model-value="updateButton(entry.row, $event)"
            />
            <UInputNumber
              v-if="entry.row.kind === 'scroll'"
              :model-value="entry.row.notches ?? 1"
              :min="-120"
              :max="120"
              :step="1"
              class="w-24"
              @update:model-value="updateNotches(entry.row, $event)"
            />
            <span class="text-[10px] text-dimmed">X%</span>
            <UInputNumber
              :model-value="ratioToPercent(entry.row.point?.x)"
              :min="0"
              :max="100"
              :step="1"
              class="w-24"
              @update:model-value="updatePoint(entry.row, 'x', $event)"
            />
            <span class="text-[10px] text-dimmed">Y%</span>
            <UInputNumber
              :model-value="ratioToPercent(entry.row.point?.y)"
              :min="0"
              :max="100"
              :step="1"
              class="w-24"
              @update:model-value="updatePoint(entry.row, 'y', $event)"
            />
            <template v-if="entry.row.kind === 'click'">
              <UInputNumber
                :model-value="microsecondsToMilliseconds(entry.row.durationUs)"
                :min="1"
                :max="5000"
                :step="10"
                class="w-28"
                @update:model-value="updateDuration(entry.row, $event)"
              />
              <span class="text-xs text-muted">ms</span>
            </template>
          </template>
        </div>

        <span class="truncate text-[10px] text-muted" :title="stateAfter(entry.row.endIndex)">
          {{ stateAfter(entry.row.endIndex) }}
        </span>
        <div class="flex items-center justify-end gap-0.5">
          <UDropdownMenu :items="rowAddMenuItems(entry.row.endIndex)">
            <UButton
              icon="i-tabler-plus"
              color="neutral"
              variant="ghost"
              size="xs"
              :aria-label="t('macroEditor.add_after', { n: entry.position + 1 })"
            />
          </UDropdownMenu>
          <UDropdownMenu :items="rowMenuItems(entry.row)">
            <UButton
              icon="i-tabler-dots"
              color="neutral"
              variant="ghost"
              size="xs"
              :aria-label="t('macroEditor.action_menu', { n: entry.position + 1 })"
            />
          </UDropdownMenu>
        </div>
      </article>
    </div>

    <div v-else class="flex min-h-0 flex-1 items-center justify-center px-4 py-12 text-center">
      <div>
        <UIcon name="i-tabler-list-details" class="mx-auto size-8 text-dimmed" />
        <p class="mt-2 text-sm font-medium text-highlighted">
          {{ search.trim() ? t('macroEditor.no_results') : t('macroEditor.empty') }}
        </p>
        <p class="mt-1 text-xs text-muted">{{ emptyHint }}</p>
      </div>
    </div>

    <footer
      v-if="selected.size"
      class="flex shrink-0 items-center gap-2 border-t border-default bg-primary/5 px-3 py-2"
    >
      <span class="mr-auto text-xs font-medium text-toned">
        {{ t('macroEditor.selected', { count: selected.size }) }}
      </span>
      <UButton
        size="xs"
        color="neutral"
        variant="soft"
        icon="i-tabler-copy"
        :label="t('macroEditor.duplicate_selected')"
        @click="duplicateSelected"
      />
      <UButton
        size="xs"
        color="neutral"
        variant="ghost"
        icon="i-tabler-arrow-up"
        :disabled="!canMoveSelectedUp"
        :aria-label="t('macroEditor.move_selected_up')"
        :title="t('macroEditor.move_selected_up')"
        @click="moveSelected('up')"
      />
      <UButton
        size="xs"
        color="neutral"
        variant="ghost"
        icon="i-tabler-arrow-down"
        :disabled="!canMoveSelectedDown"
        :aria-label="t('macroEditor.move_selected_down')"
        :title="t('macroEditor.move_selected_down')"
        @click="moveSelected('down')"
      />
      <UButton
        size="xs"
        color="neutral"
        variant="ghost"
        :label="t('common.cancel')"
        @click="selected.clear()"
      />
      <UButton
        size="xs"
        color="error"
        variant="soft"
        icon="i-tabler-trash"
        :label="t('macroEditor.delete_selected')"
        @click="deleteSelected"
      />
    </footer>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { MacroAction, MacroActionKind } from '@/stores/recording'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'
import {
  analyzeMacroActions,
  canonicalBrowserKey,
  cloneMacroAction,
  duplicateMacroRows,
  insertMacroActions,
  moveMacroRows,
  patchMacroRow,
  projectMacroRows,
  type MacroEditorIssue,
  type MacroEditorRow,
  type MacroEditorRowKind,
} from './macroEditorModel'

type ViewMode = 'simple' | 'atomic'
type InsertKind = MacroActionKind | 'key-press'

const props = defineProps<{ modelValue: MacroAction[] }>()
const emit = defineEmits<{
  'update:modelValue': [value: MacroAction[]]
  validity: [valid: boolean]
}>()
const { t } = useI18n()
const viewMode = ref<ViewMode>('simple')
const search = ref('')
const selected = reactive(new Set<string>())
const draggedRowIDs = ref<string[]>([])
const dragTarget = ref<{ rowId: string; insertAt: number } | null>(null)

const buttonItems = computed(() => [
  { label: t('macroEditor.button_left'), value: 'left' },
  { label: t('macroEditor.button_middle'), value: 'middle' },
  { label: t('macroEditor.button_right'), value: 'right' },
])
const editorRows = computed(() => projectMacroRows(props.modelValue, viewMode.value === 'simple'))
const selectedRows = computed(() => editorRows.value.filter((row) => selected.has(row.id)))
const selectedRowPositions = computed(() =>
  editorRows.value
    .map((row, index) => (selected.has(row.id) ? index : -1))
    .filter((index) => index >= 0),
)
const canMoveSelectedUp = computed(
  () => selectedRowPositions.value.length > 0 && Math.min(...selectedRowPositions.value) > 0,
)
const canMoveSelectedDown = computed(
  () =>
    selectedRowPositions.value.length > 0 &&
    Math.max(...selectedRowPositions.value) < editorRows.value.length - 1,
)
const selectedEndIndex = computed(() =>
  selectedRows.value.length
    ? Math.max(...selectedRows.value.map((row) => row.endIndex))
    : props.modelValue.length - 1,
)
const addMenuItems = computed(() => actionMenu(selectedEndIndex.value))
const addButtonLabel = computed(() =>
  selected.size ? t('macroEditor.add_after_selected') : t('macroEditor.add'),
)
const visibleRows = computed(() => {
  const needle = search.value.trim().toLocaleLowerCase()
  return editorRows.value
    .map((row, position) => ({ row, position }))
    .filter(({ row }) =>
      needle
        ? `${actionLabel(row.kind)} ${row.key ?? ''} ${row.button ?? ''}`
            .toLocaleLowerCase()
            .includes(needle)
        : true,
    )
})
const allVisibleSelected = computed(
  () => visibleRows.value.length > 0 && visibleRows.value.every(({ row }) => selected.has(row.id)),
)
const analysis = computed(() => analyzeMacroActions(props.modelValue))
const issueMessage = computed(() =>
  analysis.value.issues[0] ? translateIssue(analysis.value.issues[0]) : '',
)
const durationLabel = computed(() => `${Math.round(analysis.value.durationUs / 1000)} ms`)
const summaryLabel = computed(() =>
  viewMode.value === 'simple'
    ? t('macroEditor.summary_simple', {
        count: editorRows.value.length,
        atomic: props.modelValue.length,
        duration: durationLabel.value,
      })
    : t('macroEditor.summary', {
        count: props.modelValue.length,
        duration: durationLabel.value,
      }),
)
const emptyHint = computed(() =>
  t(viewMode.value === 'simple' ? 'macroEditor.empty_hint' : 'macroEditor.empty_hint_atomic'),
)

watch(
  () => analysis.value.issues.length,
  (count) => emit('validity', count === 0),
  { immediate: true },
)
watch(
  () => editorRows.value.map((row) => row.id),
  (ids) => {
    const live = new Set(ids)
    for (const id of selected) if (!live.has(id)) selected.delete(id)
  },
)
watch(viewMode, () => selected.clear())
watch(search, () => selected.clear())

function actionMenu(afterIndex: number) {
  return [
    [
      menuAction('key-press', 'i-tabler-keyboard', afterIndex),
      menuAction('click', 'i-tabler-pointer', afterIndex),
      menuAction('scroll', 'i-tabler-mouse', afterIndex),
      menuAction('sleep', 'i-tabler-clock-pause', afterIndex),
    ],
    [
      menuAction('key-down', 'i-tabler-keyboard-show', afterIndex),
      menuAction('key-up', 'i-tabler-keyboard-hide', afterIndex),
      menuAction('mouse-down', 'i-tabler-hand-click', afterIndex),
      menuAction('mouse-up', 'i-tabler-hand-off', afterIndex),
    ],
  ]
}

function menuAction(kind: InsertKind, icon: string, afterIndex: number) {
  return { label: actionLabel(kind), icon, onSelect: () => add(kind, afterIndex) }
}

function actionLabel(kind: MacroEditorRowKind): string {
  return t(`macroEditor.kind_${kind.replaceAll('-', '_')}`)
}

function actionTone(kind: MacroEditorRowKind): 'primary' | 'info' | 'warning' | 'neutral' {
  if (kind.startsWith('key-')) return 'primary'
  if (kind === 'sleep') return 'neutral'
  if (kind === 'scroll') return 'warning'
  return 'info'
}

function add(kind: InsertKind, afterIndex: number): void {
  const point = { x: 0.5, y: 0.5, unit: 'ratio' as const }
  let additions: MacroAction[]
  if (kind === 'key-press') {
    additions = [
      { id: newActionID(), kind: 'key-down', key: 'A' },
      { id: newActionID(), kind: 'sleep', durationUs: 100_000 },
      { id: newActionID(), kind: 'key-up', key: 'A' },
    ]
  } else {
    const id = newActionID()
    const action: MacroAction =
      kind === 'key-down' || kind === 'key-up'
        ? { id, kind, key: 'A' }
        : kind === 'sleep'
          ? { id, kind, durationUs: 100_000 }
          : kind === 'scroll'
            ? { id, kind, point, notches: 1 }
            : kind === 'click'
              ? { id, kind, point, button: 'left', durationUs: 50_000 }
              : { id, kind, point, button: 'left' }
    additions = [action]
  }
  emit('update:modelValue', insertMacroActions(props.modelValue, afterIndex, additions))
  selected.clear()
}

function patchRow(row: MacroEditorRow, patch: Parameters<typeof patchMacroRow>[2]): void {
  emit('update:modelValue', patchMacroRow(props.modelValue, row, patch))
}

function captureKey(row: MacroEditorRow, event: KeyboardEvent): void {
  const key = canonicalBrowserKey(event.key)
  if (key) patchRow(row, { key })
}

function updateDuration(row: MacroEditorRow, value: unknown): void {
  patchRow(row, { durationUs: Math.max(1, Math.round((Number(value) || 1) * 1000)) })
}

function updateButton(row: MacroEditorRow, value: unknown): void {
  if (value === 'left' || value === 'middle' || value === 'right') patchRow(row, { button: value })
}

function updateNotches(row: MacroEditorRow, value: unknown): void {
  let notches = Math.max(-120, Math.min(120, Math.trunc(Number(value) || 0)))
  if (notches === 0) notches = 1
  patchRow(row, { notches })
}

function updatePoint(row: MacroEditorRow, axis: 'x' | 'y', value: unknown): void {
  const point = row.point ?? { x: 0.5, y: 0.5, unit: 'ratio' as const }
  patchRow(row, {
    point: { ...point, [axis]: Math.max(0, Math.min(1, (Number(value) || 0) / 100)) },
  })
}

function rowAddMenuItems(afterIndex: number) {
  return actionMenu(afterIndex)
}

function rowMenuItems(row: MacroEditorRow) {
  return [
    [
      {
        label: t('macroEditor.duplicate'),
        icon: 'i-tabler-copy',
        onSelect: () => duplicateRows([row]),
      },
      {
        label: t('common.delete'),
        icon: 'i-tabler-trash',
        color: 'error' as const,
        onSelect: () => removeRows([row]),
      },
    ],
  ]
}

function duplicateRows(rows: MacroEditorRow[]): void {
  emit(
    'update:modelValue',
    duplicateMacroRows(props.modelValue, rows, () => newActionID()),
  )
  selected.clear()
}

function duplicateSelected(): void {
  duplicateRows(selectedRows.value)
}

function moveSelected(direction: 'up' | 'down'): void {
  if (!selectedRows.value.length) return
  const edge =
    direction === 'up'
      ? Math.min(...selectedRowPositions.value) - 1
      : Math.max(...selectedRowPositions.value) + 1
  const neighbor = editorRows.value[edge]
  if (!neighbor) return
  const insertAt = direction === 'up' ? neighbor.startIndex : neighbor.endIndex + 1
  emit('update:modelValue', moveMacroRows(props.modelValue, selectedRows.value, insertAt))
}

function removeRows(rows: MacroEditorRow[]): void {
  const removed = new Set(rows.flatMap((row) => row.actionIds))
  emit(
    'update:modelValue',
    props.modelValue.filter((action) => !removed.has(action.id)).map(cloneMacroAction),
  )
  selected.clear()
}

function toggle(id: string, checked: boolean): void {
  if (checked) selected.add(id)
  else selected.delete(id)
}

function toggleVisible(checked: boolean): void {
  for (const { row } of visibleRows.value) toggle(row.id, checked)
}

function deleteSelected(): void {
  removeRows(selectedRows.value)
}

function beginDrag(row: MacroEditorRow, event: DragEvent): void {
  if (!selected.has(row.id)) {
    selected.clear()
    selected.add(row.id)
  }
  draggedRowIDs.value = editorRows.value
    .filter((candidate) => selected.has(candidate.id))
    .map((candidate) => candidate.id)
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
    event.dataTransfer.setData('text/plain', row.id)
  }
}

function continueDrag(row: MacroEditorRow, event: DragEvent): void {
  if (!draggedRowIDs.value.length) return
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'move'
  const bounds = (event.currentTarget as HTMLElement).getBoundingClientRect()
  const after = event.clientY >= bounds.top + bounds.height / 2
  dragTarget.value = {
    rowId: row.id,
    insertAt: after ? row.endIndex + 1 : row.startIndex,
  }
}

function dropRows(): void {
  const target = dragTarget.value
  if (!target) return endDrag()
  const moving = editorRows.value.filter((row) => draggedRowIDs.value.includes(row.id))
  emit('update:modelValue', moveMacroRows(props.modelValue, moving, target.insertAt))
  endDrag()
}

function endDrag(): void {
  draggedRowIDs.value = []
  dragTarget.value = null
}

function stateAfter(index: number): string {
  const state = analysis.value.heldAfter[index]
  if (!state || (!state.keys.length && !state.buttons.length)) return t('macroEditor.state_none')
  return [...state.keys, ...state.buttons.map((button) => `${button} mouse`)].join(', ')
}

function isKeyRow(row: MacroEditorRow): boolean {
  return row.kind === 'key-press' || row.kind === 'key-down' || row.kind === 'key-up'
}

function newActionID(): string {
  return `action-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`
}

function translateIssue(issue: MacroEditorIssue): string {
  if (issue.code === 'key-already-down')
    return t('macroEditor.error_key_down', { n: issue.index + 1, key: issue.key })
  if (issue.code === 'key-not-down')
    return t('macroEditor.error_key_up', { n: issue.index + 1, key: issue.key })
  if (issue.code === 'button-already-down')
    return t('macroEditor.error_button_down', { n: issue.index + 1 })
  if (issue.code === 'button-not-down')
    return t('macroEditor.error_button_up', { n: issue.index + 1 })
  if (issue.code === 'click-button-held')
    return t('macroEditor.error_click_held', {
      n: issue.index + 1,
      button: issue.button,
    })
  return t('macroEditor.error_held_end')
}

function microsecondsToMilliseconds(value: number | undefined): number {
  return Math.max(1, Math.round((value ?? 1_000) / 1000))
}

function ratioToPercent(value: number | undefined): number {
  return Math.round((value ?? 0.5) * 100)
}
</script>
