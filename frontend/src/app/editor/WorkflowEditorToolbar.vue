<template>
  <header class="flex h-13 shrink-0 items-center gap-2 border-b border-default bg-default px-3">
    <UButton
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
    <span v-if="dirty" class="text-[11px] font-medium text-warning">{{
      t('workflow.editor.unsaved')
    }}</span>
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
      :label="t('workflow.action.compile')"
      icon="i-tabler-file-check"
      color="neutral"
      variant="ghost"
      size="xs"
      @click="emit('compile')"
    />
    <UButton
      :label="t('workflow.action.debug')"
      icon="i-tabler-bug"
      color="neutral"
      variant="soft"
      size="xs"
      @click="emit('debug')"
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
      :label="t('workflow.action.run')"
      icon="i-tabler-player-play"
      size="xs"
      @click="emit('run')"
    />
    <UButton
      :label="t('workflow.action.save')"
      icon="i-tabler-device-floppy"
      size="xs"
      :loading="saving"
      :disabled="!dirty"
      @click="emit('save')"
    />
  </header>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{
  name: string
  revision: number
  dirty: boolean
  canUndo: boolean
  canRedo: boolean
  aiPanelOpen: boolean
  runActive: boolean
  saving: boolean
}>()
const emit = defineEmits<{
  back: []
  rename: [name: string]
  undo: []
  redo: []
  'toggle-ai': []
  compile: []
  debug: []
  run: []
  stop: []
  save: []
}>()
const { t } = useI18n()

function rename(event: Event): void {
  emit('rename', (event.target as HTMLInputElement).value)
}
</script>
