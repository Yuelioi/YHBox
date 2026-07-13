<template>
  <aside class="shrink-0 border-l border-default overflow-y-auto p-4">
    <NodeInspector
      v-if="selectedNode"
      :node="selectedNode"
      :declared-vars="declaredVars"
      :nodes="activeGraph?.nodes ?? []"
      :edges="activeGraph?.edges ?? []"
      :experience-mode="experienceMode"
      @update="$emit('config-update', $event)"
      @remove-switch-case="$emit('remove-switch-case', $event)"
      @label-update="$emit('label-update', $event)"
      @log-enabled-update="$emit('log-enabled-update', $event)"
      @delete="$emit('delete-selected')"
      @request-record="$emit('request-record', $event)"
      @declare-var="$emit('declare-var', $event)"
    />
    <SubgraphPropsPanel
      v-else-if="inSubgraph"
      :subgraph="currentSubgraph"
      :all-tags="allSubgraphTags"
      :all-categories="allSubgraphCategories"
      @update="$emit('subgraph-update', $event)"
      @to-script="$emit('subgraph-to-script')"
    />
  </aside>
</template>

<script setup lang="ts">
import type { Graph, GraphNode } from '@/lib/backend'
import type { VarType } from '@/lib/variableRef'
import type { RemoveSwitchCaseCommand } from '@/composables/containerEditor/useGraphMutations'
import type { EditorExperienceMode } from '@/composables/editor/useSidebarPrefs'
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
  declaredVars: { name: string; type: VarType }[]
  allSubgraphTags: string[]
  allSubgraphCategories: string[]
  experienceMode: EditorExperienceMode
}>()

defineEmits<{
  'config-update': [cfg: Record<string, any>]
  'remove-switch-case': [command: RemoveSwitchCaseCommand]
  'label-update': [v: string]
  'log-enabled-update': [v: boolean]
  'delete-selected': []
  'subgraph-update': [patch: Record<string, any>]
  'subgraph-to-script': []
  'request-record': [opts: { mode: 'precise' | 'simple'; replaceNodeID: string }]
  'declare-var': [args: { name: string; type: VarType; default: unknown }]
}>()
</script>
