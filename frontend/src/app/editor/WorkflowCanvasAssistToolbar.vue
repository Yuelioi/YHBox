<template>
  <aside
    data-testid="workflow-canvas-assist"
    class="absolute left-3 top-16 z-20 flex max-h-[calc(100%-5rem)] flex-col overflow-hidden rounded-xl border border-default bg-default/95 shadow-lg"
    :class="collapsed ? 'w-10' : 'w-40'"
    :aria-label="t('workflow.canvas_assist.title')"
  >
    <div class="flex min-h-0 flex-col gap-0.5 overflow-y-auto p-1">
      <AssistButton
        test-id="workflow-canvas-add-node"
        icon="i-tabler-plus"
        :label="t('workflow.canvas.add_node')"
        :show-label="showLabels"
        @click="emit('add-node')"
      />
      <AssistButton
        test-id="workflow-annotation-add"
        icon="i-tabler-note"
        :label="t('workflow.graphs.add_comment')"
        :show-label="showLabels"
        @click="emit('add-comment')"
      />

      <div class="my-1 h-px bg-default" />
      <p v-if="showLabels" class="px-2 py-1 text-[9px] font-semibold uppercase text-dimmed">
        {{ t('workflow.canvas_assist.favorites') }}
      </p>
      <AssistButton
        v-for="(item, index) in favorites"
        :key="item.nodeTypeId"
        :test-id="`workflow-canvas-favorite-${index + 1}`"
        :icon="`i-tabler-${item.icon || 'box'}`"
        :label="item.title"
        :show-label="showLabels"
        :disabled="!item.available"
        :shortcut="`Alt+${index + 1}`"
        @click="emit('add-favorite', item.nodeTypeId)"
      />

      <div class="my-1 h-px bg-default" />
      <p v-if="showLabels" class="px-2 py-1 text-[9px] font-semibold uppercase text-dimmed">
        {{ t('workflow.canvas_assist.quick_actions') }}
      </p>
      <AssistButton
        test-id="workflow-canvas-insert-node"
        icon="i-tabler-arrow-autofit-content"
        :label="t('workflow.canvas_assist.insert_node')"
        :show-label="showLabels"
        :disabled="!hasSelectedEdge"
        @click="emit('insert-node')"
      />
      <AssistButton
        test-id="workflow-canvas-make-space"
        icon="i-tabler-arrows-split-2"
        :label="t('workflow.canvas_assist.make_space')"
        :show-label="showLabels"
        :disabled="selectedNodeCount !== 2"
        @click="emit('make-space')"
      />
      <AssistButton
        test-id="workflow-layout-lr"
        icon="i-tabler-layout-board-split"
        :label="t('workflow.selection.layout_lr')"
        :show-label="showLabels"
        :loading="layouting"
        @click="emit('layout', 'LR')"
      />
      <AssistButton
        test-id="workflow-layout-tb"
        icon="i-tabler-layout-navbar-collapse"
        :label="t('workflow.selection.layout_tb')"
        :show-label="showLabels"
        :loading="layouting"
        @click="emit('layout', 'TB')"
      />
    </div>
    <div
      class="flex shrink-0 items-center gap-0.5 border-t border-default p-1"
      :class="showLabels ? 'justify-between' : 'flex-col'"
    >
      <UPopover mode="click" :ui="{ content: 'w-80 p-0' }">
        <UButton
          data-testid="workflow-canvas-assist-settings"
          icon="i-tabler-settings"
          color="neutral"
          variant="ghost"
          size="xs"
          :label="showLabels ? t('workflow.canvas_assist.settings') : undefined"
          :aria-label="t('workflow.canvas_assist.settings')"
        />
        <template #content>
          <div>
            <div class="border-b border-default px-4 py-3">
              <p class="text-xs font-semibold text-highlighted">
                {{ t('workflow.canvas_assist.configure_favorites') }}
              </p>
              <p class="mt-1 text-[11px] leading-4 text-muted">
                {{ t('workflow.canvas_assist.configure_favorites_hint') }}
              </p>
            </div>
            <div class="space-y-1.5 p-3">
              <div
                v-for="index in 5"
                :key="index"
                class="grid grid-cols-[1.5rem_minmax(0,1fr)] items-center gap-2 rounded-lg bg-elevated/45 p-1.5"
              >
                <span
                  class="grid size-6 place-items-center rounded-md bg-default font-mono text-[10px] text-muted"
                  >{{ index }}</span
                >
                <AdaptiveSelect
                  :model-value="favoriteNodeTypeIds[index - 1] ?? ''"
                  :items="nodeOptions"
                  value-key="value"
                  label-key="label"
                  width-mode="fill"
                  :placeholder="t('workflow.canvas_assist.favorite_slot', { n: index })"
                  @update:model-value="(value: string) => updateFavorite(index - 1, value)"
                />
              </div>
            </div>
          </div>
        </template>
      </UPopover>
      <UButton
        data-testid="workflow-canvas-assist-compact"
        :icon="collapsed ? 'i-tabler-chevron-right' : 'i-tabler-chevron-left'"
        color="neutral"
        variant="ghost"
        size="xs"
        :aria-label="
          t(collapsed ? 'workflow.canvas_assist.expand' : 'workflow.canvas_assist.collapse')
        "
        :aria-pressed="collapsed"
        @click="emit('update-collapsed', !collapsed)"
      />
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'
import AssistButton from './WorkflowCanvasAssistButton.vue'

export interface CanvasAssistFavorite {
  nodeTypeId: string
  title: string
  icon: string
  available: boolean
}

const props = defineProps<{
  collapsed: boolean
  favoriteNodeTypeIds: string[]
  favorites: CanvasAssistFavorite[]
  nodeOptions: Array<{ label: string; value: string }>
  selectedNodeCount: number
  hasSelectedEdge: boolean
  layouting: boolean
}>()
const emit = defineEmits<{
  'add-node': []
  'add-comment': []
  'add-favorite': [nodeTypeId: string]
  'insert-node': []
  'make-space': []
  layout: [direction: 'LR' | 'TB']
  'update-favorites': [value: string[]]
  'update-collapsed': [value: boolean]
}>()
const { t } = useI18n()
const showLabels = computed(() => !props.collapsed)

function updateFavorite(index: number, value: string): void {
  const next = Array.from(
    { length: 5 },
    (_, itemIndex) => props.favoriteNodeTypeIds[itemIndex] ?? '',
  )
  for (let itemIndex = 0; itemIndex < next.length; itemIndex++) {
    if (itemIndex !== index && next[itemIndex] === value) next[itemIndex] = ''
  }
  next[index] = value
  emit('update-favorites', next.filter(Boolean).slice(0, 5))
}
</script>
