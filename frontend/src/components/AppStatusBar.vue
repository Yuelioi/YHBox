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
      <span v-if="activeRunId" class="truncate font-mono text-[10px] text-dimmed">{{
        activeRunId
      }}</span>
      <UButton
        v-if="active"
        :label="t('workflow31.action.stop_all')"
        icon="i-tabler-square"
        color="error"
        variant="ghost"
        size="xs"
        class="ml-auto"
        @click="stopAll"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { onRunChanged, workflowTransport } from '@/app/transport/workflow31'

const activeRunId = ref('')
const status = ref('')
const { t } = useI18n()
const active = computed(() => ['QUEUED', 'RUNNING'].includes(status.value.toUpperCase()))
const label = computed(() =>
  active.value
    ? t('workflow31.status.program', { status: status.value.toLowerCase() })
    : t('workflow31.status.ready'),
)
const statusDot = computed(() =>
  active.value ? 'bg-primary animate-pulse motion-reduce:animate-none' : 'bg-accented',
)

const unsubscribe = onRunChanged((event) => {
  if (event.status) status.value = event.status
  if (active.value) activeRunId.value = event.runId
  else activeRunId.value = ''
})

onBeforeUnmount(unsubscribe)

async function stopAll(): Promise<void> {
  await workflowTransport.cancelAllRuns()
}
</script>
