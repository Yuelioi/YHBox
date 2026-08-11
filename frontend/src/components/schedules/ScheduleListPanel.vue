<template>
  <div
    v-if="list.length"
    class="min-w-[1120px]"
    role="table"
    :aria-label="t('schedule.table.caption')"
  >
    <div
      class="workspace-surface-strong grid h-9 items-center gap-3 border-b border-default px-3 text-[10px] font-semibold uppercase tracking-wide text-dimmed"
      :style="{ gridTemplateColumns }"
      role="row"
    >
      <UCheckbox
        :model-value="allSelected"
        :aria-label="t('schedule.select_page')"
        @update:model-value="$emit('select-page', Boolean($event))"
      />
      <span>{{ t('schedule.table.name') }}</span>
      <span v-if="isColumnVisible('category')">{{ t('common.category') }}</span>
      <span v-if="isColumnVisible('tags')">{{ t('common.tags') }}</span>
      <span v-if="isColumnVisible('trigger')">{{ t('schedule.table.trigger') }}</span>
      <span v-if="isColumnVisible('targets')">{{ t('schedule.table.targets') }}</span>
      <span v-if="isColumnVisible('createdAt')">{{ t('schedule.table.created') }}</span>
      <span v-if="isColumnVisible('updatedAt')">{{ t('schedule.table.updated') }}</span>
      <span v-if="isColumnVisible('lastFiredAt')">{{ t('schedule.table.last') }}</span>
      <span>{{ t('schedule.table.enabled') }}</span>
      <span class="text-right">{{ t('schedule.table.actions') }}</span>
    </div>

    <article
      v-for="schedule in list"
      :key="schedule.id"
      class="workspace-table-row grid min-h-16 items-center gap-3 border-b border-default/70 px-3 py-2 transition-colors duration-150 hover:bg-[var(--ui-surface-hover)]"
      :class="schedule.enabled ? '' : 'opacity-70'"
      :style="{ gridTemplateColumns }"
      role="row"
      data-testid="schedule-row"
      :data-schedule-id="schedule.id"
      :data-target-ids="schedule.targets.map((target) => target.id).join(',')"
      :data-last-status="schedule.lastStatus ?? ''"
      @dblclick="$emit('edit', schedule)"
    >
      <UCheckbox
        :model-value="Boolean(selected[schedule.id])"
        :aria-label="t('schedule.select_named', { name: schedule.name })"
        @update:model-value="$emit('select', schedule, Boolean($event))"
        @dblclick.stop
      />

      <div class="min-w-0">
        <div class="flex min-w-0 items-center gap-2">
          <span class="truncate text-sm font-medium text-highlighted">{{ schedule.name }}</span>
          <UBadge v-if="schedule.lastStatus" size="xs" color="neutral" variant="subtle">
            {{ lastStatusLabel(schedule.lastStatus) }}
          </UBadge>
        </div>
        <p class="mt-0.5 truncate text-[11px] text-muted">
          {{ schedule.description || t('schedule.no_description') }}
        </p>
        <p
          v-if="schedule.lastStatus === 'failed' && schedule.lastReadiness"
          class="mt-1 flex min-w-0 items-start gap-1.5 text-[10px] leading-4 text-warning"
          data-testid="schedule-readiness"
        >
          <UIcon name="i-tabler-alert-circle" class="mt-0.5 size-3 shrink-0" />
          <span class="min-w-0 flex-1 break-words">{{ lastReadinessLabel(schedule) }}</span>
          <UButton
            size="xs"
            variant="link"
            color="warning"
            class="shrink-0 p-0"
            data-testid="schedule-repair"
            @click="$emit('repair', schedule)"
          >
            {{ t('schedule.repair_action') }}
          </UButton>
        </p>
      </div>

      <div v-if="isColumnVisible('category')" class="min-w-0">
        <UBadge v-if="schedule.category" color="neutral" variant="soft" size="sm">
          {{ schedule.category }}
        </UBadge>
        <span v-else class="text-[11px] text-dimmed">{{ t('schedule.unclassified') }}</span>
      </div>

      <div v-if="isColumnVisible('tags')" class="flex min-w-0 items-center gap-1 overflow-hidden">
        <UBadge
          v-for="tag in (schedule.tags ?? []).slice(0, 3)"
          :key="tag"
          color="neutral"
          variant="subtle"
          size="sm"
        >
          {{ tag }}
        </UBadge>
        <span v-if="!(schedule.tags ?? []).length" class="text-[11px] text-dimmed">
          {{ t('schedule.no_tags') }}
        </span>
        <span v-else-if="(schedule.tags ?? []).length > 3" class="text-[10px] text-dimmed">
          +{{ (schedule.tags ?? []).length - 3 }}
        </span>
      </div>

      <span
        v-if="isColumnVisible('trigger')"
        class="inline-flex min-w-0 items-center gap-1.5 text-xs text-toned"
      >
        <UIcon :name="triggerIcon(schedule)" class="size-3.5 shrink-0" />
        <span class="truncate">{{ triggerLabel(schedule) }}</span>
      </span>

      <span v-if="isColumnVisible('targets')" class="truncate text-xs text-muted">
        {{ targetSummary(schedule) }}
      </span>

      <time
        v-if="isColumnVisible('createdAt')"
        :datetime="schedule.createdAt || undefined"
        class="text-xs text-muted"
        :title="formatExactDate(schedule.createdAt)"
      >
        {{ formatListDate(schedule.createdAt) }}
      </time>
      <time
        v-if="isColumnVisible('updatedAt')"
        :datetime="schedule.updatedAt || undefined"
        class="text-xs text-muted"
        :title="formatExactDate(schedule.updatedAt)"
      >
        {{ formatListDate(schedule.updatedAt) }}
      </time>
      <time
        v-if="isColumnVisible('lastFiredAt')"
        :datetime="schedule.lastFiredAt || undefined"
        class="text-xs text-muted"
        :title="schedule.lastFiredAt ? formatExactDate(schedule.lastFiredAt) : undefined"
      >
        {{ formatListDate(schedule.lastFiredAt) }}
      </time>

      <div class="flex items-center gap-2" @dblclick.stop>
        <USwitch
          :model-value="schedule.enabled"
          :aria-label="schedule.enabled ? t('schedule.disable') : t('schedule.enable')"
          @update:model-value="(enabled: boolean) => $emit('toggle', schedule, enabled)"
        />
        <span class="text-[10px] text-dimmed">
          {{ schedule.enabled ? t('schedule.enable') : t('schedule.disable') }}
        </span>
      </div>

      <div class="flex justify-end gap-1" @dblclick.stop>
        <UButton
          size="sm"
          variant="ghost"
          color="neutral"
          icon="i-tabler-player-play"
          data-testid="schedule-run"
          :loading="runningId === schedule.id"
          :disabled="runningId !== ''"
          :aria-label="t('schedule.run_action', { name: schedule.name })"
          @click="$emit('run', schedule)"
        />
        <UDropdownMenu :items="menuItems(schedule)">
          <UButton
            size="sm"
            variant="ghost"
            color="neutral"
            icon="i-tabler-dots"
            :aria-label="t('schedule.more_action', { name: schedule.name })"
          />
        </UDropdownMenu>
      </div>
    </article>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { Schedule } from '@/lib/backend'
import type { SourceView } from '@/app/transport/workflow'
import { readinessOutcome, runReadinessMessage } from '@/app/run/runReadiness'

const { t, locale } = useI18n()
const { list, workflows, runningId, visibleColumns, gridTemplateColumns, selected, allSelected } =
  defineProps<{
    list: Schedule[]
    workflows: SourceView[]
    runningId: string
    visibleColumns: string[]
    gridTemplateColumns: string
    selected: Record<string, Schedule>
    allSelected: boolean
  }>()
const emit = defineEmits<{
  edit: [schedule: Schedule]
  delete: [schedule: Schedule]
  toggle: [schedule: Schedule, enabled: boolean]
  run: [schedule: Schedule]
  repair: [schedule: Schedule]
  select: [schedule: Schedule, checked: boolean]
  'select-page': [checked: boolean]
}>()

function isColumnVisible(column: string): boolean {
  return visibleColumns.includes(column)
}

function triggerIcon(schedule: Schedule): string {
  if (schedule.trigger.kind === 'hotkey') return 'i-tabler-keyboard'
  if (schedule.trigger.kind === 'once') return 'i-tabler-rocket'
  if (schedule.trigger.kind === 'manual') return 'i-tabler-hand-click'
  return schedule.trigger.subKind === 'daily' ? 'i-tabler-sun' : 'i-tabler-repeat'
}

function triggerLabel(schedule: Schedule): string {
  const trigger = schedule.trigger
  if (trigger.kind === 'cron' && trigger.subKind === 'daily')
    return t('schedule.display.daily', { at: trigger.at ?? '--:--' })
  if (trigger.kind === 'cron' && trigger.subKind === 'interval')
    return t('schedule.display.interval', { mins: trigger.everyMinutes })
  if (trigger.kind === 'hotkey') return t('schedule.display.hotkey', { key: trigger.hotkey ?? '' })
  return t(`schedule.trigger.${trigger.kind}`)
}

function targetSummary(schedule: Schedule): string {
  const names = schedule.targets
    .map((target) => workflows.find((workflow) => workflow.workflowId === target.id)?.name)
    .filter((name): name is string => Boolean(name))
  if (names.length === 0) return t('schedule.workspace.no_targets')
  const visible = names.slice(0, 2).join(' → ')
  return names.length > 2
    ? t('schedule.workspace.more_targets', { names: visible, n: names.length - 2 })
    : visible
}

function formatListDate(value?: string | null): string {
  if (!value) return '—'
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return '—'
  return new Intl.DateTimeFormat(locale.value, { dateStyle: 'medium' }).format(parsed)
}

function formatExactDate(value?: string): string | undefined {
  if (!value) return undefined
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return value
  return new Intl.DateTimeFormat(locale.value, {
    dateStyle: 'long',
    timeStyle: 'medium',
  }).format(parsed)
}

function lastStatusLabel(status: string): string {
  return status === 'queued' ? t('schedule.status.queued') : t('schedule.status.failed')
}

function lastReadinessLabel(schedule: Schedule): string {
  return runReadinessMessage(readinessOutcome(schedule.lastReadiness))
}

function menuItems(schedule: Schedule) {
  return [
    [
      {
        label: t('schedule.edit_action', { name: schedule.name }),
        icon: 'i-tabler-edit',
        onSelect: () => emit('edit', schedule),
      },
    ],
    [
      {
        label: t('schedule.delete_action', { name: schedule.name }),
        icon: 'i-tabler-trash',
        color: 'error' as const,
        onSelect: () => emit('delete', schedule),
      },
    ],
  ]
}
</script>
