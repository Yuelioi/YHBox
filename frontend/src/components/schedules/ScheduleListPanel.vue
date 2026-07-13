<template>
  <div class="space-y-3">
    <EmptyState
      v-if="list.length === 0"
      icon="i-tabler-clock"
      :title="t('schedule.empty')"
      :description="t('schedule.empty_desc')"
    />
    <div v-else class="overflow-x-auto rounded-lg border border-default/60">
      <table class="w-full min-w-[760px] text-sm">
        <caption class="sr-only">
          {{
            t('schedule.table.caption')
          }}
        </caption>
        <thead class="border-b border-default bg-muted/20 text-xs text-muted">
          <tr>
            <th class="text-left p-2">{{ t('schedule.table.name') }}</th>
            <th class="text-left p-2">{{ t('schedule.table.trigger') }}</th>
            <th class="text-left p-2">{{ t('schedule.table.count') }}</th>
            <th class="text-left p-2">{{ t('schedule.table.last') }}</th>
            <th class="text-left p-2">{{ t('schedule.table.enabled') }}</th>
            <th class="p-2 text-right">{{ t('schedule.table.actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="s in list" :key="s.id" class="border-b border-default/40 hover:bg-elevated/40">
            <td class="p-2 text-default">{{ s.name }}</td>
            <td class="p-2 text-dimmed">{{ triggerLabel(s) }}</td>
            <td class="p-2 text-dimmed font-mono tabular-nums">{{ s.targets.length }}</td>
            <td class="p-2 text-dimmed font-mono tabular-nums">
              {{ s.lastFiredAt?.slice(0, 16).replace('T', ' ') ?? '—' }}
            </td>
            <td class="p-2">
              <StatusPill
                :status="s.enabled ? 'online' : 'ready'"
                :label="s.enabled ? t('schedule.enable') : t('schedule.disable')"
                :dot="s.enabled"
              />
            </td>
            <td class="p-2 text-right">
              <UButton
                size="xs"
                variant="ghost"
                color="neutral"
                icon="i-tabler-edit"
                :title="t('schedule.edit_action', { name: s.name })"
                :aria-label="t('schedule.edit_action', { name: s.name })"
                @click="$emit('edit', s)"
              />
              <UButton
                size="xs"
                variant="ghost"
                color="error"
                icon="i-tabler-trash"
                :title="t('schedule.delete_action', { name: s.name })"
                :aria-label="t('schedule.delete_action', { name: s.name })"
                @click="$emit('delete', s)"
              />
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { Schedule } from '@/lib/backend'
import EmptyState from '@/components/common/EmptyState.vue'
import StatusPill from '@/components/common/StatusPill.vue'

const { t } = useI18n()

defineProps<{ list: Schedule[] }>()
defineEmits<{ edit: [s: Schedule]; delete: [s: Schedule] }>()

function triggerLabel(s: Schedule): string {
  const tg = s.trigger
  switch (tg.kind) {
    case 'cron':
      if (tg.subKind === 'daily') return t('schedule.display.daily', { at: tg.at ?? '--:--' })
      if (tg.subKind === 'interval')
        return t('schedule.display.interval', { mins: tg.everyMinutes })
      return 'cron'
    case 'hotkey':
      return t('schedule.display.hotkey', { key: tg.hotkey ?? '' })
    case 'once':
      return t('schedule.trigger.once')
    case 'manual':
      return t('schedule.trigger.manual')
  }
  return tg.kind
}
</script>
