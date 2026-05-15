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
      <div class="shrink-0 h-11 px-3 border-b border-default flex items-center gap-2 bg-default/60">
        <UButton
          size="xs"
          variant="ghost"
          color="neutral"
          :icon="
            paletteCollapsed
              ? 'i-tabler-layout-sidebar-left-expand'
              : 'i-tabler-layout-sidebar-left-collapse'
          "
          :title="paletteCollapsed ? '展开节点面板' : '折叠节点面板'"
          @click="paletteCollapsed = !paletteCollapsed"
        />
        <UButton
          size="sm"
          :color="recordingOrCounting ? 'error' : 'primary'"
          :variant="recordingOrCounting ? 'solid' : 'soft'"
          :icon="recordingOrCounting ? 'i-tabler-square' : 'i-tabler-circle-dot'"
          @click="onRecordAction"
        >
          {{ recordingOrCounting ? '停止录制' : '录制新动作' }}
        </UButton>

        <div class="flex-1" />

        <div
          v-if="execStore.running"
          class="inline-flex items-center gap-2 rounded-md bg-emerald-500/15 border border-emerald-500/40 px-2 py-0.5 text-[11px] text-emerald-300"
        >
          <span class="size-1.5 rounded-full bg-emerald-400 animate-pulse" />
          <span>跑中</span>
          <span v-if="execStore.currentNodeKind" class="text-emerald-200/80">
            · {{ runningNodeLabel }}
          </span>
        </div>
        <UButton
          v-if="execStore.running"
          size="sm"
          color="error"
          variant="solid"
          icon="i-tabler-square"
          title="停止当前运行 + 清队列 (同 Ctrl+Shift+F9)"
          @click="onStopRun"
          >停止</UButton
        >
        <UButton
          size="sm"
          variant="soft"
          color="primary"
          icon="i-tabler-player-play"
          :disabled="dirty || execStore.running"
          :title="
            dirty ? '请先保存再试运行' : execStore.running ? '已有任务在跑，先停' : '入队运行一次'
          "
          @click="onTryRun"
          >试运行</UButton
        >
        <UButton size="sm" color="primary" icon="i-tabler-check" :disabled="!dirty" @click="onSave"
          >保存</UButton
        >
        <UButton
          size="xs"
          variant="ghost"
          color="neutral"
          :icon="
            inspectorCollapsed
              ? 'i-tabler-layout-sidebar-right-expand'
              : 'i-tabler-layout-sidebar-right-collapse'
          "
          :title="inspectorCollapsed ? '展开属性面板' : '折叠属性面板'"
          @click="inspectorCollapsed = !inspectorCollapsed"
        />
      </div>

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
            :selection-key-code="null"
            :pan-on-drag="[1, 2]"
            :selection-on-drag="true"
            select-mode="partial"
            fit-view-on-init
            class="bg-elevated/20"
            @node-click="onNodeClick"
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
        <aside
          v-show="!inspectorCollapsed"
          class="w-96 shrink-0 border-l border-default overflow-y-auto p-4"
        >
          <NodeInspector
            v-if="selectedNode"
            :node="selectedNode"
            :var-names="varNames"
            :nodes="draft?.graph.nodes ?? []"
            :edges="draft?.graph.edges ?? []"
            @update="onConfigUpdate"
            @delete="onDeleteSelected"
          />
          <ContainerPropsPanel v-else :container="draft" @update="onContainerPatch" />
        </aside>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { Window } from '@wailsio/runtime'
import { useRoute } from 'vue-router'
import { useToast } from '@nuxt/ui/composables'
import { VueFlow, MarkerType, useVueFlow } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/controls/dist/style.css'
import '@vue-flow/minimap/dist/style.css'

import { backend, type Container, type GraphNode, type GraphEdge } from '@/lib/backend'
import { useActionsStore } from '@/stores/actions'
import { useExecutionStore } from '@/stores/execution'
import { useContainersStore } from '@/stores/containers'
import NodePalette from '@/components/containers/NodePalette.vue'
import NodeInspector from '@/components/containers/NodeInspector.vue'
import ContainerPropsPanel from '@/components/containers/ContainerPropsPanel.vue'
import ContainerFlowNode from '@/components/containers/ContainerFlowNode.vue'
import { edgeKind, KIND_DEFAULTS, KIND_LABEL_ZH } from '@/components/containers/pinSpec'
import { markRaw } from 'vue'

const route = useRoute()
const toast = useToast()
const actionsStore = useActionsStore()
const execStore = useExecutionStore()
const containersStore = useContainersStore()

const runningNodeLabel = computed(
  () => KIND_LABEL_ZH[execStore.currentNodeKind] ?? execStore.currentNodeKind ?? '',
)

async function onStopRun() {
  await containersStore.stopAll()
}

const containerID = String(route.query.id ?? '')

const draft = ref<Container | null>(null)
const selectedID = ref<string | null>(null)

// 折叠侧栏：放 toolbar 上的两个按钮控制，腾画布空间
const paletteCollapsed = ref(false)
const inspectorCollapsed = ref(false)

// Vue Flow 自身的 nodes/edges 格式：{ id, position: {x,y}, data, type } / { source, target }
interface FlowNode {
  id: string
  position: { x: number; y: number }
  data: { kind: string; config?: Record<string, any> }
  label?: string
  type?: string
}
interface FlowEdge {
  id: string
  source: string
  target: string
  sourceHandle?: string
  targetHandle?: string
  type?: string
  animated?: boolean
  style?: Record<string, any>
  markerEnd?: any
}

const flowNodes = ref<FlowNode[]>([])
const flowEdges = ref<FlowEdge[]>([])

// 注册自定义节点组件：22 个 kind 都用同一个 ContainerFlowNode，按 data.kind 渲染。
const nodeTypes = markRaw(
  Object.fromEntries(
    [
      'Start',
      'Sleep',
      'Loop',
      'If',
      'Parallel',
      'Race',
      'Stop',
      'Break',
      'Continue',
      'SetVar',
      'IncVar',
      'WaitTemplate',
      'CheckTemplate',
      'ClickTemplate',
      'DetectColor',
      'InvokeAction',
      'ClickAt',
      'KeyPress',
      'MouseMoveRel',
      'Scroll',
      'OnEvent',
      'Log',
      'Toast',
    ].map((k) => [k, ContainerFlowNode]),
  ),
)

const selectedNode = computed<GraphNode | null>(() => {
  if (!draft.value || !selectedID.value) return null
  return draft.value.graph.nodes.find((n) => n.id === selectedID.value) ?? null
})

const varNames = computed<string[]>(() => (draft.value?.vars ?? []).map((v) => v.name))

const recordingOrCounting = computed(
  () => actionsStore.recording || actionsStore.countdown !== null,
)

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
  }
  return map[v?.border ?? ''] ?? '#52525b'
}

const dirty = ref(false)
// 任何 draft 变动 → 标 dirty
watch(
  draft,
  () => {
    dirty.value = true
  },
  { deep: true },
)

onMounted(async () => {
  if (!containerID) return
  const r = await backend.containers.get(containerID)
  if (r === undefined) {
    toast.add({ title: '容器不存在', color: 'error' })
    return
  }
  const c = r as unknown as Container
  draft.value = JSON.parse(JSON.stringify(c))
  syncFlowFromDraft()
  // 初次同步不算 dirty
  setTimeout(() => {
    dirty.value = false
  }, 0)
})

function syncFlowFromDraft() {
  if (!draft.value) return
  flowNodes.value = draft.value.graph.nodes.map((n) => ({
    id: n.id,
    type: n.kind, // 用自定义组件
    position: { x: n.x, y: n.y },
    data: { kind: n.kind, config: n.config },
  }))
  flowEdges.value = draft.value.graph.edges.map((e, i) => {
    const dot = e.from.indexOf('.')
    const src = e.from.slice(0, dot)
    const srcPin = e.from.slice(dot + 1)
    const dot2 = e.to.indexOf('.')
    const tgt = e.to.slice(0, dot2)
    const tgtPin = e.to.slice(dot2 + 1)
    const fromKind = draft.value!.graph.nodes.find((n) => n.id === src)?.kind ?? ''
    const isData = edgeKind(fromKind, srcPin) === 'data'
    return {
      id: 'e' + i,
      source: src,
      target: tgt,
      sourceHandle: srcPin,
      targetHandle: tgtPin,
      type: 'smoothstep',
      animated: isData,
      style: isData
        ? { stroke: '#60a5fa', strokeWidth: 1.5, strokeDasharray: '4 4' } // data edge: dashed blue
        : { stroke: '#a1a1aa', strokeWidth: 1.5 }, // exec edge: solid zinc
      markerEnd: {
        type: MarkerType.ArrowClosed,
        color: isData ? '#60a5fa' : '#a1a1aa',
      },
    }
  })
}

function genID(): string {
  return 'n_' + Math.random().toString(36).slice(2, 10)
}

function onAddNode(kind: string, atX?: number, atY?: number) {
  if (!draft.value) return
  const id = kind === 'Start' ? 'start' : genID()
  if (kind === 'Start' && draft.value.graph.nodes.some((n) => n.kind === 'Start')) {
    toast.add({ title: '只能有一个 Start 节点', color: 'warning' })
    return
  }
  const x = atX ?? 200 + Math.random() * 200
  const y = atY ?? 100 + Math.random() * 200
  // 注入合理 default config（用户拿到的不是空 form，是合理起点）
  const defaults = KIND_DEFAULTS[kind] ?? {}
  const n: GraphNode = { id, kind, x, y, config: { ...defaults } }
  draft.value.graph.nodes.push(n)
  syncFlowFromDraft()
  selectedID.value = id // 自动选中新加的节点，方便接着配
}

// Vue Flow viewport API：屏幕坐标 → canvas 坐标（考虑 zoom/pan）。
const { project, getSelectedNodes } = useVueFlow()

// 本地剪贴板（仅本次会话有效，跨容器/跨标签不共享）
const clipboard = ref<{ nodes: GraphNode[]; edges: GraphEdge[] } | null>(null)

function onCopySelection() {
  if (!draft.value) return
  const sel = getSelectedNodes.value ?? []
  if (sel.length === 0) return
  const ids = new Set(sel.map((n) => n.id))
  // 排除 Start——只能 1 个
  const nodes = draft.value.graph.nodes
    .filter((n) => ids.has(n.id) && n.kind !== 'Start')
    .map((n) => JSON.parse(JSON.stringify(n)) as GraphNode)
  if (nodes.length === 0) return
  const copiedIDs = new Set(nodes.map((n) => n.id))
  // 只复制两端都在选中里的边（保留拓扑）
  const edges = draft.value.graph.edges
    .filter((e) => {
      const fromID = e.from.split('.')[0]
      const toID = e.to.split('.')[0]
      return copiedIDs.has(fromID) && copiedIDs.has(toID)
    })
    .map((e) => ({ ...e }))
  clipboard.value = { nodes, edges }
  toast.add({ title: `已复制 ${nodes.length} 个节点`, color: 'success', duration: 1500 })
}

function onPasteSelection() {
  if (!draft.value || !clipboard.value) return
  const idMap: Record<string, string> = {}
  const cloned = clipboard.value.nodes.map((n) => {
    const newID = genID()
    idMap[n.id] = newID
    return { ...JSON.parse(JSON.stringify(n)), id: newID, x: n.x + 40, y: n.y + 40 } as GraphNode
  })
  const clonedEdges = clipboard.value.edges.map((e) => {
    const [fromID, fromPin] = e.from.split('.')
    const [toID, toPin] = e.to.split('.')
    return {
      from: `${idMap[fromID] ?? fromID}.${fromPin}`,
      to: `${idMap[toID] ?? toID}.${toPin}`,
    } as GraphEdge
  })
  draft.value.graph.nodes.push(...cloned)
  draft.value.graph.edges.push(...clonedEdges)
  syncFlowFromDraft()
  // 选中粘贴出来的新节点（用 v-model 的 selected 字段最稳）
  setTimeout(() => {
    flowNodes.value = flowNodes.value.map((n) =>
      cloned.some((c) => c.id === n.id) ? { ...n, selected: true } : { ...n, selected: false },
    )
  }, 0)
}

function onEditorKeydown(e: KeyboardEvent) {
  // 输入控件里输入时不抢
  const t = e.target as HTMLElement | null
  if (t && /^(INPUT|TEXTAREA|SELECT)$/.test(t.tagName)) return
  if (t?.isContentEditable) return
  if ((e.ctrlKey || e.metaKey) && !e.shiftKey && !e.altKey) {
    if (e.key === 'c' || e.key === 'C') {
      onCopySelection()
      e.preventDefault()
    } else if (e.key === 'v' || e.key === 'V') {
      onPasteSelection()
      e.preventDefault()
    }
  }
}

function onCanvasDragOver(e: DragEvent) {
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'copy'
}

function onCanvasDrop(e: DragEvent) {
  const kind = e.dataTransfer?.getData('application/x-yhbox-node')
  if (!kind) return
  // 拿 canvas 容器相对坐标 → 转 Vue Flow 内部坐标
  const target = e.currentTarget as HTMLElement
  const rect = target.getBoundingClientRect()
  const px = e.clientX - rect.left
  const py = e.clientY - rect.top
  const pos = project({ x: px, y: py })
  onAddNode(kind, pos.x, pos.y)
}

function onNodeClick(evt: any) {
  selectedID.value = evt.node?.id ?? null
}

function onNodesChange(changes: any[]) {
  if (!draft.value) return
  for (const ch of changes) {
    if (ch.type === 'position' && ch.position) {
      const node = draft.value.graph.nodes.find((n) => n.id === ch.id)
      if (node) {
        node.x = ch.position.x
        node.y = ch.position.y
      }
    }
    if (ch.type === 'remove') {
      draft.value.graph.nodes = draft.value.graph.nodes.filter((n) => n.id !== ch.id)
      draft.value.graph.edges = draft.value.graph.edges.filter(
        (e) => !e.from.startsWith(ch.id + '.') && !e.to.startsWith(ch.id + '.'),
      )
    }
  }
}

function onEdgeDoubleClick(evt: any) {
  if (!draft.value || !evt?.edge?.id) return
  const idx = flowEdges.value.findIndex((e) => e.id === evt.edge.id)
  if (idx < 0) return
  draft.value.graph.edges.splice(idx, 1)
  syncFlowFromDraft()
}

function onEdgesChange(changes: any[]) {
  if (!draft.value) return
  for (const ch of changes) {
    if (ch.type === 'remove') {
      // ch.id 是 vue-flow 内部 id（"e0" 等）；映射回去删 draft edge
      const idx = flowEdges.value.findIndex((e) => e.id === ch.id)
      if (idx >= 0) draft.value.graph.edges.splice(idx, 1)
    }
  }
}

function onConnect(c: any) {
  if (!draft.value) return
  const from = `${c.source}.${c.sourceHandle ?? 'out'}`
  const to = `${c.target}.${c.targetHandle ?? 'in'}`
  // 单 out 替换：from-pin 已有出边的话，先删旧的（exec 边 1:1 语义）。
  // 同时 to-pin 已有入边的话，也替换（exec-in 必须唯一，避免连完无效）。
  draft.value.graph.edges = draft.value.graph.edges.filter((e) => e.from !== from && e.to !== to)
  draft.value.graph.edges.push({ from, to })
  syncFlowFromDraft()
}

function onConfigUpdate(cfg: Record<string, any>) {
  if (!draft.value || !selectedNode.value) return
  selectedNode.value.config = cfg
}

function onDeleteSelected() {
  if (!draft.value || !selectedID.value) return
  draft.value.graph.nodes = draft.value.graph.nodes.filter((n) => n.id !== selectedID.value)
  draft.value.graph.edges = draft.value.graph.edges.filter(
    (e) => !e.from.startsWith(selectedID.value + '.') && !e.to.startsWith(selectedID.value + '.'),
  )
  selectedID.value = null
  syncFlowFromDraft()
}

// 录制新动作 → 先 auto-save 容器 → 倒计时 → 录制 → 自动放 InvokeAction 节点到画布
//
// 防丢失：用户在 dirty 状态点录制时，先存盘；否则录制完回来 draft 还在内存（路由不切走），
// 但万一用户在录制过程中关窗口，丢失也会少。
async function onRecordAction() {
  if (!draft.value) return
  if (actionsStore.recording || actionsStore.countdown !== null) return

  // 1) auto-save draft（dirty 才存）
  if (dirty.value) {
    const patch = JSON.parse(JSON.stringify(draft.value))
    const r = await backend.containers.update(draft.value.id, JSON.stringify(patch))
    if (r === undefined) {
      // 保存失败 toast 已经由 invoke 兜底；不进录制
      return
    }
    dirty.value = false
  }

  // 2) 启动录制流程（全局 RecorderDialog 会展示倒计时 + 进度）
  await actionsStore.reload()
  const before = new Set(actionsStore.list.map((a) => a.id))
  await actionsStore.toggleRecording()

  // 3) 等录制真正结束
  await new Promise<void>((resolve) => {
    const t = setInterval(() => {
      if (!actionsStore.recording && actionsStore.countdown === null) {
        clearInterval(t)
        resolve()
      }
    }, 200)
  })

  // 4) 录到新 action → 在画布上 drop 一个 InvokeAction
  await actionsStore.reload()
  const fresh = actionsStore.list.find((a) => !before.has(a.id))
  if (!fresh || !draft.value) return
  const node: GraphNode = {
    id: 'n_' + Math.random().toString(36).slice(2, 10),
    kind: 'InvokeAction',
    x: 300 + Math.random() * 150,
    y: 300 + Math.random() * 150,
    config: { actionId: fresh.id },
  }
  draft.value.graph.nodes.push(node)
  syncFlowFromDraft()
  toast.add({
    title: `已添加 InvokeAction: ${fresh.name}`,
    color: 'success',
    icon: 'i-tabler-check',
  })
}

async function onSave() {
  if (!draft.value) return
  const patch = JSON.parse(JSON.stringify(draft.value))
  const ok = await backend.containers.update(draft.value.id, JSON.stringify(patch))
  if (ok !== undefined) {
    toast.add({ title: '已保存', color: 'success', icon: 'i-tabler-check' })
    dirty.value = false
  }
}

async function onTryRun() {
  if (!draft.value || dirty.value) return
  await backend.containers.run(draft.value.id)
  toast.add({
    title: '已加入运行队列',
    color: 'primary',
    icon: 'i-tabler-player-play',
  })
}

// 容器属性 patch（来自 ContainerPropsPanel）
function onContainerPatch(patch: Partial<Container>) {
  if (!draft.value) return
  Object.assign(draft.value, patch)
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
  window.history.length > 1 ? window.history.back() : (window.location.hash = '#/tasks')
}

// 窗口控件
const isMaximised = ref(false)
let pollTimer: ReturnType<typeof setInterval> | null = null
async function pollMax() {
  try {
    isMaximised.value = await Window.IsMaximised()
  } catch {
    /* ignore */
  }
}
function onMinimise() {
  Window.Minimise()
}
function onToggleMaximise() {
  Window.ToggleMaximise()
  setTimeout(pollMax, 50)
}
function onClose() {
  if (dirty.value) {
    pendingNav.value = 'close'
    confirmCloseOpen.value = true
    return
  }
  Window.Close()
}

onMounted(() => {
  pollMax()
  pollTimer = setInterval(pollMax, 500)
  window.addEventListener('keydown', onEditorKeydown)
})
onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
  window.removeEventListener('keydown', onEditorKeydown)
})

// Dirty 关闭确认
const confirmCloseOpen = ref(false)
const pendingNav = ref<'back' | 'close' | null>(null)

function onConfirmDiscardAndClose() {
  confirmCloseOpen.value = false
  const nav = pendingNav.value
  pendingNav.value = null
  dirty.value = false
  if (nav === 'close') Window.Close()
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
  if (nav === 'close') Window.Close()
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
