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
    <ContainerPropsPanel
      v-else
      :container="container"
      @update="$emit('container-update', $event)"
    />
  </aside>
</template>

<script setup lang="ts">
import type { Container, Graph, GraphNode } from '@/lib/backend'
import type { SubgraphSummary } from '@/stores/containerEditor'
import NodeInspector from '@/components/containers/NodeInspector.vue'
import SubgraphPropsPanel from '@/components/containers/SubgraphPropsPanel.vue'
import ContainerPropsPanel from '@/components/containers/ContainerPropsPanel.vue'

defineProps<{
  selectedNode: GraphNode | null
  inSubgraph: boolean
  // 这里走 store 侧的 SubgraphSummary（无 createdAt 字段）而非 backend.Subgraph，
  // 因为 currentSubgraph 来自 useEditorPath → editorStore.subgraphsForCurrentContainer。
  // SubgraphPropsPanel 内部用结构相容的 SubgraphLike 接收。
  currentSubgraph: SubgraphSummary | null
  container: Container | null
  activeGraph: Graph | null
  varNames: string[]
  allSubgraphTags: string[]
}>()

defineEmits<{
  'config-update': [cfg: Record<string, any>]
  'delete-selected': []
  'subgraph-update': [patch: Record<string, any>]
  'container-update': [patch: Partial<Container>]
  'request-record': [opts: { mode: 'precise' | 'simple'; replaceNodeID: string }]
}>()
</script>
