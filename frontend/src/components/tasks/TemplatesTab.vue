<template>
  <div class="space-y-4">
    <TemplateManager>
      <template #toolbar-right>
        <UButton color="primary" icon="i-tabler-camera" @click="onNewTemplate">截图新模板</UButton>
      </template>
    </TemplateManager>

    <p class="text-xs text-dimmed">
      全局模板库。容器节点 WaitTemplate / CheckTemplate / ClickTemplate 引用这里的 key。
    </p>
  </div>
</template>

<script setup lang="ts">
import { backend } from '@/lib/backend'
import { useTemplatesStore } from '@/stores/templates'
import { awaitWailsEvent } from '@/composables/useWailsEvent'

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
    await tplStore.reload()
  }
}
</script>
