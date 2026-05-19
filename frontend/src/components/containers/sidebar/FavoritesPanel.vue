<!-- Favorites panel. Drag-out → useEditorDragDrop ('node-spec' payload) → 画布建节点. -->
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
    <div v-else class="space-y-1 max-h-48 overflow-y-auto pr-1">
      <div
        v-for="kind in store.favorites"
        :key="kind"
        class="group px-2 py-1 bg-elevated/30 rounded text-[11px] flex items-center gap-2 cursor-grab hover:bg-elevated/50"
        draggable="true"
        :title="`拖到画布建 ${kind} 节点 · 悬停点 × 移除收藏`"
        @dragstart="(e) => onDragStart(kind, e)"
      >
        <UIcon name="i-tabler-grip-vertical" class="size-3 text-dimmed" />
        <span class="flex-1">{{ labelFor(kind) }}</span>
        <button
          type="button"
          class="text-dimmed hover:text-error px-0.5 opacity-0 group-hover:opacity-100 transition-opacity"
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
