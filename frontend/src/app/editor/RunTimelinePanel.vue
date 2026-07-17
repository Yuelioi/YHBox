<template>
  <section class="flex max-h-64 shrink-0 flex-col border-t border-default bg-default">
    <header class="flex items-center gap-3 border-b border-default px-4 py-2.5">
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <span class="size-2 rounded-full" :class="statusColor" aria-hidden="true" />
          <h2 class="text-xs font-semibold text-highlighted">
            {{ t('workflow.timeline.run_status', { status: run.status }) }}
          </h2>
        </div>
        <p class="mt-0.5 truncate font-mono text-[10px] text-dimmed">{{ run.runId }}</p>
      </div>
      <UButton
        v-if="canCancel"
        :label="t('workflow.action.stop')"
        icon="i-tabler-square"
        color="error"
        variant="soft"
        size="xs"
        @click="emit('cancel')"
      />
      <UButton
        :label="t('workflow.action.refresh')"
        icon="i-tabler-refresh"
        color="neutral"
        variant="ghost"
        size="xs"
        @click="emit('refresh')"
      />
      <UButton
        icon="i-tabler-x"
        color="neutral"
        variant="ghost"
        size="xs"
        :aria-label="t('workflow.timeline.close')"
        @click="emit('close')"
      />
    </header>

    <div class="min-h-0 flex-1 overflow-y-auto px-4 py-3">
      <div v-if="run.failure" class="mb-3 rounded-lg border border-error/35 bg-error/10 px-3 py-2">
        <p class="text-xs font-medium text-error">{{ run.failure.code }}</p>
        <p class="mt-1 text-[11px] text-muted">
          {{ run.failure.category }}{{ run.failure.nodeId ? ` / ${run.failure.nodeId}` : '' }}
        </p>
      </div>
      <p v-if="!run.timeline.length" class="py-3 text-center text-xs text-muted">
        {{ t('workflow.timeline.empty') }}
      </p>
      <ol v-else class="space-y-2">
        <li v-for="entry in run.timeline" :key="entry.sequence" class="rounded-lg bg-elevated/45">
          <UButton
            color="neutral"
            variant="ghost"
            class="grid h-auto w-full grid-cols-[72px_minmax(0,1fr)_auto] items-start justify-stretch gap-3 px-3 py-2 text-left"
            :disabled="!entry.nodeId"
            @click="entry.nodeId && emit('focus-node', entry.graphPath, entry.nodeId)"
          >
            <span class="font-mono text-[10px] text-dimmed"
              >#{{ entry.sequence }} {{ entry.kind }}</span
            >
            <span class="min-w-0">
              <span class="block truncate text-xs text-toned">{{
                entry.nodeId || entry.summary.code
              }}</span>
              <span
                v-if="entry.attemptOutcome || entry.action || entry.statusCode || entry.errorCode"
                class="mt-0.5 block truncate font-mono text-[10px] text-muted"
              >
                {{ entry.attemptOutcome || entry.action || entry.statusCode || entry.errorCode }}
              </span>
            </span>
            <span class="font-mono text-[10px] text-dimmed">
              {{ t('workflow.timeline.attempt', { n: entry.attempt }) }}
            </span>
          </UButton>
        </li>
      </ol>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { RunView } from '@/app/transport/workflow'

const props = defineProps<{ run: RunView }>()
const emit = defineEmits<{
  cancel: []
  refresh: []
  close: []
  'focus-node': [graphPath: string[], nodeId: string]
}>()
const { t } = useI18n()

const canCancel = computed(() => ['QUEUED', 'RUNNING'].includes(props.run.status.toUpperCase()))
const statusColor = computed(() => {
  switch (props.run.status.toUpperCase()) {
    case 'SUCCEEDED':
      return 'bg-success'
    case 'FAILED':
    case 'INTERRUPTED':
      return 'bg-error'
    case 'CANCELLED':
      return 'bg-warning'
    default:
      return 'bg-primary animate-pulse motion-reduce:animate-none'
  }
})
</script>
