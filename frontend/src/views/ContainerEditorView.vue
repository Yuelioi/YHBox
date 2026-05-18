<template>
  <div class="flex flex-col h-screen bg-default text-default">
    <!-- 顶部窗口拖区：左返回+标题，右窗口控件（min/max/close） -->
    <header
      class="h-12 shrink-0 flex items-center gap-2 border-b border-default pl-3 pr-0"
      style="--wails-draggable: drag"
    >
      <UButton
        size="xs"
        variant="ghost"
        color="neutral"
        icon="i-tabler-arrow-left"
        @click="goBack"
        style="--wails-draggable: no-drag"
        >返回</UButton
      >
      <UIcon name="i-tabler-schema" class="size-3.5 text-dimmed shrink-0" />
      <h3 class="text-xs font-medium truncate text-toned">
        {{ draft?.name ?? '加载中...' }}
      </h3>
      <span v-if="dirty" class="text-[10px] text-amber-300/80 shrink-0">· 未保存</span>

      <div class="flex-1" />

      <!-- 窗口控件（min / max-restore / close）-->
      <div class="flex items-stretch h-full" style="--wails-draggable: no-drag">
        <button
          type="button"
          class="w-11 flex items-center justify-center text-muted hover:bg-elevated/60 hover:text-highlighted transition-colors"
          title="最小化"
          @click="onMinimise"
        >
          <UIcon name="i-tabler-minus" class="size-4" />
        </button>
        <button
          type="button"
          class="w-11 flex items-center justify-center text-muted hover:bg-elevated/60 hover:text-highlighted transition-colors"
          :title="isMaximised ? '还原' : '最大化'"
          @click="onToggleMaximise"
        >
          <UIcon :name="isMaximised ? 'i-tabler-copy' : 'i-tabler-square'" class="size-3.5" />
        </button>
        <button
          type="button"
          class="w-11 flex items-center justify-center text-muted hover:bg-error hover:text-highlighted transition-colors"
          title="关闭"
          @click="onClose"
        >
          <UIcon name="i-tabler-x" class="size-4" />
        </button>
      </div>
    </header>

    <!-- Dirty 关闭确认 -->
    <UModal
      :open="confirmCloseOpen"
      :ui="{ content: 'sm:max-w-[460px]' }"
      @update:open="
        (v: boolean) => {
          if (!v) confirmCloseOpen = false
        }
      "
    >
      <template #content>
        <div class="p-6 space-y-4 bg-default">
          <div class="flex items-center gap-2">
            <UIcon name="i-tabler-alert-triangle" class="size-4 text-warning" />
            <h3 class="text-sm font-medium">未保存的修改</h3>
          </div>
          <p class="text-xs text-muted">当前容器有未保存的修改。继续将丢失这些改动。</p>
          <div class="flex justify-end gap-2 pt-2">
            <UButton variant="ghost" color="neutral" @click="confirmCloseOpen = false"
              >取消</UButton
            >
            <UButton
              class="ml-auto"
              color="error"
              icon="i-tabler-x"
              @click="onConfirmDiscardAndClose"
              >丢弃并关闭</UButton
            >
            <UButton color="primary" icon="i-tabler-check" @click="onSaveAndClose"
              >保存并关闭</UButton
            >
          </div>
        </div>
      </template>
    </UModal>

    <div v-if="!draft" class="flex-1 flex items-center justify-center text-sm text-muted">
      加载中...
    </div>

    <div v-else class="flex flex-col flex-1 min-h-0">
      <!-- Toolbar 独立一行：左 [折叠 palette] [录制] [折叠 inspector]，右 [运行状态] [试运行/停止] [保存] -->
      <ContainerEditorToolbar
        v-model:palette-collapsed="paletteCollapsed"
        v-model:inspector-collapsed="inspectorCollapsed"
        :is-recording="recordStore.isRecording"
        :countdown-sec="countdownSec"
        :selected-count="selectedCount"
        :exec-store-running="execStore.running"
        :running-node-kind="execStore.currentNodeKind ?? undefined"
        :running-node-label="runningNodeLabel"
        :dirty="dirty"
        @record="(mode) => startRecording(mode)"
        @stop-record="stopRecording"
        @cancel-countdown="startRecording('precise')"
        @fold="onFoldSelection"
        @try-run="onTryRun"
        @stop-run="onStopRun"
        @save="onSave"
        @auto-layout="onAutoLayout"
        @align-selected="onAlignSelected"
        @validate="onValidate"
      />

      <!-- 面包屑栏：主图 > 子图层级导航 + 当前层级节点数 -->
      <ContainerEditorBreadcrumb
        :root-label="draft?.name"
        :editor-path="editorStore.editorPath"
        :sg-label-fn="sgLabel"
        :active-node-count="activeGraph?.nodes?.length ?? null"
        @pop="editorStore.popPath()"
        @goto="editorStore.gotoPathIndex($event)"
      />

      <div class="flex flex-1 min-h-0">
        <!-- Left palette -->
        <aside
          v-show="!paletteCollapsed"
          class="w-44 shrink-0 border-r border-default overflow-y-auto p-3"
        >
          <NodePalette @add="onAddNode" />
        </aside>

        <!-- Canvas -->
        <div
          class="flex-1 min-w-0 relative"
          @dragover.prevent="onCanvasDragOver"
          @drop.prevent="onCanvasDrop"
        >
          <!-- 操作提示 -->
          <div
            class="absolute bottom-2 left-1/2 -translate-x-1/2 z-20 text-[10px] text-dimmed pointer-events-none bg-default/70 px-2 py-1 rounded"
          >
            左键拖空白框选 · 中键拖拽视图 · Ctrl+C/V 复制粘贴 · Delete 删除
          </div>
          <VueFlow
            v-model:nodes="flowNodes"
            v-model:edges="flowEdges"
            :node-types="nodeTypes as any"
            :default-edge-options="{ type: 'smoothstep' }"
            :delete-key-code="['Delete', 'Backspace']"
            :multi-selection-key-code="['Shift', 'Control', 'Meta']"
            :selection-key-code="true"
            :pan-on-drag="[1, 2]"
            :selection-mode="SelectionMode.Partial"
            :nodes-draggable="true"
            :elements-selectable="true"
            fit-view-on-init
            class="bg-elevated/20"
            @node-click="onNodeClick"
            @node-double-click="onNodeDoubleClick"
            @pane-click="selectedID = null"
            @edge-double-click="onEdgeDoubleClick"
            @nodes-change="onNodesChange"
            @edges-change="onEdgesChange"
            @connect="onConnect"
          >
            <Background pattern-color="#3f3f46" :gap="20" />
            <Controls position="bottom-left" />
            <MiniMap
              pannable
              zoomable
              position="bottom-right"
              mask-color="rgba(9, 9, 11, 0.6)"
              :node-color="miniNodeColor"
              node-stroke-color="#52525b"
              :node-border-radius="2"
            />
          </VueFlow>
        </div>

        <!-- Right panel：选中节点显示 Inspector，否则显示容器属性 -->
        <ContainerEditorInspector
          v-show="!inspectorCollapsed"
          :selected-node="selectedNode"
          :in-subgraph="editorStore.editorPath.length > 0"
          :current-subgraph="currentSubgraph"
          :container="draft"
          :active-graph="activeGraph"
          :var-names="varNames"
          :all-subgraph-tags="allSubgraphTags"
          @config-update="onConfigUpdate"
          @delete-selected="onDeleteSelected"
          @subgraph-update="onSubgraphPropsUpdate"
          @container-update="onContainerPatch"
          @request-record="(e) => startRecording(e.mode, { replaceNodeID: e.replaceNodeID })"
        />
      </div>

      <!-- 底部日志面板 (VSCode 风格): 订阅 container:log / container:node-enter, 默认展开 -->
      <ContainerLogPanel />
    </div>

    <ValidationErrorPanel
      :open="validationPanelOpen"
      :errors="validationErrors"
      @close="validationPanelOpen = false"
      @run="onValidationPanelRun"
      @fix-missing-window-target="onFixMissingWindowTarget"
    />

  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useWindowControls } from '@/composables/useWindowControls'
import { useRoute } from 'vue-router'
import { useToast } from '@nuxt/ui/composables'
import { VueFlow, useVueFlow, SelectionMode } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/controls/dist/style.css'
import '@vue-flow/minimap/dist/style.css'

import { backend, type Container, type GraphNode, type GraphEdge, type ValidationError } from '@/lib/backend'
import { useRecordingStore } from '@/stores/recording'
import { useExecutionStore } from '@/stores/execution'
import { useContainersStore } from '@/stores/containers'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { useContainerDraft } from '@/composables/containerEditor/useContainerDraft'
import { useEditorPath } from '@/composables/containerEditor/useEditorPath'
import { useSubgraphLifecycle } from '@/composables/containerEditor/useSubgraphLifecycle'
import { useFlowInteraction } from '@/composables/containerEditor/useFlowInteraction'
import { useFolding } from '@/composables/containerEditor/useFolding'
import { useRecording } from '@/composables/containerEditor/useRecording'
import { useEditorSave } from '@/composables/containerEditor/useEditorSave'
import { useNodeClipboard } from '@/composables/containerEditor/useNodeClipboard'
import { useGraphLayout, type AlignMode } from '@/composables/containerEditor/useGraphLayout'
import { useGraphMutations } from '@/composables/containerEditor/useGraphMutations'
import NodePalette from '@/components/containers/NodePalette.vue'
import ContainerFlowNode from '@/components/containers/ContainerFlowNode.vue'
import CommentBoxNode from '@/components/containers/CommentBoxNode.vue'
import ContainerEditorToolbar from '@/components/containers/ContainerEditorToolbar.vue'
import ContainerEditorBreadcrumb from '@/components/containers/ContainerEditorBreadcrumb.vue'
import ContainerEditorInspector from '@/components/containers/ContainerEditorInspector.vue'
import ValidationErrorPanel from '@/components/containers/ValidationErrorPanel.vue'
import ContainerLogPanel from '@/components/containers/ContainerLogPanel.vue'
import { KIND_DEFAULTS, KIND_LABEL_ZH, PIN_SPECS } from '@/components/containers/pinSpec'
import { markRaw } from 'vue'

const route = useRoute()
const toast = useToast()
const recordStore = useRecordingStore()
const execStore = useExecutionStore()
const containersStore = useContainersStore()

const editorStore = useContainerEditorStore()

const runningNodeLabel = computed(
  () => KIND_LABEL_ZH[execStore.currentNodeKind] ?? execStore.currentNodeKind ?? '',
)

async function onStopRun() {
  await containersStore.stopAll()
}

const containerID = String(route.query.id ?? '')

const {
  draft,
  dirty,
  activeGraph,
  flowNodes,
  flowEdges,
  syncFlowFromDraft,
  refreshSubgraphStore,
} = useContainerDraft(containerID)

// 编辑路径 + 当前子图（useEditorPath，转发 editorStore）
const { sgLabel, currentSubgraph } = useEditorPath()

// 子图 metadata 外部编辑 (NodeInspector / SubgraphPropsPanel) 改的是 store 里 sg 对象,
// useContainerDraft 的 deep watch 自动标 dirty — 之前的 window 总线桥接已删除.

const {
  autoCreateSubgraphForNewNode,
  countSubgraphReferencesIncludeMain,
  findNodeAcrossGraphs,
  deleteSubgraphCascade,
  deepCloneSubgraphForCopy,
  gcOrphanSubgraphs,
} = useSubgraphLifecycle({ draft, activeGraph, syncFlowFromDraft, refreshSubgraphStore })

const selectedID = ref<string | null>(null)

// 折叠侧栏：放 toolbar 上的两个按钮控制，腾画布空间
const paletteCollapsed = ref(false)
const inspectorCollapsed = ref(false)

// FlowNode / FlowEdge 类型从 useContainerDraft export (公共声明), view 不再局部重复定义.

// 注册自定义节点组件：从 PIN_SPECS keys 自动派生，无需手维护。
// 加新 kind 只需在 pinSpec.ts 里加一条 PIN_SPECS 即可——nodeTypes / NodePalette / FlowNode 自动响应，避免漏注册。
// v4 §9.1: CommentBox uses its own visual-only Vue component (no handles).
// All other kinds share ContainerFlowNode.
const nodeTypes = markRaw({
  ...Object.fromEntries(
    Object.keys(PIN_SPECS)
      .filter((k) => k !== 'CommentBox')
      .map((k) => [k, ContainerFlowNode]),
  ),
  CommentBox: CommentBoxNode,
})

const selectedNode = computed<GraphNode | null>(() => {
  if (!selectedID.value) return null
  const g = activeGraph.value
  if (!g) return null
  return (g.nodes as GraphNode[]).find((n) => n.id === selectedID.value) ?? null
})

const varNames = computed<string[]>(() => (draft.value?.vars ?? []).map((v) => v.name))

// recording 状态 — 三态在 toolbar 内部判断 (isRecording / countdownSec / idle); 这里只暴露
// recordStore.isRecording 给 RecordingOverlay (countdownSec 通过 useRecording 返回).

// Minimap 节点颜色 — 按 kind 取调色板里的 border 色（深色基调）
import { KIND_VISUAL } from '@/components/containers/pinSpec'
function miniNodeColor(node: any): string {
  const k = node?.data?.kind ?? ''
  const v = KIND_VISUAL[k]
  // 取 border-class 转 hex（粗略映射）
  const map: Record<string, string> = {
    'border-emerald-500/40': '#10b981',
    'border-zinc-500/40': '#71717a',
    'border-blue-500/40': '#3b82f6',
    'border-rose-500/40': '#f43f5e',
    'border-amber-500/40': '#f59e0b',
    'border-violet-500/40': '#8b5cf6',
    'border-fuchsia-500/40': '#d946ef',
    'border-orange-500/40': '#f97316',
    'border-pink-500/40': '#ec4899',
    'border-slate-500/40': '#64748b',
    'border-sky-500/40': '#0ea5e9',
    'border-yellow-500/40': '#eab308',
    'border-cyan-500/40': '#06b6d4',
  }
  return map[v?.border ?? ''] ?? '#52525b'
}

function genID(): string {
  return 'n_' + Math.random().toString(36).slice(2, 10)
}

async function onAddNode(
  kind: string,
  atX?: number,
  atY?: number,
): Promise<string | null> {
  if (!draft.value) return null
  // v2 Plan B：把节点 push 到 activeGraph（主图 / 当前子图层级），而不是总往主图塞
  const targetGraph = activeGraph.value
  if (!targetGraph) {
    toast.add({ title: '当前层级 graph 不可用', color: 'error' })
    return null
  }
  const id = kind === 'Start' ? 'start' : genID()
  if (kind === 'Start' && targetGraph.nodes.some((n) => n.kind === 'Start')) {
    toast.add({ title: '只能有一个 Start 节点', color: 'warning' })
    return null
  }
  const x = atX ?? 200 + Math.random() * 200
  const y = atY ?? 100 + Math.random() * 200
  const defaults = KIND_DEFAULTS[kind] ?? {}
  const n: GraphNode = { id, kind, x, y, config: { ...defaults } }

  if (kind === 'Subgraph') {
    const ok = await autoCreateSubgraphForNewNode(n)
    if (!ok) {
      toast.add({ title: '建子图失败，请重试', color: 'error' })
      return null
    }
  }

  targetGraph.nodes.push(n)
  syncFlowFromDraft()
  selectedID.value = id
  return id
}

// Vue Flow viewport API：屏幕坐标 → canvas 坐标（考虑 zoom/pan）。
const { project, getSelectedNodes, removeNodes } = useVueFlow()

// 画布 drag/drop 交互（NodePalette → Canvas + LibraryView 卡片 → Canvas copy-on-use）
const { onCanvasDragOver, onCanvasDrop } = useFlowInteraction({
  project, onAddNode,
  draft, activeGraph, syncFlowFromDraft, refreshSubgraphStore, toast,
})

// 折叠选中节点为新子图
const { onFoldSelection } = useFolding({
  draft, activeGraph, refreshSubgraphStore, syncFlowFromDraft, getSelectedNodes, toast,
})

// 保存 + 孤儿 GC（onSaveAndClose 留在 view 因为依赖 view-local close 状态）
// 提前到 useRecording 之前: 录制完成自动 save 需要 onSave.
const { onSave } = useEditorSave({ draft, dirty, gcOrphanSubgraphs, toast })

// 录制流程 (v2): 拿 subgraphID → refreshSubgraphStore 让 editorStore 知道新子图 →
// activeGraph 加 Subgraph 引用节点 + autoConnect Start + 自动保存. 双击节点能进编辑.
const { startRecording, stopRecording, countdownSec } = useRecording({
  draft, activeGraph, syncFlowFromDraft, refreshSubgraphStore, saveDraft: onSave, toast,
})

// 节点剪贴板 (Ctrl+C/V) + Subgraph 1:1 复制独立子图副本
useNodeClipboard({
  draft, activeGraph, flowNodes,
  syncFlowFromDraft, refreshSubgraphStore,
  deepCloneSubgraphForCopy, getSelectedNodes,
  genID, toast,
})

// 自动布局 (dagre) + 对齐
const { autoLayout, alignSelected } = useGraphLayout({
  activeGraph, getSelectedNodes, syncFlowFromDraft, dirty, toast,
})
function onAutoLayout(direction: 'LR' | 'TB') {
  autoLayout(direction)
}
function onAlignSelected(mode: AlignMode) {
  alignSelected(mode)
}

// 本地剪贴板：含节点 + edges + 被复制 Subgraph 节点绑定的子图 deep copy（1:1 联动用）
// v2：clipboard 在 activeGraph 层级生效（主图 / 子图层级都能 copy/paste）
// clipboard / onCopySelection / onPasteSelection / Ctrl+C/V 监听 由 useNodeClipboard composable 提供（见 setup 顶部）

// onCanvasDragOver / onCanvasDrop 由 useFlowInteraction 提供（见 setup 顶部）

function onNodeClick(evt: any) {
  selectedID.value = evt.node?.id ?? null
}

function onNodeDoubleClick(evt: any) {
  const n = evt.node
  // v4 §9.2: CollapsedNode 跟 Subgraph 共享 navigation 语义 (both wrap a subgraph by ID).
  if (n?.data?.kind === 'Subgraph' || n?.data?.kind === 'CollapsedNode') {
    const sgID = n.data.config?.subgraphId
    if (!sgID) {
      toast.add({ title: '该节点未指定子图', color: 'warning' })
      return
    }
    editorStore.pushPath(sgID)
    selectedID.value = null
  }
}


// 所有 graph mutation 走 useGraphMutations 唯一写入点 (内部 activeGraph)
// 避免 6 个 handler 各自写错 graph 引用的整类 bug
const { onNodesChange, onEdgeDoubleClick, onEdgesChange, onConnect } = useGraphMutations({
  activeGraph,
  flowEdges,
  syncFlowFromDraft,
  findNodeAcrossGraphs,
  deleteSubgraphCascade,
})

function onConfigUpdate(cfg: Record<string, any>) {
  if (!draft.value || !selectedNode.value) return
  selectedNode.value.config = cfg
  // 配置变更可能改变 exec out pin 集 (Switch.cases / Parallel.n / Race.n) —
  // flowNodes 持有旧 config 引用快照, computed pins 不会重算 → handle 不刷新.
  // 这里重建 flow 让 ContainerFlowNode 拿到新 config 引用.
  syncFlowFromDraft()
}

// v4 D8: Expr fusion listener (Inspector 触发 'expr-fuse' CustomEvent).
import { useExprFusion } from '@/composables/containerEditor/useExprFusion'
const { fuse: fuseExpr } = useExprFusion({ activeGraph, syncFlowFromDraft })
function onExprFuseEvent(ev: Event) {
  const detail = (ev as CustomEvent).detail as { sourceID: string; targetID: string; targetPin: string }
  if (!detail) return
  const ok = fuseExpr(detail.sourceID, detail.targetID, detail.targetPin)
  if (ok) {
    selectedID.value = null
    toast.add({ title: 'Expr 合并完成', color: 'success' })
  } else {
    toast.add({ title: 'Expr 合并失败 (前置条件不满足)', color: 'warning' })
  }
}
onMounted(() => window.addEventListener('expr-fuse', onExprFuseEvent))
onUnmounted(() => window.removeEventListener('expr-fuse', onExprFuseEvent))

function onDeleteSelected() {
  if (!draft.value || !selectedID.value) return
  // 走 vue-flow removeNodes → 触发 onNodesChange(remove) → 统一处理 Subgraph cascade
  removeNodes([selectedID.value])
  selectedID.value = null
}

// 录制流程 (Phase 4): start → 后端落盘 InputClip → 主图加 PlayClip 节点 (config.clipID).
// startRecording / stopRecording / countdownSec 由 useRecording composable 提供.
// sgLabel / currentSubgraph 由 useEditorPath 提供.

const selectedCount = computed(() => getSelectedNodes.value.length)

// onFoldSelection 由 useFolding composable 提供（见 setup 顶部）

// onSave 由 useEditorSave composable 提供（见 setup 顶部）

async function onTryRun() {
  if (!draft.value || dirty.value) return
  // 试运行前先 validate, 有错弹 panel; 无错才真的 run (避免 backend run 抛 RuntimeError 单行 message)
  try {
    const errs = (await backend.containers.validate(draft.value.id)) as ValidationError[]
    const realErrs = (errs ?? []).filter((e) => e.severity === 'error')
    if (realErrs.length > 0) {
      validationErrors.value = errs
      validationPanelOpen.value = true
      return
    }
  } catch (e) {
    toast.add({ title: '校验失败', description: String(e), color: 'error' })
    return
  }
  await backend.containers.run(draft.value.id)
  toast.add({
    title: '已加入运行队列',
    color: 'primary',
    icon: 'i-tabler-player-play',
  })
}

// "检查" 按钮: 主动跑 validate, 始终弹 panel (即使全通过也告知用户)
const validationPanelOpen = ref(false)
const validationErrors = ref<ValidationError[]>([])
async function onValidate() {
  if (!draft.value || dirty.value) return
  try {
    const errs = (await backend.containers.validate(draft.value.id)) as ValidationError[]
    validationErrors.value = errs ?? []
    validationPanelOpen.value = true
  } catch (e) {
    toast.add({ title: '校验调用失败', description: String(e), color: 'error' })
  }
}

async function onValidationPanelRun() {
  validationPanelOpen.value = false
  if (!draft.value) return
  await backend.containers.run(draft.value.id)
  toast.add({
    title: '已加入运行队列',
    color: 'primary',
    icon: 'i-tabler-player-play',
  })
}

// 一键修复 MISSING_WINDOW_TARGET: 往主图 push 一个空 WindowTarget 节点 (用户后续在
// NodeInspector 里填 match/runtime 或点 "捕获前台窗口"). 必须落主图, 不是 activeGraph.
function onFixMissingWindowTarget() {
  if (!draft.value) return
  const mainGraph = draft.value.graph
  // 已存在则不重复加
  if ((mainGraph.nodes as GraphNode[]).some((n) => n.kind === 'WindowTarget')) {
    toast.add({ title: '主图已经有 WindowTarget 节点了', color: 'warning' })
    return
  }
  const defaults = KIND_DEFAULTS.WindowTarget ?? {}
  const newNode: GraphNode = {
    id: 'wt_' + Math.random().toString(36).slice(2, 8),
    kind: 'WindowTarget',
    x: 40,
    y: 40,
    config: JSON.parse(JSON.stringify(defaults)),
    createdAt: new Date().toISOString(),
  }
  ;(mainGraph.nodes as GraphNode[]).push(newNode)
  syncFlowFromDraft()
  validationPanelOpen.value = false
  toast.add({
    title: '已添加 WindowTarget 节点',
    description: '请打开节点 Inspector 配置目标窗口 (或点"捕获前台窗口")',
    color: 'success',
    icon: 'i-tabler-check',
  })
}

// 容器属性 patch（来自 ContainerPropsPanel）
function onContainerPatch(patch: Partial<Container>) {
  if (!draft.value) return
  Object.assign(draft.value, patch)
}

// currentSubgraph 由 useEditorPath 提供

const allSubgraphTags = computed(() => {
  const set = new Set<string>()
  for (const sg of editorStore.subgraphsForCurrentContainer) {
    for (const t of sg.tags ?? []) set.add(t)
  }
  return [...set]
})

function onSubgraphPropsUpdate(patch: Record<string, any>) {
  if (!currentSubgraph.value) return
  if (patch.__resetRecording) {
    toast.add({
      title: '重置录制元数据需要重新录制此子图（v1 仅提示）',
      color: 'warning',
    })
    return
  }
  Object.assign(currentSubgraph.value, patch)
  dirty.value = true
}

function goBack() {
  if (dirty.value) {
    pendingNav.value = 'back'
    confirmCloseOpen.value = true
    return
  }
  doBack()
}

function doBack() {
  // 独立 Frameless 窗口：返回 = 关窗（dashboard 在另一个窗口里继续显示）
  closeImmediate()
}

// 窗口控件 (isMaximised + min/max 全 useWindowControls 提供; close 在下方包一层 dirty 拦截)
const { isMaximised, onMinimise, onToggleMaximise, closeImmediate } = useWindowControls()
function onClose() {
  if (dirty.value) {
    pendingNav.value = 'close'
    confirmCloseOpen.value = true
    return
  }
  closeImmediate()
}

// Dirty 关闭确认
const confirmCloseOpen = ref(false)
const pendingNav = ref<'back' | 'close' | null>(null)

function onConfirmDiscardAndClose() {
  confirmCloseOpen.value = false
  const nav = pendingNav.value
  pendingNav.value = null
  dirty.value = false
  if (nav === 'close') closeImmediate()
  else doBack()
}

async function onSaveAndClose() {
  await onSave()
  if (dirty.value) {
    // 保存失败 → 不关闭
    confirmCloseOpen.value = false
    return
  }
  confirmCloseOpen.value = false
  const nav = pendingNav.value
  pendingNav.value = null
  if (nav === 'close') closeImmediate()
  else doBack()
}
</script>

<style scoped>
/* 自定义节点容器（ContainerFlowNode 自己有 bg/border，这里不再覆盖） */

/* ---- Controls (左下角缩放/fit) 深色 ---- */
:deep(.vue-flow__controls) {
  display: flex;
  flex-direction: column;
  background: rgba(24, 24, 27, 0.85); /* zinc-900/85 */
  border: 1px solid rgba(63, 63, 70, 0.8); /* zinc-700 */
  border-radius: 6px;
  overflow: hidden;
  box-shadow: 0 4px 10px rgba(0, 0, 0, 0.4);
}
:deep(.vue-flow__controls-button) {
  background: transparent;
  border: none;
  border-bottom: 1px solid rgba(63, 63, 70, 0.6);
  color: #d4d4d8; /* zinc-300 */
  width: 26px;
  height: 26px;
  fill: currentColor;
}
:deep(.vue-flow__controls-button:last-child) {
  border-bottom: none;
}
:deep(.vue-flow__controls-button:hover) {
  background: rgba(63, 63, 70, 0.6);
  color: #f4f4f5;
}
:deep(.vue-flow__controls-button svg) {
  fill: currentColor;
  max-width: 14px;
  max-height: 14px;
}

/* ---- MiniMap (右下角) 深色 ---- */
:deep(.vue-flow__minimap) {
  background: rgba(24, 24, 27, 0.85);
  border: 1px solid rgba(63, 63, 70, 0.8);
  border-radius: 6px;
  overflow: hidden;
}
:deep(.vue-flow__minimap-mask) {
  fill: rgba(9, 9, 11, 0.55);
}

/* ---- Attribution 左下角 "Vue Flow" 水印淡化 ---- */
:deep(.vue-flow__attribution) {
  background: transparent;
  color: rgba(161, 161, 170, 0.5);
  font-size: 9px;
}
</style>
