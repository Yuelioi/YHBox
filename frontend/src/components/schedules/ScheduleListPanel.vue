<template>
  <EmptyState
    v-if="list.length === 0"
    icon="i-tabler-calendar-off"
    :title="t('schedule.empty')"
    :description="t('schedule.empty_desc')"
  />

  <div v-else class="schedule-list" role="list" :aria-label="t('schedule.table.caption')">
    <article
      v-for="schedule in list"
      :key="schedule.id"
      class="schedule-row"
      :class="schedule.enabled ? '' : 'schedule-row--disabled'"
      role="listitem"
      data-testid="schedule-row"
      :data-schedule-id="schedule.id"
      :data-target-ids="schedule.targets.map((target) => target.id).join(',')"
      @dblclick="$emit('edit', schedule)"
    >
      <span class="schedule-row__trigger">
        <UIcon :name="triggerIcon(schedule)" class="size-5" />
      </span>

      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-2">
          <h2 class="truncate text-sm font-semibold text-highlighted">{{ schedule.name }}</h2>
          <StatusPill
            :status="schedule.enabled ? 'online' : 'paused'"
            :label="schedule.enabled ? t('schedule.enable') : t('schedule.disable')"
            :dot="schedule.enabled"
          />
          <UBadge v-if="schedule.lastStatus" size="xs" color="neutral" variant="subtle">
            {{ lastStatusLabel(schedule.lastStatus) }}
          </UBadge>
        </div>
        <div class="mt-1.5 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-dimmed">
          <span class="inline-flex items-center gap-1.5 text-toned">
            <UIcon name="i-tabler-bolt" class="size-3.5" />
            {{ triggerLabel(schedule) }}
          </span>
          <span class="inline-flex items-center gap-1.5">
            <UIcon name="i-tabler-stack-2" class="size-3.5" />
            {{ targetSummary(schedule) }}
          </span>
          <span class="inline-flex items-center gap-1.5">
            <UIcon name="i-tabler-history" class="size-3.5" />
            {{ lastFiredLabel(schedule) }}
          </span>
        </div>
      </div>

      <div class="schedule-row__policy">
        <span class="text-xs text-dimmed">{{ t('schedule.workspace.error_policy') }}</span>
        <strong class="text-xs font-medium text-toned">{{
          t(`schedule.error_mode.${schedule.onError}`)
        }}</strong>
      </div>

      <div class="flex shrink-0 items-center gap-1">
        <USwitch
          :model-value="schedule.enabled"
          :aria-label="schedule.enabled ? t('schedule.disable') : t('schedule.enable')"
          @click.stop
          @update:model-value="(enabled: boolean) => $emit('toggle', schedule, enabled)"
        />
        <UButton
          size="sm"
          variant="ghost"
          color="neutral"
          icon="i-tabler-edit"
          data-testid="schedule-edit"
          :title="t('schedule.edit_action', { name: schedule.name })"
          :aria-label="t('schedule.edit_action', { name: schedule.name })"
          @click="$emit('edit', schedule)"
        />
        <UDropdownMenu :items="menuItems(schedule)">
          <UButton
            size="sm"
            variant="ghost"
            color="neutral"
            icon="i-tabler-dots-vertical"
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
import EmptyState from '@/components/common/EmptyState.vue'
import StatusPill from '@/components/common/StatusPill.vue'

const { t } = useI18n()
const { list, workflows } = defineProps<{
  list: Schedule[]
  workflows: SourceView[]
}>()
const emit = defineEmits<{
  edit: [schedule: Schedule]
  delete: [schedule: Schedule]
  toggle: [schedule: Schedule, enabled: boolean]
}>()

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

function lastFiredLabel(schedule: Schedule): string {
  if (!schedule.lastFiredAt) return t('schedule.workspace.never_run')
  const value = new Date(schedule.lastFiredAt)
  return t('schedule.workspace.last_run', {
    value: Number.isNaN(value.getTime()) ? schedule.lastFiredAt : value.toLocaleString(),
  })
}

function lastStatusLabel(status: string): string {
  return status === 'queued' ? t('schedule.status.queued') : status
}

function menuItems(schedule: Schedule) {
  return [
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
