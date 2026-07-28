<template>
  <section
    class="flex shrink-0 flex-col border-t border-default bg-default transition-[height] duration-200 ease-out motion-reduce:transition-none"
    :style="{
      height: open
        ? expanded
          ? 'min(70vh, calc(100vh - 96px))'
          : 'clamp(220px, 32vh, 380px)'
        : '32px',
    }"
    data-testid="workflow-runtime-workbench"
  >
    <header class="flex h-8 shrink-0 items-center border-b border-default px-2">
      <div class="flex h-full min-w-0 flex-1 items-center gap-1" role="tablist">
        <UButton
          :label="t('workflow.workbench.diagnostics')"
          icon="i-tabler-alert-triangle"
          color="neutral"
          :variant="open && tab === 'diagnostics' ? 'soft' : 'ghost'"
          size="xs"
          role="tab"
          :aria-selected="open && tab === 'diagnostics'"
          :disabled="!diagnostics.length"
          @click="activate('diagnostics')"
        />
        <UButton
          :label="t('workflow.workbench.logs')"
          icon="i-tabler-terminal-2"
          color="neutral"
          :variant="open && tab === 'logs' ? 'soft' : 'ghost'"
          size="xs"
          role="tab"
          :aria-selected="open && tab === 'logs'"
          @click="activate('logs')"
        />
        <UButton
          :label="t('workflow.workbench.timeline')"
          icon="i-tabler-timeline-event"
          color="neutral"
          :variant="open && tab === 'timeline' ? 'soft' : 'ghost'"
          size="xs"
          role="tab"
          :aria-selected="open && tab === 'timeline'"
          :disabled="!run"
          @click="activate('timeline')"
        />
        <UButton
          :label="t('workflow.workbench.debug')"
          icon="i-tabler-bug"
          color="neutral"
          :variant="open && tab === 'debug' ? 'soft' : 'ghost'"
          size="xs"
          role="tab"
          :aria-selected="open && tab === 'debug'"
          :disabled="!snapshot"
          @click="activate('debug')"
        />
      </div>
      <UButton
        v-if="open"
        :label="t(expanded ? 'workflow.workbench.restore' : 'workflow.workbench.expand')"
        :icon="expanded ? 'i-tabler-arrows-minimize' : 'i-tabler-arrows-maximize'"
        color="neutral"
        variant="ghost"
        size="xs"
        :aria-pressed="expanded"
        @click="expanded = !expanded"
      />
      <UButton
        :icon="open ? 'i-tabler-chevron-down' : 'i-tabler-chevron-up'"
        color="neutral"
        variant="ghost"
        size="xs"
        :aria-label="t(open ? 'workflow.workbench.close' : 'workflow.workbench.open')"
        @click="emit('update:open', !open)"
      />
    </header>

    <div v-if="open" class="min-h-0 flex-1 overflow-hidden">
      <WorkflowDiagnosticsPanel
        v-if="tab === 'diagnostics' && diagnostics.length"
        :diagnostics="diagnostics"
        @focus="emit('focus', $event)"
        @close="emit('update:open', false)"
      />
      <LogPanel v-else-if="tab === 'logs'" embedded />
      <RunTimelinePanel
        v-else-if="tab === 'timeline' && run"
        embedded
        :run="run"
        :node-labels="nodeLabels"
        :unhandled-routes="unhandledRoutes"
        :exporting="timelineExporting"
        @cancel="emit('cancel')"
        @refresh="emit('refresh')"
        @export="emit('export-timeline')"
        @page="emit('page', $event)"
        @focus-node="(path, nodeId) => emit('focus-node', path, nodeId)"
      />
      <WorkflowDebuggerPanel
        v-else-if="tab === 'debug' && snapshot"
        embedded
        :snapshot="snapshot"
        :busy="debugBusy"
        :node-labels="nodeLabels"
        @continue="emit('continue')"
        @pause="emit('pause')"
        @step="emit('step')"
        @stop="emit('cancel')"
        @focus-node="(path, nodeId) => emit('focus-node', path, nodeId)"
      />
      <div v-else class="grid h-full place-items-center text-xs text-muted">
        {{ t('workflow.workbench.unavailable') }}
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { defineAsyncComponent, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DebugSnapshot, RunView } from '@/app/transport/workflow'
import type { WorkflowDiagnostic } from '@/app/editor/workflowDiagnostics'
import RunTimelinePanel from './RunTimelinePanel.vue'
import WorkflowDebuggerPanel from './WorkflowDebuggerPanel.vue'

type WorkbenchTab = 'diagnostics' | 'logs' | 'timeline' | 'debug'

const LogPanel = defineAsyncComponent(() => import('@/components/LogPanel.vue'))
const WorkflowDiagnosticsPanel = defineAsyncComponent(
  () => import('./WorkflowDiagnosticsPanel.vue'),
)
const props = defineProps<{
  open: boolean
  tab: WorkbenchTab
  run?: RunView | null
  snapshot?: DebugSnapshot | null
  debugBusy?: boolean
  nodeLabels?: Record<string, string>
  unhandledRoutes?: string[]
  timelineExporting?: boolean
  diagnostics: WorkflowDiagnostic[]
}>()
const emit = defineEmits<{
  'update:open': [open: boolean]
  'update:tab': [tab: WorkbenchTab]
  cancel: []
  refresh: []
  page: [page: number]
  'export-timeline': []
  continue: []
  pause: []
  step: []
  'focus-node': [graphPath: string[], nodeId: string]
  focus: [diagnostic: WorkflowDiagnostic]
}>()
const { t } = useI18n()
const expanded = ref(false)

function activate(tab: WorkbenchTab): void {
  if (props.open && props.tab === tab) {
    emit('update:open', false)
    return
  }
  emit('update:tab', tab)
  emit('update:open', true)
}
</script>
