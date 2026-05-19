<!-- Recent panel (LRU max 8). Drag-out → useEditorDragDrop ('node-spec') → 画布建节点. -->
<template>
  <SidebarSection
    title="最近"
    icon="i-tabler-clock"
    title-color="indigo"
    :count="store.recent.length"
    :expanded="expanded"
    @update:expanded="$emit('update:expanded', $event)"
  >
    <p v-if="store.recent.length === 0" class="text-[10px] text-dimmed italic px-1">
      暂无最近. 拖节点入画布会自动加入.
    </p>
    <div v-else class="space-y-1 max-h-48 overflow-y-auto pr-1">
      <div
        v-for="kind in store.recent"
        :key="kind"
        class="px-2 py-1 bg-elevated/30 rounded text-[11px] flex items-center gap-2 cursor-grab hover:bg-elevated/50"
        draggable="true"
        :title="`拖到画布建 ${kind} 节点`"
        @dragstart="(e) => onDragStart(kind, e)"
      >
        <UIcon name="i-tabler-grip-vertical" class="size-3 text-dimmed" />
        <span>{{ labelFor(kind) }}</span>
        <button
          type="button"
          class="ml-auto text-dimmed hover:text-amber-400 px-0.5"
          :title="store.favorites.includes(kind) ? '已收藏' : '加入收藏'"
          @click.stop="store.toggleFavorite(kind)"
        >
          <UIcon
            :name="store.favorites.includes(kind) ? 'i-tabler-star-filled' : 'i-tabler-star'"
            class="size-3"
            :class="store.favorites.includes(kind) ? 'text-amber-400' : ''"
          />
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
