<template>
  <div class="space-y-3">
    <div
      v-if="list.length === 0"
      class="rounded-xl bg-default/50 border border-default/60 border-dashed py-12 px-6 text-center"
    >
      <UIcon name="i-tabler-clock" class="size-8 text-dimmed mx-auto mb-3" />
      <p class="text-sm text-muted">{{ t('schedule.empty') }}</p>
      <p class="text-xs text-dimmed mt-1">
        {{ t('schedule.empty_desc') }}
      </p>
    </div>
    <table v-else class="w-full text-sm">
      <thead class="text-xs text-dimmed uppercase tracking-wider border-b border-default">
        <tr>
          <th class="text-left p-2">{{ t('schedule.table.name') }}</th>
          <th class="text-left p-2">{{ t('schedule.table.trigger') }}</th>
          <th class="text-left p-2">{{ t('schedule.table.count') }}</th>
          <th class="text-left p-2">{{ t('schedule.table.last') }}</th>
          <th class="text-left p-2">{{ t('schedule.table.enabled') }}</th>
          <th class="p-2"></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="s in list" :key="s.id" class="border-b border-default/40 hover:bg-elevated/40">
          <td class="p-2 text-default">{{ s.name }}</td>
          <td class="p-2 text-dimmed">{{ triggerLabel(s) }}</td>
          <td class="p-2 text-dimmed">{{ s.targets.length }}</td>
          <td class="p-2 text-dimmed">
            {{ s.lastFiredAt?.slice(0, 16).replace('T', ' ') ?? '—' }}
          </td>
          <td class="p-2">
            <span
              class="text-[10px] px-1.5 py-0.5 rounded"
              :class="
                s.enabled
                  ? 'bg-primary/10 text-primary border border-primary/20'
                  : 'bg-elevated text-dimmed'
              "
            >
              {{ s.enabled ? t('schedule.enable') : t('schedule.disable') }}
            </span>
          </td>
          <td class="p-2 text-right">
            <UButton
              size="xs"
              variant="ghost"
              color="neutral"
              icon="i-tabler-edit"
              @click="$emit('edit', s)"
            />
            <UButton
              size="xs"
              variant="ghost"
              color="error"
              icon="i-tabler-trash"
              @click="$emit('delete', s)"
            />
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { Schedule } from '@/lib/backend'

const { t } = useI18n()

defineProps<{ list: Schedule[] }>()
defineEmits<{ edit: [s: Schedule]; delete: [s: Schedule] }>()

function triggerLabel(s: Schedule): string {
  const tg = s.trigger
  switch (tg.kind) {
    case 'cron':
      if (tg.subKind === 'daily') return t('schedule.display.daily', { at: tg.at ?? '--:--' })
      if (tg.subKind === 'interval') return t('schedule.display.interval', { mins: tg.everyMinutes })
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
