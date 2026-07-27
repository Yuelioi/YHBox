<template>
  <div
    class="flex h-8 shrink-0 select-none items-center justify-between border-t border-default bg-default px-4 text-xs text-muted"
  >
    <div class="flex min-w-0 flex-1 items-center gap-3">
      <span class="size-1.5 shrink-0 rounded-full" :class="statusDot" aria-hidden="true" />
      <span
        role="status"
        aria-live="polite"
        class="min-w-0 truncate font-medium"
        :class="active ? 'text-primary' : 'text-dimmed'"
      >
        {{ label }}
      </span>
      <UButton
        v-if="active"
        :label="t('workflow.action.stop_all')"
        icon="i-tabler-square"
        color="error"
        variant="ghost"
        size="xs"
        class="ml-auto"
        :loading="stopBusy"
        @click="stopAll"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { onRunChanged, workflowTransport } from '@/app/transport/workflow'

const activeRunIds = ref(new Set<string>())
const stopBusy = ref(false)
const { t } = useI18n()
const active = computed(() => activeRunIds.value.size > 0)
const label = computed(() =>
  active.value
    ? t('workflow.status.active_count', { n: activeRunIds.value.size })
    : t('workflow.status.ready'),
)
const statusDot = computed(() =>
  active.value ? 'bg-primary animate-pulse motion-reduce:animate-none' : 'bg-accented',
)

const unsubscribe = onRunChanged((event) => {
  if (!event.runId) return
  const next = new Set(activeRunIds.value)
  if (['queued', 'running'].includes(event.status.toLowerCase())) next.add(event.runId)
  else next.delete(event.runId)
  activeRunIds.value = next
})

onBeforeUnmount(unsubscribe)

async function stopAll(): Promise<void> {
  if (stopBusy.value) return
  stopBusy.value = true
  try {
    await workflowTransport.cancelAllRuns()
  } finally {
    stopBusy.value = false
  }
}
</script>
