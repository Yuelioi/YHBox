<template>
  <div class="flex flex-col h-full bg-default text-default">
    <TabToolbar :tabs="sourceTabs" v-model:activeTab="sourceTab">
      <template #actions>
        <UButton size="sm" variant="ghost" color="neutral" icon="i-tabler-refresh" @click="reload">
          刷新
        </UButton>
      </template>
    </TabToolbar>

    <div v-if="sourceTab === 'local'" class="flex-1 flex overflow-hidden min-h-0">
      <div class="flex-1 flex flex-col overflow-hidden min-h-0">
        <div class="px-4 pt-3 pb-2 flex items-center gap-2 shrink-0">
          <UInput
            v-model="search"
            placeholder="搜索子图..."
            icon="i-tabler-search"
            size="sm"
            class="flex-1"
          />
          <UButtonGroup size="sm">
            <UButton
              :variant="viewMode === 'grid' ? 'solid' : 'soft'"
              color="neutral"
              icon="i-tabler-layout-grid"
              title="网格视图"
              @click="setViewMode('grid')"
            />
            <UButton
              :variant="viewMode === 'list' ? 'solid' : 'soft'"
              color="neutral"
              icon="i-tabler-list"
              title="列表视图"
              @click="setViewMode('list')"
            />
          </UButtonGroup>
        </div>

        <div class="flex-1 overflow-y-auto px-4 pb-4">
          <div
            v-if="filtered.length > 0"
            :class="viewMode === 'grid' ? 'grid grid-cols-3 gap-3' : 'flex flex-col gap-1.5'"
          >
            <LibraryCard
              v-for="sgID in filtered"
              :key="sgID"
              :sg-i-d="sgID"
              :pkg="libraryStore.packages[sgID]"
              :selected="selectedSgID === sgID"
              :view-mode="viewMode"
              @select="selectedSgID = $event"
              @import="openImportDialog($event)"
            />
          </div>
          <p v-else class="text-xs text-dimmed text-center py-8">
            {{
              libraryStore.loading
                ? '加载中...'
                : search
                  ? '没有匹配的子图'
                  : '库还是空的，从容器编辑器把子图分享到库即可'
            }}
          </p>
        </div>
      </div>

      <LibraryDetailPanel
        :sg-i-d="selectedSgID"
        @import="openImportDialog($event)"
        @cleared="selectedSgID = null"
      />
    </div>

    <ComingSoonSection v-else-if="sourceTab === 'online'" />
  </div>

  <ImportToContainerDialog
    v-model:open="importDialog.open"
    :lib-sg-id="importDialog.sgID"
  />
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import TabToolbar from '@/components/common/TabToolbar.vue'
import LibraryCard from '@/components/library/LibraryCard.vue'
import LibraryDetailPanel from '@/components/library/LibraryDetailPanel.vue'
import ComingSoonSection from '@/components/library/ComingSoonSection.vue'
import ImportToContainerDialog from '@/components/library/ImportToContainerDialog.vue'
import { useLibraryStore } from '@/stores/library'
import { useLibraryViewMode } from '@/composables/useLibraryViewMode'

const libraryStore = useLibraryStore()
const { mode: viewMode, set: setViewMode } = useLibraryViewMode()

const selectedSgID = ref<string | null>(null)
const search = ref('')

const sourceTabs = [
  { label: '本机', value: 'local' },
  { label: '在线（即将上线）', value: 'online' },
]
const sourceTab = ref('local')

const filtered = computed<string[]>(() => {
  if (!search.value) return libraryStore.subgraphIds
  const q = search.value.toLowerCase()
  return libraryStore.subgraphIds.filter((id) => {
    const pkg = libraryStore.packages[id]
    if (!pkg) return id.toLowerCase().includes(q)
    const label = pkg.root.label ?? ''
    const desc = pkg.root.description ?? ''
    return (
      id.toLowerCase().includes(q) ||
      label.toLowerCase().includes(q) ||
      desc.toLowerCase().includes(q)
    )
  })
})

watch(() => libraryStore.subgraphIds, () => {
  if (selectedSgID.value && !libraryStore.subgraphIds.includes(selectedSgID.value)) {
    selectedSgID.value = null
  }
})

const importDialog = ref({ open: false, sgID: '' })

function openImportDialog(sgID: string) {
  importDialog.value = { open: true, sgID }
}

async function reload() {
  await libraryStore.reload()
}

onMounted(() => {
  void reload()
})
</script>
