<template>
  <!-- h-full 撑满父 (嵌入: <main>; 子窗: App.vue isStandalone div). overflow-hidden 防 minimap/canvas
       撑出来触发 <main> 的 overflow-auto 滚动条 — 编辑器自管 canvas 缩放, 不允许外层滚. -->
  <div class="flex flex-col h-full min-h-0 overflow-hidden bg-default text-default">
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
        >{{ t('editor.header.back') }}</UButton
      >
      <UIcon name="i-tabler-schema" class="size-3.5 text-dimmed shrink-0" />
      <h3 class="text-xs font-medium truncate text-toned">
        {{ draft?.name ?? t('editor.header.loading') }}
      </h3>
      <span v-if="dirty" class="text-[10px] text-amber-300/80 shrink-0">{{ t('editor.header.dirty_dot') }}</span>

      <div class="flex-1" />

      <!-- 窗口控件（min / max-restore / close）-->
      <div class="flex items-stretch h-full" style="--wails-draggable: no-drag">
        <button
          type="button"
          class="w-11 flex items-center justify-center text-muted hover:bg-elevated/60 hover:text-highlighted transition-colors"
          :title="t('editor.window.minimize')"
          @click="onMinimise"
        >
          <UIcon name="i-tabler-minus" class="size-4" />
        </button>
        <button
          type="button"
          class="w-11 flex items-center justify-center text-muted hover:bg-elevated/60 hover:text-highlighted transition-colors"
          :title="isMaximised ? t('editor.window.restore') : t('editor.window.maximize')"
          @click="onToggleMaximise"
        >
          <UIcon :name="isMaximised ? 'i-tabler-copy' : 'i-tabler-square'" class="size-3.5" />
        </button>
        <button
          type="button"
          class="w-11 flex items-center justify-center text-muted hover:bg-error hover:text-highlighted transition-colors"
          :title="t('editor.window.close')"
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
            <h3 class="text-sm font-medium">{{ t('editor.dirty.title') }}</h3>
          </div>
          <p class="text-xs text-muted">{{ t('editor.dirty.desc') }}</p>
          <div class="flex justify-end gap-2 pt-2">
            <UButton variant="ghost" color="neutral" @click="confirmCloseOpen = false"
              >{{ t('editor.dirty.cancel') }}</UButton
            >
            <UButton
              class="ml-auto"
              color="error"
              icon="i-tabler-x"
              @click="onConfirmDiscardAndClose"
              >{{ t('editor.dirty.discard') }}</UButton
            >
            <UButton color="primary" icon="i-tabler-check" @click="onSaveAndClose"
              >{{ t('editor.dirty.save_and_close') }}</UButton
            >
          </div>
        </div>
      </template>
    </UModal>

    <div v-if="!draft" class="flex-1 flex items-center justify-center text-sm text-muted">
      {{ t('editor.header.loading') }}
    </div>

    <div v-else class="flex flex-col flex-1 min-h-0">
      <!-- Toolbar 独立一行：左 [折叠 palette] [录制] [折叠 inspector]，右 [运行状态] [试运行/停止] [保存] -->
      <ContainerEditorToolbar
        v-model:palette-collapsed="sidebarPrefs.leftSidebarCollapsed"
        v-model:inspector-collapsed="sidebarPrefs.inspectorCollapsed"
        :is-standalone="isStandalone"
        :is-recording="recordStore.isRecording || recordStore.isPaused"
        :recording-target-name="recordingTargetName"
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
        @back-to-list="onBackToList"
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
              {{ t('editor.sidebar.palette_tab') }}
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
            {{ t('editor.canvas.hint') }}
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
      @save="onSettingsSave"
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
// 给 <keep-alive include="ContainerEditorView"> 用 — 切到 settings/help 等再回来,
// draft / canvas viewport / selection / dirty 全保留, 不重新 load.
defineOptions({ name: 'ContainerEditorView' })

import { computed, nextTick, onBeforeUnmount, onMounted, provide, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ContainerCanvasApiKey } from '@/composables/containerEditor/pinLiterals'
import { useWindowControls } from '@/composables/useWindowControls'
import { useRoute, useRouter, onBeforeRouteLeave } from 'vue-router'
import { useToast } from '@nuxt/ui/composables'
import { useConfirm } from '@/composables/useConfirm'
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
  centerOnNode,
} from '@/composables/containerEditor/constants'
import { useSnapEngine } from '@/composables/containerEditor/useSnapEngine'
import { useEditorHotkeys } from '@/composables/containerEditor/useEditorHotkeys'
import { useNodeSearch } from '@/composables/containerEditor/useNodeSearch'
import { useInlineMenu } from '@/composables/containerEditor/useInlineMenu'
import { useCommandPalette } from '@/composables/containerEditor/useCommandPalette'
import { useContextMenuRouter } from '@/composables/containerEditor/useContextMenuRouter'
import { useNodeCreation } from '@/composables/containerEditor/useNodeCreation'
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
import NodeContextMenu from '@/components/containers/menus/NodeContextMenu.vue'
import MultiNodeContextMenu from '@/components/containers/menus/MultiNodeContextMenu.vue'
import EdgeContextMenu from '@/components/containers/menus/EdgeContextMenu.vue'
import PinContextMenu from '@/components/containers/menus/PinContextMenu.vue'
import CommandPalette from '@/components/containers/CommandPalette.vue'
import PromoteToVarModal, { type PromoteContext } from '@/components/containers/PromoteToVarModal.vue'
import FindReferencesModal, { type RefEntry } from '@/components/containers/FindReferencesModal.vue'
import NodeSearchModal from '@/components/containers/NodeSearchModal.vue'
import SnapGuideOverlay from '@/components/containers/SnapGuideOverlay.vue'
import { useSnippetsStore, eventToShortcutKey, type Snippet } from '@/stores/snippets'
import { KIND_DEFAULTS, KIND_LABEL_ZH, PIN_SPECS } from '@/components/containers/pinSpec'
import { markRaw } from 'vue'
import { readDragPayload } from '@/composables/editor/useEditorDragDrop'
import { getSpec } from '@/components/containers/nodeRegistry/registry'
import SplitHandle from '@/components/common/SplitHandle.vue'
import { useSplitpane } from '@/composables/useSplitpane'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const isStandalone = computed(() => route.query.standalone === '1')
const toast = useToast()
const { confirm } = useConfirm()
const recordStore = useRecordingStore()
const execStore = useExecutionStore()
const containersStore = useContainersStore()
const tplStore = useTemplatesStore()

const editorStore = useContainerEditorStore()

const runningNodeLabel = computed(() => {
  const k = execStore.currentNodeKind
  if (!k) return ''
  const key = KIND_LABEL_ZH[k]
  return key ? t(key) : k
})

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

// 嵌入态 toolbar '返回列表' 入口. ContainersView mount 时 clearLastEditing →
// 之后 sidebar '容器' 不再跳回该编辑器, 回归列表行为.
function onBackToList() {
  router.push('/containers')
}

// 嵌入路由 /containers/:id/edit 用 params.id; 子窗口同款路由 + ?standalone=1 也走 params.
// query.id 仅作兜底.
const containerID = String(route.params.id ?? route.query.id ?? '')

// 标 "正在编辑这个容器" — 侧栏 '容器' 跳法用. ContainersView mount 时会 clear.
if (containerID) editorStore.setLastEditing(containerID)

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

// A4: 录制态显眼标 "录制中 → 容器名". target 来源是 recordStore 单一值; 名字优先本容器 draft.name
// (正常录制就是录本容器), fallback 容器列表 / 裸 ID.
const recordingTargetName = computed(() => {
  const id = recordStore.activeTargetContainerID
  if (!id) return ''
  if (id === containerID) return draft.value?.name ?? id
  return containersStore.list.find((c) => c.id === id)?.name ?? id
})

// A3: 录制进行中离开"正在录的容器"编辑器 → 确认. 留下 → 录完正常 autoConnect 接节点;
// 确认离开 → 放行 (子图已落盘, 但不自动接入当前视图; onSubgraphCreated 的 mismatch 守卫兜底不 dangling).
onBeforeRouteLeave(async () => {
  if ((recordStore.isRecording || recordStore.isPaused) && recordStore.activeTargetContainerID === containerID) {
    const ok = await confirm({
      title: t('recordComposable.leave_title'),
      description: t('recordComposable.leave_during_recording'),
      color: 'warning',
      confirmText: t('recordComposable.leave_confirm'),
      cancelText: t('common.cancel'),
    })
    return ok === true
  }
  return true
})

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

// 节点创建 pipeline (drop / picker / programmatic add) — 9 处 GraphNode push 散落抽这里.
const {
  dropVar, dropNodeSpec, dropSnippet,
  onInsertIncVar, onApplySnippet,
  onPickKind, onPickLibrarySubgraph,
  onAddNode,
} = useNodeCreation({
  draft, activeGraph, selectedID,
  applyDraftMutation, syncFlowFromDraft, refreshSubgraphStore,
  autoCreateSubgraphForNewNode, toast,
})

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
const {
  query: nodeSearchQuery,
  results: nodeSearchResults,
  onPick: onNodeSearchPick,
} = useNodeSearch({ open: nodeSearchOpen, draft, activeGraph, selectedID })

// 4 个右键菜单 (Node / Multi / Edge / Pin) 状态 + 路由 → useContextMenuRouter (调用见下方).

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

// dropVar / dropNodeSpec / dropSnippet / onInsertIncVar / 等 — useNodeCreation 已抽
// (调用在上方 selectedID 之后)

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

function onOpenLibraryExplorer() {
  libraryExplorerOpen.value = true
}

// onPickKind / onPickLibrarySubgraph / onAddNode 等 — 已抽 useNodeCreation
// InlineContextMenu (右键画布空白 / pin drag-to-empty → 添加节点) 整块抽 useInlineMenu.
// markConnectSuccess: 给 view 的 onConnect wrap 用 — 通知 onVfConnectEnd 别开 menu.
const {
  inlineMenu,
  isValidVueFlowConnection,
  onVfConnectStart,
  onVfConnectEnd,
  onCanvasContextMenu,
  onPaneDoubleClick,
  onInlineMenuPick,
  markConnectSuccess,
} = useInlineMenu({ activeGraph, applyDraftMutation, syncFlowFromDraft })

// 4 menu router + actions + onFindRefsPick + onPromoteConfirm — useContextMenuRouter
// 调用在下方 onAlignSelected 之后 (依赖 onCopy/Paste/Fold/Align/AutoLayout 全部声明).

// 命令面板 commands 列表 → useCommandPalette (调用见下方 onValidate 后, 所有 action
// 必须先声明). open ref view 持有, 跟 useEditorHotkeys 共享.

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


// Vue Flow viewport API：屏幕坐标 → canvas 坐标（考虑 zoom/pan）。
const { project, getSelectedNodes, removeNodes, screenToFlowCoordinate, setCenter } = useVueFlow()

// Pin-aware snap engine (PS smart-guides) — 抽到 useSnapEngine.
// 拖拽位置必读 event.node.position, 不读 flowNodes (vmodel shallow sync 下 flowNodes 滞后).
const { snapGuides, onSnapNodeDrag, onSnapNodeDragStop } = useSnapEngine({
  sidebarPrefs,
  activeGraph,
  applyDraftMutation,
})

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

// ⚙ 容器设置 (name/hotkey/description/tags/runMode) 改完即落盘 —— 不必等保存整个蓝图。
// 容器热键靠后端 containers.update → emitChange → binder.Refresh 注册到热键中心;
// 只 mutate draft 不落盘 → 热键永远进不了注册中心 (「快捷键」页无容器分组)。
// 只 patch 元数据字段, 不带 graph/vars → Update 的 Unmarshal 只覆盖这几个键, 蓝图 draft 不受影响。
async function onSettingsSave(form: { name: string; hotkey: string; description: string; tags: string[]; runMode: string }) {
  applyDraftMutation((d) => Object.assign(d, form))
  if (!draft.value) return
  await backend.containers.update(draft.value.id, JSON.stringify({
    name: form.name,
    hotkey: form.hotkey,
    description: form.description,
    tags: form.tags,
    runMode: form.runMode,
  }))
}

// 全局快捷键: Ctrl+K palette / Ctrl+F search / Ctrl+S save / Ctrl+, settings /
// Ctrl+Z undo / Ctrl+Shift+Z(Y) redo / Tab toggle NodeExplorer.
// dedup 原 5 处 isTypingTarget. composable 内 onMounted/onUnmounted 自挂 keydown listener.
// 放 useEditorSave 之后 — onSave 在那里声明.
useEditorHotkeys({
  commandPaletteOpen, nodeSearchOpen, settingsOpen, nodeExplorerOpen,
  dirty, onSave, undo, redo,
})

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

// 4 个右键菜单路由 (Node / Multi / Edge / Pin) + action dispatchers + onFindRefsPick
// + onPromoteConfirm. promoteCtx / findRefsState 仍 view 持有 (modal state), 内部写回.
const {
  nodeMenu, multiMenu, edgeMenu, pinMenu,
  onNodeContextMenu, onSelectionContextMenu, onEdgeContextMenu, onCanvasContextMenuCapture,
  onNodeMenuAction, onMultiMenuAction, onEdgeMenuAction, onPinMenuAction,
  onFindRefsPick, onPromoteConfirm,
} = useContextMenuRouter({
  containerID, draft, activeGraph, selectedID,
  promoteCtx, findRefsState,
  applyDraftMutation, varMutations,
  onCopySelection, onPasteSelection, onFoldSelection,
  onAlignSelected, onAutoLayout,
  emitSaveSnippetIntent,
  toast,
})

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
      toast.add({ title: t('toast.subgraph_not_set'), color: 'warning' })
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

// Wrap onConnect: markConnectSuccess 让 useInlineMenu.onVfConnectEnd 退出 (不开 menu).
function onConnect(c: Parameters<typeof _onConnectBase>[0]) {
  markConnectSuccess()
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

// 画布内联 pin literal 编辑入口 — ContainerFlowNode inject 调用。
// activeGraph.value 指向 draft 当前层级图 (main/子图), mutate 它的 node 即 mutate draft。
// 写回走 applyDraftMutation (单一 mutation 入口: dirty + history 200ms 合并 + syncFlowFromDraft)。
// edges 同源 activeGraph → 切子图时一起切, 连线判定不会用错图。
provide(ContainerCanvasApiKey, {
  setPinLiteral(nodeId: string, pin: string, value: unknown) {
    applyDraftMutation(() => {
      const n = activeGraph.value?.nodes.find((x) => x.id === nodeId)
      if (!n) return
      const literal = { ...((n.config?.literal as Record<string, unknown>) ?? {}), [pin]: value }
      n.config = { ...(n.config ?? {}), literal }
    })
  },
  edges: computed(() => activeGraph.value?.edges ?? []),
})

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

// Expr fusion — NodeInspector 通过 editorBus.requestExprFusion 触发, 这里 watch 处理.
import { useExprFusion } from '@/composables/containerEditor/useExprFusion'
import { useEditorBusStore } from '@/stores/editorBus'
const { fuse: fuseExpr } = useExprFusion({ activeGraph, syncFlowFromDraft })
const editorBus = useEditorBusStore()
watch(() => editorBus.pendingExprFusion, (req) => {
  if (!req) return
  const ok = fuseExpr(req.sourceID, req.targetID, req.targetPin)
  editorBus.clearExprFusion()
  if (ok) {
    selectedID.value = null
    toast.add({ title: t('toast.expr_fuse_ok'), color: 'success' })
  } else {
    toast.add({ title: t('toast.expr_fuse_failed'), color: 'warning' })
  }
})
onMounted(() => {
  tplStore.setContainer(containerID)
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
    toast.add({ title: t('toast.validate_failed'), description: String(e), color: 'error' })
    return
  }
  await backend.containers.run(draft.value.id)
  toast.add({
    title: t('toast.runqueue_added'),
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
    toast.add({ title: t('toast.validate_call_failed'), description: String(e), color: 'error' })
  }
}

// 命令面板 — 所有 action 已声明, 安全调用.
const { commands } = useCommandPalette({
  canUndo, canRedo, dirty, sidebarPrefs,
  nodeExplorerOpen, libraryExplorerOpen, settingsOpen, nodeSearchOpen,
  undo, redo,
  onCopySelection, onPasteSelection, onFoldSelection,
  onAlignSelected, onAutoLayout,
  onSave, onValidate, onTryRun, onStopRun, onAddVar,
})

async function onValidationPanelRun() {
  validationPanelOpen.value = false
  if (!draft.value) return
  await backend.containers.run(draft.value.id)
  toast.add({
    title: t('toast.runqueue_added'),
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
    toast.add({ title: t('toast.window_target_exists'), color: 'warning' })
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
    title: t('toast.window_target_added_title'),
    description: t('toast.window_target_added_desc'),
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
      title: t('toast.subgraph_recording_reset_warn'),
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

/* ---- Edge selected: vue-flow 默认 #555 在深 canvas 上看不清, override 走 primary + 加粗 ---- */
:deep(.vue-flow__edge.selected .vue-flow__edge-path),
:deep(.vue-flow__edge:focus .vue-flow__edge-path),
:deep(.vue-flow__edge:focus-visible .vue-flow__edge-path) {
  stroke: var(--ui-primary, #6366f1);
  stroke-width: 2.5;
}
</style>
