<template>
  <aside v-if="summary.visible" class="debug-panel pointer-events-auto" :class="toneClass">
    <div class="flex items-center gap-2 min-w-0">
      <UIcon name="i-tabler-bug" class="size-4 shrink-0" />
      <span class="text-xs font-semibold">{{ t('editor.debug_panel.title') }}</span>
      <UBadge size="xs" :color="badgeColor" variant="soft">{{ t(summary.statusKey) }}</UBadge>
      <span v-if="summary.queueCount > 0" class="ml-auto text-[10px] text-dimmed">
        {{ t('editor.debug_panel.queue_count', { n: summary.queueCount }) }}
      </span>
      <span v-else class="ml-auto" />
      <UButton
        size="xs"
        color="neutral"
        variant="ghost"
        icon="i-tabler-x"
        :title="t('editor.debug_panel.close_tip')"
        @click="$emit('stop')"
      />
    </div>

    <div v-if="summary.focusNodeID" class="debug-row">
      <span class="debug-key">{{ t(summary.focusRoleKey) }}</span>
      <span class="debug-val">
        {{ kindLabel(summary.focusNodeKind) }}
        <code class="debug-code">{{ summary.focusNodeID }}</code>
      </span>
    </div>

    <div v-if="summary.lastNodeID" class="debug-row">
      <span class="debug-key">{{ t('editor.debug_panel.last') }}</span>
      <span class="debug-val">
        {{ kindLabel(summary.lastNodeKind) }}
        <code class="debug-code">{{ summary.lastNodeID }}</code>
        <span v-if="summary.lastExit" class="text-dimmed">· {{ summary.lastExit }}</span>
      </span>
    </div>

    <div v-if="summary.lastOutputPreview" class="debug-row">
      <span class="debug-key">{{ t('editor.debug_panel.output') }}</span>
      <span class="debug-val">{{ summary.lastOutputPreview }}</span>
    </div>

    <div v-if="summary.varsPreview" class="debug-row">
      <span class="debug-key">{{ t('editor.debug_panel.vars') }}</span>
      <span class="debug-val">{{ summary.varsPreview }}</span>
    </div>

    <div v-if="summary.queuePreview" class="debug-row">
      <span class="debug-key">{{ t('editor.debug_panel.queue') }}</span>
      <span class="debug-val">{{ summary.queuePreview }}</span>
    </div>

    <div class="debug-note warning">
      <UIcon name="i-tabler-alert-triangle" class="size-3.5 shrink-0" />
      <span>{{ t('editor.debug_panel.side_effect_warning') }}</span>
    </div>

    <div v-if="summary.warnings.length > 0" class="debug-note warning">
      <UIcon name="i-tabler-alert-triangle" class="size-3.5 shrink-0" />
      <span>{{ warningText(summary.warnings[0]) }}</span>
    </div>

    <div v-if="summary.error" class="debug-note error">
      <UIcon name="i-tabler-circle-x" class="size-3.5 shrink-0" />
      <span>{{
        summary.error.message || summary.error.code || t('editor.debug_panel.unknown_error')
      }}</span>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useExecutionStore, type DebugSessionState, type DebugWarning } from '@/stores/execution'
import { summarizeDebugSession } from '@/composables/editor/debugPanel'
import { KIND_LABEL_ZH } from '@/components/containers/pinSpec'

const { t, te } = useI18n()
const execStore = useExecutionStore()
defineEmits<{ stop: [] }>()

const state = computed<Partial<DebugSessionState>>(() => ({
  sessionId: execStore.debugSessionID,
  containerId: execStore.debugContainerID,
  status: execStore.debugStatus,
  mode: execStore.debugMode,
  currentNodeId: execStore.debugCurrentNodeID || execStore.debugNextNodeID,
  currentNodeKind: execStore.debugCurrentNodeKind || execStore.debugNextNodeKind,
  runningNodeId: execStore.debugRunningNodeID,
  runningNodeKind: execStore.debugRunningNodeKind,
  lastNodeId: execStore.debugLastNodeID,
  lastNodeKind: execStore.debugLastNodeKind,
  lastExit: execStore.debugLastExit,
  lastOutput: execStore.debugLastOutput,
  vars: execStore.debugVars,
  queue: execStore.debugQueue,
  warnings: execStore.debugWarnings,
  error: execStore.debugError,
}))

const summary = computed(() => summarizeDebugSession(state.value))

const badgeColor = computed(() => {
  const tone = summary.value.tone
  return tone === 'primary' || tone === 'warning' || tone === 'error' || tone === 'success'
    ? tone
    : 'neutral'
})

const toneClass = computed(() => `tone-${summary.value.tone}`)

function kindLabel(kind: string): string {
  if (!kind) return ''
  const key = KIND_LABEL_ZH[kind]
  return key ? t(key) : kind
}

function warningText(w: DebugWarning): string {
  const key = `editor.debug_panel.warning.${w.code}`
  return te(key) ? t(key) : w.message || w.code
}
</script>

<style scoped>
.debug-panel {
  position: absolute;
  left: 12px;
  top: 12px;
  z-index: 22;
  width: min(360px, calc(100% - 24px));
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px 12px;
  border: 1px solid var(--ui-border);
  border-radius: 8px;
  background: color-mix(in oklab, var(--ui-bg) 88%, transparent);
  color: var(--ui-text-default);
  box-shadow: 0 12px 36px rgba(0, 0, 0, 0.32);
  backdrop-filter: blur(8px);
}
.debug-panel.tone-primary {
  border-color: color-mix(in oklab, var(--ui-primary) 48%, var(--ui-border));
}
.debug-panel.tone-warning {
  border-color: color-mix(in oklab, var(--ui-warning) 55%, var(--ui-border));
}
.debug-panel.tone-error {
  border-color: color-mix(in oklab, var(--ui-error) 60%, var(--ui-border));
}
.debug-panel.tone-success {
  border-color: color-mix(in oklab, var(--ui-success) 50%, var(--ui-border));
}
.debug-row {
  display: grid;
  grid-template-columns: 64px minmax(0, 1fr);
  gap: 8px;
  align-items: baseline;
  font-size: 11px;
}
.debug-key {
  color: var(--ui-text-dimmed);
}
.debug-val {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--ui-text-toned);
}
.debug-code {
  margin-left: 6px;
  padding: 1px 4px;
  border-radius: 4px;
  background: color-mix(in oklab, var(--ui-bg-elevated) 70%, transparent);
  color: var(--ui-text-dimmed);
  font-size: 10px;
}
.debug-note {
  display: flex;
  gap: 6px;
  align-items: flex-start;
  font-size: 11px;
  line-height: 1.45;
}
.debug-note.warning {
  color: var(--ui-warning);
}
.debug-note.error {
  color: var(--ui-error);
}
</style>
