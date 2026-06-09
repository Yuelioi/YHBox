<template>
  <div class="space-y-4">
    <TemplateManager>
      <template #toolbar-right>
        <UButton color="primary" icon="i-tabler-camera" @click="onNewTemplate">{{ t('template.capture.title') }}</UButton>
      </template>
    </TemplateManager>

    <p class="text-xs text-dimmed">
      {{ t('template.tab_intro') }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { backend } from '@/lib/backend'
import { useTemplatesStore } from '@/stores/templates'
import { awaitWailsEvent } from '@/composables/useWailsEvent'

const { t } = useI18n()

const tplStore = useTemplatesStore()

async function onNewTemplate() {
  const id = 'tpl-' + Math.random().toString(36).slice(2, 10) + '-' + Date.now()
  const waiter = awaitWailsEvent<{ id: string; mode: string; payload: any }>(
    'tools:picker-result',
    (p) => p?.id === id,
  )
  await backend.tools.openScreenPicker('template_save', id, tplStore.containerId)
  const result = await waiter
  if (!result.payload?.cancelled) {
    // payload.guid (新) 或 payload.cancelled; reload 刷新全局列表
    await tplStore.reload()
  }
}
</script>
