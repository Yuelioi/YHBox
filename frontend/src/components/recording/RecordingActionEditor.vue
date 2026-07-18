<template>
  <section class="space-y-3 rounded-xl border border-default bg-sunken p-3">
    <div class="flex flex-wrap items-center gap-2">
      <div class="mr-auto">
        <h3 class="text-sm font-medium text-highlighted">{{ t('recordingEditor.title') }}</h3>
        <p class="mt-0.5 text-xs text-muted">
          {{ t('recordingEditor.summary', { count: modelValue.length }) }}
        </p>
      </div>
      <UButton
        size="xs"
        color="neutral"
        variant="soft"
        icon="i-tabler-keyboard"
        @click="add('keys')"
      >
        {{ t('recordingEditor.add_keys') }}
      </UButton>
      <UButton
        size="xs"
        color="neutral"
        variant="soft"
        icon="i-tabler-pointer"
        @click="add('click')"
      >
        {{ t('recordingEditor.add_click') }}
      </UButton>
      <UButton
        size="xs"
        color="neutral"
        variant="soft"
        icon="i-tabler-mouse"
        @click="add('scroll')"
      >
        {{ t('recordingEditor.add_scroll') }}
      </UButton>
    </div>

    <div v-if="modelValue.length" class="max-h-80 space-y-2 overflow-y-auto pr-1">
      <article
        v-for="{ action, index } in visibleActions"
        :key="index"
        class="rounded-lg border border-default bg-default p-3"
      >
        <div class="flex items-center gap-2">
          <UBadge color="primary" variant="soft" size="sm">
            {{ index + 1 }} · {{ t(`recordingEditor.kind_${action.kind}`) }}
          </UBadge>
          <div class="ml-auto flex items-center gap-1">
            <UButton
              icon="i-tabler-arrow-up"
              color="neutral"
              variant="ghost"
              size="xs"
              :disabled="index === 0"
              :aria-label="t('recordingEditor.move_up')"
              @click="move(index, -1)"
            />
            <UButton
              icon="i-tabler-arrow-down"
              color="neutral"
              variant="ghost"
              size="xs"
              :disabled="index === modelValue.length - 1"
              :aria-label="t('recordingEditor.move_down')"
              @click="move(index, 1)"
            />
            <UButton
              icon="i-tabler-trash"
              color="error"
              variant="ghost"
              size="xs"
              :aria-label="t('common.delete')"
              @click="remove(index)"
            />
          </div>
        </div>

        <div class="mt-3 grid grid-cols-2 gap-3 lg:grid-cols-4">
          <UFormField :label="t('recordingEditor.delay_ms')">
            <UInputNumber
              :model-value="microsecondsToMilliseconds(action.delayUs)"
              :min="0"
              :max="3_600_000"
              :step="10"
              :disabled="index === 0"
              class="w-32"
              @update:model-value="updateNumber(index, 'delayUs', $event)"
            />
          </UFormField>
          <UFormField v-if="action.kind !== 'scroll'" :label="t('recordingEditor.duration_ms')">
            <UInputNumber
              :model-value="microsecondsToMilliseconds(action.durationUs)"
              :min="0"
              :max="3_600_000"
              :step="10"
              class="w-32"
              @update:model-value="updateNumber(index, 'durationUs', $event)"
            />
          </UFormField>
          <UFormField
            v-if="action.kind === 'keys'"
            :label="t('recordingEditor.keys')"
            class="col-span-2"
          >
            <UInput
              :model-value="action.keys?.join(' + ') ?? ''"
              :placeholder="t('recordingEditor.keys_placeholder')"
              @update:model-value="updateKeys(index, String($event))"
            />
          </UFormField>
          <UFormField v-if="action.kind === 'click'" :label="t('recordingEditor.button')">
            <USelect
              :model-value="action.button ?? 'left'"
              :items="buttonItems"
              value-key="value"
              label-key="label"
              @update:model-value="updateButton(index, $event)"
            />
          </UFormField>
          <UFormField v-if="action.kind === 'scroll'" :label="t('recordingEditor.notches')">
            <UInputNumber
              :model-value="action.notches ?? 1"
              :min="-120"
              :max="120"
              :step="1"
              class="w-32"
              @update:model-value="updateNotches(index, $event)"
            />
          </UFormField>
          <template v-if="action.kind !== 'keys'">
            <UFormField :label="t('recordingEditor.point_x')">
              <UInputNumber
                :model-value="ratioToPercent(action.point?.x)"
                :min="0"
                :max="100"
                :step="1"
                class="w-28"
                @update:model-value="updatePoint(index, 'x', $event)"
              />
            </UFormField>
            <UFormField :label="t('recordingEditor.point_y')">
              <UInputNumber
                :model-value="ratioToPercent(action.point?.y)"
                :min="0"
                :max="100"
                :step="1"
                class="w-28"
                @update:model-value="updatePoint(index, 'y', $event)"
              />
            </UFormField>
          </template>
        </div>
      </article>
    </div>
    <p
      v-else
      class="rounded-lg border border-dashed border-default px-3 py-6 text-center text-xs text-muted"
    >
      {{ t('recordingEditor.empty') }}
    </p>
    <div v-if="pageCount > 1" class="flex items-center justify-end gap-2">
      <span class="mr-auto text-xs text-dimmed">
        {{ t('recordingEditor.page', { page, pages: pageCount }) }}
      </span>
      <UButton
        icon="i-tabler-chevron-left"
        color="neutral"
        variant="ghost"
        size="xs"
        :disabled="page <= 1"
        @click="page--"
      />
      <UButton
        icon="i-tabler-chevron-right"
        color="neutral"
        variant="ghost"
        size="xs"
        :disabled="page >= pageCount"
        @click="page++"
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { RecordingAction } from '@/stores/recording'

const props = defineProps<{ modelValue: RecordingAction[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: RecordingAction[]] }>()
const { t } = useI18n()
const pageSize = 50
const page = ref(1)
const pageCount = computed(() => Math.max(1, Math.ceil(props.modelValue.length / pageSize)))
const visibleActions = computed(() => {
  const start = (page.value - 1) * pageSize
  return props.modelValue.slice(start, start + pageSize).map((action, offset) => ({
    action,
    index: start + offset,
  }))
})

watch(pageCount, (count) => {
  if (page.value > count) page.value = count
})

const buttonItems = computed(() => [
  { label: t('recordingEditor.button_left'), value: 'left' },
  { label: t('recordingEditor.button_middle'), value: 'middle' },
  { label: t('recordingEditor.button_right'), value: 'right' },
])

function add(kind: RecordingAction['kind']): void {
  const action: RecordingAction =
    kind === 'keys'
      ? { kind, delayUs: 0, durationUs: 50_000, keys: ['A'] }
      : kind === 'click'
        ? {
            kind,
            delayUs: 0,
            durationUs: 50_000,
            button: 'left',
            point: { x: 0.5, y: 0.5, unit: 'ratio' },
          }
        : {
            kind,
            delayUs: 0,
            durationUs: 0,
            notches: 1,
            point: { x: 0.5, y: 0.5, unit: 'ratio' },
          }
  emit('update:modelValue', [...props.modelValue, action])
  page.value = Math.ceil((props.modelValue.length + 1) / pageSize)
}

function remove(index: number): void {
  emit(
    'update:modelValue',
    props.modelValue.filter((_, current) => current !== index),
  )
}

function move(index: number, offset: -1 | 1): void {
  const nextIndex = index + offset
  if (nextIndex < 0 || nextIndex >= props.modelValue.length) return
  const next = props.modelValue.map(cloneAction)
  const [action] = next.splice(index, 1)
  if (!action) return
  next.splice(nextIndex, 0, action)
  emit('update:modelValue', next)
}

function updateAction(index: number, patch: Partial<RecordingAction>): void {
  emit(
    'update:modelValue',
    props.modelValue.map((action, current) =>
      current === index ? { ...cloneAction(action), ...patch } : cloneAction(action),
    ),
  )
}

function updateNumber(index: number, field: 'delayUs' | 'durationUs', value: unknown): void {
  const milliseconds = Math.max(0, Number(value) || 0)
  updateAction(index, { [field]: Math.round(milliseconds * 1000) })
}

function updateKeys(index: number, value: string): void {
  const keys = value
    .split('+')
    .map((key) => key.trim())
    .filter(Boolean)
  updateAction(index, { keys })
}

function updateButton(index: number, value: unknown): void {
  if (value === 'left' || value === 'middle' || value === 'right')
    updateAction(index, { button: value })
}

function updateNotches(index: number, value: unknown): void {
  let notches = Math.trunc(Number(value) || 0)
  if (notches === 0) notches = 1
  updateAction(index, { notches })
}

function updatePoint(index: number, axis: 'x' | 'y', value: unknown): void {
  const action = props.modelValue[index]
  if (!action) return
  const point = action.point ?? { x: 0.5, y: 0.5, unit: 'ratio' as const }
  updateAction(index, {
    point: { ...point, [axis]: Math.min(1, Math.max(0, (Number(value) || 0) / 100)) },
  })
}

function cloneAction(action: RecordingAction): RecordingAction {
  return {
    ...action,
    keys: action.keys ? [...action.keys] : undefined,
    point: action.point ? { ...action.point } : undefined,
  }
}

function microsecondsToMilliseconds(value: number): number {
  return Math.round(value / 1000)
}

function ratioToPercent(value: number | undefined): number {
  return Math.round((value ?? 0.5) * 100)
}
</script>
