<!-- 子图库快查 modal. 入口: toolbar 📚. -->
<template>
  <BaseModal
    v-model:open="modelOpen"
    :title="t('library.explorer.title')"
    icon="i-tabler-books"
    size="4xl"
  >
    <div class="space-y-3">
      <!-- Search -->
      <div class="flex items-center gap-3">
        <UInput
          ref="searchInputRef"
          v-model="query"
          :placeholder="t('library.explorer.search')"
          icon="i-tabler-search"
          size="sm"
          class="flex-1"
          @keydown.escape="modelOpen = false"
        />
        <span class="text-[10px] text-dimmed">{{ t('library.explorer.esc_close') }}</span>
      </div>

      <!-- List -->
      <div>
          <div
            v-if="filteredItems.length === 0"
            class="text-center text-xs text-dimmed py-8 italic"
          >
            <span v-if="lib.loading">{{ t('library.loading') }}</span>
            <span v-else-if="lib.subgraphs.length === 0"
              >{{ t('library.explorer.empty') }}</span
            >
            <span v-else>{{ t('library.explorer.no_match') }}</span>
          </div>

          <div v-else class="space-y-2">
            <!-- Group by primary tag -->
            <template v-for="group in groupedItems" :key="group.tag">
              <!-- Section header -->
              <div class="text-[10px] font-semibold text-dimmed uppercase tracking-wider px-1 pt-2 pb-0.5">
                {{ group.tag }}
              </div>
              <div
                v-for="item in group.items"
                :key="item.id"
                class="bg-elevated/30 hover:bg-elevated/60 rounded p-3 cursor-pointer"
                @click="onPick(item.id)"
              >
                <div class="flex items-start gap-2">
                  <UIcon name="i-tabler-package" class="size-4 text-primary mt-0.5 shrink-0" />
                  <div class="flex-1 min-w-0">
                    <div class="text-sm font-medium">{{ item.label }}</div>
                    <div
                      v-if="item.description"
                      class="text-[11px] text-dimmed mt-0.5 line-clamp-2"
                    >
                      {{ item.description }}
                    </div>
                    <div
                      v-if="item.tags && item.tags.length > 0"
                      class="flex flex-wrap gap-1 mt-1"
                    >
                      <span
                        v-for="t in item.tags"
                        :key="t"
                        class="px-1.5 py-0 bg-elevated/60 text-[9px] rounded text-dimmed"
                      >
                        {{ t }}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            </template>
          </div>
      </div>
    </div>
  </BaseModal>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLibraryStore } from '@/stores/library'
import { useDialogOpen } from '@/composables/editor/useDialogOpen'
import { useAutoFocusOnOpen } from '@/composables/editor/useAutoFocusOnOpen'
import BaseModal from '@/components/common/BaseModal.vue'
import type { Subgraph } from '@/lib/backend'

const { t } = useI18n()

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{
  'update:open': [v: boolean]
  'pick-subgraph': [libraryID: string]
}>()

const modelOpen = useDialogOpen(props, emit)

const query = ref('')
const searchInputRef = ref<any>(null)

const lib = useLibraryStore()

// Hydrate on mount; refresh when modal opens (cheap — backend caches).
async function refreshLibrary() {
  await lib.reload()
}

onMounted(() => refreshLibrary())
useAutoFocusOnOpen(modelOpen, searchInputRef, {
  onOpen: () => { void refreshLibrary(); query.value = '' },
})

const filteredItems = computed<Subgraph[]>(() => {
  const q = query.value.toLowerCase().trim()
  if (!q) return lib.subgraphs
  return lib.subgraphs.filter((item) => {
    const hay =
      `${item.label} ${item.description ?? ''} ${(item.tags ?? []).join(' ')}`.toLowerCase()
    return hay.includes(q)
  })
})

// Group by primary tag (first tag); untagged go under "(未分类)"
interface TagGroup {
  tag: string
  items: Subgraph[]
}

const groupedItems = computed<TagGroup[]>(() => {
  const map = new Map<string, Subgraph[]>()
  for (const item of filteredItems.value) {
    const primaryTag = (item.tags ?? [])[0] ?? t('library.explorer.uncategorized')
    if (!map.has(primaryTag)) map.set(primaryTag, [])
    map.get(primaryTag)!.push(item)
  }
  return Array.from(map.entries()).map(([tag, items]) => ({ tag, items }))
})

function onPick(libraryID: string) {
  emit('pick-subgraph', libraryID)
  modelOpen.value = false
}
</script>
