<template>
  <header
    class="flex h-13 shrink-0 items-center gap-2 overflow-x-auto whitespace-nowrap border-b border-default bg-default px-3"
  >
    <UButton
      data-testid="workflow-editor-back"
      icon="i-tabler-arrow-left"
      color="neutral"
      variant="ghost"
      size="xs"
      :aria-label="t('workflow.editor.back')"
      @click="emit('back')"
    />
    <UInput
      :model-value="name"
      class="w-56"
      :aria-label="t('workflow.editor.workflow_name')"
      @change="rename"
    />
    <span class="font-mono text-[10px] text-dimmed">{{
      t('workflow.editor.revision', { n: revision })
    }}</span>
    <span
      v-if="dirty"
      data-testid="workflow-unsaved"
      class="text-[11px] font-medium text-warning"
      >{{ t('workflow.editor.unsaved') }}</span
    >
    <div class="mx-2 h-5 w-px bg-default" />
    <UButton
      icon="i-tabler-arrow-back-up"
      color="neutral"
      variant="ghost"
      size="xs"
      :disabled="!canUndo"
      :aria-label="t('workflow.action.undo')"
      @click="emit('undo')"
    />
    <UButton
      icon="i-tabler-arrow-forward-up"
      color="neutral"
      variant="ghost"
      size="xs"
      :disabled="!canRedo"
      :aria-label="t('workflow.action.redo')"
      @click="emit('redo')"
    />
    <UButton
      data-testid="workflow-find-node"
      :label="t('workflow.node_search.action')"
      icon="i-tabler-search"
      color="neutral"
      variant="ghost"
      size="xs"
      :title="t('workflow.node_search.shortcut')"
      @click="emit('find-node')"
    />
    <div class="flex-1" />
    <UButton
      data-testid="ai-workflow-review-open"
      :label="t('workflow.ai.open')"
      icon="i-tabler-sparkles"
      color="neutral"
      :variant="aiPanelOpen ? 'soft' : 'ghost'"
      size="xs"
      :aria-pressed="aiPanelOpen"
      @click="emit('toggle-ai')"
    />
    <UButton
      data-testid="workflow-state-open"
      :label="t('workflow.inspector.state_title')"
      icon="i-tabler-database"
      color="neutral"
      :variant="statePanelOpen ? 'soft' : 'ghost'"
      size="xs"
      :aria-pressed="statePanelOpen"
      @click="emit('toggle-state')"
    />
    <template v-if="recordingPhase !== 'idle' && recordingPhase !== 'pending'">
      <UButton
        :label="
          recordingPhase === 'paused'
            ? t('workflow.recording.resume')
            : t('workflow.recording.pause')
        "
        :icon="recordingPhase === 'paused' ? 'i-tabler-player-play' : 'i-tabler-player-pause'"
        color="warning"
        variant="soft"
        size="xs"
        :disabled="recordingPhase === 'finalizing'"
        @click="toggleRecordingPause"
      />
      <UButton
        data-testid="workflow-recording-stop"
        :label="t('workflow.recording.finish')"
        icon="i-tabler-square"
        color="error"
        variant="soft"
        size="xs"
        :loading="recordingPhase === 'finalizing'"
        @click="emit('stop-recording')"
      />
    </template>
    <UButton
      v-else-if="recordingPhase === 'pending'"
      :label="t('recordingSave.pending')"
      icon="i-tabler-clock-edit"
      color="warning"
      variant="soft"
      size="xs"
      disabled
    />
    <UButton
      :label="
        compileSucceeded ? t('workflow.action.compile_succeeded') : t('workflow.action.compile')
      "
      :icon="compileSucceeded ? 'i-tabler-check' : 'i-tabler-file-check'"
      :color="compileSucceeded ? 'success' : 'neutral'"
      :variant="compileSucceeded ? 'soft' : 'ghost'"
      size="xs"
      @click="emit('compile')"
    />
    <UButton
      v-if="diagnosticCount"
      :label="t('workflow.diagnostics.badge', { n: diagnosticCount })"
      icon="i-tabler-alert-triangle"
      color="warning"
      :variant="diagnosticsOpen ? 'soft' : 'ghost'"
      size="xs"
      :aria-pressed="diagnosticsOpen"
      @click="emit('toggle-diagnostics')"
    />
    <UButton
      v-if="hasRunTimeline"
      :label="t('workflow.timeline.open')"
      icon="i-tabler-timeline-event"
      color="neutral"
      :variant="runTimelineOpen ? 'soft' : 'ghost'"
      size="xs"
      :aria-pressed="runTimelineOpen"
      @click="emit('toggle-timeline')"
    />
    <UButton
      v-if="hasDebug"
      :label="t('workflow.debug.title')"
      icon="i-tabler-bug"
      color="warning"
      :variant="debuggerOpen ? 'soft' : 'ghost'"
      size="xs"
      :aria-pressed="debuggerOpen"
      @click="emit('toggle-debugger')"
    />
    <UButton
      v-if="!runActive"
      data-testid="workflow-debug-start"
      :label="t('workflow.debug.start')"
      icon="i-tabler-bug"
      color="neutral"
      variant="soft"
      size="xs"
      @click="emit('start-debug')"
    />
    <UButton
      v-if="runActive"
      :label="t('workflow.action.stop')"
      icon="i-tabler-square"
      color="error"
      variant="soft"
      size="xs"
      @click="emit('stop')"
    />
    <UButton
      v-else
      data-testid="workflow-run-timeline"
      :label="t('workflow.action.run_timeline')"
      icon="i-tabler-player-play"
      size="xs"
      @click="emit('run')"
    />
    <UButton
      data-testid="workflow-save"
      :label="saveSucceeded ? t('workflow.action.saved') : t('workflow.action.save')"
      :icon="saveSucceeded ? 'i-tabler-check' : 'i-tabler-device-floppy'"
      :color="saveSucceeded ? 'success' : 'neutral'"
      variant="soft"
      size="xs"
      :loading="saving"
      :disabled="!dirty"
      @click="emit('save')"
    />
  </header>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  name: string
  revision: number
  dirty: boolean
  canUndo: boolean
  canRedo: boolean
  aiPanelOpen: boolean
  statePanelOpen: boolean
  runActive: boolean
  saving: boolean
  compileSucceeded: boolean
  saveSucceeded: boolean
  diagnosticCount: number
  diagnosticsOpen: boolean
  hasRunTimeline: boolean
  runTimelineOpen: boolean
  hasDebug: boolean
  debuggerOpen: boolean
  recordingPhase: 'idle' | 'recording' | 'paused' | 'finalizing' | 'pending'
}>()
const emit = defineEmits<{
  back: []
  rename: [name: string]
  undo: []
  redo: []
  'find-node': []
  'toggle-ai': []
  'toggle-state': []
  compile: []
  'toggle-diagnostics': []
  'toggle-timeline': []
  'toggle-debugger': []
  'start-debug': []
  'start-recording': [mode: 'simple' | 'precise']
  'pause-recording': []
  'resume-recording': []
  'stop-recording': []
  run: []
  stop: []
  save: []
}>()
const { t } = useI18n()

function toggleRecordingPause(): void {
  if (props.recordingPhase === 'paused') {
    emit('resume-recording')
    return
  }
  emit('pause-recording')
}

function rename(event: Event): void {
  emit('rename', (event.target as HTMLInputElement).value)
}
</script>
