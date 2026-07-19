<template>
  <section class="overflow-hidden rounded-xl border border-default bg-default">
    <header class="flex flex-wrap items-center gap-2 border-b border-default bg-elevated/25 p-3">
      <div class="min-w-48 flex-1">
        <h3 class="text-sm font-semibold text-highlighted">{{ t('macroEditor.title') }}</h3>
        <p class="mt-0.5 text-xs text-muted">
          {{ t('macroEditor.summary', { count: modelValue.length, duration: durationLabel }) }}
        </p>
      </div>
      <UInput
        v-model="search"
        icon="i-tabler-search"
        size="sm"
        class="w-52"
        :placeholder="t('macroEditor.search')"
      />
      <UDropdownMenu :items="addMenuItems">
        <UButton size="sm" icon="i-tabler-plus" :label="t('macroEditor.add')" />
      </UDropdownMenu>
    </header>

    <div
      v-if="analysis.issues.length"
      class="flex items-start gap-2 border-b border-error/30 bg-error/10 px-3 py-2 text-xs text-error"
      role="alert"
    >
      <UIcon name="i-tabler-alert-triangle" class="mt-0.5 size-4 shrink-0" />
      <span>{{ issueMessage }}</span>
    </div>

    <div v-if="visibleActions.length" class="max-h-[28rem] overflow-auto">
      <div
        class="sticky top-0 z-10 grid min-w-[900px] grid-cols-[2.25rem_3rem_8rem_minmax(18rem,1fr)_8rem_3rem] items-center gap-2 border-b border-default bg-elevated px-3 py-2 text-[10px] font-semibold uppercase tracking-wide text-dimmed"
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
        v-for="entry in visibleActions"
        :key="entry.action.id"
        class="grid min-h-12 min-w-[900px] grid-cols-[2.25rem_3rem_8rem_minmax(18rem,1fr)_8rem_3rem] items-center gap-2 border-b border-default/70 px-3 py-2 hover:bg-elevated/35"
        :class="dragOverIndex === entry.index ? 'bg-primary/10' : ''"
        :draggable="!search.trim()"
        @dragstart="beginDrag(entry.index)"
        @dragover.prevent="dragOverIndex = entry.index"
        @dragleave="dragOverIndex = -1"
        @drop.prevent="dropAt(entry.index)"
        @dragend="endDrag"
      >
        <UCheckbox
          :model-value="selected.has(entry.action.id)"
          :aria-label="t('macroEditor.select_action', { n: entry.index + 1 })"
          @update:model-value="toggle(entry.action.id, Boolean($event))"
        />
        <span class="flex items-center gap-1 font-mono text-[10px] text-dimmed">
          <UIcon
            name="i-tabler-grip-vertical"
            class="size-3.5"
            :class="search.trim() ? 'opacity-25' : 'cursor-grab'"
          />
          {{ entry.index + 1 }}
        </span>
        <UBadge :color="actionTone(entry.action.kind)" variant="soft" size="sm">
          {{ actionLabel(entry.action.kind) }}
        </UBadge>

        <div class="flex min-w-0 items-center gap-2">
          <template v-if="isKeyAction(entry.action)">
            <UInput
              :model-value="entry.action.key ?? ''"
              icon="i-tabler-keyboard"
              class="w-44"
              :placeholder="t('macroEditor.press_key')"
              readonly
              @keydown.prevent.stop="captureKey(entry.index, $event)"
            />
            <span class="text-xs text-muted">{{ t('macroEditor.press_key_hint') }}</span>
          </template>
          <template v-else-if="entry.action.kind === 'sleep'">
            <UInputNumber
              :model-value="microsecondsToMilliseconds(entry.action.durationUs)"
              :min="1"
              :max="3_600_000"
              :step="10"
              class="w-36"
              @update:model-value="updateDuration(entry.index, $event)"
            />
            <span class="text-xs text-muted">ms</span>
          </template>
          <template v-else>
            <AdaptiveSelect
              v-if="entry.action.kind !== 'scroll'"
              :model-value="entry.action.button ?? 'left'"
              :items="buttonItems"
              class="w-28"
              width-mode="fixed"
              @update:model-value="updateButton(entry.index, $event)"
            />
            <UInputNumber
              v-if="entry.action.kind === 'scroll'"
              :model-value="entry.action.notches ?? 1"
              :min="-120"
              :max="120"
              :step="1"
              class="w-24"
              @update:model-value="updateNotches(entry.index, $event)"
            />
            <span class="text-[10px] text-dimmed">X%</span>
            <UInputNumber
              :model-value="ratioToPercent(entry.action.point?.x)"
              :min="0"
              :max="100"
              :step="1"
              class="w-24"
              @update:model-value="updatePoint(entry.index, 'x', $event)"
            />
            <span class="text-[10px] text-dimmed">Y%</span>
            <UInputNumber
              :model-value="ratioToPercent(entry.action.point?.y)"
              :min="0"
              :max="100"
              :step="1"
              class="w-24"
              @update:model-value="updatePoint(entry.index, 'y', $event)"
            />
            <template v-if="entry.action.kind === 'click'">
              <UInputNumber
                :model-value="microsecondsToMilliseconds(entry.action.durationUs)"
                :min="1"
                :max="5000"
                :step="10"
                class="w-28"
                @update:model-value="updateDuration(entry.index, $event)"
              />
              <span class="text-xs text-muted">ms</span>
            </template>
          </template>
        </div>

        <span class="truncate text-[10px] text-muted" :title="stateAfter(entry.index)">
          {{ stateAfter(entry.index) }}
        </span>
        <UDropdownMenu :items="rowMenuItems(entry.index)">
          <UButton
            icon="i-tabler-dots"
            color="neutral"
            variant="ghost"
            size="xs"
            :aria-label="t('macroEditor.action_menu', { n: entry.index + 1 })"
          />
        </UDropdownMenu>
      </article>
    </div>

    <div v-else class="px-4 py-12 text-center">
      <UIcon name="i-tabler-list-details" class="mx-auto size-8 text-dimmed" />
      <p class="mt-2 text-sm font-medium text-highlighted">
        {{ search.trim() ? t('macroEditor.no_results') : t('macroEditor.empty') }}
      </p>
      <p class="mt-1 text-xs text-muted">{{ t('macroEditor.empty_hint') }}</p>
    </div>

    <footer
      v-if="selected.size"
      class="flex items-center gap-2 border-t border-default bg-primary/5 px-3 py-2"
    >
      <span class="mr-auto text-xs font-medium text-toned">
        {{ t('macroEditor.selected', { count: selected.size }) }}
      </span>
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
  duplicateMacroAction,
  moveMacroAction,
  type MacroEditorIssue,
} from './macroEditorModel'

const props = defineProps<{ modelValue: MacroAction[] }>()
const emit = defineEmits<{
  'update:modelValue': [value: MacroAction[]]
  validity: [valid: boolean]
}>()
const { t } = useI18n()
const search = ref('')
const selected = reactive(new Set<string>())
const draggedIndex = ref(-1)
const dragOverIndex = ref(-1)

const buttonItems = computed(() => [
  { label: t('macroEditor.button_left'), value: 'left' },
  { label: t('macroEditor.button_middle'), value: 'middle' },
  { label: t('macroEditor.button_right'), value: 'right' },
])
const addMenuItems = computed(() => [
  [
    menuAction('key-down', 'i-tabler-keyboard-show'),
    menuAction('key-up', 'i-tabler-keyboard-hide'),
    menuAction('sleep', 'i-tabler-clock-pause'),
  ],
  [
    menuAction('click', 'i-tabler-pointer'),
    menuAction('mouse-down', 'i-tabler-hand-click'),
    menuAction('mouse-up', 'i-tabler-hand-off'),
    menuAction('scroll', 'i-tabler-mouse'),
  ],
])
const visibleActions = computed(() => {
  const needle = search.value.trim().toLocaleLowerCase()
  return props.modelValue
    .map((action, index) => ({ action, index }))
    .filter(({ action }) =>
      needle
        ? `${actionLabel(action.kind)} ${action.key ?? ''} ${action.button ?? ''}`
            .toLocaleLowerCase()
            .includes(needle)
        : true,
    )
})
const allVisibleSelected = computed(
  () =>
    visibleActions.value.length > 0 &&
    visibleActions.value.every(({ action }) => selected.has(action.id)),
)
const analysis = computed(() => analyzeMacroActions(props.modelValue))
const issueMessage = computed(() =>
  analysis.value.issues[0] ? translateIssue(analysis.value.issues[0]) : '',
)
const durationLabel = computed(() => `${Math.round(analysis.value.durationUs / 1000)} ms`)

watch(
  () => analysis.value.issues.length,
  (count) => emit('validity', count === 0),
  { immediate: true },
)
watch(
  () => props.modelValue.map((action) => action.id),
  (ids) => {
    const live = new Set(ids)
    for (const id of selected) if (!live.has(id)) selected.delete(id)
  },
)

function menuAction(kind: MacroActionKind, icon: string) {
  return { label: actionLabel(kind), icon, onSelect: () => add(kind) }
}

function actionLabel(kind: MacroActionKind): string {
  return t(`macroEditor.kind_${kind.replaceAll('-', '_')}`)
}

function actionTone(kind: MacroActionKind): 'primary' | 'info' | 'warning' | 'neutral' {
  if (kind.startsWith('key-')) return 'primary'
  if (kind === 'sleep') return 'neutral'
  if (kind === 'scroll') return 'warning'
  return 'info'
}

function add(kind: MacroActionKind): void {
  const id = newActionID()
  const point = { x: 0.5, y: 0.5, unit: 'ratio' as const }
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
  emit('update:modelValue', [...props.modelValue.map(cloneMacroAction), action])
}

function updateAction(index: number, patch: Partial<MacroAction>): void {
  emit(
    'update:modelValue',
    props.modelValue.map((action, current) =>
      current === index
        ? ({ ...cloneMacroAction(action), ...patch } as MacroAction)
        : cloneMacroAction(action),
    ),
  )
}

function captureKey(index: number, event: KeyboardEvent): void {
  const key = canonicalBrowserKey(event.key)
  if (key) updateAction(index, { key })
}

function updateDuration(index: number, value: unknown): void {
  updateAction(index, { durationUs: Math.max(1, Math.round((Number(value) || 1) * 1000)) })
}

function updateButton(index: number, value: unknown): void {
  if (value === 'left' || value === 'middle' || value === 'right')
    updateAction(index, { button: value })
}

function updateNotches(index: number, value: unknown): void {
  let notches = Math.max(-120, Math.min(120, Math.trunc(Number(value) || 0)))
  if (notches === 0) notches = 1
  updateAction(index, { notches })
}

function updatePoint(index: number, axis: 'x' | 'y', value: unknown): void {
  const current = props.modelValue[index]
  if (!current) return
  const point = current.point ?? { x: 0.5, y: 0.5, unit: 'ratio' as const }
  updateAction(index, {
    point: { ...point, [axis]: Math.max(0, Math.min(1, (Number(value) || 0) / 100)) },
  })
}

function rowMenuItems(index: number) {
  return [
    [
      {
        label: t('macroEditor.duplicate'),
        icon: 'i-tabler-copy',
        onSelect: () => duplicate(index),
      },
      {
        label: t('common.delete'),
        icon: 'i-tabler-trash',
        color: 'error' as const,
        onSelect: () => remove(index),
      },
    ],
  ]
}

function duplicate(index: number): void {
  emit('update:modelValue', duplicateMacroAction(props.modelValue, index, newActionID()))
}

function remove(index: number): void {
  emit(
    'update:modelValue',
    props.modelValue.filter((_, current) => current !== index).map(cloneMacroAction),
  )
}

function toggle(id: string, checked: boolean): void {
  if (checked) selected.add(id)
  else selected.delete(id)
}

function toggleVisible(checked: boolean): void {
  for (const { action } of visibleActions.value) toggle(action.id, checked)
}

function deleteSelected(): void {
  emit(
    'update:modelValue',
    props.modelValue.filter((action) => !selected.has(action.id)).map(cloneMacroAction),
  )
  selected.clear()
}

function beginDrag(index: number): void {
  if (search.value.trim()) return
  draggedIndex.value = index
}

function dropAt(index: number): void {
  const from = draggedIndex.value
  if (from < 0 || from === index) return endDrag()
  emit('update:modelValue', moveMacroAction(props.modelValue, from, index))
  endDrag()
}

function endDrag(): void {
  draggedIndex.value = -1
  dragOverIndex.value = -1
}

function stateAfter(index: number): string {
  const state = analysis.value.heldAfter[index]
  if (!state || (!state.keys.length && !state.buttons.length)) return t('macroEditor.state_none')
  return [...state.keys, ...state.buttons.map((button) => `${button} mouse`)].join(', ')
}

function isKeyAction(action: MacroAction): boolean {
  return action.kind === 'key-down' || action.kind === 'key-up'
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
