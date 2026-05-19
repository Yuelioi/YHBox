<template>
  <aside class="w-96 shrink-0 border-l border-default overflow-y-auto p-4">
    <NodeInspector
      v-if="selectedNode"
      :node="selectedNode"
      :var-names="varNames"
      :nodes="activeGraph?.nodes ?? []"
      :edges="activeGraph?.edges ?? []"
      @update="$emit('config-update', $event)"
      @delete="$emit('delete-selected')"
      @request-record="$emit('request-record', $event)"
    />
    <SubgraphPropsPanel
      v-else-if="inSubgraph"
      :subgraph="currentSubgraph"
      :all-tags="allSubgraphTags"
      @update="$emit('subgraph-update', $event)"
    />
    <div v-else class="p-4 space-y-3 text-xs text-muted">
      <p class="text-default font-medium">未选节点</p>
      <div class="space-y-2 text-[11px]">
        <p class="text-dimmed">快捷开始:</p>
        <ul class="space-y-1.5 pl-1">
          <li class="flex items-center gap-2">
            <kbd class="px-1.5 py-0.5 bg-elevated rounded border border-default text-[10px]">Tab</kbd>
            <span>打开节点 Explorer</span>
          </li>
          <li class="flex items-center gap-2">
            <UIcon name="i-tabler-mouse" class="size-3.5" />
            <span>拖左侧变量 → 建 GetVar 节点</span>
          </li>
          <li class="flex items-center gap-2">
            <UIcon name="i-tabler-mouse" class="size-3.5" />
            <span>右键画布 → 添加节点菜单</span>
          </li>
          <li class="flex items-center gap-2">
            <kbd class="px-1.5 py-0.5 bg-elevated rounded border border-default text-[10px]">Ctrl+K</kbd>
            <span>命令面板</span>
          </li>
          <li class="flex items-center gap-2">
            <kbd class="px-1.5 py-0.5 bg-elevated rounded border border-default text-[10px]">Ctrl+,</kbd>
            <span>容器设置</span>
          </li>
        </ul>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import type { Graph, GraphNode } from '@/lib/backend'
import type { SubgraphSummary } from '@/stores/containerEditor'
import NodeInspector from '@/components/containers/NodeInspector.vue'
import SubgraphPropsPanel from '@/components/containers/SubgraphPropsPanel.vue'

defineProps<{
  selectedNode: GraphNode | null
  inSubgraph: boolean
  // 这里走 store 侧的 SubgraphSummary（无 createdAt 字段）而非 backend.Subgraph，
  // 因为 currentSubgraph 来自 useEditorPath → editorStore.subgraphsForCurrentContainer。
  // SubgraphPropsPanel 内部用结构相容的 SubgraphLike 接收。
  currentSubgraph: SubgraphSummary | null
  activeGraph: Graph | null
  varNames: string[]
  allSubgraphTags: string[]
}>()

defineEmits<{
  'config-update': [cfg: Record<string, any>]
  'delete-selected': []
  'subgraph-update': [patch: Record<string, any>]
  'request-record': [opts: { mode: 'precise' | 'simple'; replaceNodeID: string }]
}>()
</script>
