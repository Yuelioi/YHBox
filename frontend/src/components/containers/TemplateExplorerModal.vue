<!-- 模板管理 modal (编辑器内入口: 左 rail 📷). 复用全局 TemplateManager + 顶部「新建截图」.
     模板是全局资产; tplStore.containerId(编辑器 setContainer 注入)仅供截图定位当前容器目标窗口。 -->
<template>
  <BaseModal
    v-model:open="modelOpen"
    :title="t('template.manager.title')"
    icon="i-tabler-photo"
    size="5xl"
  >
    <TemplateManager>
      <template #toolbar-right>
        <UButton color="primary" icon="i-tabler-camera" size="xs" @click="onNewTemplate">
          {{ t('template.capture.title') }}
        </UButton>
      </template>
    </TemplateManager>
  </BaseModal>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { backend } from '@/lib/backend'
import { useTemplatesStore } from '@/stores/templates'
import { useDialogOpen } from '@/composables/editor/useDialogOpen'
import { awaitWailsEvent } from '@/composables/useWailsEvent'
import BaseModal from '@/components/common/BaseModal.vue'
import TemplateManager from '@/components/templates/TemplateManager.vue'

const { t } = useI18n()
const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ 'update:open': [v: boolean] }>()
const modelOpen = useDialogOpen(props, emit)

const tplStore = useTemplatesStore()

// 搬自原 TemplatesTab.onNewTemplate — 开 ScreenPicker 截图存模板, 完事 reload 全局列表.
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
