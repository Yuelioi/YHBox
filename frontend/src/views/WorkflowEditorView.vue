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
        @back="router.push('/workflows')"
        @rename="renameWorkflow"
        @undo="session.undo()"
        @redo="session.redo()"
        @toggle-ai="toggleAIReview"
        @toggle-state="toggleStatePanel"
        @compile="compile"
        @debug="startDebug"
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
      <div
        v-if="session.diagnostics.length"
        class="flex max-h-28 shrink-0 gap-2 overflow-x-auto border-b border-default bg-elevated/30 px-4 py-2"
      >
        <button
          v-for="(diagnostic, index) in session.diagnostics"
          :key="`${diagnostic.code}:${index}`"
          type="button"
          class="shrink-0 rounded-lg border px-3 py-1.5 text-left text-[11px]"
          :class="
            diagnostic.severity === 'error'
              ? 'border-error/35 bg-error/10 text-error'
              : 'border-warning/35 bg-warning/10 text-warning'
          "
          @click="selectDiagnostic(diagnostic.nodeId)"
        >
          <span class="font-mono">{{ diagnostic.code }}</span>
          <span v-if="diagnostic.nodeId" class="ml-2 text-muted">{{ diagnostic.nodeId }}</span>
        </button>
      </div>

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
            fit-view-on-init
            :min-zoom="0.2"
            :max-zoom="2"
            class="workflow-flow"
            @connect="connect"
            @node-click="selectNode"
            @pane-click="selectedNodeId = ''"
            @node-drag-stop="moveNode"
            @edge-double-click="disconnect"
          >
            <template #node-workflow="slotProps">
              <WorkflowNode
                :node="slotProps.data.node"
                :projection="slotProps.data.projection"
                :selected="slotProps.selected"
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
        v-if="session.activeRun"
        :run="session.activeRun"
        @cancel="cancelRun"
        @refresh="refreshRun"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import { useToast } from '@nuxt/ui/composables'
import {
  VueFlow,
  useVueFlow,
  type Connection,
  type Edge as FlowEdge,
  type EdgeMouseEvent,
  type Node as FlowNode,
  type NodeDragEvent,
  type NodeMouseEvent,
} from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/controls/dist/style.css'
import '@vue-flow/minimap/dist/style.css'
import { useI18n } from 'vue-i18n'
import { type EditorCommand, type Node, type NodeProjection } from '@/app/editor/EditorSession'
import { createEditorSession } from '@/app/editor/createEditorSession'
import { graphHandle, parseGraphHandle } from '@/app/editor/graphHandles'
import { onRunChanged, workflowTransport } from '@/app/transport/workflow'
import { useConfirm } from '@/composables/useConfirm'
import WorkflowNode from '@/app/editor/WorkflowNode.vue'
import WorkflowInspector from '@/app/editor/WorkflowInspector.vue'
import AIWorkflowReviewPanel from '@/app/editor/AIWorkflowReviewPanel.vue'
import RunTimelinePanel from '@/app/editor/RunTimelinePanel.vue'
import WorkflowEditorToolbar from '@/app/editor/WorkflowEditorToolbar.vue'
import WorkflowStatePanel from '@/app/editor/WorkflowStatePanel.vue'

defineOptions({ name: 'WorkflowEditorView' })

interface WorkflowNodeData {
  node: Node
  projection: NodeProjection
}

const route = useRoute()
const router = useRouter()
const toast = useToast()
const { confirm } = useConfirm()
const { t, te } = useI18n()
const session = createEditorSession(workflowTransport)
const selectedNodeId = ref('')
const nodeDragActive = ref(false)
const aiPanelOpen = ref(false)
const statePanelOpen = ref(false)
const catalogQuery = ref('')
const compileSucceeded = ref(false)
const saveSucceeded = ref(false)
const { screenToFlowCoordinate } = useVueFlow()
let unsubscribeRun: (() => void) | undefined
let compileFlashTimer: ReturnType<typeof setTimeout> | undefined
let saveFlashTimer: ReturnType<typeof setTimeout> | undefined
let nextPosition = 0

const NODE_TYPE_DRAG_FORMAT = 'application/x-yotta-node-type'
const RUN_STARTED_NODE_ID = 'https://schemas.yotta.dev/nodes/event/run-started'

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

const flowNodes = computed<FlowNode<WorkflowNodeData, Record<string, never>, 'workflow'>[]>(() =>
  (session.currentGraph?.nodes ?? []).flatMap((node) => {
    const projection = session.nodeProjection(node.nodeRef.nodeTypeId)
    if (!projection) return []
    return [
      {
        id: node.id,
        type: 'workflow',
        position: node.position,
        selected: node.id === selectedNodeId.value,
        data: { node, projection },
      },
    ]
  }),
)

const flowEdges = computed<FlowEdge[]>(() =>
  (session.currentGraph?.edges ?? []).map((edge) => ({
    id: edgeId(edge),
    source: edge.from.nodeId,
    target: edge.to.nodeId,
    sourceHandle: graphHandle(edge.channel, 'output', edge.from.portId),
    targetHandle: graphHandle(edge.channel === 'data' ? 'data' : 'exec', 'input', edge.to.portId),
    animated: edge.channel !== 'data',
    style: {
      stroke:
        edge.channel === 'error' ? '#f87171' : edge.channel === 'exec' ? '#a1a1aa' : '#10b981',
    },
  })),
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
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleEditorKeydown)
  unsubscribeRun?.()
  clearTimeout(compileFlashTimer)
  clearTimeout(saveFlashTimer)
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

function applyCommand(command: EditorCommand): void {
  try {
    session.apply(command)
    if (command.kind === 'remove-node' && command.nodeId === selectedNodeId.value)
      selectedNodeId.value = ''
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
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
  if (!selectedNodeId.value || event.ctrlKey || event.metaKey || event.altKey) return
  if (event.key !== 'Delete' && event.key !== 'Backspace') return
  const target = event.target as HTMLElement | null
  if (
    target?.matches('input, textarea, select, [contenteditable="true"]') ||
    target?.closest('[role="dialog"]')
  )
    return
  event.preventDefault()
  applyCommand({ kind: 'remove-node', nodeId: selectedNodeId.value })
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
  const source = parseGraphHandle(connection.sourceHandle)
  const target = parseGraphHandle(connection.targetHandle)
  if (!source || !target || source.direction !== 'output' || target.direction !== 'input') return
  if (source.channel === 'data' ? target.channel !== 'data' : target.channel !== 'exec') return
  applyCommand({
    kind: 'connect',
    edge: {
      channel: source.channel,
      from: { nodeId: connection.source, portId: source.portId },
      to: { nodeId: connection.target, portId: target.portId },
    },
  })
}

function disconnect(event: EdgeMouseEvent): void {
  const edge = session.currentGraph?.edges.find((candidate) => edgeId(candidate) === event.edge.id)
  if (edge) session.apply({ kind: 'disconnect', edge })
}

function selectNode(event: NodeMouseEvent): void {
  selectedNodeId.value = event.node.id
}

function moveNode(event: NodeDragEvent): void {
  session.apply({ kind: 'move-node', nodeId: event.node.id, position: event.node.position })
}

function renameWorkflow(name: string): void {
  if (name.trim() && name !== session.source?.workflow.name)
    session.apply({ kind: 'rename-workflow', name })
}

async function compile(): Promise<void> {
  setCompileSucceeded(false)
  try {
    const result = await session.validate()
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
    await session.run()
  } catch (error) {
    showError(t('workflow.toast.run_failed'), error)
  }
}

async function startDebug(): Promise<void> {
  try {
    await session.debug()
  } catch (error) {
    showError(t('workflow.toast.debug_failed'), error)
  }
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

function selectDiagnostic(nodeId?: string): void {
  if (nodeId) selectedNodeId.value = nodeId
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
