<!-- 资产停靠面板宿主: UTabs 收 模板 / 子图库 / Clip / 资源管理 (替原 3 个 5xl modal).
     宽态停靠 (~600px), 挤画布不盖。模板支持 pick 模式 (从节点字段唤起, 实时回写)。 -->
<template>
  <div class="flex h-full min-h-0 flex-col">
    <UTabs
      :model-value="tab"
      :items="tabItems"
      :content="false"
      size="sm"
      class="min-w-0 shrink-0 px-2 pt-2"
      @update:model-value="(v: string | number) => emit('update:tab', v as AssetTab)"
    />
    <div class="flex-1 min-h-0 overflow-hidden">
      <TemplateAssetPanel
        v-if="tab === 'templates'"
        :pick-mode="templatePickMode"
        :model-value="templateSelected"
        @update:model-value="(v: string[]) => emit('update:template-selected', v)"
      />
      <LibraryAssetPanel
        v-else-if="tab === 'library'"
        @pick-subgraph="(id: string) => emit('pick-subgraph', id)"
      />
      <ClipAssetPanel
        v-else-if="tab === 'clips'"
        @pick-clip="(id: string) => emit('pick-clip', id)"
      />
      <AssetMaintenancePanel
        v-else-if="tab === 'maintenance'"
        @navigate="(target) => emit('update:tab', target)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import TemplateAssetPanel from './TemplateAssetPanel.vue'
import LibraryAssetPanel from './LibraryAssetPanel.vue'
import ClipAssetPanel from './ClipAssetPanel.vue'
import AssetMaintenancePanel from './AssetMaintenancePanel.vue'

type AssetTab = 'templates' | 'library' | 'clips' | 'maintenance'

defineProps<{
  tab: AssetTab
  templatePickMode: boolean
  templateSelected: string[]
}>()
const emit = defineEmits<{
  'update:tab': [v: AssetTab]
  'update:template-selected': [v: string[]]
  'pick-subgraph': [id: string]
  'pick-clip': [id: string]
}>()

const { t } = useI18n()
const tabItems = computed(() => [
  { value: 'templates', label: t('template.manager.title'), icon: 'i-tabler-photo' },
  { value: 'library', label: t('editor.toolbar.library_explorer'), icon: 'i-tabler-books' },
  { value: 'clips', label: t('clip.manager.title'), icon: 'i-tabler-movie' },
  { value: 'maintenance', label: t('assetMaintenance.title'), icon: 'i-tabler-database-cog' },
])
</script>
