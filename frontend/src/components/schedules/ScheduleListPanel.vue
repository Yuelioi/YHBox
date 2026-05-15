<template>
  <div class="space-y-3">
    <div
      v-if="list.length === 0"
      class="rounded-xl bg-default/50 border border-default/60 border-dashed py-12 px-6 text-center"
    >
      <UIcon name="i-tabler-clock" class="size-8 text-dimmed mx-auto mb-3" />
      <p class="text-sm text-muted">还没有计划</p>
      <p class="text-xs text-dimmed mt-1">
        计划绑定 cron / 热键 / 启动后一次，触发后顺序跑指定容器。
      </p>
    </div>
    <table v-else class="w-full text-sm">
      <thead class="text-xs text-dimmed uppercase tracking-wider border-b border-default">
        <tr>
          <th class="text-left p-2">名称</th>
          <th class="text-left p-2">触发</th>
          <th class="text-left p-2">容器数</th>
          <th class="text-left p-2">上次触发</th>
          <th class="text-left p-2">启用</th>
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
              {{ s.enabled ? '启用' : '停用' }}
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
import type { Schedule } from '@/lib/backend'

defineProps<{ list: Schedule[] }>()
defineEmits<{ edit: [s: Schedule]; delete: [s: Schedule] }>()

function triggerLabel(s: Schedule): string {
  const t = s.trigger
  switch (t.kind) {
    case 'cron':
      if (t.subKind === 'daily') return `每日 ${t.at ?? '--:--'}`
      if (t.subKind === 'interval') return `每 ${t.everyMinutes}m`
      return 'cron'
    case 'hotkey':
      return `热键 ${t.hotkey ?? ''}`
    case 'once':
      return '启动后一次'
    case 'manual':
      return '仅手动'
  }
  return t.kind
}
</script>
