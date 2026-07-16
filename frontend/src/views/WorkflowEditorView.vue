<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden bg-default">
    <div v-if="session.phase === 'loading'" class="flex flex-1 items-center justify-center px-8">
      <div class="w-full max-w-xl space-y-3" :aria-label="t('workflow31.editor.loading')">
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
          {{ t('workflow31.editor.open_failed') }}
        </h1>
        <p class="mt-2 text-xs leading-5 text-muted">{{ session.failure }}</p>
        <UButton
          class="mt-4"
          :label="t('workflow31.editor.back')"
          color="neutral"
          @click="router.push('/workflows')"
        />
      </div>
    </div>

    <template v-else-if="session.source && session.authoring">
      <header class="flex h-13 shrink-0 items-center gap-2 border-b border-default bg-default px-3">
        <UButton
          icon="i-tabler-arrow-left"
          color="neutral"
          variant="ghost"
          size="xs"
          :aria-label="t('workflow31.editor.back')"
          @click="router.push('/workflows')"
        />
        <UInput
          :model-value="session.source.workflow.name"
          class="w-56"
          :aria-label="t('workflow31.editor.workflow_name')"
          @change="renameWorkflow"
        />
        <span class="font-mono text-[10px] text-dimmed">
          {{ t('workflow31.editor.revision', { n: session.baseRevision }) }}
        </span>
        <span v-if="session.dirty" class="text-[11px] font-medium text-warning">
          {{ t('workflow31.editor.unsaved') }}
        </span>

        <div class="mx-2 h-5 w-px bg-default" />
        <UButton
          icon="i-tabler-arrow-back-up"
          color="neutral"
          variant="ghost"
          size="xs"
          :disabled="!session.canUndo"
          :aria-label="t('workflow31.action.undo')"
          @click="session.undo()"
        />
        <UButton
          icon="i-tabler-arrow-forward-up"
          color="neutral"
          variant="ghost"
          size="xs"
          :disabled="!session.canRedo"
          :aria-label="t('workflow31.action.redo')"
          @click="session.redo()"
        />

        <div class="flex-1" />
        <UButton
          :label="t('workflow31.action.compile')"
          icon="i-tabler-file-check"
          color="neutral"
          variant="ghost"
          size="xs"
          @click="compile"
        />
        <UButton
          :label="t('workflow31.action.debug')"
          icon="i-tabler-bug"
          color="neutral"
          variant="soft"
          size="xs"
          @click="startDebug"
        />
        <UButton
          v-if="runActive"
          :label="t('workflow31.action.stop')"
          icon="i-tabler-square"
          color="error"
          variant="soft"
          size="xs"
          @click="cancelRun"
        />
        <UButton
          v-else
          :label="t('workflow31.action.run')"
          icon="i-tabler-player-play"
          size="xs"
          @click="startRun"
        />
        <UButton
          :label="t('workflow31.action.save')"
          icon="i-tabler-device-floppy"
          size="xs"
          :loading="session.phase === 'saving'"
          :disabled="!session.dirty"
          @click="save"
        />
      </header>

      <div
        v-if="session.saveConflict"
        class="border-b border-error/35 bg-error/10 px-4 py-2 text-xs text-error"
        role="alert"
      >
        {{ t('workflow31.editor.save_conflict', { message: session.saveConflict }) }}
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
              {{ t('workflow31.editor.node_catalog') }}
            </h2>
            <p class="mt-1 text-[11px] leading-4 text-muted">
              {{ t('workflow31.editor.catalog_description') }}
            </p>
          </div>
          <div class="flex-1 space-y-1 overflow-y-auto p-2">
            <button
              v-for="projection in session.authoring.body.nodes"
              :key="projection.nodeRef.nodeTypeId"
              type="button"
              draggable="true"
              data-testid="node-catalog-item"
              :data-node-type-id="projection.nodeRef.nodeTypeId"
              class="group flex w-full cursor-grab items-center gap-2 rounded-lg px-2.5 py-2 text-left transition-colors hover:bg-elevated active:cursor-grabbing active:translate-y-px"
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
              <UIcon name="i-tabler-plus" class="size-3.5 text-dimmed group-hover:text-primary" />
            </button>
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
            <MiniMap position="bottom-right" :pannable="true" :zoomable="true" />
          </VueFlow>
        </div>

        <WorkflowInspector
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
import { onRunChanged, workflowTransport } from '@/app/transport/workflow31'
import WorkflowNode from '@/app/editor/WorkflowNode.vue'
import WorkflowInspector from '@/app/editor/WorkflowInspector.vue'
import RunTimelinePanel from '@/app/editor/RunTimelinePanel.vue'

defineOptions({ name: 'WorkflowEditorView' })

interface WorkflowNodeData {
  node: Node
  projection: NodeProjection
}

const route = useRoute()
const router = useRouter()
const toast = useToast()
const { t, te } = useI18n()
const session = createEditorSession(workflowTransport)
const selectedNodeId = ref('')
const nodeDragActive = ref(false)
const { screenToFlowCoordinate } = useVueFlow()
let unsubscribeRun: (() => void) | undefined
let nextPosition = 0

const NODE_TYPE_DRAG_FORMAT = 'application/x-yotta-node-type'

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

onBeforeUnmount(() => unsubscribeRun?.())
onBeforeRouteLeave(() => !session.dirty || window.confirm(t('workflow31.editor.discard_confirm')))

function applyCommand(command: EditorCommand): void {
  try {
    session.apply(command)
    if (command.kind === 'remove-node' && command.nodeId === selectedNodeId.value)
      selectedNodeId.value = ''
  } catch (error) {
    showError(t('workflow31.toast.edit_rejected'), error)
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

function renameWorkflow(event: Event): void {
  const name = (event.target as HTMLInputElement).value
  if (name.trim() && name !== session.source?.workflow.name)
    session.apply({ kind: 'rename-workflow', name })
}

async function compile(): Promise<void> {
  try {
    const result = await session.validate()
    toast.add({
      title: result.diagnostics.length
        ? t('workflow31.toast.compile_diagnostics')
        : t('workflow31.toast.compile_succeeded'),
      description: result.programHash || result.sourceHash,
      color: result.diagnostics.some((diagnostic) => diagnostic.severity === 'error')
        ? 'warning'
        : 'success',
    })
  } catch (error) {
    showError(t('workflow31.toast.compile_failed'), error)
  }
}

async function save(): Promise<void> {
  try {
    await session.save()
    toast.add({ title: t('workflow31.toast.saved'), color: 'success' })
  } catch (error) {
    showError(t('workflow31.toast.save_failed'), error)
  }
}

async function startRun(): Promise<void> {
  try {
    const run = await session.run()
    if (run)
      toast.add({
        title: t('workflow31.toast.queued'),
        description: run.runId,
        color: 'success',
      })
  } catch (error) {
    showError(t('workflow31.toast.run_failed'), error)
  }
}

async function startDebug(): Promise<void> {
  try {
    const run = await session.debug()
    if (run)
      toast.add({
        title: t('workflow31.toast.debug_started'),
        description: run.runId,
        color: 'success',
      })
  } catch (error) {
    showError(t('workflow31.toast.debug_failed'), error)
  }
}

async function cancelRun(): Promise<void> {
  try {
    await session.cancelRun()
  } catch (error) {
    showError(t('workflow31.toast.stop_failed'), error)
  }
}

async function refreshRun(): Promise<void> {
  try {
    await session.refreshRun()
  } catch (error) {
    showError(t('workflow31.toast.refresh_failed'), error)
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
</script>

<style scoped>
.workflow-flow :deep(.vue-flow__edge-path) {
  stroke-width: 2;
}

.workflow-flow :deep(.vue-flow__controls),
.workflow-flow :deep(.vue-flow__minimap) {
  border: 1px solid var(--ui-border);
  border-radius: 8px;
  overflow: hidden;
  background: var(--ui-bg-elevated);
}

@media (prefers-reduced-motion: reduce) {
  .workflow-flow :deep(.vue-flow__edge-path) {
    animation: none !important;
  }
}
</style>
