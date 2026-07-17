<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden bg-default">
    <div v-if="session.phase === 'loading'" class="flex flex-1 items-center justify-center px-8">
      <div class="w-full max-w-xl space-y-3" :aria-label="t('workflow.editor.loading')">
        <USkeleton class="h-10 w-2/3 rounded-lg" />
        <USkeleton class="h-72 w-full rounded-lg" />
      </div>
    </div>

    <div
      v-else-if="session.failure && !session.source"
      class="flex flex-1 items-center justify-center p-8"
    >
      <div class="max-w-lg rounded-lg border border-error/35 bg-error/10 p-5" role="alert">
        <h1 class="text-sm font-semibold text-error">
          {{ t('workflow.editor.open_failed') }}
        </h1>
        <p class="mt-2 text-xs leading-5 text-muted">{{ session.failure }}</p>
        <UButton
          class="mt-4"
          :label="t('workflow.editor.back')"
          color="neutral"
          @click="router.push('/workflows')"
        />
      </div>
    </div>

    <template v-else-if="session.source && session.authoring">
      <WorkflowEditorToolbar
        :name="session.source.workflow.name"
        :revision="session.baseRevision"
        :dirty="session.dirty"
        :can-undo="session.canUndo"
        :can-redo="session.canRedo"
        :ai-panel-open="aiPanelOpen"
        :state-panel-open="statePanelOpen"
        :run-active="runActive"
        :saving="session.phase === 'saving'"
        :compile-succeeded="compileSucceeded"
        :save-succeeded="saveSucceeded"
        :diagnostic-count="session.diagnostics.length"
        :diagnostics-open="diagnosticsOpen"
        :has-run-timeline="Boolean(session.activeRun)"
        :run-timeline-open="runTimelineOpen"
        :has-debug="Boolean(session.debugSnapshot)"
        :debugger-open="debuggerOpen"
        @back="router.push('/workflows')"
        @rename="renameWorkflow"
        @undo="session.undo()"
        @redo="session.redo()"
        @toggle-ai="toggleAIReview"
        @toggle-state="toggleStatePanel"
        @compile="compile"
        @toggle-diagnostics="diagnosticsOpen = !diagnosticsOpen"
        @toggle-timeline="runTimelineOpen = !runTimelineOpen"
        @toggle-debugger="debuggerOpen = !debuggerOpen"
        @start-debug="startDebug"
        @run="startRun"
        @stop="cancelRun"
        @save="save"
      />

      <div
        v-if="session.saveConflict"
        class="border-b border-error/35 bg-error/10 px-4 py-2 text-xs text-error"
        role="alert"
      >
        {{ t('workflow.editor.save_conflict', { message: session.saveConflict }) }}
      </div>
      <div
        v-else-if="session.failure"
        class="border-b border-error/35 bg-error/10 px-4 py-2 text-xs text-error"
        role="alert"
      >
        {{ session.failure }}
      </div>
      <WorkflowDiagnosticsPanel
        v-if="diagnosticsOpen && session.diagnostics.length"
        :diagnostics="session.diagnostics"
        @focus="focusDiagnostic"
        @close="diagnosticsOpen = false"
      />

      <div class="flex min-h-0 flex-1">
        <aside class="flex w-56 shrink-0 flex-col border-r border-default bg-default">
          <div class="border-b border-default px-4 py-3">
            <h2 class="text-xs font-semibold text-highlighted">
              {{ t('workflow.editor.node_catalog') }}
            </h2>
            <p class="mt-1 text-[11px] leading-4 text-muted">
              {{ t('workflow.editor.catalog_description') }}
            </p>
            <UInput
              v-model="catalogQuery"
              data-testid="workflow-catalog-search"
              icon="i-tabler-search"
              size="sm"
              class="mt-3"
              :placeholder="t('workflow.catalog.search_placeholder')"
              :aria-label="t('workflow.catalog.search_placeholder')"
            />
          </div>
          <div class="flex-1 overflow-y-auto p-2">
            <div v-if="catalogGroups.length" class="space-y-3">
              <section v-for="group in catalogGroups" :key="group.key">
                <div class="flex items-center justify-between px-2 pb-1">
                  <h3 class="text-[10px] font-semibold uppercase tracking-wider text-dimmed">
                    {{ group.label }}
                  </h3>
                  <span class="font-mono text-[9px] text-dimmed">{{ group.nodes.length }}</span>
                </div>
                <div class="space-y-1">
                  <button
                    v-for="projection in group.nodes"
                    :key="projection.nodeRef.nodeTypeId"
                    type="button"
                    draggable="true"
                    data-testid="node-catalog-item"
                    :data-node-type-id="projection.nodeRef.nodeTypeId"
                    class="group flex w-full cursor-grab items-center gap-2 rounded-lg px-2.5 py-2 text-left transition-colors hover:bg-elevated focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary active:cursor-grabbing active:translate-y-px"
                    @click="addNode(projection.nodeRef.nodeTypeId)"
                    @dragstart="startNodeDrag($event, projection.nodeRef.nodeTypeId)"
                    @dragend="finishNodeDrag"
                  >
                    <UIcon
                      :name="`i-tabler-${projection.icon || 'box'}`"
                      class="size-4 shrink-0 text-primary"
                    />
                    <span class="min-w-0 flex-1">
                      <span class="block truncate text-xs font-medium text-toned">{{
                        projectionTitle(projection)
                      }}</span>
                      <span class="block truncate font-mono text-[10px] text-dimmed">{{
                        projection.execution.class
                      }}</span>
                    </span>
                    <UIcon
                      name="i-tabler-plus"
                      class="size-3.5 text-dimmed group-hover:text-primary"
                    />
                  </button>
                </div>
              </section>
            </div>
            <div v-else class="px-3 py-10 text-center">
              <UIcon name="i-tabler-search-off" class="mx-auto mb-2 size-5 text-dimmed" />
              <p class="text-xs text-muted">{{ t('workflow.catalog.no_results') }}</p>
            </div>
          </div>
          <div class="border-t border-default px-3 py-2 font-mono text-[10px] text-dimmed">
            {{ session.authoring.projectionDigest.slice(0, 24) }}
          </div>
        </aside>

        <div
          ref="canvasElement"
          data-testid="workflow-canvas"
          class="relative min-w-0 flex-1 bg-elevated/15 transition-shadow"
          :class="nodeDragActive ? 'ring-1 ring-inset ring-primary/60' : ''"
          @dragover="continueNodeDrag"
          @dragleave.self="finishNodeDrag"
          @drop="dropNode"
        >
          <VueFlow
            :nodes="flowNodes"
            :edges="flowEdges"
            :delete-key-code="null"
            :is-valid-connection="isValidConnection"
            fit-view-on-init
            :min-zoom="0.2"
            :max-zoom="2"
            class="workflow-flow"
            @connect="connect"
            @connect-start="startConnection"
            @connect-end="endConnection"
            @node-click="selectNode"
            @pane-click="handlePaneClick"
            @nodes-change="handleNodesChange"
            @node-drag-start="trackNodeDrag"
            @node-drag="trackNodeDrag"
            @node-drag-stop="moveNode"
            @edge-double-click="disconnect"
          >
            <template #node-workflow="slotProps">
              <WorkflowNode
                :node="slotProps.data.node"
                :projection="slotProps.data.projection"
                :selected="slotProps.selected"
                :run-status="nodeRunStatusById.get(slotProps.data.node.id)"
                :breakpoint="hasBreakpoint(session.currentGraph?.id ?? '', slotProps.data.node.id)"
                :debug-current="
                  isDebugCurrent(session.currentGraph?.id ?? '', slotProps.data.node.id)
                "
                @toggle-breakpoint="
                  toggleBreakpoint(session.currentGraph?.id ?? '', slotProps.data.node.id)
                "
              />
            </template>
            <Background :gap="20" :size="1" pattern-color="rgb(113 113 122 / 0.26)" />
            <Controls position="bottom-left" />
            <MiniMap
              position="bottom-right"
              :pannable="true"
              :zoomable="true"
              node-color="var(--ui-bg-accented)"
              node-stroke-color="var(--ui-border-accented)"
              :node-stroke-width="1"
              mask-color="color-mix(in oklab, var(--ui-bg) 72%, transparent)"
            />
          </VueFlow>
          <div
            v-if="snapGuides.x !== undefined"
            class="pointer-events-none absolute inset-y-0 z-10 w-px bg-primary/70"
            :style="{ left: `${snapGuides.x}px` }"
          />
          <div
            v-if="snapGuides.y !== undefined"
            class="pointer-events-none absolute inset-x-0 z-10 h-px bg-primary/70"
            :style="{ top: `${snapGuides.y}px` }"
          />
          <WorkflowSelectionToolbar
            v-if="selectedNodeIds.size"
            :count="selectedNodeIds.size"
            :layouting="layouting"
            @align="alignSelection"
            @distribute="distributeSelection"
            @auto-layout="autoLayout"
            @copy="copySelection"
            @cut="cutSelection"
            @duplicate="duplicateSelection"
            @remove="removeSelection"
          />
          <div
            v-else
            class="absolute right-3 top-3 z-20 flex gap-1 rounded-lg border border-default bg-default/95 p-1 shadow-lg"
          >
            <UButton
              data-testid="workflow-layout-lr"
              icon="i-tabler-layout-board-split"
              color="neutral"
              variant="ghost"
              size="xs"
              :loading="layouting"
              :aria-label="t('workflow.selection.layout_lr')"
              @click="autoLayout('LR')"
            />
            <UButton
              data-testid="workflow-layout-tb"
              icon="i-tabler-layout-navbar-collapse"
              color="neutral"
              variant="ghost"
              size="xs"
              :loading="layouting"
              :aria-label="t('workflow.selection.layout_tb')"
              @click="autoLayout('TB')"
            />
          </div>
          <div
            v-if="connectionHint"
            class="pointer-events-none absolute left-1/2 top-3 z-20 -translate-x-1/2 rounded-lg border border-default bg-default/95 px-3 py-1.5 text-[11px] text-muted shadow-lg"
            role="status"
          >
            {{ connectionHint }}
          </div>
          <WorkflowConnectionMenu
            v-if="connectionMenu"
            :position="connectionMenu.canvasPosition"
            :compatible-candidates="compatibleConnectionCandidates"
            :all-candidates="allConnectionCandidates"
            @select="selectConnectionCandidate"
            @close="closeConnectionMenu"
          />
          <div
            v-if="flowNodes.length === 0"
            data-testid="workflow-empty-canvas"
            class="pointer-events-none absolute inset-0 z-10 flex items-center justify-center p-8"
          >
            <div
              class="pointer-events-auto max-w-sm rounded-xl border border-default bg-default/95 p-6 text-center shadow-xl"
            >
              <div
                class="mx-auto mb-3 flex size-11 items-center justify-center rounded-xl bg-primary/10 text-primary"
              >
                <UIcon name="i-tabler-player-play" class="size-5" />
              </div>
              <h2 class="text-sm font-semibold text-highlighted">
                {{ t('workflow.empty_canvas.title') }}
              </h2>
              <p class="mt-2 text-xs leading-5 text-muted">
                {{ t('workflow.empty_canvas.description') }}
              </p>
              <UButton
                class="mt-4"
                icon="i-tabler-player-play"
                :label="t('workflow.empty_canvas.add_start')"
                @click="addNode(RUN_STARTED_NODE_ID, { x: 120, y: 160 })"
              />
            </div>
          </div>
        </div>

        <AIWorkflowReviewPanel
          v-if="aiPanelOpen"
          :workflow-id="session.workflowId"
          :base-revision="session.baseRevision"
          :dirty="session.dirty"
          @close="aiPanelOpen = false"
          @accepted="acceptAIProposal"
        />
        <WorkflowStatePanel
          v-else-if="statePanelOpen"
          :variables="session.source?.variables ?? []"
          :types="session.authoring?.body.types ?? []"
          @command="applyCommand"
          @close="statePanelOpen = false"
        />
        <WorkflowInspector
          v-else
          :node="selectedNode"
          :projection="selectedProjection"
          :variables="session.source?.variables ?? []"
          :types="session.authoring?.body.types ?? []"
          @command="applyCommand"
        />
      </div>

      <RunTimelinePanel
        v-if="session.activeRun && runTimelineOpen"
        :run="session.activeRun"
        @cancel="cancelRun"
        @refresh="refreshRun"
        @focus-node="focusNode"
        @close="runTimelineOpen = false"
      />
      <WorkflowDebuggerPanel
        v-if="session.debugSnapshot && debuggerOpen"
        :snapshot="session.debugSnapshot"
        @continue="controlDebug('continue')"
        @pause="controlDebug('pause')"
        @step="controlDebug('step')"
        @stop="cancelRun"
        @close="debuggerOpen = false"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import { useToast } from '@nuxt/ui/composables'
import {
  VueFlow,
  useVueFlow,
  type Connection,
  type Edge as FlowEdge,
  type EdgeMouseEvent,
  type NodeDragEvent,
  type NodeChange,
  type NodeMouseEvent,
  type OnConnectStartParams,
} from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/controls/dist/style.css'
import '@vue-flow/minimap/dist/style.css'
import { useI18n } from 'vue-i18n'
import {
  type Edge,
  type EditorCommand,
  type Node,
  type NodeProjection,
} from '@/app/editor/EditorSession'
import { createEditorSession } from '@/app/editor/createEditorSession'
import { graphHandle, parseGraphHandle, type ParsedHandle } from '@/app/editor/graphHandles'
import {
  onDebugChanged,
  onRunChanged,
  workflowTransport,
  type DebugBreakpoint,
} from '@/app/transport/workflow'
import { useConfirm } from '@/composables/useConfirm'
import WorkflowNode from '@/app/editor/WorkflowNode.vue'
import WorkflowInspector from '@/app/editor/WorkflowInspector.vue'
import AIWorkflowReviewPanel from '@/app/editor/AIWorkflowReviewPanel.vue'
import RunTimelinePanel from '@/app/editor/RunTimelinePanel.vue'
import WorkflowDiagnosticsPanel from '@/app/editor/WorkflowDiagnosticsPanel.vue'
import WorkflowDebuggerPanel from '@/app/editor/WorkflowDebuggerPanel.vue'
import WorkflowEditorToolbar from '@/app/editor/WorkflowEditorToolbar.vue'
import WorkflowStatePanel from '@/app/editor/WorkflowStatePanel.vue'
import WorkflowConnectionMenu, {
  type WorkflowConnectionCandidate,
} from '@/app/editor/WorkflowConnectionMenu.vue'
import WorkflowSelectionToolbar from '@/app/editor/WorkflowSelectionToolbar.vue'
import { nodeRunStatuses } from '@/app/editor/runTrace'
import type { WorkflowDiagnostic } from '@/app/editor/workflowDiagnostics'
import {
  compatibleCandidatePorts,
  type ConnectionIssue,
} from '@/app/editor/connectionCompatibility'
import {
  createWorkflowNodeGestureState,
  projectWorkflowFlowNodes,
} from '@/app/editor/workflowFlowProjection'
import {
  alignNodePositions,
  autoLayoutNodePositions,
  distributeNodePositions,
  snapNodePosition,
  type AlignMode,
  type DistributeMode,
  type SizedWorkflowNode,
} from '@/app/editor/workflowLayout'

defineOptions({ name: 'WorkflowEditorView' })

const route = useRoute()
const router = useRouter()
const toast = useToast()
const { confirm } = useConfirm()
const { t, te } = useI18n()
const session = createEditorSession(workflowTransport)
const selectedNodeId = ref('')
const selectedNodeIds = ref(new Set<string>())
const nodeDragActive = ref(false)
const aiPanelOpen = ref(false)
const statePanelOpen = ref(false)
const catalogQuery = ref('')
const compileSucceeded = ref(false)
const saveSucceeded = ref(false)
const canvasElement = ref<HTMLElement | null>(null)
const connectionStart = ref<ConnectionAnchor | null>(null)
const connectionMenu = ref<ConnectionMenuState | null>(null)
const connectionHint = ref('')
const snapGuides = ref<{ x?: number; y?: number }>({})
const layouting = ref(false)
const diagnosticsOpen = ref(false)
const runTimelineOpen = ref(false)
const debuggerOpen = ref(false)
const breakpointKeys = ref(new Set<string>())
const {
  addSelectedNodes,
  findNode,
  fitView,
  flowToScreenCoordinate,
  getSelectedNodes,
  removeSelectedNodes,
  screenToFlowCoordinate,
  setCenter,
  updateNode,
} = useVueFlow()
const nodeGestures = createWorkflowNodeGestureState()
let unsubscribeRun: (() => void) | undefined
let unsubscribeDebug: (() => void) | undefined
let compileFlashTimer: ReturnType<typeof setTimeout> | undefined
let saveFlashTimer: ReturnType<typeof setTimeout> | undefined
let connectionEndTimer: ReturnType<typeof setTimeout> | undefined
let connectionMadeThisGesture = false
let workflowClipboard: WorkflowSelectionClipboard | null = null
let pasteOffset = 0
let nextPosition = 0

const NODE_TYPE_DRAG_FORMAT = 'application/x-yotta-node-type'
const RUN_STARTED_NODE_ID = 'https://schemas.yotta.dev/nodes/event/run-started'

interface ConnectionAnchor {
  nodeId: string
  handle: ParsedHandle
}

interface ConnectionMenuState {
  anchor: ConnectionAnchor
  flowPosition: { x: number; y: number }
  canvasPosition: { x: number; y: number }
}

interface WorkflowSelectionClipboard {
  format: 'yotta.workflow-selection'
  version: 1
  nodes: Node[]
  edges: Edge[]
}

const catalogGroups = computed(() => {
  const query = catalogQuery.value.trim().toLocaleLowerCase()
  const grouped = new Map<string, NodeProjection[]>()
  for (const projection of session.authoring?.body.nodes ?? []) {
    if (query && !catalogSearchText(projection).includes(query)) continue
    const key = projection.category || 'other'
    const nodes = grouped.get(key) ?? []
    nodes.push(projection)
    grouped.set(key, nodes)
  }
  return [...grouped.entries()]
    .sort(([left], [right]) => categoryLabel(left).localeCompare(categoryLabel(right)))
    .map(([key, nodes]) => ({
      key,
      label: categoryLabel(key),
      nodes: nodes.sort((left, right) =>
        projectionTitle(left).localeCompare(projectionTitle(right)),
      ),
    }))
})

const flowNodes = computed(() =>
  projectWorkflowFlowNodes(
    session.currentGraph?.nodes ?? [],
    session.nodeProjection.bind(session),
    nodeGestures.positions,
  ),
)

const flowEdges = computed<FlowEdge[]>(() =>
  (session.currentGraph?.edges ?? []).map((edge) => ({
    id: edgeId(edge),
    source: edge.from.nodeId,
    target: edge.to.nodeId,
    sourceHandle: graphHandle(edge.channel, 'output', edge.from.portId),
    targetHandle: graphHandle(edge.channel, 'input', edge.to.portId),
    animated: edge.channel !== 'data',
    style: {
      stroke:
        edge.channel === 'error' ? '#f87171' : edge.channel === 'exec' ? '#a1a1aa' : '#10b981',
    },
  })),
)

const compatibleConnectionCandidates = computed<WorkflowConnectionCandidate[]>(() => {
  const menu = connectionMenu.value
  if (!menu) return []
  const anchorNode = session.currentGraph?.nodes.find((node) => node.id === menu.anchor.nodeId)
  if (!anchorNode) return []
  const anchorProjection = session.nodeProjection(anchorNode.nodeRef.nodeTypeId)
  if (!anchorProjection) return []
  return (session.authoring?.body.nodes ?? [])
    .flatMap((projection) =>
      compatibleCandidatePorts(anchorProjection, menu.anchor.handle, projection).map((port) => ({
        key: `${projection.nodeRef.nodeTypeId}:${port.handle.channel}:${port.handle.portId}`,
        nodeTypeId: projection.nodeRef.nodeTypeId,
        title: projectionTitle(projection),
        icon: projection.icon,
        searchText: catalogSearchText(projection),
        handle: port.handle,
      })),
    )
    .sort(
      (left, right) => left.title.localeCompare(right.title) || left.key.localeCompare(right.key),
    )
})

const allConnectionCandidates = computed<WorkflowConnectionCandidate[]>(() =>
  (session.authoring?.body.nodes ?? [])
    .filter((projection) => projection.instruction.kind !== 'run-root')
    .map((projection) => ({
      key: projection.nodeRef.nodeTypeId,
      nodeTypeId: projection.nodeRef.nodeTypeId,
      title: projectionTitle(projection),
      icon: projection.icon,
      searchText: catalogSearchText(projection),
    }))
    .sort((left, right) => left.title.localeCompare(right.title)),
)

const selectedNode = computed(
  () => session.currentGraph?.nodes.find((node) => node.id === selectedNodeId.value) ?? null,
)
const selectedProjection = computed(() =>
  selectedNode.value
    ? (session.nodeProjection(selectedNode.value.nodeRef.nodeTypeId) ?? null)
    : null,
)
const runActive = computed(() =>
  session.activeRun
    ? ['QUEUED', 'RUNNING'].includes(session.activeRun.status.toUpperCase())
    : false,
)
const nodeRunStatusById = computed(() =>
  nodeRunStatuses(session.activeRun, session.currentGraph?.id ?? ''),
)

onMounted(async () => {
  document.addEventListener('keydown', handleEditorKeydown)
  const workflowId = String(route.params.id ?? '')
  try {
    await session.load(workflowId)
  } catch {
    return
  }
  unsubscribeRun = onRunChanged((event) => {
    if (event.runId === session.activeRun?.runId) void refreshRun()
  })
  unsubscribeDebug = onDebugChanged((event) => {
    if (!session.acceptDebugSnapshot(event.runId, event.snapshot)) return
    debuggerOpen.value = true
    if (event.snapshot.status === 'paused' && event.snapshot.nodeId) {
      void focusNode(event.snapshot.graphId ? [event.snapshot.graphId] : [], event.snapshot.nodeId)
    }
  })
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleEditorKeydown)
  unsubscribeRun?.()
  unsubscribeDebug?.()
  clearTimeout(compileFlashTimer)
  clearTimeout(saveFlashTimer)
  clearTimeout(connectionEndTimer)
})
onBeforeRouteLeave(async () => {
  if (!session.dirty) return true
  return (
    (await confirm({
      title: t('workflow.editor.discard_title'),
      description: t('workflow.editor.discard_confirm'),
      confirmText: t('workflow.editor.discard_action'),
      color: 'warning',
    })) === true
  )
})

function applyCommand(command: EditorCommand): boolean {
  try {
    session.apply(command)
    if (command.kind === 'remove-node' || command.kind === 'remove-nodes') {
      const removed = new Set(command.kind === 'remove-node' ? [command.nodeId] : command.nodeIds)
      selectedNodeIds.value = new Set(
        [...selectedNodeIds.value].filter((nodeId) => !removed.has(nodeId)),
      )
      if (removed.has(selectedNodeId.value)) selectedNodeId.value = ''
    }
    return true
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
    return false
  }
}

function addNode(nodeTypeId: string, position?: { x: number; y: number }): void {
  const offset = position ? 0 : nextPosition++ * 28
  applyCommand({
    kind: 'add-node',
    nodeTypeId,
    position: position ?? { x: 100 + offset, y: 100 + offset },
  })
}

function toggleAIReview(): void {
  aiPanelOpen.value = !aiPanelOpen.value
  if (aiPanelOpen.value) statePanelOpen.value = false
}

function toggleStatePanel(): void {
  statePanelOpen.value = !statePanelOpen.value
  if (statePanelOpen.value) aiPanelOpen.value = false
}

function handleEditorKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape' && connectionMenu.value) {
    event.preventDefault()
    closeConnectionMenu()
    return
  }
  const target = event.target as HTMLElement | null
  if (
    target?.matches('input, textarea, select, [contenteditable="true"]') ||
    target?.closest('[role="dialog"]')
  )
    return
  const modifier = event.ctrlKey || event.metaKey
  if (modifier && !event.altKey) {
    const key = event.key.toLocaleLowerCase()
    if (key === 'c' && selectedNodeIds.value.size) {
      event.preventDefault()
      void copySelection()
      return
    }
    if (key === 'x' && selectedNodeIds.value.size) {
      event.preventDefault()
      void cutSelection()
      return
    }
    if (key === 'v') {
      event.preventDefault()
      void pasteSelection()
      return
    }
    if (key === 'd' && selectedNodeIds.value.size) {
      event.preventDefault()
      duplicateSelection()
      return
    }
    return
  }
  if (event.altKey || (event.key !== 'Delete' && event.key !== 'Backspace')) return
  if (!selectedNodeIds.value.size && !selectedNodeId.value) return
  event.preventDefault()
  removeSelection()
}

function startNodeDrag(event: DragEvent, nodeTypeId: string): void {
  if (!event.dataTransfer) return
  event.dataTransfer.effectAllowed = 'copy'
  event.dataTransfer.setData(NODE_TYPE_DRAG_FORMAT, nodeTypeId)
  nodeDragActive.value = true
}

function continueNodeDrag(event: DragEvent): void {
  if (!event.dataTransfer?.types.includes(NODE_TYPE_DRAG_FORMAT)) return
  event.preventDefault()
  event.dataTransfer.dropEffect = 'copy'
  nodeDragActive.value = true
}

function finishNodeDrag(): void {
  nodeDragActive.value = false
}

function dropNode(event: DragEvent): void {
  const nodeTypeId = event.dataTransfer?.getData(NODE_TYPE_DRAG_FORMAT)
  if (nodeTypeId) event.preventDefault()
  finishNodeDrag()
  if (!nodeTypeId) return
  addNode(nodeTypeId, screenToFlowCoordinate({ x: event.clientX, y: event.clientY }))
}

function connect(connection: Connection): void {
  const edge = connectionEdge(connection)
  if (!edge) return
  const compatibility = session.connectionCompatibility(edge)
  if (!compatibility.valid) {
    connectionHint.value = connectionIssueText(compatibility.issue)
    return
  }
  if (applyCommand({ kind: 'connect', edge })) {
    connectionMadeThisGesture = true
    connectionHint.value = ''
  }
}

function isValidConnection(connection: Connection): boolean {
  const edge = connectionEdge(connection)
  if (!edge) return false
  const compatibility = session.connectionCompatibility(edge)
  connectionHint.value = compatibility.valid ? '' : connectionIssueText(compatibility.issue)
  return compatibility.valid
}

function startConnection(params: OnConnectStartParams): void {
  connectionMadeThisGesture = false
  connectionHint.value = ''
  closeConnectionMenu()
  const handle = parseGraphHandle(params.handleId)
  connectionStart.value = params.nodeId && handle ? { nodeId: params.nodeId, handle } : null
}

function endConnection(event?: MouseEvent | TouchEvent): void {
  const anchor = connectionStart.value
  connectionStart.value = null
  const point = eventClientPoint(event)
  clearTimeout(connectionEndTimer)
  connectionEndTimer = setTimeout(() => {
    if (connectionMadeThisGesture) {
      connectionMadeThisGesture = false
      return
    }
    if (anchor && point) openConnectionMenu(anchor, point)
  }, 0)
}

function openConnectionMenu(anchor: ConnectionAnchor, point: { x: number; y: number }): void {
  const bounds = canvasElement.value?.getBoundingClientRect()
  if (!bounds) return
  connectionMenu.value = {
    anchor,
    flowPosition: screenToFlowCoordinate(point),
    canvasPosition: {
      x: Math.max(8, Math.min(point.x - bounds.left, Math.max(8, bounds.width - 328))),
      y: Math.max(8, Math.min(point.y - bounds.top, Math.max(8, bounds.height - 424))),
    },
  }
  connectionHint.value = ''
}

function closeConnectionMenu(): void {
  connectionMenu.value = null
}

function selectConnectionCandidate(candidate: WorkflowConnectionCandidate): void {
  const menu = connectionMenu.value
  if (!menu) return
  if (!candidate.handle) {
    addNode(candidate.nodeTypeId, menu.flowPosition)
    closeConnectionMenu()
    return
  }
  try {
    selectedNodeId.value = session.insertConnectedNode(
      menu.anchor.nodeId,
      menu.anchor.handle,
      candidate.nodeTypeId,
      candidate.handle,
      menu.flowPosition,
    )
    closeConnectionMenu()
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

function handlePaneClick(): void {
  selectedNodeId.value = ''
  selectedNodeIds.value = new Set()
  closeConnectionMenu()
}

function dragPositions(
  event: NodeDragEvent,
): Array<{ nodeId: string; position: { x: number; y: number } }> {
  const dragged = event.nodes.length ? event.nodes : [event.node]
  const draggedIds = new Set(dragged.map((node) => node.id))
  const primary = sizedFlowNode(event.node.id, event.node.position)
  const disableSnap = event.event instanceof MouseEvent && event.event.altKey
  const snapped = disableSnap
    ? { position: event.node.position }
    : snapNodePosition(
        primary,
        (session.currentGraph?.nodes ?? []).flatMap((node) => {
          if (draggedIds.has(node.id)) return []
          return [sizedFlowNode(node.id, node.position)]
        }),
      )
  const delta = {
    x: snapped.position.x - event.node.position.x,
    y: snapped.position.y - event.node.position.y,
  }
  updateSnapGuides(snapped.guideX, snapped.guideY)
  return dragged.map((node) => ({
    nodeId: node.id,
    position: { x: node.position.x + delta.x, y: node.position.y + delta.y },
  }))
}

function sizedFlowNode(nodeId: string, position: { x: number; y: number }): SizedWorkflowNode {
  const dimensions = findNode(nodeId)?.dimensions
  const element = [
    ...(canvasElement.value?.querySelectorAll<HTMLElement>('.vue-flow__node') ?? []),
  ].find((candidate) => candidate.dataset.id === nodeId)
  return {
    id: nodeId,
    position,
    width: element?.offsetWidth || dimensions?.width || 230,
    height: element?.offsetHeight || dimensions?.height || 90,
  }
}

function selectedSizedNodes(): SizedWorkflowNode[] {
  return [...selectedNodeIds.value].flatMap((nodeId) => {
    const node = session.currentGraph?.nodes.find((candidate) => candidate.id === nodeId)
    return node ? [sizedFlowNode(nodeId, node.position)] : []
  })
}

function updateSnapGuides(guideX?: number, guideY?: number): void {
  const bounds = canvasElement.value?.getBoundingClientRect()
  if (!bounds) {
    snapGuides.value = {}
    return
  }
  snapGuides.value = {
    x:
      guideX === undefined
        ? undefined
        : flowToScreenCoordinate({ x: guideX, y: 0 }).x - bounds.left,
    y:
      guideY === undefined ? undefined : flowToScreenCoordinate({ x: 0, y: guideY }).y - bounds.top,
  }
}

function alignSelection(mode: AlignMode): void {
  const positions = alignNodePositions(selectedSizedNodes(), mode)
  if (positions.length) applyCommand({ kind: 'move-nodes', positions })
}

function distributeSelection(mode: DistributeMode): void {
  const positions = distributeNodePositions(selectedSizedNodes(), mode)
  if (positions.length) applyCommand({ kind: 'move-nodes', positions })
}

async function autoLayout(direction: 'LR' | 'TB'): Promise<void> {
  if (layouting.value) return
  const graph = session.currentGraph
  const source = session.source
  if (!graph || !source || graph.nodes.length === 0) return
  const nodes = graph.nodes.map((node) => sizedFlowNode(node.id, node.position))
  layouting.value = true
  try {
    const positions = await autoLayoutNodePositions(nodes, graph.edges, direction)
    if (session.source !== source || session.currentGraph?.id !== graph.id) return
    if (applyCommand({ kind: 'move-nodes', positions })) {
      await nextTick()
      await fitView({ padding: 0.18, duration: 180 })
    }
  } catch (error) {
    showError(t('workflow.selection.layout_failed'), error)
  } finally {
    layouting.value = false
  }
}

function removeSelection(): void {
  const ids = selectedNodeIds.value.size
    ? [...selectedNodeIds.value]
    : selectedNodeId.value
      ? [selectedNodeId.value]
      : []
  if (ids.length && applyCommand({ kind: 'remove-nodes', nodeIds: ids })) {
    selectedNodeId.value = ''
    selectedNodeIds.value = new Set()
  }
}

function duplicateSelection(): void {
  try {
    const ids = session.duplicateNodes([...selectedNodeIds.value])
    if (ids.length) void selectInsertedNodes(ids)
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

async function copySelection(): Promise<void> {
  const snapshot = session.selectionSnapshot([...selectedNodeIds.value])
  if (!snapshot.nodes.length) return
  workflowClipboard = {
    format: 'yotta.workflow-selection',
    version: 1,
    ...snapshot,
  }
  pasteOffset = 0
  try {
    await navigator.clipboard?.writeText(JSON.stringify(workflowClipboard))
  } catch {
    return
  }
}

async function cutSelection(): Promise<void> {
  await copySelection()
  removeSelection()
}

async function pasteSelection(): Promise<void> {
  let clipboard = workflowClipboard
  try {
    const text = await navigator.clipboard?.readText()
    if (text) clipboard = parseWorkflowClipboard(text)
  } catch {
    if (!clipboard) {
      showError(t('workflow.selection.clipboard_failed'), new Error('clipboard is unavailable'))
      return
    }
  }
  if (!clipboard) return
  try {
    pasteOffset += 24
    const ids = session.insertNodeSelection(clipboard, { x: pasteOffset, y: pasteOffset })
    if (ids.length) await selectInsertedNodes(ids)
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

async function selectInsertedNodes(nodeIds: string[]): Promise<void> {
  await nextTick()
  removeSelectedNodes(getSelectedNodes.value)
  const nodes = nodeIds.flatMap((nodeId) => {
    const node = findNode(nodeId)
    return node ? [node] : []
  })
  if (nodes.length) addSelectedNodes(nodes)
  selectedNodeIds.value = new Set(nodeIds)
  selectedNodeId.value = nodeIds.at(-1) ?? ''
}

function parseWorkflowClipboard(value: string): WorkflowSelectionClipboard {
  if (value.length > 1_000_000) throw new Error('workflow clipboard exceeds size budget')
  const parsed = JSON.parse(value) as Partial<WorkflowSelectionClipboard>
  if (
    parsed.format !== 'yotta.workflow-selection' ||
    parsed.version !== 1 ||
    !Array.isArray(parsed.nodes) ||
    !Array.isArray(parsed.edges)
  ) {
    throw new Error('clipboard does not contain a workflow selection')
  }
  return parsed as WorkflowSelectionClipboard
}

function connectionEdge(connection: Connection): Edge | null {
  const source = parseGraphHandle(connection.sourceHandle)
  const target = parseGraphHandle(connection.targetHandle)
  if (!source || !target || source.direction !== 'output' || target.direction !== 'input')
    return null
  if (source.channel !== target.channel) return null
  return {
    channel: source.channel,
    from: { nodeId: connection.source, portId: source.portId },
    to: { nodeId: connection.target, portId: target.portId },
  }
}

function connectionIssueText(issue?: ConnectionIssue): string {
  return t(`workflow.connection.issue.${issue ?? 'port'}`)
}

function eventClientPoint(event?: MouseEvent | TouchEvent): { x: number; y: number } | null {
  if (!event) return null
  if (event instanceof MouseEvent) return { x: event.clientX, y: event.clientY }
  const touch = event.changedTouches[0]
  return touch ? { x: touch.clientX, y: touch.clientY } : null
}

function disconnect(event: EdgeMouseEvent): void {
  const edge = session.currentGraph?.edges.find((candidate) => edgeId(candidate) === event.edge.id)
  if (edge) session.apply({ kind: 'disconnect', edge })
}

function selectNode(event: NodeMouseEvent): void {
  selectedNodeId.value = event.node.id
}

function handleNodesChange(changes: NodeChange[]): void {
  const selected = new Set(selectedNodeIds.value)
  let changed = false
  for (const change of changes) {
    if (change.type !== 'select') continue
    changed = true
    if (change.selected) selected.add(change.id)
    else selected.delete(change.id)
  }
  if (changed) {
    selectedNodeIds.value = selected
    if (!selected.has(selectedNodeId.value)) selectedNodeId.value = [...selected].at(-1) ?? ''
  }
}

function trackNodeDrag(event: NodeDragEvent): void {
  const positions = dragPositions(event)
  for (const item of positions) {
    nodeGestures.track(item.nodeId, item.position)
    updateNode(item.nodeId, { position: item.position })
  }
}

function moveNode(event: NodeDragEvent): void {
  const positions = dragPositions(event)
  for (const item of positions) nodeGestures.track(item.nodeId, item.position)
  applyCommand({ kind: 'move-nodes', positions })
  for (const item of positions) nodeGestures.clear(item.nodeId)
  snapGuides.value = {}
}

function renameWorkflow(name: string): void {
  if (name.trim() && name !== session.source?.workflow.name)
    session.apply({ kind: 'rename-workflow', name })
}

async function compile(): Promise<void> {
  setCompileSucceeded(false)
  try {
    const result = await session.validate()
    diagnosticsOpen.value = result.diagnostics.length > 0
    if (result.diagnostics.length === 0) setCompileSucceeded(true)
  } catch (error) {
    showError(t('workflow.toast.compile_failed'), error)
  }
}

async function save(): Promise<void> {
  setSaveSucceeded(false)
  try {
    await session.save()
    setSaveSucceeded(true)
  } catch (error) {
    showError(t('workflow.toast.save_failed'), error)
  }
}

async function acceptAIProposal(): Promise<void> {
  selectedNodeId.value = ''
  try {
    await session.load(session.workflowId)
  } catch (error) {
    showError(t('workflow.ai.refresh_failed'), error)
  }
}

async function startRun(): Promise<void> {
  try {
    const run = await session.run()
    diagnosticsOpen.value = session.diagnostics.length > 0
    if (run) runTimelineOpen.value = true
  } catch (error) {
    showError(t('workflow.toast.run_failed'), error)
  }
}

async function startDebug(): Promise<void> {
  try {
    const run = await session.startDebug(debugBreakpoints())
    diagnosticsOpen.value = session.diagnostics.length > 0
    if (!run) return
    debuggerOpen.value = true
    runTimelineOpen.value = false
    const snapshot = session.debugSnapshot
    if (snapshot?.status === 'paused' && snapshot.nodeId) {
      await focusNode(snapshot.graphId ? [snapshot.graphId] : [], snapshot.nodeId)
    }
  } catch (error) {
    showError(t('workflow.toast.debug_failed'), error)
  }
}

async function controlDebug(action: 'continue' | 'pause' | 'step'): Promise<void> {
  try {
    await session.controlDebug(action)
  } catch (error) {
    showError(t('workflow.toast.debug_failed'), error)
  }
}

async function toggleBreakpoint(graphId: string, nodeId: string): Promise<void> {
  if (!graphId || !nodeId) return
  const key = breakpointKey(graphId, nodeId)
  const hadBreakpoint = breakpointKeys.value.has(key)
  const next = new Set(breakpointKeys.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  breakpointKeys.value = next
  if (!session.debugSnapshot || session.debugSnapshot.status === 'completed') return
  try {
    await session.setDebugBreakpoints(debugBreakpoints())
  } catch (error) {
    const rollback = new Set(breakpointKeys.value)
    if (hadBreakpoint) rollback.add(key)
    else rollback.delete(key)
    breakpointKeys.value = rollback
    showError(t('workflow.toast.debug_failed'), error)
  }
}

function debugBreakpoints(): DebugBreakpoint[] {
  return [...breakpointKeys.value].map((key) => {
    const separator = key.indexOf('\u0000')
    return { graphId: key.slice(0, separator), nodeId: key.slice(separator + 1) } as DebugBreakpoint
  })
}

function hasBreakpoint(graphId: string, nodeId: string): boolean {
  return breakpointKeys.value.has(breakpointKey(graphId, nodeId))
}

function isDebugCurrent(graphId: string, nodeId: string): boolean {
  const snapshot = session.debugSnapshot
  return snapshot?.status === 'paused' && snapshot.graphId === graphId && snapshot.nodeId === nodeId
}

function breakpointKey(graphId: string, nodeId: string): string {
  return `${graphId}\u0000${nodeId}`
}

async function cancelRun(): Promise<void> {
  try {
    await session.cancelRun()
  } catch (error) {
    showError(t('workflow.toast.stop_failed'), error)
  }
}

async function refreshRun(): Promise<void> {
  try {
    await session.refreshRun()
  } catch (error) {
    showError(t('workflow.toast.refresh_failed'), error)
  }
}

async function focusDiagnostic(diagnostic: WorkflowDiagnostic): Promise<void> {
  if (!diagnostic.nodeId) return
  await focusNode(diagnostic.graphPath ?? [], diagnostic.nodeId)
}

async function focusNode(graphPath: string[], nodeId: string): Promise<void> {
  try {
    session.openGraphPath(graphPath)
  } catch (error) {
    showError(t('workflow.diagnostics.locate_failed'), error)
    return
  }
  await nextTick()
  removeSelectedNodes(getSelectedNodes.value)
  const node = findNode(nodeId)
  if (!node) return
  addSelectedNodes([node])
  selectedNodeIds.value = new Set([nodeId])
  selectedNodeId.value = nodeId
  const width = node.dimensions.width || 230
  const height = node.dimensions.height || 116
  await setCenter(node.position.x + width / 2, node.position.y + height / 2, {
    zoom: 1,
    duration: 180,
  })
}

function projectionTitle(projection: NodeProjection): string {
  if (projection.titleKey && te(projection.titleKey)) return t(projection.titleKey)
  return (
    projection.nodeRef.nodeTypeId.split('/').filter(Boolean).at(-2) ?? projection.nodeRef.nodeTypeId
  )
}

function categoryLabel(category: string): string {
  const key = `workflow.catalog.category.${category}`
  return te(key) ? t(key) : category
}

function catalogSearchText(projection: NodeProjection): string {
  const description =
    projection.descriptionKey && te(projection.descriptionKey) ? t(projection.descriptionKey) : ''
  return [
    projectionTitle(projection),
    description,
    projection.category,
    projection.execution.class,
    projection.nodeRef.nodeTypeId,
    ...projection.tags,
  ]
    .filter(Boolean)
    .join(' ')
    .toLocaleLowerCase()
}

function edgeId(edge: {
  channel: string
  from: { nodeId: string; portId: string }
  to: { nodeId: string; portId: string }
}): string {
  return `${edge.channel}:${edge.from.nodeId}:${edge.from.portId}:${edge.to.nodeId}:${edge.to.portId}`
}

function showError(title: string, error: unknown): void {
  toast.add({
    title,
    description: error instanceof Error ? error.message : String(error),
    color: 'error',
  })
}

function setCompileSucceeded(value: boolean): void {
  clearTimeout(compileFlashTimer)
  compileSucceeded.value = value
  if (value) compileFlashTimer = setTimeout(() => (compileSucceeded.value = false), 1600)
}

function setSaveSucceeded(value: boolean): void {
  clearTimeout(saveFlashTimer)
  saveSucceeded.value = value
  if (value) saveFlashTimer = setTimeout(() => (saveSucceeded.value = false), 1600)
}
</script>

<style scoped src="./WorkflowEditorView.css"></style>
