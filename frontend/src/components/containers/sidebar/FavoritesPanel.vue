<!-- frontend/src/components/containers/sidebar/FavoritesPanel.vue -->
<!-- Editor v2 B — real Favorites panel (replaces FavoritesPanelPlaceholder).
     Drag-out → build node via useEditorDragDrop ('node-spec' payload).
     Spec: editor-v2-node-discovery-design.md §6.3. -->
<template>
  <SidebarSection
    title="收藏"
    icon="i-tabler-star"
    title-color="amber"
    :count="store.favorites.length"
    :expanded="expanded"
    @update:expanded="$emit('update:expanded', $event)"
  >
    <p v-if="store.favorites.length === 0" class="text-[10px] text-dimmed italic px-1">
      暂无收藏. 在节点 Explorer 中 ☆ 加入.
    </p>
    <div v-else class="space-y-1">
      <div
        v-for="kind in store.favorites"
        :key="kind"
        class="px-2 py-1 bg-elevated/30 rounded text-[11px] flex items-center gap-2 cursor-grab hover:bg-elevated/50"
        draggable="true"
        :title="`拖到画布建 ${kind} 节点 · 点 ★ 移除收藏`"
        @dragstart="(e) => onDragStart(kind, e)"
      >
        <UIcon name="i-tabler-grip-vertical" class="size-3 text-dimmed" />
        <span class="text-amber-400">★</span>
        <span>{{ labelFor(kind) }}</span>
        <button
          type="button"
          class="ml-auto text-dimmed hover:text-amber-400 px-0.5"
          title="移除收藏"
          @click.stop="store.toggleFavorite(kind)"
        >
          <UIcon name="i-tabler-x" class="size-3" />
        </button>
      </div>
    </div>
  </SidebarSection>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useDiscoveryStore } from '@/stores/discovery'
import { startEditorDrag } from '@/composables/editor/useEditorDragDrop'
import SidebarSection from './SidebarSection.vue'
import { getSpec } from '@/components/containers/nodeRegistry/registry'

defineProps<{ expanded: boolean }>()
defineEmits<{ 'update:expanded': [v: boolean] }>()

const store = useDiscoveryStore()
onMounted(() => store.loadFromLocalStorage())

function onDragStart(kind: string, e: DragEvent) {
  startEditorDrag({ type: 'node-spec', kind }, e)
}

function labelFor(kind: string): string {
  const spec = getSpec(kind)
  return spec?.labelZh ?? kind
}
</script>
