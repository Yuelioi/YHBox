<template>
  <div class="flex flex-col h-full bg-default text-default">
    <TabToolbar :tabs="sourceTabs" v-model:activeTab="sourceTab">
      <template #actions>
        <UButton size="sm" variant="ghost" color="neutral" icon="i-tabler-refresh" @click="reload">
          {{ t('library.refresh') }}
        </UButton>
      </template>
    </TabToolbar>

    <div v-if="sourceTab === 'local'" class="flex-1 flex overflow-hidden min-h-0">
      <div class="flex-1 flex flex-col overflow-hidden min-h-0">
        <div class="px-4 pt-3 pb-2 flex items-center gap-2 shrink-0">
          <UInput
            v-model="search"
            :placeholder="t('library.search_placeholder')"
            icon="i-tabler-search"
            size="sm"
            class="flex-1"
          />
          <UButtonGroup size="sm">
            <UButton
              :variant="viewMode === 'grid' ? 'solid' : 'soft'"
              color="neutral"
              icon="i-tabler-layout-grid"
              :title="t('library.view_grid')"
              @click="setViewMode('grid')"
            />
            <UButton
              :variant="viewMode === 'list' ? 'solid' : 'soft'"
              color="neutral"
              icon="i-tabler-list"
              :title="t('library.view_list')"
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
              v-for="sg in filtered"
              :key="sg.id"
              :sg="sg"
              :selected="selectedSgID === sg.id"
              :view-mode="viewMode"
              @select="selectedSgID = $event"
            />
          </div>
          <p v-else class="text-xs text-dimmed text-center py-8">
            {{
              libraryStore.loading
                ? t('library.loading')
                : search
                  ? t('library.no_match')
                  : t('library.empty')
            }}
          </p>
        </div>
      </div>

      <LibraryDetailPanel
        :sg-i-d="selectedSgID"
        @cleared="selectedSgID = null"
      />
    </div>

    <ComingSoonSection v-else-if="sourceTab === 'online'" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import TabToolbar from '@/components/common/TabToolbar.vue'

const { t } = useI18n()
import LibraryCard from '@/components/library/LibraryCard.vue'
import LibraryDetailPanel from '@/components/library/LibraryDetailPanel.vue'
import ComingSoonSection from '@/components/library/ComingSoonSection.vue'
import { useLibraryStore } from '@/stores/library'
import { useLibraryViewMode } from '@/composables/useLibraryViewMode'
import type { Subgraph } from '@/lib/backend'

const libraryStore = useLibraryStore()
const { mode: viewMode, set: setViewMode } = useLibraryViewMode()

const selectedSgID = ref<string | null>(null)
const search = ref('')

const sourceTabs = computed(() => [
  { label: t('library.tab.local'), value: 'local' },
  { label: t('library.tab.online'), value: 'online' },
])
const sourceTab = ref('local')

const filtered = computed<Subgraph[]>(() => {
  if (!search.value) return libraryStore.subgraphs
  const q = search.value.toLowerCase()
  return libraryStore.subgraphs.filter((sg) => {
    const hay = `${sg.id} ${sg.label ?? ''} ${sg.description ?? ''} ${(sg.tags ?? []).join(' ')}`.toLowerCase()
    return hay.includes(q)
  })
})

watch(() => libraryStore.subgraphs, (list) => {
  if (selectedSgID.value && !list.some((s) => s.id === selectedSgID.value)) {
    selectedSgID.value = null
  }
})

async function reload() {
  await libraryStore.reload()
}

onMounted(() => {
  void reload()
})
</script>
