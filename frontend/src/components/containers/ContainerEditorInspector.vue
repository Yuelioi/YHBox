<template>
  <aside class="shrink-0 border-l border-default overflow-y-auto p-4">
    <NodeInspector
      v-if="selectedNode"
      :node="selectedNode"
      :declared-vars="declaredVars"
      :nodes="activeGraph?.nodes ?? []"
      :edges="activeGraph?.edges ?? []"
      @update="$emit('config-update', $event)"
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
      @update="$emit('subgraph-update', $event)"
    />
    <div v-else class="p-4 space-y-4 text-xs text-muted">
      <!-- 容器概览：选中节点前的默认面板 — 统计 + 热键 -->
      <div class="space-y-2">
        <p class="text-default font-medium">{{ t('editor.inspector.empty.overview_title') }}</p>
        <div class="flex flex-wrap gap-1.5">
          <span class="px-1.5 py-0.5 rounded bg-elevated/60 border border-default/60 text-[11px]">
            {{ t('editor.inspector.empty.stats_nodes', { n: activeGraph?.nodes?.length ?? 0 }) }}
          </span>
          <span class="px-1.5 py-0.5 rounded bg-elevated/60 border border-default/60 text-[11px]">
            {{ t('editor.inspector.empty.stats_vars', { n: declaredVars.length }) }}
          </span>
          <span class="px-1.5 py-0.5 rounded bg-elevated/60 border border-default/60 text-[11px]">
            {{ t('editor.inspector.empty.stats_subgraphs', { n: subgraphCount }) }}
          </span>
        </div>
        <div class="flex items-center gap-2 text-[11px]">
          <span class="text-dimmed">{{ t('editor.inspector.empty.hotkey_label') }}</span>
          <kbd
            v-if="hotkey"
            class="px-1.5 py-0.5 bg-elevated rounded border border-default text-[10px]"
          >{{ hotkey }}</kbd>
          <span v-else class="text-dimmed">{{ t('editor.inspector.empty.hotkey_none') }}</span>
        </div>
      </div>

      <!-- 快捷开始 -->
      <div class="space-y-2 text-[11px]">
        <p class="text-dimmed">{{ t('editor.inspector.empty.quick_start') }}</p>
        <ul class="space-y-1.5 pl-1">
          <li class="flex items-center gap-2">
            <kbd class="px-1.5 py-0.5 bg-elevated rounded border border-default text-[10px]">Tab</kbd>
            <span>{{ t('editor.inspector.empty.tab_explorer') }}</span>
          </li>
          <li class="flex items-center gap-2">
            <UIcon name="i-tabler-mouse" class="size-3.5" />
            <span>{{ t('editor.inspector.empty.drag_var') }}</span>
          </li>
          <li class="flex items-center gap-2">
            <UIcon name="i-tabler-mouse" class="size-3.5" />
            <span>{{ t('editor.inspector.empty.right_click_canvas') }}</span>
          </li>
          <li class="flex items-center gap-2">
            <kbd class="px-1.5 py-0.5 bg-elevated rounded border border-default text-[10px]">Ctrl+K</kbd>
            <span>{{ t('editor.inspector.empty.command_palette') }}</span>
          </li>
          <li class="flex items-center gap-2">
            <kbd class="px-1.5 py-0.5 bg-elevated rounded border border-default text-[10px]">Ctrl+,</kbd>
            <span>{{ t('editor.inspector.empty.container_settings') }}</span>
          </li>
        </ul>
      </div>

      <!-- 帮助入口 — 打开多 tab 帮助弹窗 (快速上手 / 错误码 / 快捷键 / 节点速查) -->
      <button
        type="button"
        class="flex items-center gap-2 w-full px-3 py-2 rounded-md bg-elevated/30 border border-default/40 text-left hover:bg-elevated/50 transition-colors"
        @click="$emit('open-help')"
      >
        <UIcon name="i-tabler-help-circle" class="size-3.5 text-primary shrink-0" />
        <span class="text-[12px] font-medium text-toned flex-1">{{ t('editor.inspector.empty.open_help') }}</span>
        <UIcon name="i-tabler-chevron-right" class="size-3.5 text-dimmed shrink-0" />
      </button>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { Graph, GraphNode } from '@/lib/backend'
import type { VarType } from '@/lib/variableRef'
import type { SubgraphSummary } from '@/stores/containerEditor'
import NodeInspector from '@/components/containers/NodeInspector.vue'
import SubgraphPropsPanel from '@/components/containers/SubgraphPropsPanel.vue'

const { t } = useI18n()

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
  // 空状态「容器概览」用：热键 + 子图数（节点数/变量数从 activeGraph/declaredVars 取）。
  hotkey: string
  subgraphCount: number
}>()

defineEmits<{
  'config-update': [cfg: Record<string, any>]
  'label-update': [v: string]
  'log-enabled-update': [v: boolean]
  'delete-selected': []
  'subgraph-update': [patch: Record<string, any>]
  'request-record': [opts: { mode: 'precise' | 'simple'; replaceNodeID: string }]
  'declare-var': [args: { name: string; type: VarType; default: unknown }]
  'open-help': []
}>()
</script>
