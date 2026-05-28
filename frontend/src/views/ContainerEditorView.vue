<template>
  <div class="flex flex-col h-screen bg-default text-default">
    <!-- 子窗口形态自画 header (高度跟主壳 AppTitleBar 一致 h-14); 嵌入主壳时由 AppTitleBar 接管 -->
    <header
      v-if="isStandalone"
      class="h-14 shrink-0 flex items-center gap-2 border-b border-default pl-3 pr-0"
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
        v-model:palette-collapsed="sidebarPrefs.leftSidebarCollapsed"
        v-model:inspector-collapsed="sidebarPrefs.inspectorCollapsed"
        :is-standalone="isStandalone"
        :is-recording="recordStore.isRecording"
        :countdown-sec="countdownSec"
        :selected-count="selectedCount"
        :exec-store-running="execStore.running"
        :running-node-kind="execStore.currentNodeKind ?? undefined"
        :running-node-label="runningNodeLabel"
        :dirty="dirty"
        :can-undo="canUndo"
        :can-redo="canRedo"
        :snap-enabled="sidebarPrefs.snapEnabled"
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
        @open-node-explorer="onOpenNodeExplorer"
        @open-library-explorer="onOpenLibraryExplorer"
        @open-settings="settingsOpen = true"
        @undo="undo"
        @redo="redo"
        @toggle-snap="sidebarPrefs.snapEnabled = !sidebarPrefs.snapEnabled"
        @open-new-window="onOpenNewWindow"
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
        <!-- Left sidebar: 3 collapsible panels -->
        <aside
          v-show="!sidebarPrefs.leftSidebarCollapsed"
          :style="{ width: leftPane.width.value + 'px' }"
          class="shrink-0 border-r border-default overflow-y-auto flex flex-col"
        >
          <!-- Sidebar tabs (Palette | Snippets) — segmented toggle 顶部 -->
          <div class="flex border-b border-default bg-elevated/20">
            <button
              type="button"
              class="flex-1 flex items-center justify-center gap-1.5 py-2 text-[11px] font-medium transition-colors"
              :class="
                sidebarPrefs.leftSidebarTab === 'palette'
                  ? 'text-primary border-b-2 border-primary -mb-px'
                  : 'text-dimmed hover:text-default'
              "
              @click="sidebarPrefs.leftSidebarTab = 'palette'"
            >
              <UIcon name="i-tabler-box" class="size-3.5" />
              节点库
            </button>
            <button
              type="button"
              class="flex-1 flex items-center justify-center gap-1.5 py-2 text-[11px] font-medium transition-colors"
              :class="
                sidebarPrefs.leftSidebarTab === 'snippets'
                  ? 'text-primary border-b-2 border-primary -mb-px'
                  : 'text-dimmed hover:text-default'
              "
              @click="sidebarPrefs.leftSidebarTab = 'snippets'"
            >
              <UIcon name="i-tabler-bookmarks" class="size-3.5" />
              Snippets
            </button>
          </div>

          <SnippetsPanel
            v-if="sidebarPrefs.leftSidebarTab === 'snippets'"
            @apply="onApplySnippet"
            @edit="onEditSnippet"
          />

          <VarsPanel
            v-show="sidebarPrefs.leftSidebarTab === 'palette'"
            :vars="draft?.vars ?? []"
            :usage-count="totalVarUsageCount"
            v-model:expanded="sidebarPrefs.varsExpanded"
            @add-var="onAddVar"
            @rename-var="onRenameVar"
            @update-var-field="onUpdateVarField"
            @request-delete="onRequestDeleteVar"
            @reorder-vars="onReorderVars"
            @insert-incvar="onInsertIncVar"
          />
        </aside>
        <SplitHandle
          v-show="!sidebarPrefs.leftSidebarCollapsed"
          :model-value="leftPane.width.value"
          @update:model-value="leftPane.setWidth"
          :min="200"
          :max="480"
        />

        <!-- Canvas -->
        <div
          class="flex-1 min-w-0 relative"
          @dragover.prevent="onCanvasDragOver"
          @drop.prevent="onCanvasDrop"
          @contextmenu.capture="onCanvasContextMenuCapture"
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
            class="canvas-bg"
            @node-click="onNodeClick"
            @node-double-click="onNodeDoubleClick"
            @selection-change="onSelectionChange"
            @pane-click="selectedID = null"
            @pane-context-menu="onCanvasContextMenu"
            @node-context-menu="onNodeContextMenu"
            @selection-context-menu="onSelectionContextMenu"
            @pane-double-click="onPaneDoubleClick"
            :is-valid-connection="isValidVueFlowConnection"
            @connect-start="onVfConnectStart"
            @connect-end="onVfConnectEnd"
            @edge-double-click="onEdgeDoubleClick"
            @edge-context-menu="onEdgeContextMenu"
            @nodes-change="onNodesChange"
            @edges-change="onEdgesChange"
            @connect="onConnect"
            @node-drag="onSnapNodeDrag"
            @node-drag-stop="onSnapNodeDragStop"
          >
            <Background pattern-color="#3a3a4d" :gap="22" :size="1.2" />
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
            <!-- Drag-time alignment guides (PS smart-guides style) -->
            <SnapGuideOverlay :guides="snapGuides" />
          </VueFlow>
        </div>

        <SplitHandle
          v-show="!sidebarPrefs.inspectorCollapsed"
          reverse
          :model-value="rightPane.width.value"
          @update:model-value="rightPane.setWidth"
          :min="200"
          :max="480"
        />

        <!-- Right panel：选中节点显示 Inspector，否则显示引导空状态 -->
        <ContainerEditorInspector
          v-show="!sidebarPrefs.inspectorCollapsed"
          :style="{ width: rightPane.width.value + 'px' }"
          :selected-node="selectedNode"
          :in-subgraph="editorStore.editorPath.length > 0"
          :current-subgraph="currentSubgraph"
          :active-graph="activeGraph"
          :var-names="varNames"
          :all-subgraph-tags="allSubgraphTags"
          @config-update="onConfigUpdate"
          @label-update="onLabelUpdate"
          @delete-selected="onDeleteSelected"
          @subgraph-update="onSubgraphPropsUpdate"
          @request-record="(e) => startRecording(e.mode, { replaceNodeID: e.replaceNodeID })"
        />
      </div>

    </div>

    <ValidationErrorPanel
      :open="validationPanelOpen"
      :errors="validationErrors"
      @close="validationPanelOpen = false"
      @run="onValidationPanelRun"
      @fix-missing-window-target="onFixMissingWindowTarget"
    />

    <ContainerSettingsModal
      v-if="draft"
      v-model:open="settingsOpen"
      :initial="{
        name: draft.name,
        hotkey: draft.hotkey ?? '',
        description: draft.description ?? '',
        tags: draft.tags ?? [],
        runMode: draft.runMode || 'background',
      }"
      :all-tags="allSubgraphTags"
      @save="(form) => applyDraftMutation((d) => Object.assign(d, form))"
    />

    <DeleteVarConfirmModal
      v-if="deleteConfirm"
      :open="true"
      :var-name="deleteConfirm.name"
      :ref-i-ds="deleteConfirm.refIDs"
      @update:open="(v) => { if (!v) deleteConfirm = null }"
      @confirm="onDeleteConfirm"
    />

    <NodeExplorerModal
      v-model:open="nodeExplorerOpen"
      @pick-kind="onPickKind"
    />

    <LibraryExplorerModal
      v-model:open="libraryExplorerOpen"
      @pick-subgraph="onPickLibrarySubgraph"
    />

    <InlineContextMenu
      :open="inlineMenu.open"
      :position="inlineMenu.position"
      :pin-context="inlineMenu.pinContext"
      @update:open="(v) => { inlineMenu.open = v }"
      @pick="onInlineMenuPick"
    />

    <!-- 右键菜单 (节点 / 多选 / 边 / pin) -->
    <NodeContextMenu
      v-if="nodeMenu.node"
      :open="nodeMenu.open"
      :position="nodeMenu.position"
      :node="nodeMenu.node"
      @update:open="(v) => { nodeMenu.open = v }"
      @action="onNodeMenuAction"
    />
    <MultiNodeContextMenu
      :open="multiMenu.open"
      :position="multiMenu.position"
      :count="multiMenu.count"
      @update:open="(v) => { multiMenu.open = v }"
      @action="onMultiMenuAction"
    />
    <EdgeContextMenu
      v-if="edgeMenu.edge"
      :open="edgeMenu.open"
      :position="edgeMenu.position"
      :edge="edgeMenu.edge"
      @update:open="(v) => { edgeMenu.open = v }"
      @action="onEdgeMenuAction"
    />
    <PinContextMenu
      v-if="pinMenu.pin"
      :open="pinMenu.open"
      :position="pinMenu.position"
      :pin="pinMenu.pin"
      @update:open="(v) => { pinMenu.open = v }"
      @action="onPinMenuAction"
    />

    <!-- 命令面板 Ctrl+K -->
    <CommandPalette
      v-model:open="commandPaletteOpen"
      :commands="commands"
    />

    <!-- Promote-to-Variable modal -->
    <PromoteToVarModal
      v-if="promoteCtx"
      :open="!!promoteCtx"
      :context="promoteCtx"
      :existing-var-names="(draft?.vars ?? []).map(v => v.name)"
      @update:open="(v) => { if (!v) promoteCtx = null }"
      @confirm="onPromoteConfirm"
    />

    <!-- Save Snippet drawer (右侧抽屉) -->
    <!-- :key 强制 remount: open=true 时父先 set state 再 v-if=true, drawer setup 新跑,
         fillFromProps 拿到 final props.editingId. 解决 edit prefill 时序问题. -->
    <SaveSnippetDrawer
      v-if="saveSnippetState.open"
      :key="`drawer-${saveSnippetState.editingId || 'new'}-${saveSnippetState.sourceKind || ''}`"
      :open="true"
      :source-kind="saveSnippetState.sourceKind"
      :source-config="saveSnippetState.sourceConfig"
      :editing-id="saveSnippetState.editingId"
      @update:open="(v) => { if (!v) saveSnippetState.open = false }"
      @saved="onSnippetSaved"
    />

    <!-- Find-References modal -->
    <FindReferencesModal
      v-if="findRefsState"
      :open="!!findRefsState"
      :var-name="findRefsState.varName"
      :refs="findRefsState.refs"
      @update:open="(v) => { if (!v) findRefsState = null }"
      @pick="onFindRefsPick"
    />

    <!-- Ctrl+F canvas node search (UE Blueprint "Find in Blueprint" equivalent) -->
    <NodeSearchModal
      :open="nodeSearchOpen"
      :results="nodeSearchResults"
      @update:open="(v) => { nodeSearchOpen = v }"
      @update:query="(q) => { nodeSearchQuery = q }"
      @pick="onNodeSearchPick"
    />

  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, onUnmounted, ref } from 'vue'
import { useWindowControls } from '@/composables/useWindowControls'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from '@nuxt/ui/composables'
import { VueFlow, useVueFlow, SelectionMode, type NodeDragEvent, type NodeMouseEvent, type EdgeMouseEvent } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/controls/dist/style.css'
import '@vue-flow/minimap/dist/style.css'

import { backend, type Container, type GraphNode, type GraphEdge, type ValidationError, type VarDecl } from '@/lib/backend'
import { useRecordingStore } from '@/stores/recording'
import { useExecutionStore } from '@/stores/execution'
import { useContainersStore } from '@/stores/containers'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { useTemplatesStore } from '@/stores/templates'
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
import {
  AUTO_CONNECT_THRESHOLD_FLOW_PX,
  SNAP_EPSILON,
  SNAP_ANCHOR_Y_OFFSET,
  centerOnNode,
} from '@/composables/containerEditor/constants'
import { newNodeID, genNodeID, randID } from '@/composables/containerEditor/ids'
import ContainerFlowNode from '@/components/containers/ContainerFlowNode.vue'
import CommentBoxNode from '@/components/containers/CommentBoxNode.vue'
import ContainerEditorToolbar from '@/components/containers/ContainerEditorToolbar.vue'
import ContainerEditorBreadcrumb from '@/components/containers/ContainerEditorBreadcrumb.vue'
import ContainerEditorInspector from '@/components/containers/ContainerEditorInspector.vue'
import ValidationErrorPanel from '@/components/containers/ValidationErrorPanel.vue'
import { useSidebarPrefs } from '@/composables/editor/useSidebarPrefs'
import { useVarMutations } from '@/composables/containerEditor/useVarMutations'
import SnippetsPanel from '@/components/snippets/SnippetsPanel.vue'
import SaveSnippetDrawer from '@/components/snippets/SaveSnippetDrawer.vue'
import VarsPanel from '@/components/containers/sidebar/VarsPanel.vue'
import ContainerSettingsModal from '@/components/containers/ContainerSettingsModal.vue'
import DeleteVarConfirmModal from '@/components/containers/sidebar/DeleteVarConfirmModal.vue'
import NodeExplorerModal from '@/components/containers/NodeExplorerModal.vue'
import LibraryExplorerModal from '@/components/containers/LibraryExplorerModal.vue'
import InlineContextMenu, { type PinContext as InlinePinContext } from '@/components/containers/InlineContextMenu.vue'
import NodeContextMenu, { type NodeMenuAction } from '@/components/containers/menus/NodeContextMenu.vue'
import MultiNodeContextMenu, { type MultiMenuAction } from '@/components/containers/menus/MultiNodeContextMenu.vue'
import EdgeContextMenu, { type EdgeMenuAction } from '@/components/containers/menus/EdgeContextMenu.vue'
import PinContextMenu, { type PinMenuAction, type PinInfo } from '@/components/containers/menus/PinContextMenu.vue'
import CommandPalette, { type Command } from '@/components/containers/CommandPalette.vue'
import PromoteToVarModal, { type PromoteContext } from '@/components/containers/PromoteToVarModal.vue'
import FindReferencesModal, { type RefEntry } from '@/components/containers/FindReferencesModal.vue'
import NodeSearchModal, { type NodeSearchResult } from '@/components/containers/NodeSearchModal.vue'
import SnapGuideOverlay, { type SnapGuide } from '@/components/containers/SnapGuideOverlay.vue'
import { useSnippetsStore, eventToShortcutKey, type Snippet } from '@/stores/snippets'
import { useLibraryStore } from '@/stores/library'
import { KIND_DEFAULTS, KIND_LABEL_ZH, PIN_SPECS } from '@/components/containers/pinSpec'
import { markRaw } from 'vue'
import { readDragPayload, type EditorDragPayload } from '@/composables/editor/useEditorDragDrop'
import { dataInTypeFor, dataOutTypeFor, getSpec } from '@/components/containers/nodeRegistry/registry'
import { isCompatibleType, type VarType } from '@/lib/variableRef'
import SplitHandle from '@/components/common/SplitHandle.vue'
import { useSplitpane } from '@/composables/useSplitpane'

const route = useRoute()
const router = useRouter()
const isStandalone = computed(() => route.query.standalone === '1')
const toast = useToast()
const recordStore = useRecordingStore()
const execStore = useExecutionStore()
const containersStore = useContainersStore()
const tplStore = useTemplatesStore()

const editorStore = useContainerEditorStore()

const runningNodeLabel = computed(
  () => KIND_LABEL_ZH[execStore.currentNodeKind] ?? execStore.currentNodeKind ?? '',
)

async function onStopRun() {
  await containersStore.stopAll()
}

async function onOpenNewWindow() {
  const id = containerID
  if (!id) return
  try {
    await backend.containers.openInWindow(id)
    // 嵌入态主动放手: 跳回 /containers 列表 (子窗口接管 edit acquisition)
    router.push('/containers')
  } catch (e) {
    console.error('OpenInWindow failed:', e)
  }
}

// 嵌入路由 /containers/:id/edit 用 params.id; 子窗口同款路由 + ?standalone=1 也走 params.
// 老 /container-editor?id=xxx 已删, query.id fallback 仅兜底极少数老链接.
const containerID = String(route.params.id ?? route.query.id ?? '')

const {
  draft,
  dirty,
  activeGraph,
  flowNodes,
  flowEdges,
  syncFlowFromDraft,
  refreshSubgraphStore,
  applyDraftMutation,
  undo,
  redo,
  canUndo,
  canRedo,
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

// 折叠侧栏：持久化到 localStorage via useSidebarPrefs
const { prefs: sidebarPrefs } = useSidebarPrefs()
const leftPane = useSplitpane('editor.splitpane.left', { default: 280, min: 200, max: 480 })
const rightPane = useSplitpane('editor.splitpane.right', { default: 320, min: 200, max: 480 })
const settingsOpen = ref(false)
const nodeExplorerOpen = ref(false)
const libraryExplorerOpen = ref(false)

// ===== 命令面板 =====
const commandPaletteOpen = ref(false)

// ===== Ctrl+F canvas node search =====
const nodeSearchOpen = ref(false)
const nodeSearchQuery = ref('')

// ===== 右键菜单状态 =====
const nodeMenu = ref<{ open: boolean; position: { x: number; y: number }; node: GraphNode | null }>({
  open: false, position: { x: 0, y: 0 }, node: null,
})
const multiMenu = ref<{ open: boolean; position: { x: number; y: number }; count: number }>({
  open: false, position: { x: 0, y: 0 }, count: 0,
})
const edgeMenu = ref<{ open: boolean; position: { x: number; y: number }; edge: GraphEdge | null }>({
  open: false, position: { x: 0, y: 0 }, edge: null,
})
const pinMenu = ref<{ open: boolean; position: { x: number; y: number }; pin: PinInfo | null }>({
  open: false, position: { x: 0, y: 0 }, pin: null,
})

// ===== Promote-to-Variable =====
const promoteCtx = ref<PromoteContext | null>(null)

// ===== Find-References modal =====
const findRefsState = ref<{ varName: string; refs: RefEntry[] } | null>(null)

const varMutations = useVarMutations(draft)

function onAddVar() {
  // Auto-name v1/v2/v3 — VarsPanel later allows rename via inline edit.
  applyDraftMutation((d) => {
    let n = 1
    const vars = d.vars ?? []
    while (vars.some(v => v.name === `v${n}`)) n++
    if (!d.vars) d.vars = []
    d.vars.push({ name: `v${n}`, type: 'number', default: 0 })
  })
}

// Var CRUD handlers — wired to VarsPanel emits.
const deleteConfirm = ref<{ name: string; refIDs: string[] } | null>(null)

function onRequestDeleteVar(name: string) {
  const refs = varMutations.listUsageNodeIDs(name)
  if (refs.length === 0) {
    // 0 references → delete directly without confirm
    applyDraftMutation(() => varMutations.deleteVar(name, { cascade: false }))
  } else {
    deleteConfirm.value = { name, refIDs: refs }
  }
}

function onDeleteConfirm(cascade: boolean) {
  if (!deleteConfirm.value) return
  const name = deleteConfirm.value.name
  applyDraftMutation(() => varMutations.deleteVar(name, { cascade }))
  deleteConfirm.value = null
}

function onRenameVar(oldName: string, newName: string) {
  applyDraftMutation(() => varMutations.renameVar(oldName, newName))
}

function onUpdateVarField(name: string, field: 'type' | 'default', value: unknown) {
  applyDraftMutation((d) => {
    const v = (d.vars ?? []).find(x => x.name === name)
    if (!v) return
    if (field === 'type') {
      v.type = value as 'number' | 'bool' | 'string' | 'point' | 'any'
    } else {
      v.default = value
    }
  })
}

function onReorderVars(fromIdx: number, toIdx: number) {
  applyDraftMutation(() => varMutations.reorderVars(fromIdx, toIdx))
}

// ===== Drag-drop helpers =====


function defaultLiteralFor(type: VarDecl['type']): unknown {
  switch (type) {
    case 'number': return 0
    case 'string': return ''
    case 'bool': return false
    case 'point': return { x: 0.5, y: 0.5 }
    default: return null  // 'any' — no useful default
  }
}

function findVarType(name: string): VarDecl['type'] {
  if (!draft.value) return 'any'
  const v = (draft.value.vars ?? []).find(x => x.name === name)
  return (v?.type ?? 'any') as VarDecl['type']
}

// ===== Pin-aware auto-connect =====


interface PinAtPosition {
  nodeID: string
  pinName: string
  pinType: VarType
  dist: number
}

/**
 * Find the nearest eligible data-in pin within AUTO_CONNECT_THRESHOLD_FLOW_PX of flowPos.
 *
 * Data-in handles in this codebase use Position.Bottom + type="target", so vue-flow
 * renders them with data-handlepos="bottom" and CSS class "target". Exec-in handles
 * use Position.Left, so the selector `.vue-flow__handle[data-handlepos="bottom"].target`
 * isolates data-in pins only.
 *
 * Node ID comes from data-nodeid on the handle element itself (no parent traversal needed).
 */
function findNearestEligibleDataInPin(
  flowPos: { x: number; y: number },
  srcVarType: VarType,
): PinAtPosition | null {
  const handles = document.querySelectorAll<HTMLElement>(
    '.vue-flow__handle[data-handlepos="bottom"].target',
  )
  let best: PinAtPosition | null = null

  for (const handleEl of Array.from(handles)) {
    const nodeID = handleEl.getAttribute('data-nodeid') ?? ''
    const pinName = handleEl.getAttribute('data-handleid') ?? ''
    if (!nodeID || !pinName) continue

    const node = activeGraph.value?.nodes?.find(
      (n) => n.id === nodeID,
    ) ?? null
    if (!node) continue
    // Skip disabled nodes — they don't participate in auto-connect.
    if (node.disabled === true) continue

    const pinType = dataInTypeFor(node.kind, pinName, node.config as Record<string, unknown>)
    if (!pinType) continue  // not a data-in pin for this kind

    if (!isCompatibleType(srcVarType, pinType as VarType)) continue

    const rect = handleEl.getBoundingClientRect()
    const screenCenter = { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 }
    const flowCenter = screenToFlowCoordinate(screenCenter)
    const dx = flowCenter.x - flowPos.x
    const dy = flowCenter.y - flowPos.y
    const dist = Math.sqrt(dx * dx + dy * dy)
    if (dist > AUTO_CONNECT_THRESHOLD_FLOW_PX) continue

    if (!best || dist < best.dist) {
      best = { nodeID, pinName, pinType: pinType as VarType, dist }
    }
  }
  return best
}

function dropVar(
  payload: Extract<EditorDragPayload, { type: 'var' }>,
  pos: { x: number; y: number },
) {
  const kind = payload.modifier === 'alt' ? 'SetVar' : 'GetVar'
  const config: Record<string, unknown> = { varName: payload.ref.name, scope: 'auto' }
  if (kind === 'SetVar') {
    config.literal = { value: defaultLiteralFor(payload.ref.type) }
  }

  // Pin-aware auto-connect: detect nearest eligible data-in pin before mutation
  // (DOM query must run before applyDraftMutation triggers re-render)
  const autoConnectTarget = kind === 'GetVar'
    ? findNearestEligibleDataInPin(pos, payload.ref.type as VarType)
    : null

  applyDraftMutation((d) => {
    const g = activeGraph.value
    if (!g) return
    const node: GraphNode = {
      id: newNodeID(kind),
      kind,
      x: pos.x,
      y: pos.y,
      config,
      createdAt: new Date().toISOString(),
    } as GraphNode
    g.nodes.push(node)

    // Auto-connect GetVar.value → nearest eligible data-in pin
    if (autoConnectTarget) {
      ;(g.edges as GraphEdge[]).push({
        from: `${node.id}.value`,
        to: `${autoConnectTarget.nodeID}.${autoConnectTarget.pinName}`,
      } as GraphEdge)
    }
  })
}

function dropNodeSpec(
  payload: Extract<EditorDragPayload, { type: 'node-spec' }>,
  pos: { x: number; y: number },
) {
  const kind = payload.kind
  applyDraftMutation((d) => {
    const g = activeGraph.value
    if (!g) return
    const node: GraphNode = {
      id: newNodeID(kind),
      kind,
      x: pos.x,
      y: pos.y,
      config: getSpec(kind)?.defaults ?? {},
      createdAt: new Date().toISOString(),
    } as GraphNode
    g.nodes.push(node)
  })
}

/** snippet 拖到画布 → 在 pos 生成节点 with snippet.payload.config */
function dropSnippet(
  payload: Extract<EditorDragPayload, { type: 'snippet' }>,
  pos: { x: number; y: number },
) {
  const s = useSnippetsStore().getById(payload.snippetID)
  if (!s) return
  applyDraftMutation((d) => {
    const g = activeGraph.value
    if (!g) return
    const node: GraphNode = {
      id: newNodeID(s.payload.kind),
      kind: s.payload.kind,
      x: pos.x,
      y: pos.y,
      config: JSON.parse(JSON.stringify(s.payload.config)),
      label: s.name, // 用 snippet 名当节点 label, 一眼能看出是哪个 snippet 实例
      createdAt: new Date().toISOString(),
    } as GraphNode
    g.nodes.push(node)
  })
  useSnippetsStore().markUsed(payload.snippetID)
}

function onInsertIncVar(name: string) {
  // VarRow "+" hover button → insert IncVar at viewport center
  const center = screenToFlowCoordinate({
    x: window.innerWidth / 2,
    y: window.innerHeight / 2,
  })
  applyDraftMutation(() => {
    const g = activeGraph.value
    if (!g) return
    const node: GraphNode = {
      id: newNodeID('IncVar'),
      kind: 'IncVar',
      x: center.x,
      y: center.y,
      config: {
        varName: name,
        scope: 'auto',
        literal: { delta: 1 },
      },
      createdAt: new Date().toISOString(),
    } as GraphNode
    g.nodes.push(node)
  })
}

// ========== Snippet system wiring (Stage 2/3) ==========

const saveSnippetState = ref<{
  open: boolean
  sourceKind?: string
  sourceConfig?: Record<string, unknown>
  editingId?: string
}>({ open: false })

/** 从节点 ContextMenu 'save-as-snippet' action 触发: 打开 drawer create 模式. */
function emitSaveSnippetIntent(node: GraphNode) {
  saveSnippetState.value = {
    open: true,
    sourceKind: node.kind,
    sourceConfig: JSON.parse(JSON.stringify(node.config ?? {})),
    editingId: undefined,
  }
}

/** SnippetsPanel pencil 按钮 → 打开 drawer edit 模式. */
function onEditSnippet(s: Snippet) {
  saveSnippetState.value = {
    open: true,
    editingId: s.id,
  }
}

function onSnippetSaved(_s: Snippet) {
  // toast 反馈 (optional). 已经 persist 到 localStorage 由 store 自己干.
}

/** SnippetsPanel 单击 snippet (非 drag) → 在画布中心生成节点 with config. */
function onApplySnippet(s: Snippet) {
  applyDraftMutation(() => {
    const g = activeGraph.value
    if (!g) return
    // 画布中心 fallback (没 viewport center API 时用固定 offset)
    const pos = { x: 240 + Math.random() * 60, y: 180 + Math.random() * 60 }
    const node: GraphNode = {
      id: newNodeID(s.payload.kind),
      kind: s.payload.kind,
      x: pos.x,
      y: pos.y,
      config: JSON.parse(JSON.stringify(s.payload.config)),
      label: s.name,
      createdAt: new Date().toISOString(),
    } as GraphNode
    g.nodes.push(node)
  })
  useSnippetsStore().markUsed(s.id)
}

// 全局快捷键监听: snippet.shortcut 触发 → 在最后已知鼠标位置生成节点
const lastMousePos = ref({ x: 240, y: 180 })
function trackMouse(e: MouseEvent) {
  lastMousePos.value = { x: e.clientX, y: e.clientY }
}
function onSnippetShortcutKeydown(e: KeyboardEvent) {
  // 文本输入聚焦时不触发 (避免破坏 textarea/input)
  const t = e.target as HTMLElement | null
  if (t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || t.isContentEditable)) return
  const key = eventToShortcutKey(e)
  if (!key) return
  const s = useSnippetsStore().byShortcut.get(key)
  if (!s) return
  e.preventDefault()
  e.stopPropagation()
  const pos = screenToFlowCoordinate(lastMousePos.value)
  applyDraftMutation(() => {
    const g = activeGraph.value
    if (!g) return
    const node: GraphNode = {
      id: newNodeID(s.payload.kind),
      kind: s.payload.kind,
      x: pos.x,
      y: pos.y,
      config: JSON.parse(JSON.stringify(s.payload.config)),
      label: s.name,
      createdAt: new Date().toISOString(),
    } as GraphNode
    g.nodes.push(node)
  })
  useSnippetsStore().markUsed(s.id)
}
onMounted(() => {
  useSnippetsStore().load()
  window.addEventListener('keydown', onSnippetShortcutKeydown, true)
  window.addEventListener('mousemove', trackMouse, { passive: true })
})
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onSnippetShortcutKeydown, true)
  window.removeEventListener('mousemove', trackMouse)
})

// ========== /Snippet system ==========

function onCanvasDrop(e: DragEvent) {
  e.preventDefault()
  // MIME-based dispatch via useEditorDragDrop (var / node-spec / library-subgraph).
  const payload = readDragPayload(e)
  if (payload) {
    const pos = screenToFlowCoordinate({ x: e.clientX, y: e.clientY })
    switch (payload.type) {
      case 'var': return dropVar(payload, pos)
      case 'node-spec': return dropNodeSpec(payload, pos)
      case 'snippet': return dropSnippet(payload, pos)
      case 'library-subgraph': return  // not used yet
    }
    return
  }
  // Legacy: fallback to existing NodePalette / LibraryView drop logic
  _legacyCanvasDrop(e)
}

// Real usage count — sum across all vars (derive from draft, reactive)
const totalVarUsageCount = computed(() => {
  if (!draft.value) return 0
  return (draft.value.vars ?? []).reduce(
    (sum, v) => sum + varMutations.countUsage(v.name),
    0,
  )
})

function onOpenNodeExplorer() {
  nodeExplorerOpen.value = !nodeExplorerOpen.value
}

function onPickKind(kind: string) {
  applyDraftMutation((d) => {
    const g = activeGraph.value
    if (!g) return
    const node: GraphNode = {
      id: newNodeID(kind),
      kind,
      x: 200 + Math.random() * 100,
      y: 200 + Math.random() * 100,
      config: { ...(getSpec(kind)?.defaults ?? {}) },
      createdAt: new Date().toISOString(),
    } as GraphNode
    g.nodes.push(node)
  })
  // pushRecent 已删 (Snippet 系统替代 Recent)
}

function onOpenLibraryExplorer() {
  libraryExplorerOpen.value = true
}

async function onPickLibrarySubgraph(libraryID: string) {
  if (!draft.value) return
  try {
    await backend.library.importToContainer(libraryID, draft.value.id, '')
    const newSubgraphID = libraryID
    await refreshSubgraphStore()
    applyDraftMutation(() => {
      const g = activeGraph.value
      if (!g) return
      const node: GraphNode = {
        id: newNodeID('Subgraph'),
        kind: 'Subgraph',
        x: 200 + Math.random() * 100,
        y: 200 + Math.random() * 100,
        config: { SubgraphID: newSubgraphID },
        createdAt: new Date().toISOString(),
      } as GraphNode
      g.nodes.push(node)
    })
    // pushRecent 已删
    useLibraryStore().reload()
    toast.add({
      title: '子图已从库导入',
      description: `SubgraphID: ${newSubgraphID}`,
      color: 'success',
      icon: 'i-tabler-check',
    })
  } catch (e: any) {
    console.error('LibraryExplorer pick failed:', e)
    toast.add({
      title: '从库导入失败',
      description: String(e?.message ?? e),
      color: 'error',
    })
  }
}
// ===== InlineContextMenu (右键 canvas + pin drag → 添加节点) =====

interface InlineMenuState {
  open: boolean
  position: { x: number; y: number }  // viewport (screen) coords
  flowPos: { x: number; y: number }   // flow canvas coords
  pinContext?: InlinePinContext
  sourcePin?: { nodeID: string; pinName: string; side: 'input' | 'output'; isExec?: boolean }
}

const inlineMenu = ref<InlineMenuState>({
  open: false,
  position: { x: 0, y: 0 },
  flowPos: { x: 0, y: 0 },
})

// Track the pin that started the most recent connection drag
const connectionStart = ref<{
  nodeId: string
  handleId: string
  handleType: 'source' | 'target'
} | null>(null)
// Set to true by onVfConnect (success path) so onVfConnectEnd knows NOT to open the inline menu.
const connectMadeThisGesture = ref(false)

function isValidVueFlowConnection(conn: {
  source: string
  target: string
  sourceHandle?: string | null
  targetHandle?: string | null
}): boolean {
  if (!conn.sourceHandle || !conn.targetHandle) return true
  const srcNode = activeGraph.value?.nodes?.find(
    (n) => n.id === conn.source,
  )
  const tgtNode = activeGraph.value?.nodes?.find(
    (n) => n.id === conn.target,
  )
  if (!srcNode || !tgtNode) return true

  const srcOutType = dataOutTypeFor(srcNode.kind, conn.sourceHandle)
  const tgtInType = dataInTypeFor(tgtNode.kind, conn.targetHandle, tgtNode.config as Record<string, unknown>)

  // If either side is not a data pin (exec pin or unknown kind), allow
  if (!srcOutType || !tgtInType) return true

  return isCompatibleType(srcOutType as VarType, tgtInType as VarType)
}

function onVfConnectStart(params: {
  event?: MouseEvent
  nodeId?: string
  handleId: string | null
  handleType?: 'source' | 'target'
}) {
  if (params.nodeId && params.handleId && params.handleType) {
    connectionStart.value = {
      nodeId: params.nodeId,
      handleId: params.handleId,
      handleType: params.handleType,
    }
  }
}

function onVfConnectEnd(event?: MouseEvent) {
  if (!connectionStart.value) return

  const clientX = event?.clientX ?? 0
  const clientY = event?.clientY ?? 0

  const start = connectionStart.value
  connectionStart.value = null

  // Only open when the drag was released on empty space (no connection was made).
  // We defer via setTimeout so that vue-flow's @connect fires first if it will.
  // If a connection completed, vue-flow fires @connect synchronously in the same tick,
  // which sets connectMadeThisGesture = true in our onConnect wrapper.
  const startCopy = { ...start }
  setTimeout(() => {
    // Successful connect: onConnect already handled it — skip menu.
    if (connectMadeThisGesture.value) {
      connectMadeThisGesture.value = false
      return
    }

    const node = activeGraph.value?.nodes?.find(
      (n) => n.id === startCopy.nodeId,
    )
    if (!node) return

    const pinType =
      startCopy.handleType === 'source'
        ? dataOutTypeFor(node.kind, startCopy.handleId)
        : dataInTypeFor(node.kind, startCopy.handleId, node.config as Record<string, unknown>)

    const isExec = !pinType
    const flowPos = screenToFlowCoordinate({ x: clientX, y: clientY })

    inlineMenu.value = {
      open: true,
      position: { x: clientX, y: clientY },
      flowPos,
      // Exec pins have no type filter — show all nodes (menu header: "添加节点").
      // Data pins filter by compatible type (menu header: "⊕ 接受 <type>").
      pinContext: isExec
        ? undefined
        : {
            pinType: pinType as VarType,
            side: startCopy.handleType === 'source' ? 'output' : 'input',
          },
      sourcePin: {
        nodeID: startCopy.nodeId,
        pinName: startCopy.handleId,
        side: startCopy.handleType === 'source' ? 'output' : 'input',
        isExec,
      },
    }
  }, 0)
}

function openInlineMenuAt(clientX: number, clientY: number) {
  const flowPos = screenToFlowCoordinate({ x: clientX, y: clientY })
  inlineMenu.value = {
    open: true,
    position: { x: clientX, y: clientY },
    flowPos,
    pinContext: undefined,
    sourcePin: undefined,
  }
}

function onCanvasContextMenu(e: MouseEvent) {
  e.preventDefault()
  openInlineMenuAt(e.clientX, e.clientY)
}

// ===== 右键菜单事件处理 =====

function onNodeContextMenu(event: NodeMouseEvent) {
  event.event.preventDefault()
  const clientX = event.event instanceof MouseEvent ? event.event.clientX : 0
  const clientY = event.event instanceof MouseEvent ? event.event.clientY : 0
  const node = activeGraph.value?.nodes?.find(
    (n) => n.id === event.node.id,
  )
  if (!node) return
  // 多选且点击节点在选中集合内 → 显示多选菜单
  const sel = getSelectedNodes.value ?? []
  if (sel.length > 1 && sel.some((n: any) => n.id === node.id)) {
    multiMenu.value = {
      open: true,
      position: { x: clientX, y: clientY },
      count: sel.length,
    }
    nodeMenu.value.open = false
    return
  }
  nodeMenu.value = {
    open: true,
    position: { x: clientX, y: clientY },
    node,
  }
  multiMenu.value.open = false
}

function onSelectionContextMenu(event: { event: MouseEvent; nodes: any[] }) {
  event.event.preventDefault()
  multiMenu.value = {
    open: true,
    position: { x: event.event.clientX, y: event.event.clientY },
    count: event.nodes.length,
  }
  nodeMenu.value.open = false
}

function onNodeMenuAction(a: NodeMenuAction) {
  const node = nodeMenu.value.node
  if (!node) return
  switch (a) {
    case 'copy':
      onCopySelection()
      return
    case 'cut':
      onCopySelection()
      // 复制后删除选中节点 (剪切语义)
      getSelectedNodes.value.forEach((n: any) => removeNodes([n.id]))
      return
    case 'paste':
      void onPasteSelection()
      return
    case 'duplicate': {
      // 单节点复刻: 先选中该节点, 复制, 再粘贴
      onCopySelection()
      void onPasteSelection()
      return
    }
    case 'delete':
      removeNodes([node.id])
      if (selectedID.value === node.id) selectedID.value = null
      return
    case 'toggle-disable':
      applyDraftMutation((_d) => {
        const g = activeGraph.value
        const n = g?.nodes.find((x) => x.id === node.id) as (GraphNode & { disabled?: boolean }) | undefined
        if (!n) return
        n.disabled = !n.disabled
      })
      return
    case 'save-as-snippet':
      // Stage 2 实施: 触发 SaveSnippetDrawer 打开. 暂占位.
      emitSaveSnippetIntent(node)
      return
    case 'find-references': {
      const varName = (node.config as Record<string, unknown> | undefined)?.varName as string | undefined
      if (!varName) return
      const ids = varMutations.listUsageNodeIDs(varName)
      const refs: RefEntry[] = []
      const walk = (nodes: GraphNode[], location: string) => {
        for (const n of nodes) {
          if (ids.includes(n.id)) {
            refs.push({ id: n.id, kind: n.kind, label: n.label, location })
          }
        }
      }
      if (draft.value) {
        walk(draft.value.graph.nodes, '主图')
        for (const sg of draft.value.subgraphs ?? []) {
          if (sg.graph) walk(sg.graph.nodes, `子图: ${sg.label || sg.id}`)
        }
      }
      findRefsState.value = { varName, refs }
      return
    }
    case 'promote-to-var': {
      const lit = (node.config as Record<string, unknown> | undefined)?.literal as Record<string, unknown> | undefined
      if (!lit || Object.keys(lit).length === 0) {
        toast.add({ title: 'Promote', description: '此节点无 literal pin 可提取', color: 'warning' })
        return
      }
      // 单选节点 promote 只挑第一个 literal pin — 多 pin 走 PinContextMenu 单 pin promote.
      const pinName = Object.keys(lit)[0]
      const literal = lit[pinName]
      const pinType = dataInTypeFor(node.kind, pinName, node.config as Record<string, unknown>) as VarType | ''
      if (!pinType) {
        toast.add({ title: 'Promote', description: `pin ${pinName} 不是 data-in 类型`, color: 'warning' })
        return
      }
      promoteCtx.value = {
        nodeID: node.id,
        pinName,
        pinType: pinType as VarType,
        literal,
      }
      return
    }
    case 'jump-to-subgraph': {
      const sgID = (node.config as Record<string, unknown> | undefined)?.SubgraphID as string | undefined
      if (sgID) editorStore.pushPath(sgID)
      return
    }
    case 'share-to-library': {
      const sgID = (node.config as Record<string, unknown> | undefined)?.SubgraphID as string | undefined
      if (!sgID) return
      void shareSubgraphToLibrary(sgID)
      return
    }
  }
}

async function shareSubgraphToLibrary(sgID: string) {
  if (!containerID) return
  if (!window.confirm(`将子图 "${sgID}" 及其依赖打包分享到库（覆盖已有同名 package）？`)) return
  try {
    await backend.library.exportSubgraph(containerID, sgID, true)
    toast.add({ title: `已分享 ${sgID} 到库`, color: 'success', icon: 'i-tabler-check' })
  } catch (e: any) {
    toast.add({ title: '分享失败', description: String(e?.message ?? e), color: 'error' })
  }
}

function onMultiMenuAction(a: MultiMenuAction) {
  switch (a) {
    case 'copy':
      onCopySelection()
      return
    case 'cut':
      onCopySelection()
      getSelectedNodes.value.forEach((n: any) => removeNodes([n.id]))
      return
    case 'paste':
      void onPasteSelection()
      return
    case 'duplicate':
      onCopySelection()
      void onPasteSelection()
      return
    case 'delete':
      getSelectedNodes.value.forEach((n: any) => removeNodes([n.id]))
      return
    case 'toggle-disable-all':
      applyDraftMutation((_d) => {
        const g = activeGraph.value
        if (!g) return
        const sel = new Set(getSelectedNodes.value.map((n: any) => n.id))
        const allDisabled = (g.nodes as Array<GraphNode & { disabled?: boolean }>)
          .filter((n) => sel.has(n.id))
          .every((n) => n.disabled === true)
        for (const n of g.nodes as Array<GraphNode & { disabled?: boolean }>) {
          if (sel.has(n.id)) n.disabled = !allDisabled
        }
      })
      return
    case 'fold':
      onFoldSelection()
      return
    case 'auto-layout-lr':
      onAutoLayout('LR')
      return
    case 'auto-layout-tb':
      onAutoLayout('TB')
      return
    case 'align-left':
      onAlignSelected('left')
      return
    case 'align-right':
      onAlignSelected('right')
      return
    case 'align-top':
      onAlignSelected('top')
      return
    case 'align-bottom':
      onAlignSelected('bottom')
      return
    case 'align-center-h':
      onAlignSelected('center-h')
      return
    case 'align-center-v':
      onAlignSelected('center-v')
      return
    case 'distribute-h':
      onAlignSelected('h-equal')
      return
    case 'distribute-v':
      onAlignSelected('v-equal')
      return
  }
}

// ===== Find-References pick handler =====

async function onFindRefsPick(nodeID: string) {
  const currentNodes = activeGraph.value?.nodes
  const inCurrent = currentNodes?.some(n => n.id === nodeID) ?? false
  if (!inCurrent) {
    // Find which subgraph contains this node and jump there
    let targetSgID: string | null = null
    let targetNode: GraphNode | undefined
    for (const sg of draft.value?.subgraphs ?? []) {
      const found = sg.graph?.nodes?.find(n => n.id === nodeID)
      if (found) {
        targetSgID = sg.id
        targetNode = found
        break
      }
    }
    if (targetSgID) {
      editorStore.editorPath = [targetSgID]
      await nextTick()
      selectedID.value = nodeID
      if (targetNode) {
        centerOnNode(setCenter, targetNode)
      }
    } else {
      toast.add({ title: '跳转失败', description: `节点 ${nodeID} 不在当前容器`, color: 'warning' })
    }
    findRefsState.value = null
    return
  }
  selectedID.value = nodeID
  // Center viewport on the node
  const targetNode = currentNodes?.find(n => n.id === nodeID)
  if (targetNode) {
    centerOnNode(setCenter, targetNode)
  }
  findRefsState.value = null
}

// ===== Edge + Pin context menus =====

function onEdgeContextMenu(event: EdgeMouseEvent) {
  if (event.event instanceof MouseEvent) event.event.preventDefault()
  const clientX = event.event instanceof MouseEvent ? event.event.clientX : 0
  const clientY = event.event instanceof MouseEvent ? event.event.clientY : 0
  // Match the flow edge back to a GraphEdge via from/to composed from source/sourceHandle/target/targetHandle
  const flowEdge = activeGraph.value?.edges.find(
    (ed) =>
      ed.from === `${event.edge.source}.${event.edge.sourceHandle}` &&
      ed.to === `${event.edge.target}.${event.edge.targetHandle}`,
  )
  if (!flowEdge) return
  edgeMenu.value = {
    open: true,
    position: { x: clientX, y: clientY },
    edge: flowEdge,
  }
  // Close other menus
  nodeMenu.value.open = false
  multiMenu.value.open = false
  pinMenu.value.open = false
}

function onEdgeMenuAction(a: EdgeMenuAction) {
  const edge = edgeMenu.value.edge
  if (!edge) return
  switch (a) {
    case 'delete':
      applyDraftMutation(() => {
        const g = activeGraph.value
        if (!g) return
        g.edges = g.edges.filter((e) => !(e.from === edge.from && e.to === edge.to))
      })
      return
  }
}

function onCanvasContextMenuCapture(e: MouseEvent) {
  const t = e.target as HTMLElement | null
  if (!t) return
  const handleEl = t.closest('.vue-flow__handle') as HTMLElement | null
  if (!handleEl) return  // not a pin — let pane/node/edge-context-menu handlers deal
  // It's a pin handle — intercept
  e.preventDefault()
  e.stopPropagation()

  const nodeEl = handleEl.closest('[data-id]') as HTMLElement | null
  const nodeID = nodeEl?.getAttribute('data-id') ?? ''
  const pinName = handleEl.getAttribute('data-handleid') ?? ''
  const handleType = handleEl.getAttribute('data-handletype') ?? ''
  const side: 'input' | 'output' = handleType === 'source' ? 'output' : 'input'

  const node = activeGraph.value?.nodes.find((n) => n.id === nodeID)
  if (!node) return

  // Resolve pin type
  let pinType: string | undefined
  if (side === 'input') {
    pinType = dataInTypeFor(node.kind, pinName, node.config as Record<string, unknown>) || undefined
  } else {
    pinType = dataOutTypeFor(node.kind, pinName) || undefined
  }

  // Count attached edges
  const edges = activeGraph.value?.edges ?? []
  const matchFrom = `${nodeID}.${pinName}`
  const edgeCount = edges.filter((ed) => ed.from === matchFrom || ed.to === matchFrom).length

  pinMenu.value = {
    open: true,
    position: { x: e.clientX, y: e.clientY },
    pin: { nodeID, pinName, side, pinType, edgeCount },
  }
  // Close other menus
  nodeMenu.value.open = false
  multiMenu.value.open = false
  edgeMenu.value.open = false
}

function onPinMenuAction(a: PinMenuAction) {
  const pin = pinMenu.value.pin
  if (!pin) return
  const matchID = `${pin.nodeID}.${pin.pinName}`
  switch (a) {
    case 'break-all-connections':
      applyDraftMutation(() => {
        const g = activeGraph.value
        if (!g) return
        g.edges = g.edges.filter((ed) => ed.from !== matchID && ed.to !== matchID)
      })
      return
    case 'promote-to-var': {
      const node = activeGraph.value?.nodes.find(n => n.id === pin.nodeID)
      if (!node) return
      const lit = (node.config as Record<string, unknown> | undefined)?.literal as Record<string, unknown> | undefined
      const literal = lit?.[pin.pinName]
      if (literal === undefined) {
        toast.add({ title: 'Promote', description: `pin ${pin.pinName} 无 literal 可提取 (可能已被边驱动)`, color: 'warning' })
        return
      }
      promoteCtx.value = {
        nodeID: pin.nodeID,
        pinName: pin.pinName,
        pinType: (pin.pinType ?? 'any') as VarType,
        literal,
      }
      return
    }
    case 'reset-to-literal':
      applyDraftMutation(() => {
        const g = activeGraph.value
        if (!g) return
        // Break incoming edges to this input pin
        g.edges = g.edges.filter((ed) => ed.to !== matchID)
        const node = g.nodes.find((n) => n.id === pin.nodeID)
        if (!node) return
        const cfg = node.config as Record<string, unknown>
        const lit = (cfg.literal as Record<string, unknown> | undefined) ?? {}
        const def =
          pin.pinType === 'number' ? 0
          : pin.pinType === 'string' ? ''
          : pin.pinType === 'bool' ? false
          : pin.pinType === 'point' ? { x: 0.5, y: 0.5 }
          : null
        lit[pin.pinName] = def
        cfg.literal = lit
      })
      return
    case 'show-schema':
      toast.add({
        title: `Pin schema: ${pin.pinName}`,
        description: `${pin.side} · type: ${pin.pinType ?? '(exec)'} · edges: ${pin.edgeCount}`,
        color: 'info',
      })
      return
  }
}

// ===== Promote-to-Variable confirm handler =====
function onPromoteConfirm(args: { varName: string; varType: VarType }) {
  const ctx = promoteCtx.value
  if (!ctx) return
  applyDraftMutation((d) => {
    const g = activeGraph.value
    if (!g) return

    // 1. Add new var to Container.Vars
    if (!d.vars) d.vars = []
    d.vars.push({
      name: args.varName,
      type: args.varType as 'number' | 'bool' | 'string' | 'point' | 'any',
      default: ctx.literal,
    })

    // 2. Remove literal from original node's config
    const origNode = g.nodes.find(n => n.id === ctx.nodeID)
    if (!origNode) return
    const cfg = origNode.config as Record<string, unknown>
    const lit = cfg.literal as Record<string, unknown> | undefined
    if (lit) {
      delete lit[ctx.pinName]
    }

    // 3. Insert GetVar node 200px to the left
    const getVarID = newNodeID('GetVar')
    g.nodes.push({
      id: getVarID,
      kind: 'GetVar',
      x: (origNode.x ?? 0) - 200,
      y: origNode.y ?? 0,
      config: { varName: args.varName, scope: 'auto' },
      createdAt: new Date().toISOString(),
    } as GraphNode)

    // 4. Add data edge: GetVar.value → originalNode.pinName
    g.edges.push({
      from: `${getVarID}.value`,
      to: `${ctx.nodeID}.${ctx.pinName}`,
    } as GraphEdge)
  })
  promoteCtx.value = null
  // pushRecent 已删
}

function onPaneDoubleClick(e: MouseEvent) {
  openInlineMenuAt(e.clientX, e.clientY)
}

function onInlineMenuPick(kind: string) {
  const ctx = inlineMenu.value
  applyDraftMutation((_d) => {
    const g = activeGraph.value
    if (!g) return

    const newID = newNodeID(kind)
    const node: GraphNode = {
      id: newID,
      kind,
      x: ctx.flowPos.x,
      y: ctx.flowPos.y,
      config: { ...getSpec(kind)?.defaults },
      createdAt: new Date().toISOString(),
    } as GraphNode
    g.nodes.push(node)

    // Pin-context: auto-connect the new node back to the source pin
    if (ctx.sourcePin) {
      const spec = getSpec(kind)
      if (spec) {
        if (ctx.sourcePin.isExec) {
          // Exec-out drag → wire source exec-out → newNode first exec-in (usually "in")
          if (ctx.sourcePin.side === 'output') {
            const firstExecIn = spec.execIn?.[0]
            if (firstExecIn) {
              ;(g.edges as GraphEdge[]).push({
                from: `${ctx.sourcePin.nodeID}.${ctx.sourcePin.pinName}`,
                to: `${newID}.${firstExecIn}`,
              } as GraphEdge)
            }
          } else {
            // Exec-in drag → wire newNode first exec-out → source exec-in
            const firstExecOut = spec.execOut?.[0] ?? 'out'
            ;(g.edges as GraphEdge[]).push({
              from: `${newID}.${firstExecOut}`,
              to: `${ctx.sourcePin.nodeID}.${ctx.sourcePin.pinName}`,
            } as GraphEdge)
          }
        } else if (ctx.pinContext) {
          if (ctx.pinContext.side === 'output') {
            // User dragged from output pin → wire source.pin → newNode.firstCompatibleDataIn
            const candidates = Object.entries(spec.dataIn ?? {}).filter(([, t]) =>
              isCompatibleType(ctx.pinContext!.pinType, t as VarType),
            )
            if (candidates.length > 0) {
              ;(g.edges as GraphEdge[]).push({
                from: `${ctx.sourcePin.nodeID}.${ctx.sourcePin.pinName}`,
                to: `${newID}.${candidates[0][0]}`,
              } as GraphEdge)
            }
          } else {
            // User dragged from input pin → wire newNode.firstCompatibleDataOut → source.pin
            const candidates = Object.entries(spec.dataOut ?? {}).filter(([, t]) =>
              isCompatibleType(t as VarType, ctx.pinContext!.pinType),
            )
            if (candidates.length > 0) {
              ;(g.edges as GraphEdge[]).push({
                from: `${newID}.${candidates[0][0]}`,
                to: `${ctx.sourcePin.nodeID}.${ctx.sourcePin.pinName}`,
              } as GraphEdge)
            }
          }
        }
      }
    }
  })
  // pushRecent 已删 (Snippet 系统替代 Recent)
  syncFlowFromDraft()
}

// ===== End InlineContextMenu =====

// ===== 命令面板命令列表 =====
const commands = computed<Command[]>(() => {
  const sel = getSelectedNodes.value ?? []
  const selCount = sel.length
  return [
    // ── edit ──
    {
      id: 'edit.copy', label: '复制选中', group: 'edit', icon: 'i-tabler-copy', shortcut: 'Ctrl+C',
      disabled: selCount === 0,
      exec: () => onCopySelection(),
    },
    {
      id: 'edit.paste', label: '粘贴', group: 'edit', icon: 'i-tabler-clipboard', shortcut: 'Ctrl+V',
      exec: () => void onPasteSelection(),
    },
    {
      id: 'edit.delete', label: '删除选中', group: 'edit', icon: 'i-tabler-trash', shortcut: 'Del',
      disabled: selCount === 0,
      exec: () => sel.forEach((n: any) => removeNodes([n.id])),
    },
    {
      id: 'edit.undo', label: '撤销', group: 'edit', icon: 'i-tabler-arrow-back-up', shortcut: 'Ctrl+Z',
      disabled: !canUndo.value,
      exec: () => undo(),
    },
    {
      id: 'edit.redo', label: '重做', group: 'edit', icon: 'i-tabler-arrow-forward-up', shortcut: 'Ctrl+Y',
      disabled: !canRedo.value,
      exec: () => redo(),
    },
    {
      id: 'edit.fold', label: '折叠为子图', group: 'edit', icon: 'i-tabler-package-import',
      keywords: ['fold', 'subgraph', '折叠'],
      disabled: selCount === 0,
      exec: () => onFoldSelection(),
    },
    {
      id: 'edit.align-left', label: '对齐 - 左', group: 'edit', keywords: ['align', '对齐'],
      disabled: selCount < 2,
      exec: () => onAlignSelected('left'),
    },
    {
      id: 'edit.align-right', label: '对齐 - 右', group: 'edit', keywords: ['align', '对齐'],
      disabled: selCount < 2,
      exec: () => onAlignSelected('right'),
    },
    {
      id: 'edit.align-top', label: '对齐 - 顶', group: 'edit', keywords: ['align', '对齐'],
      disabled: selCount < 2,
      exec: () => onAlignSelected('top'),
    },
    {
      id: 'edit.align-bottom', label: '对齐 - 底', group: 'edit', keywords: ['align', '对齐'],
      disabled: selCount < 2,
      exec: () => onAlignSelected('bottom'),
    },
    {
      id: 'edit.align-center-h', label: '水平居中', group: 'edit', keywords: ['align', '对齐', '居中'],
      disabled: selCount < 2,
      exec: () => onAlignSelected('center-h'),
    },
    {
      id: 'edit.align-center-v', label: '垂直居中', group: 'edit', keywords: ['align', '对齐', '居中'],
      disabled: selCount < 2,
      exec: () => onAlignSelected('center-v'),
    },
    {
      id: 'edit.distribute-h', label: '水平等距分布', group: 'edit', keywords: ['distribute', '分布'],
      disabled: selCount < 3,
      exec: () => onAlignSelected('h-equal'),
    },
    {
      id: 'edit.distribute-v', label: '垂直等距分布', group: 'edit', keywords: ['distribute', '分布'],
      disabled: selCount < 3,
      exec: () => onAlignSelected('v-equal'),
    },
    {
      id: 'edit.auto-layout-lr', label: '自动布局 (横向)', group: 'edit',
      icon: 'i-tabler-layout-board-split', shortcut: 'Ctrl+L',
      keywords: ['layout', 'dagre', '布局'],
      exec: () => onAutoLayout('LR'),
    },
    {
      id: 'edit.auto-layout-tb', label: '自动布局 (纵向)', group: 'edit',
      icon: 'i-tabler-layout-rows', shortcut: 'Ctrl+Shift+L',
      keywords: ['layout', 'dagre', '布局'],
      exec: () => onAutoLayout('TB'),
    },

    // ── view ──
    {
      id: 'view.toggle-left-sidebar', label: '折叠/展开 左侧栏', group: 'view',
      keywords: ['sidebar', 'panel', '侧栏'],
      exec: () => { sidebarPrefs.value.leftSidebarCollapsed = !sidebarPrefs.value.leftSidebarCollapsed },
    },
    {
      id: 'view.toggle-inspector', label: '折叠/展开 右 Inspector', group: 'view',
      keywords: ['inspector', 'panel', '检查器'],
      exec: () => { sidebarPrefs.value.inspectorCollapsed = !sidebarPrefs.value.inspectorCollapsed },
    },

    // ── navigate ──
    {
      id: 'navigate.node-explorer', label: '打开 Node Explorer', group: 'navigate',
      icon: 'i-tabler-grid-dots', shortcut: 'Tab',
      keywords: ['explorer', 'node', '节点'],
      exec: () => { nodeExplorerOpen.value = !nodeExplorerOpen.value },
    },
    {
      id: 'navigate.library', label: '打开子图库 Explorer', group: 'navigate',
      icon: 'i-tabler-books',
      keywords: ['library', 'subgraph', '库', '子图'],
      exec: () => { libraryExplorerOpen.value = true },
    },
    {
      id: 'navigate.settings', label: '打开容器设置', group: 'navigate',
      icon: 'i-tabler-settings', shortcut: 'Ctrl+,',
      keywords: ['settings', 'config', '设置'],
      exec: () => { settingsOpen.value = true },
    },
    {
      id: 'navigate.back', label: '跳到主图 / 上一层级', group: 'navigate',
      keywords: ['back', 'up', '返回', '主图'],
      disabled: editorStore.editorPath.length === 0,
      exec: () => editorStore.popPath(),
    },

    // ── run ──
    {
      id: 'run.save', label: '保存', group: 'run',
      icon: 'i-tabler-check', shortcut: 'Ctrl+S',
      keywords: ['save', '保存'],
      disabled: !dirty.value,
      exec: () => void onSave(),
    },
    {
      id: 'run.validate', label: '检查 (Validate)', group: 'run',
      icon: 'i-tabler-checks',
      keywords: ['validate', 'check', '校验', '检查'],
      disabled: dirty.value,
      exec: () => void onValidate(),
    },
    {
      id: 'run.try-run', label: '试运行', group: 'run',
      icon: 'i-tabler-player-play',
      keywords: ['run', 'play', '运行'],
      disabled: dirty.value || execStore.running,
      exec: () => void onTryRun(),
    },
    {
      id: 'run.stop', label: '停止运行', group: 'run',
      icon: 'i-tabler-square',
      keywords: ['stop', 'halt', '停止'],
      disabled: !execStore.running,
      exec: () => void onStopRun(),
    },

    // ── var ──
    {
      id: 'var.add', label: '添加变量', group: 'var',
      icon: 'i-tabler-circle-plus',
      keywords: ['variable', 'add', '变量', '添加'],
      exec: () => onAddVar(),
    },

    // ── navigate: find-node (Ctrl+F) ──
    {
      id: 'navigate.find-node', label: '在画布找节点...', group: 'navigate',
      icon: 'i-tabler-search', shortcut: 'Ctrl+F',
      keywords: ['find', 'search', '搜索', '查找'],
      exec: () => { nodeSearchOpen.value = true },
    },
  ]
})

// ===== Node search computed results =====
const nodeSearchResults = computed<NodeSearchResult[]>(() => {
  if (!draft.value) return []
  const q = nodeSearchQuery.value.toLowerCase().trim()
  if (!q) return []
  const out: NodeSearchResult[] = []
  const walk = (nodes: any[], location: string, sgID: string | null) => {
    for (const n of nodes) {
      const hay = `${n.id} ${n.kind} ${n.label ?? ''}`.toLowerCase()
      if (hay.includes(q)) {
        out.push({ id: n.id, kind: n.kind, label: n.label, location, sgID })
      }
    }
  }
  walk(draft.value.graph.nodes, '主图', null)
  for (const sg of draft.value.subgraphs ?? []) {
    if (sg.graph) walk(sg.graph.nodes, `子图: ${sg.label || sg.id}`, sg.id)
  }
  return out
})

async function onNodeSearchPick(r: NodeSearchResult) {
  const inCurrent = (!r.sgID && editorStore.editorPath.length === 0)
    || (!!r.sgID && editorStore.editorPath[editorStore.editorPath.length - 1] === r.sgID)
  if (!inCurrent) {
    if (r.sgID) {
      editorStore.editorPath = [r.sgID]
    } else {
      editorStore.editorPath = []
    }
    await nextTick()
  }
  selectedID.value = r.id
  const target = activeGraph.value?.nodes.find((n: any) => n.id === r.id)
  if (target) {
    try {
      centerOnNode(setCenter, target as { x: number; y: number })
    } catch {}
  }
  nodeSearchOpen.value = false
}

function onGlobalKeydown(e: KeyboardEvent) {
  // Ctrl+K → 命令面板
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
    const t = e.target as HTMLElement | null
    const tag = t?.tagName?.toLowerCase()
    if (tag === 'input' || tag === 'textarea' || tag === 'select' || t?.isContentEditable) return
    e.preventDefault()
    commandPaletteOpen.value = true
    return
  }
  // Ctrl+F → 画布节点搜索
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'f') {
    const t = e.target as HTMLElement | null
    const tag = t?.tagName?.toLowerCase()
    if (tag === 'input' || tag === 'textarea' || tag === 'select' || t?.isContentEditable) return
    e.preventDefault()
    nodeSearchOpen.value = !nodeSearchOpen.value
    return
  }
  // Ctrl+S → 保存
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') {
    e.preventDefault()
    if (dirty.value) void onSave()
    return
  }
  // Ctrl+, → open settings (Mac Cmd+, also works)
  if ((e.ctrlKey || e.metaKey) && e.key === ',') {
    e.preventDefault()
    settingsOpen.value = true
    return
  }
  // Ctrl+Z → undo (skip when input/textarea/select/contentEditable is focused)
  if ((e.ctrlKey || e.metaKey) && !e.shiftKey && e.key.toLowerCase() === 'z') {
    const t = e.target as HTMLElement | null
    const tag = t?.tagName?.toLowerCase()
    if (tag === 'input' || tag === 'textarea' || tag === 'select' || t?.isContentEditable) return
    e.preventDefault()
    undo()
    return
  }
  // Ctrl+Shift+Z / Ctrl+Y → redo
  if (
    (e.ctrlKey || e.metaKey) &&
    ((e.shiftKey && e.key.toLowerCase() === 'z') || (!e.shiftKey && e.key.toLowerCase() === 'y'))
  ) {
    const t = e.target as HTMLElement | null
    const tag = t?.tagName?.toLowerCase()
    if (tag === 'input' || tag === 'textarea' || tag === 'select' || t?.isContentEditable) return
    e.preventDefault()
    redo()
    return
  }
  // Tab → toggle NodeExplorer
  if (e.key === 'Tab') {
    // If Explorer already open, always allow Tab to close it (even from inside the modal's search input)
    if (nodeExplorerOpen.value) {
      e.preventDefault()
      nodeExplorerOpen.value = false
      return
    }
    // Modal not open — open it, but skip when typing in an unrelated input
    const t = e.target as HTMLElement | null
    const tag = t?.tagName?.toLowerCase()
    if (tag === 'input' || tag === 'textarea' || tag === 'select' || t?.isContentEditable) return
    e.preventDefault()
    nodeExplorerOpen.value = true
    return
  }
}

// FlowNode / FlowEdge 类型从 useContainerDraft export (公共声明), view 不再局部重复定义.

// 注册自定义节点组件：从 PIN_SPECS keys 自动派生，无需手维护。
// 加新 kind 只需在 pinSpec.ts 里加一条 PIN_SPECS 即可——nodeTypes / NodePalette / FlowNode 自动响应，避免漏注册。
// CommentBox 是 visual-only (no handles) — 用独立 Vue 组件, 其他 kind 共享 ContainerFlowNode.
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
  return g.nodes.find((n) => n.id === selectedID.value) ?? null
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
  const id = kind === 'Start' ? 'start' : genNodeID()
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
const { project, getSelectedNodes, removeNodes, screenToFlowCoordinate, setCenter, updateNode } = useVueFlow()

// ===== Pin-aware snap guides (PS smart-guides style) =====

const snapGuides = ref<SnapGuide[]>([])

function onSnapNodeDrag(event: NodeDragEvent) {
  // wantSnap: toolbar toggle XOR Alt key — Alt inverts whichever the toggle is (PS/Figma pattern)
  const wantSnap = sidebarPrefs.value.snapEnabled !== event.event.altKey
  if (!wantSnap) {
    snapGuides.value = []
    return
  }

  const draggedID: string = event.node.id
  const yOff = SNAP_ANCHOR_Y_OFFSET

  // event.node.position is the live flow coordinate (updated by vue-flow during drag)
  const draggedX: number = event.node.position?.x ?? 0
  const draggedY: number = event.node.position?.y ?? 0
  const draggedAnchorY = draggedY + yOff

  const guides: SnapGuide[] = []
  const nodes = activeGraph.value?.nodes ?? []
  for (const other of nodes) {
    if (other.id === draggedID) continue
    const otherAnchorY = other.y + SNAP_ANCHOR_Y_OFFSET
    if (Math.abs(otherAnchorY - draggedAnchorY) <= SNAP_EPSILON) {
      // Horizontal cyan guide at otherAnchorY — coords stay in flow space;
      // SnapGuideOverlay applies the vue-flow viewport transform via <g :transform>.
      const lineFlowY = otherAnchorY
      const leftFlowX = Math.min(other.x, draggedX) - 30
      const rightFlowX = Math.max(other.x + 220, draggedX + 220) + 30
      guides.push({
        axis: 'y',
        x1: leftFlowX,
        y1: lineFlowY,
        x2: rightFlowX,
        y2: lineFlowY,
      })
    }
    // X-axis (vertical magenta guide) — left-edge alignment
    if (Math.abs(other.x - draggedX) <= SNAP_EPSILON) {
      const lineFlowX = other.x
      const topFlowY = Math.min(other.y, draggedY) - 30
      const botFlowY = Math.max(other.y + 80, draggedY + 80) + 30
      guides.push({
        axis: 'x',
        x1: lineFlowX,
        y1: topFlowY,
        x2: lineFlowX,
        y2: botFlowY,
      })
    }
  }
  snapGuides.value = guides
}

function onSnapNodeDragStop(event: NodeDragEvent) {
  snapGuides.value = []

  // wantSnap: toolbar toggle XOR Alt key — Alt inverts whichever the toggle is (PS/Figma pattern)
  const wantSnap = sidebarPrefs.value.snapEnabled !== event.event.altKey
  if (!wantSnap) return

  const draggedID: string = event.node.id

  // event.node.position 是 vue-flow store 的 live 坐标 (跟 onSnapNodeDrag 同源).
  // 不能读 flowNodes.value: v-model:nodes 的 store→model watcher (vue-flow-core
  // pauseStore) shallow 监听 array ref + length, drag 中 element.position 突变
  // 不触发同步 → flowNodes[i].position 在 dragStop 时仍是拖动前坐标 → snap 比对全错.
  const yOff = SNAP_ANCHOR_Y_OFFSET
  const draggedX = event.node.position?.x ?? 0
  const draggedY = event.node.position?.y ?? 0
  const draggedAnchorY = draggedY + yOff

  let bestY: { delta: number; targetAnchorY: number } | null = null
  let bestX: { delta: number; targetX: number } | null = null

  for (const other of activeGraph.value?.nodes ?? []) {
    if (other.id === draggedID) continue
    const otherAnchorY = other.y + SNAP_ANCHOR_Y_OFFSET
    const dy = otherAnchorY - draggedAnchorY
    if (Math.abs(dy) <= SNAP_EPSILON) {
      if (!bestY || Math.abs(dy) < Math.abs(bestY.delta)) {
        bestY = { delta: dy, targetAnchorY: otherAnchorY }
      }
    }
    const dx = other.x - draggedX
    if (Math.abs(dx) <= SNAP_EPSILON) {
      if (!bestX || Math.abs(dx) < Math.abs(bestX.delta)) {
        bestX = { delta: dx, targetX: other.x }
      }
    }
  }

  if (bestY || bestX) {
    const finalX = bestX ? bestX.targetX : draggedX
    const finalY = bestY ? bestY.targetAnchorY - yOff : draggedY

    // 1. Tell vue-flow's internal node store about the snapped position.
    //    Position 突变本身不触发 v-model 同步 (见上); 这里直显 store 让 computedPosition
    //    watcher (vue-flow-core.mjs:9534) 立刻重算 → DOM transform 跳到 snap target.
    updateNode(draggedID, { position: { x: finalX, y: finalY } })

    // 2. Persist to draft (authoritative source-of-truth for save/reload).
    applyDraftMutation((d) => {
      const g = activeGraph.value
      if (!g) return
      const node = g.nodes.find((n) => n.id === draggedID)
      if (!node) return
      node.x = finalX
      node.y = finalY
    })
  }
}

// ===== End snap guides =====

// 画布 drag/drop 交互（NodePalette → Canvas + LibraryView 卡片 → Canvas copy-on-use）
const { onCanvasDragOver, onCanvasDrop: _legacyCanvasDrop } = useFlowInteraction({
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
const { onCopySelection, onPasteSelection } = useNodeClipboard({
  draft, activeGraph, flowNodes,
  syncFlowFromDraft, refreshSubgraphStore,
  deepCloneSubgraphForCopy, getSelectedNodes,
  genID: genNodeID, toast,
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

// vue-flow selection-change: 拖完后 click 偶尔不发 node-click, 但 internal selection
// 仍变 (节点视觉 selected 但 selectedID 没同步 → 属性栏 80% 不更新). selection-change
// 比 node-click 可靠, 单选 → 同步 ID, 多/空选 → null (multi 菜单 / inspector hide 处理).
function onSelectionChange(evt: { nodes: any[]; edges: any[] }) {
  if (evt.nodes?.length === 1) {
    selectedID.value = evt.nodes[0].id
  } else if (!evt.nodes?.length) {
    selectedID.value = null
  }
}

function onNodeDoubleClick(evt: any) {
  const n = evt.node
  // CollapsedNode 跟 Subgraph 共享 navigation 语义 (both wrap a subgraph by ID).
  if (n?.data?.kind === 'Subgraph' || n?.data?.kind === 'CollapsedNode') {
    const sgID = n.data.config?.SubgraphID
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
const { onNodesChange, onEdgeDoubleClick, onEdgesChange, onConnect: _onConnectBase } = useGraphMutations({
  activeGraph,
  flowEdges,
  syncFlowFromDraft,
  findNodeAcrossGraphs,
  deleteSubgraphCascade,
})

// Wrap onConnect: set connectMadeThisGesture so onVfConnectEnd skips the inline menu.
function onConnect(c: Parameters<typeof _onConnectBase>[0]) {
  connectMadeThisGesture.value = true
  connectionStart.value = null
  _onConnectBase(c)
}

function onConfigUpdate(cfg: Record<string, any>) {
  if (!draft.value || !selectedNode.value) return
  selectedNode.value.config = cfg
  // 配置变更可能改变 exec out pin 集 (Switch.cases / Parallel.n / Race.n) —
  // flowNodes 持有旧 config 引用快照, computed pins 不会重算 → handle 不刷新.
  // 这里重建 flow 让 ContainerFlowNode 拿到新 config 引用.
  syncFlowFromDraft()
}

function onLabelUpdate(newLabel: string) {
  if (!selectedNode.value) return
  const targetID = selectedNode.value.id
  applyDraftMutation((d) => {
    const g = activeGraph.value
    if (!g) return
    const n = g.nodes.find((x) => x.id === targetID)
    if (!n) return
    const trimmed = newLabel.trim()
    if (trimmed) n.label = trimmed
    else delete n.label
  })
}

// Expr fusion listener — Inspector 通过 'expr-fuse' CustomEvent 触发. (TODO: 走 Pinia store, 见 backlog C10)
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
onMounted(() => {
  window.addEventListener('expr-fuse', onExprFuseEvent)
  window.addEventListener('keydown', onGlobalKeydown)
  tplStore.setContainer(containerID)
})
onUnmounted(() => {
  window.removeEventListener('expr-fuse', onExprFuseEvent)
  window.removeEventListener('keydown', onGlobalKeydown)
})

function onDeleteSelected() {
  if (!draft.value || !selectedID.value) return
  // 走 vue-flow removeNodes → 触发 onNodesChange(remove) → 统一处理 Subgraph cascade
  removeNodes([selectedID.value])
  selectedID.value = null
}

const selectedCount = computed(() => getSelectedNodes.value.length)

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
  if (mainGraph.nodes.some((n) => n.kind === 'WindowTarget')) {
    toast.add({ title: '主图已经有 WindowTarget 节点了', color: 'warning' })
    return
  }
  const defaults = KIND_DEFAULTS.WindowTarget ?? {}
  const newNode: GraphNode = {
    id: newNodeID('WindowTarget'),
    kind: 'WindowTarget',
    x: 40,
    y: 40,
    config: JSON.parse(JSON.stringify(defaults)),
    createdAt: new Date().toISOString(),
  }
  mainGraph.nodes.push(newNode)
  syncFlowFromDraft()
  validationPanelOpen.value = false
  toast.add({
    title: '已添加 WindowTarget 节点',
    description: '请打开节点 Inspector 配置目标窗口 (或点"捕获前台窗口")',
    color: 'success',
    icon: 'i-tabler-check',
  })
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

/* ---- Canvas 背景: 深色 radial gradient + 微妙 vignette + 网格 dots ---- */
.canvas-bg {
  background:
    radial-gradient(
      ellipse 80% 60% at 50% 0%,
      rgba(99, 102, 241, 0.08) 0%,
      transparent 70%
    ),
    radial-gradient(
      ellipse 60% 50% at 50% 100%,
      rgba(6, 182, 212, 0.05) 0%,
      transparent 65%
    ),
    linear-gradient(180deg, #0c0c14 0%, #07070c 100%);
}
.canvas-bg::after {
  /* vignette: 边缘略暗, 中心略亮, 把焦点收回画布中央 */
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: radial-gradient(
    ellipse at center,
    transparent 50%,
    rgba(0, 0, 0, 0.35) 100%
  );
  z-index: 0;
}
/* vue-flow 内部 viewport / pane / nodes 都 z-index > 0, vignette 不挡 */

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
