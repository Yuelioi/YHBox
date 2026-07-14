<!-- 资产浏览宿主：模板 / 蓝图 / Clip 是产品导航，维护能力作为独立工具入口。
     dock 使用紧凑浏览，workspace 展开为分类、内容、详情三栏。 -->
<template>
  <div class="asset-dock flex h-full min-h-0 flex-col" :data-workspace="workspace">
    <header class="shrink-0 border-b border-default bg-elevated/15 px-3 py-2.5">
      <div class="mb-2.5 flex items-center gap-3">
        <div class="flex min-w-0 flex-1 items-center gap-2">
          <div
            class="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary"
          >
            <UIcon name="i-tabler-folders" class="size-4.5" />
          </div>
          <div class="min-w-0">
            <p class="truncate text-sm font-semibold text-highlighted">
              {{ t('assetBrowser.workspaceTitle') }}
            </p>
            <p class="truncate text-xs text-dimmed">{{ t('assetBrowser.workspaceSubtitle') }}</p>
          </div>
        </div>

        <UButton
          v-if="tab !== 'maintenance'"
          size="xs"
          variant="ghost"
          color="neutral"
          icon="i-tabler-database-cog"
          class="size-7 shrink-0 p-0"
          :aria-label="t('assetMaintenance.title')"
          :title="t('assetMaintenance.title')"
          @click="emit('update:tab', 'maintenance')"
        />
        <UButton
          v-if="!workspace"
          size="xs"
          variant="ghost"
          color="neutral"
          icon="i-tabler-arrows-maximize"
          class="size-7 shrink-0 p-0"
          :aria-label="t('assetBrowser.openWorkspace')"
          :title="t('assetBrowser.openWorkspace')"
          @click="emit('open-workspace')"
        />
        <UButton
          v-else
          size="xs"
          variant="ghost"
          color="neutral"
          icon="i-tabler-x"
          class="size-7 shrink-0 p-0"
          :aria-label="t('common.close')"
          @click="emit('close-workspace')"
        />
      </div>

      <div v-if="tab === 'maintenance'" class="flex items-center gap-2">
        <UButton
          size="xs"
          variant="ghost"
          color="neutral"
          icon="i-tabler-arrow-left"
          @click="emit('update:tab', 'templates')"
        >
          {{ t('assetBrowser.backToAssets') }}
        </UButton>
        <span class="text-xs text-muted">{{ t('assetMaintenance.title') }}</span>
      </div>
      <UTabs
        v-else
        :model-value="tab"
        :items="tabItems"
        :content="false"
        size="sm"
        class="min-w-0"
        @update:model-value="(v: string | number) => emit('update:tab', v as AssetTab)"
      />
    </header>
    <div class="flex-1 min-h-0 overflow-hidden">
      <TemplateAssetPanel
        v-if="tab === 'templates'"
        :pick-mode="templatePickMode"
        :model-value="templateSelected"
        :workspace="workspace"
        @update:model-value="(v: string[]) => emit('update:template-selected', v)"
      />
      <LibraryAssetPanel
        v-else-if="tab === 'library'"
        :workspace="workspace"
        @pick-subgraph="(id: string) => emit('pick-subgraph', id)"
      />
      <ClipAssetPanel
        v-else-if="tab === 'clips'"
        :workspace="workspace"
        @pick-clip="(id: string) => emit('pick-clip', id)"
      />
      <AssetMaintenancePanel v-else-if="tab === 'maintenance'" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import TemplateAssetPanel from './TemplateAssetPanel.vue'
import LibraryAssetPanel from './LibraryAssetPanel.vue'
import ClipAssetPanel from './ClipAssetPanel.vue'
import AssetMaintenancePanel from './AssetMaintenancePanel.vue'
import { useTemplatesStore } from '@/stores/templates'
import { useLibraryStore } from '@/stores/library'
import { useClipsStore } from '@/stores/clips'

type AssetTab = 'templates' | 'library' | 'clips' | 'maintenance'

defineProps<{
  tab: AssetTab
  templatePickMode: boolean
  templateSelected: string[]
  workspace?: boolean
}>()
const emit = defineEmits<{
  'update:tab': [v: AssetTab]
  'update:template-selected': [v: string[]]
  'pick-subgraph': [id: string]
  'pick-clip': [id: string]
  'open-workspace': []
  'close-workspace': []
}>()

const { t } = useI18n()
const templates = useTemplatesStore()
const library = useLibraryStore()
const clips = useClipsStore()
const tabItems = computed(() => [
  {
    value: 'templates',
    label: `${t('assetBrowser.visualTemplates')} ${Object.keys(templates.map).length}`,
    icon: 'i-tabler-photo',
  },
  {
    value: 'library',
    label: `${t('assetBrowser.automationBlueprints')} ${library.subgraphs.length}`,
    icon: 'i-tabler-hierarchy',
  },
  {
    value: 'clips',
    label: `${t('assetBrowser.actionClips')} ${clips.clips.length}`,
    icon: 'i-tabler-player-record',
  },
])

onMounted(() => {
  void Promise.allSettled([templates.reload(), library.reload(), clips.refresh()])
})
</script>

<style scoped>
.asset-dock {
  container-type: inline-size;
}

@container (width < 520px) {
  .asset-dock :deep([role='tab']) {
    padding-inline: 0.55rem;
  }
}
</style>
