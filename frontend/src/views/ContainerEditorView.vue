<template>
  <!-- h-full 撑满父 <main>. overflow-hidden 防 minimap/canvas
       撑出来触发 <main> 的 overflow-auto 滚动条 — 编辑器自管 canvas 缩放, 不允许外层滚. -->
  <div class="flex flex-col h-full min-h-0 overflow-hidden bg-default text-default">
    <!-- Dirty 关闭确认 -->
    <UModal
      :open="confirmCloseOpen"
      :ui="{ content: 'sm:max-w-[460px]' }"
      @update:open="
        (v: boolean) => {
          if (!v) resolveDirty('cancel')
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
            <UButton variant="ghost" color="neutral" @click="resolveDirty('cancel')">{{
              t('editor.dirty.cancel')
            }}</UButton>
            <UButton
              class="ml-auto"
              color="error"
              icon="i-tabler-x"
              @click="resolveDirty('discard')"
              >{{ t('editor.dirty.discard') }}</UButton
            >
            <UButton color="primary" icon="i-tabler-check" @click="resolveDirty('save')">{{
              t('editor.dirty.save_and_close')
            }}</UButton>
          </div>
        </div>
      </template>
    </UModal>

    <div v-if="!draft" class="flex-1 flex items-center justify-center text-sm text-muted">
      {{ t('editor.header.loading') }}
    </div>

    <div v-else class="flex flex-col flex-1 min-h-0">
      <!-- Toolbar：左 [面包屑][撤销/重做]，中间空，右 [录制][运行状态][试运行/停止][保存][折叠 inspector]。
           加内容(节点库/子图库)在左侧 rail。 -->
      <ContainerEditorToolbar
        :is-recording="recordStore.isRecording || recordStore.isPaused"
        :recording-target-name="recordingTargetName"
        :countdown-sec="countdownSec"
        :exec-store-running="execStore.running"
        :running-node-kind="execStore.currentNodeKind ?? undefined"
        :running-node-label="runningNodeLabel"
        :debug-active="execStore.debugActive"
        :debug-can-step="execStore.debugCanStep"
        :debug-can-continue="execStore.debugCanContinue"
        :debug-can-pause="execStore.debugCanPause"
        :debug-running-node-kind="
          execStore.debugRunningNodeKind || execStore.debugNextNodeKind || undefined
        "
        :debug-running-node-label="debugNodeLabel"
        :dirty="dirty"
        :save-flash="saveFlash"
        :can-undo="canUndo"
        :can-redo="canRedo"
        :snap-enabled="sidebarPrefs.snapEnabled"
        :edge-style="sidebarPrefs.edgeStyle"
        :root-label="draft?.name"
        :editor-path="editorStore.editorPath"
        :sg-label-fn="sgLabel"
        :node-count="activeGraph?.nodes?.length ?? 0"
        :var-count="declaredVars.length"
        :subgraph-count="editorStore.visibleSubgraphs.length"
        :overview-hotkey="draft?.hotkey ?? ''"
        @record="(mode) => startRecording(mode)"
        @stop-record="stopRecording"
        @cancel-countdown="startRecording('precise')"
        @try-run="onTryRun"
        @stop-run="onStopRun"
        @debug-start="onDebugStart"
        @debug-step="onDebugStep"
        @debug-continue="onDebugContinue"
        @debug-pause="onDebugPause"
        @debug-stop="onDebugStop"
        @save="onSave"
        @reload="onReload"
        @auto-layout="onAutoLayout"
        @validate="onValidate"
        @open-settings="settingsOpen = true"
        @open-js-console="jsConsoleOpen = true"
        @undo="undo"
        @redo="redo"
        @toggle-snap="sidebarPrefs.snapEnabled = !sidebarPrefs.snapEnabled"
        @set-edge-style="sidebarPrefs.edgeStyle = $event"
        @back-to-list="onBackToList"
        @open-help="helpModalOpen = true"
        @goto="editorStore.gotoPathIndex($event)"
      />

      <div
        ref="workspace"
        class="editor-workspace flex flex-1 min-h-0"
        :data-layout="workspaceLayout"
      >
        <!-- 左活动栏 rail (常驻细栏, VS Code 式): 变量/Snippets 开收停靠 drawer,
             节点库/子图库 点开 5xl modal。录制在 toolbar 右区 (主操作要显眼)。加节点也可走 Tab / 右键画布。 -->
        <nav
          class="shrink-0 w-11 border-r border-default flex flex-col items-center py-2 gap-1 bg-elevated/20"
        >
          <button
            v-for="item in leftRail"
            :key="item.key"
            type="button"
            class="size-8 flex items-center justify-center rounded-md transition-colors"
            :class="
              railActive(item)
                ? 'text-primary bg-primary/10'
                : 'text-dimmed hover:text-default hover:bg-elevated/60'
            "
            :title="item.title"
            @click="onRailClick(item)"
          >
            <UIcon :name="item.icon" class="size-5" />
          </button>
        </nav>

        <!-- 停靠区: 选中的面板滑出 (挤画布, 不盖)。面板各自带标题/搜索, 外壳不加 header。
             窄态(节点库/变量/Snippets) ↔ 宽态(资产网格) 由 dock 自适应宽度。 -->
        <ContainerEditorDock
          v-if="sidebarPrefs.leftDrawer"
          :wide="sidebarPrefs.leftDrawer === 'assets'"
          :overlay="dockOverlay"
        >
          <NodeLibraryPanel
            v-if="sidebarPrefs.leftDrawer === 'nodes'"
            @pick-kind="(k: string) => onPickKind(k)"
          />
          <VarsPanel
            v-else-if="sidebarPrefs.leftDrawer === 'vars'"
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
          <SnippetsPanel
            v-else-if="sidebarPrefs.leftDrawer === 'snippets'"
            @apply="onApplySnippet"
            @edit="onEditSnippet"
          />
          <AssetDockPanel
            v-else-if="sidebarPrefs.leftDrawer === 'assets'"
            v-model:tab="sidebarPrefs.assetTab"
            :template-pick-mode="!!assetPickRequest"
            :template-selected="assetPickRequest?.selected ?? []"
            @update:template-selected="updateAssetPick"
            @pick-subgraph="onPickLibrarySubgraph"
            @pick-clip="onPickLibraryClip"
          />
        </ContainerEditorDock>

        <!-- Canvas -->
        <div
          class="editor-canvas-shell flex-1 min-w-0 relative"
          @dragover.prevent="onCanvasDragOver"
          @drop.prevent="onCanvasDrop"
          @contextmenu.capture="onCanvasContextMenuCapture"
        >
          <!-- 浮动上下文条：选中节点时画布顶部居中浮现 (折叠 / 对齐 / 分布 / 删除) -->
          <CanvasContextBar
            v-if="selectedCount > 0"
            :selected-count="selectedCount"
            @fold="onFoldSelection"
            @align-selected="onAlignSelected"
            @delete-selected="onDeleteSelectedNodes"
          />
          <!-- 画布空态: 无节点时居中显「快捷开始」(取代原 Inspector 常占栏)。 -->
          <CanvasEmptyState v-if="canvasEmpty" />
          <ContainerDebugPanel @stop="onDebugStop" />
          <!-- 操作提示 -->
          <div
            class="absolute bottom-2 left-1/2 -translate-x-1/2 z-20 text-[10px] text-dimmed pointer-events-none bg-default/70 px-2 py-1 rounded"
          >
            {{ t('editor.canvas.hint') }}
          </div>
          <!-- pan-on-drag 只用中键(1), 不能含右键(2): vue-flow Pane.onContextMenu 见 panOnDrag 含 2
               会 preventDefault+return 不 emit paneContextMenu → 右键加节点菜单永远弹不出. -->
          <VueFlow
            v-model:nodes="flowNodes"
            v-model:edges="flowEdges"
            :node-types="nodeTypes as any"
            :default-edge-options="{ type: 'smoothstep' }"
            :delete-key-code="['Delete', 'Backspace']"
            :multi-selection-key-code="['Shift', 'Control', 'Meta']"
            :selection-key-code="true"
            :pan-on-drag="[1]"
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
            <!-- 网点/minimap mask/stroke = 画布景深专用色 (有意非 token), 与 .canvas-bg 渐变同族 -->
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

        <!-- Inspector 折叠/展开单一 toggle: 贴画布右边缘 (替原 toolbar ⊟)。
             collapsed 态(根图空选)无内容可切 → 不显。 -->
        <button
          v-if="inspectorMode !== 'collapsed'"
          type="button"
          class="shrink-0 w-5 self-stretch flex items-center justify-center border-l border-default text-dimmed hover:text-default hover:bg-elevated/40 transition-colors"
          :title="
            showInspector
              ? t('editor.toolbar.inspector_collapse')
              : t('editor.toolbar.inspector_expand')
          "
          @click="sidebarPrefs.inspectorCollapsed = !sidebarPrefs.inspectorCollapsed"
        >
          <UIcon
            :name="showInspector ? 'i-tabler-chevron-right' : 'i-tabler-chevron-left'"
            class="size-4"
          />
        </button>

        <SplitHandle
          v-show="showInspector"
          reverse
          :model-value="rightPane.width.value"
          @update:model-value="rightPane.setWidth"
          :min="200"
          :max="workspaceLayout === 'spacious' ? 480 : 320"
        />

        <!-- Right panel：选中节点显示 Inspector，否则显示引导空状态 -->
        <ContainerEditorInspector
          v-show="showInspector"
          :style="{ width: inspectorWidth + 'px' }"
          :selected-node="selectedNode"
          :in-subgraph="editorStore.editorPath.length > 0"
          :current-subgraph="currentSubgraph"
          :active-graph="activeGraph"
          :declared-vars="declaredVars"
          :all-subgraph-tags="allSubgraphTags"
          :all-subgraph-categories="allSubgraphCategories"
          @config-update="onConfigUpdate"
          @remove-switch-case="removeSwitchCase"
          @declare-var="onDeclareVar"
          @label-update="onLabelUpdate"
          @log-enabled-update="onLogEnabledUpdate"
          @delete-selected="onDeleteSelected"
          @subgraph-update="onSubgraphPropsUpdate"
          @subgraph-to-script="onSubgraphPanelToScript"
          @request-record="(e) => startRecording(e.mode, { replaceNodeID: e.replaceNodeID })"
        />
      </div>

      <!-- 底部「问题」条: 校验错误列表 (跳转/修复), 停靠编辑器列底、挤画布不盖。 -->
      <EditorProblemsBar
        v-model:expanded="problemsExpanded"
        :errors="validationErrors"
        :validated="validationRan"
        @jump="onJumpToErrorNode"
        @fix="onFixError"
        @fix-missing-win32-window-target="onFixMissingWin32WindowTarget"
        @run="onValidationPanelRun"
      />
    </div>

    <ContainerSettingsModal
      v-if="draft"
      v-model:open="settingsOpen"
      :initial="{
        name: draft.name,
        hotkey: draft.hotkey ?? '',
        description: draft.description ?? '',
        tags: draft.tags ?? [],
        category: draft.category ?? '',
        inputBackend: draft.inputBackend || 'postmessage',
        captureBackend: draft.captureBackend || 'auto',
        scaleTolerance: draft.scaleTolerance ?? 2.0,
      }"
      :all-tags="allSubgraphTags"
      :all-categories="allContainerCategories"
      @save="onSettingsSave"
    />

    <RecordingSaveModal
      v-if="pendingRecording"
      :pending="pendingRecording"
      :busy="pendingBusy"
      :replace-mode="pendingReplaceMode"
      @save="finalizePending"
      @discard="discardPending"
    />

    <DeleteVarConfirmModal
      v-if="deleteConfirm"
      :open="true"
      :var-name="deleteConfirm.name"
      :ref-i-ds="deleteConfirm.refIDs"
      @update:open="
        (v) => {
          if (!v) deleteConfirm = null
        }
      "
      @confirm="onDeleteConfirm"
    />

    <ContainerHelpModal v-model:open="helpModalOpen" />

    <SubgraphScriptPreviewModal
      v-model:open="toScriptState.open"
      :sg-label="toScriptState.sgLabel"
      :code="toScriptState.code"
      :unsupported="toScriptState.unsupported"
      @insert="insertConvertedScript"
    />

    <InlineContextMenu
      :open="inlineMenu.open"
      :position="inlineMenu.position"
      :pin-context="inlineMenu.pinContext"
      @update:open="
        (v) => {
          inlineMenu.open = v
        }
      "
      @pick="onInlineMenuPick"
    />

    <!-- 右键菜单 (节点 / 多选 / 边 / pin) -->
    <NodeContextMenu
      v-if="nodeMenu.node"
      :open="nodeMenu.open"
      :position="nodeMenu.position"
      :node="nodeMenu.node"
      @update:open="
        (v) => {
          nodeMenu.open = v
        }
      "
      @action="onNodeMenuAction"
    />
    <MultiNodeContextMenu
      :open="multiMenu.open"
      :position="multiMenu.position"
      :count="multiMenu.count"
      @update:open="
        (v) => {
          multiMenu.open = v
        }
      "
      @action="onMultiMenuAction"
    />
    <EdgeContextMenu
      v-if="edgeMenu.edge"
      :open="edgeMenu.open"
      :position="edgeMenu.position"
      :edge="edgeMenu.edge"
      @update:open="
        (v) => {
          edgeMenu.open = v
        }
      "
      @action="onEdgeMenuAction"
    />
    <PinContextMenu
      v-if="pinMenu.pin"
      :open="pinMenu.open"
      :position="pinMenu.position"
      :pin="pinMenu.pin"
      @update:open="
        (v) => {
          pinMenu.open = v
        }
      "
      @action="onPinMenuAction"
    />

    <!-- 命令面板 Ctrl+K -->
    <CommandPalette v-model:open="commandPaletteOpen" :commands="commands" />

    <YtConsoleModal v-model:open="jsConsoleOpen" :run="ytConsole.run" />

    <!-- Promote-to-Variable modal -->
    <PromoteToVarModal
      v-if="promoteCtx"
      :open="!!promoteCtx"
      :context="promoteCtx"
      :existing-var-names="(draft?.vars ?? []).map((v) => v.name)"
      @update:open="
        (v) => {
          if (!v) promoteCtx = null
        }
      "
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
      @update:open="
        (v) => {
          if (!v) saveSnippetState.open = false
        }
      "
      @saved="onSnippetSaved"
    />

    <!-- Find-References modal -->
    <FindReferencesModal
      v-if="findRefsState"
      :open="!!findRefsState"
      :var-name="findRefsState.varName"
      :refs="findRefsState.refs"
      @update:open="
        (v) => {
          if (!v) findRefsState = null
        }
      "
      @pick="onFindRefsPick"
    />

    <!-- Ctrl+F canvas node search (UE Blueprint "Find in Blueprint" equivalent) -->
    <NodeSearchModal
      :open="nodeSearchOpen"
      :results="nodeSearchResults"
      @update:open="
        (v) => {
          nodeSearchOpen = v
        }
      "
      @update:query="
        (q) => {
          nodeSearchQuery = q
        }
      "
      @pick="onNodeSearchPick"
    />
  </div>
</template>

<script setup lang="ts">
// 给 <keep-alive include="ContainerEditorView"> 用 — 切到 settings/help 等再回来,
// draft / canvas viewport / selection / dirty 全保留, 不重新 load.
defineOptions({ name: 'ContainerEditorView' })

import {
  computed,
  nextTick,
  onActivated,
  onBeforeUnmount,
  onMounted,
  provide,
  ref,
  useTemplateRef,
  watch,
} from 'vue'
import { useElementSize } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { ContainerCanvasApiKey } from '@/composables/containerEditor/pinLiterals'
import { useRoute, useRouter, onBeforeRouteLeave, onBeforeRouteUpdate } from 'vue-router'
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

import { backend, type GraphNode, type ValidationError, type VarDecl } from '@/lib/backend'
import { type VarType } from '@/lib/variableRef'
import { errorMessage } from '@/lib/invoke'
import { useRecordingStore } from '@/stores/recording'
import { useExecutionStore } from '@/stores/execution'
import { useContainersStore } from '@/stores/containers'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { useTemplatesStore } from '@/stores/templates'
import { useClipsStore } from '@/stores/clips'
import { useContainerDraft } from '@/composables/containerEditor/useContainerDraft'
import { useEditorPath } from '@/composables/containerEditor/useEditorPath'
import { useSubgraphLifecycle } from '@/composables/containerEditor/useSubgraphLifecycle'
import { useFolding } from '@/composables/containerEditor/useFolding'
import { useRecording } from '@/composables/containerEditor/useRecording'
import { useEditorSave } from '@/composables/containerEditor/useEditorSave'
import { useNodeClipboard } from '@/composables/containerEditor/useNodeClipboard'
import { useGraphLayout, type AlignMode } from '@/composables/containerEditor/useGraphLayout'
import { useGraphMutations } from '@/composables/containerEditor/useGraphMutations'
import { centerOnNode, SUBGRAPH_ENTRY_DEFAULT } from '@/composables/containerEditor/constants'
import { useSnapEngine } from '@/composables/containerEditor/useSnapEngine'
import { useEditorHotkeys } from '@/composables/containerEditor/useEditorHotkeys'
import { useNodeSearch } from '@/composables/containerEditor/useNodeSearch'
import { safeCoerceForFix } from '@/components/containers/inline/coerceLiteral'
import { useInlineMenu } from '@/composables/containerEditor/useInlineMenu'
import { useCommandPalette } from '@/composables/containerEditor/useCommandPalette'
import { useYtConsole } from '@/composables/containerEditor/useYtConsole'
import { useContextMenuRouter } from '@/composables/containerEditor/useContextMenuRouter'
import { useSubgraphToScript } from '@/composables/containerEditor/useSubgraphToScript'
import { useNodeCreation } from '@/composables/containerEditor/useNodeCreation'
import { useInsertPoint } from '@/composables/containerEditor/useInsertPoint'
import { newNodeID, genNodeID } from '@/composables/containerEditor/ids'
import ContainerFlowNode from '@/components/containers/ContainerFlowNode.vue'
import RecordingSaveModal from '@/components/containers/RecordingSaveModal.vue'
import CommentBoxNode from '@/components/containers/CommentBoxNode.vue'
import ContainerEditorToolbar from '@/components/containers/ContainerEditorToolbar.vue'
import CanvasContextBar from '@/components/containers/CanvasContextBar.vue'
import CanvasEmptyState from '@/components/containers/CanvasEmptyState.vue'
import ContainerEditorInspector from '@/components/containers/ContainerEditorInspector.vue'
import ContainerDebugPanel from '@/components/containers/ContainerDebugPanel.vue'
import EditorProblemsBar from '@/components/containers/EditorProblemsBar.vue'
import { useSidebarPrefs } from '@/composables/editor/useSidebarPrefs'
import { resolveInspectorMode } from '@/composables/editor/inspectorMode'
import { resolveEditorWorkspaceLayout } from '@/composables/editor/workspaceLayout'
import { useAssetPicker } from '@/composables/editor/useAssetPicker'
import { useVarMutations } from '@/composables/containerEditor/useVarMutations'
import SnippetsPanel from '@/components/snippets/SnippetsPanel.vue'
import SaveSnippetDrawer from '@/components/snippets/SaveSnippetDrawer.vue'
import VarsPanel from '@/components/containers/sidebar/VarsPanel.vue'
import ContainerSettingsModal from '@/components/containers/ContainerSettingsModal.vue'
import DeleteVarConfirmModal from '@/components/containers/sidebar/DeleteVarConfirmModal.vue'
import ContainerEditorDock from '@/components/containers/dock/ContainerEditorDock.vue'
import NodeLibraryPanel from '@/components/containers/dock/NodeLibraryPanel.vue'
import AssetDockPanel from '@/components/containers/dock/AssetDockPanel.vue'
import ContainerHelpModal from '@/components/containers/ContainerHelpModal.vue'
import InlineContextMenu from '@/components/containers/InlineContextMenu.vue'
import SubgraphScriptPreviewModal from '@/components/containers/SubgraphScriptPreviewModal.vue'
import NodeContextMenu from '@/components/containers/menus/NodeContextMenu.vue'
import MultiNodeContextMenu from '@/components/containers/menus/MultiNodeContextMenu.vue'
import EdgeContextMenu from '@/components/containers/menus/EdgeContextMenu.vue'
import PinContextMenu from '@/components/containers/menus/PinContextMenu.vue'
import CommandPalette from '@/components/containers/CommandPalette.vue'
import YtConsoleModal from '@/components/containers/YtConsoleModal.vue'
import PromoteToVarModal, {
  type PromoteContext,
} from '@/components/containers/PromoteToVarModal.vue'
import FindReferencesModal, { type RefEntry } from '@/components/containers/FindReferencesModal.vue'
import NodeSearchModal from '@/components/containers/NodeSearchModal.vue'
import SnapGuideOverlay from '@/components/containers/SnapGuideOverlay.vue'
import { useSnippetsStore, eventToShortcutKey, type Snippet } from '@/stores/snippets'
import { KIND_DEFAULTS, KIND_LABEL_ZH, PIN_SPECS } from '@/components/containers/pinSpec'
import { markRaw } from 'vue'
import { readDragPayload } from '@/composables/editor/useEditorDragDrop'
import SplitHandle from '@/components/common/SplitHandle.vue'
import { useSplitpane } from '@/composables/useSplitpane'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const toast = useToast()
const { confirm } = useConfirm()
const recordStore = useRecordingStore()
const execStore = useExecutionStore()
const containersStore = useContainersStore()
const tplStore = useTemplatesStore()
const clipsStore = useClipsStore()

const editorStore = useContainerEditorStore()

const runningNodeLabel = computed(() => {
  const k = execStore.currentNodeKind
  if (!k) return ''
  const key = KIND_LABEL_ZH[k]
  return key ? t(key) : k
})

const debugNodeLabel = computed(() => {
  const k =
    execStore.debugRunningNodeKind || execStore.debugNextNodeKind || execStore.debugCurrentNodeKind
  if (!k) return ''
  const key = KIND_LABEL_ZH[k]
  return key ? t(key) : k
})

function displayNodeLabel(node: GraphNode): string {
  if (node.label) return node.label
  const key = KIND_LABEL_ZH[node.kind]
  return key ? t(key) : node.kind
}

async function onStopRun() {
  await containersStore.stopAll()
}

async function startDebug(startNodeID = '', startNodeLabel = '') {
  if (!draft.value) return
  if (dirty.value) {
    toast.add({
      title: t('toast.debug_save_first'),
      color: 'warning',
      icon: 'i-tabler-device-floppy',
    })
    return
  }
  if (execStore.running) {
    toast.add({ title: t('toast.debug_run_busy'), color: 'warning', icon: 'i-tabler-player-play' })
    return
  }
  if (startNodeID) {
    const ok = await confirm({
      title: t('editor.debug.confirm_from_here_title'),
      description: t('editor.debug.confirm_from_here_desc', {
        node: startNodeLabel || startNodeID,
      }),
      color: 'warning',
      confirmText: t('editor.debug.confirm_from_here_action'),
      cancelText: t('common.cancel'),
    })
    if (ok !== true) return
  }
  const state = await backend.containers.debugStart(
    draft.value.id,
    startNodeID ? { startNodeId: startNodeID } : {},
  )
  if (!state) return
  execStore.applyDebugState(state)
}

function applyDebugCommandState(state: Awaited<ReturnType<typeof backend.containers.debugStep>>) {
  if (state) {
    execStore.applyDebugState(state)
  } else {
    execStore.clearDebugState()
  }
}

async function onDebugStart() {
  await startDebug()
}

async function onDebugStep() {
  const sid = execStore.debugSessionID
  if (!sid) return
  const state = await backend.containers.debugStep(sid)
  applyDebugCommandState(state)
}

async function onDebugContinue() {
  const sid = execStore.debugSessionID
  if (!sid) return
  const state = await backend.containers.debugContinue(sid)
  applyDebugCommandState(state)
}

async function onDebugPause() {
  const sid = execStore.debugSessionID
  if (!sid) return
  const state = await backend.containers.debugPause(sid)
  applyDebugCommandState(state)
}

async function onDebugStop() {
  const sid = execStore.debugSessionID
  if (!sid) return
  const state = await backend.containers.debugStop(sid)
  applyDebugCommandState(state)
}

async function resyncDebugState() {
  const sid = execStore.debugSessionID
  if (!sid) return
  const state = await backend.containers.debugState(sid)
  applyDebugCommandState(state)
}

// 工具栏「重载」: 从磁盘重读当前容器 (MCP / 外部改盘后同步)。
// 脏 (有未保存改动) → 先确认; 干净 → 直接重载。重载强制回主图根 (见 useContainerDraft.reload)。
async function onReload() {
  if (dirty.value) {
    const ok = await confirm({
      title: t('editor.reload.title'),
      description: t('editor.reload.desc'),
      color: 'warning',
      confirmText: t('editor.reload.confirm'),
      cancelText: t('editor.dirty.cancel'),
    })
    if (ok !== true) return
  }
  try {
    await reload()
  } catch (e) {
    toast.add({ title: t('editor.reload.failed'), description: errorMessage(e), color: 'error' })
  }
}

// toolbar '返回列表' 入口. ContainersView mount 时 clearLastEditing →
// 之后 sidebar '容器' 不再跳回该编辑器, 回归列表行为.
function onBackToList() {
  router.push('/containers')
}

// 路由 /containers/:id/edit 用 params.id; query.id 仅作兜底.
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
  applyBulkMutation,
  undo,
  redo,
  canUndo,
  canRedo,
  reload,
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

// 离开本编辑器的统一守卫. 两个钩子都挂:
//  - onBeforeRouteLeave: 离开编辑器路由 (返回列表 / settings 等)
//  - onBeforeRouteUpdate: 切到另一个容器 (/containers/A/edit → /B/edit, 同路由改 param,
//    leave 不触发只触发 update) —— 不挂这个, 切容器就绕过 dirty/录制守卫, 把未保存改动留在
//    keep-alive 缓存实例里, 跨容器状态串 (孤儿边 / 拿错 Win32WindowTarget)。
async function guardLeaveEditor(): Promise<boolean> {
  // A3: 录制本容器进行中 → 确认. 留下 → 录完正常 autoConnect; 确认离开 → 放行.
  if (
    (recordStore.isRecording || recordStore.isPaused) &&
    recordStore.activeTargetContainerID === containerID
  ) {
    const ok = await confirm({
      title: t('recordComposable.leave_title'),
      description: t('recordComposable.leave_during_recording'),
      color: 'warning',
      confirmText: t('recordComposable.leave_confirm'),
      cancelText: t('common.cancel'),
    })
    if (ok !== true) return false
  }
  // 未保存改动 → 保存/丢弃/取消. 丢弃要 reload 真正回盘 (本实例可能被 keep-alive 缓存,
  // 光清 dirty 会让被丢弃的改动留在内存里, 切回来还在)。
  return await guardDirty({ reloadOnDiscard: true })
}
onBeforeRouteLeave(guardLeaveEditor)
onBeforeRouteUpdate(guardLeaveEditor)

// 子图 metadata 外部编辑 (NodeInspector / SubgraphPropsPanel) 改的是 store 里 sg 对象,
// useContainerDraft 的 deep watch 自动标 dirty — 之前的 window 总线桥接已删除.

const { autoCreateSubgraphForNewNode } = useSubgraphLifecycle({
  draft,
  activeGraph,
  syncFlowFromDraft,
  refreshSubgraphStore,
})

const selectedID = ref<string | null>(null)

// 节点创建 pipeline (drop / picker / programmatic add) — 9 处 GraphNode push 散落抽这里.
const {
  dropVar,
  dropNodeSpec,
  dropSnippet,
  onInsertIncVar,
  onApplySnippet,
  onPickKind,
  onPickLibrarySubgraph,
  onPickLibraryClip,
  addNode,
} = useNodeCreation({
  draft,
  activeGraph,
  selectedID,
  applyDraftMutation,
  syncFlowFromDraft,
  refreshSubgraphStore,
  autoCreateSubgraphForNewNode,
  toast,
})

// 子图一键转脚本 (右键菜单 / 子图属性面板 → 预览 modal → 复制/插入 Script 节点)
const {
  state: toScriptState,
  convert: convertSubgraphToScript,
  convertFromNode: convertSubgraphNodeToScript,
  insert: insertConvertedScript,
} = useSubgraphToScript({ draft, addNode, toast })

// 面板入口转的是「当前正在编辑的子图」— currentSubgraph 已是含完整 graph 的子图对象。
function onSubgraphPanelToScript() {
  if (currentSubgraph.value) convertSubgraphToScript(currentSubgraph.value, null)
}

// 折叠侧栏：持久化到 localStorage via useSidebarPrefs
const { prefs: sidebarPrefs } = useSidebarPrefs()
const workspace = useTemplateRef<HTMLElement>('workspace')
const { width: workspaceWidth } = useElementSize(workspace)
const workspaceLayout = computed(() => resolveEditorWorkspaceLayout(workspaceWidth.value))
const dockOverlay = computed(() => workspaceLayout.value !== 'spacious')
// 左活动栏 rail: 4 个停靠面板 (节点库/变量/Snippets/资产), 点同图标收回。录制在 toolbar 右区。
type DockPanel = 'nodes' | 'vars' | 'snippets' | 'assets'
const leftRail = [
  { key: 'nodes' as const, icon: 'i-tabler-grid-dots', title: t('editor.toolbar.node_explorer') },
  { key: 'vars' as const, icon: 'i-tabler-variable', title: t('var.title') },
  { key: 'snippets' as const, icon: 'i-tabler-bookmarks', title: 'Snippets' },
  { key: 'assets' as const, icon: 'i-tabler-stack-2', title: t('editor.dock.assets') },
]
const {
  request: assetPickRequest,
  updateSelection: updateAssetPick,
  cancel: cancelAssetPick,
} = useAssetPicker()
function toggleDock(key: DockPanel) {
  const next = sidebarPrefs.value.leftDrawer === key ? null : key
  // 离开资产 tab → 取消可能挂着的字段 pick 上下文
  if (sidebarPrefs.value.leftDrawer === 'assets' && next !== 'assets') cancelAssetPick()
  sidebarPrefs.value.leftDrawer = next
}
function onRailClick(item: (typeof leftRail)[number]) {
  toggleDock(item.key)
}
function railActive(item: (typeof leftRail)[number]): boolean {
  return sidebarPrefs.value.leftDrawer === item.key
}
// 节点字段 (TemplatePickerField) 发起选模板 → 自动开停靠区资产·模板 tab (pick 模式).
watch(assetPickRequest, (req) => {
  if (req) {
    sidebarPrefs.value.leftDrawer = 'assets'
    sidebarPrefs.value.assetTab = 'templates'
  }
})
// 切/取消选中节点 → 丢弃挂着的 pick 上下文 (别把上个节点的指派写到新节点).
watch(selectedID, () => {
  if (assetPickRequest.value) cancelAssetPick()
})
const rightPane = useSplitpane('editor.splitpane.right', { default: 320, min: 200, max: 480 })
const inspectorWidth = computed(() =>
  workspaceLayout.value === 'spacious'
    ? rightPane.width.value
    : Math.min(rightPane.width.value, 320),
)
const settingsOpen = ref(false)
const helpModalOpen = ref(false)

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
    while (vars.some((v) => v.name === `v${n}`)) n++
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
    const v = (d.vars ?? []).find((x) => x.name === name)
    if (!v) return
    if (field === 'type') {
      v.type = value as VarDecl['type']
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

function onSnippetSaved() {
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
  const pos = screenPointToFlow(lastMousePos.value)
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
  // MIME-based dispatch via useEditorDragDrop (var / node-spec / snippet / library-subgraph / clip).
  const payload = readDragPayload(e)
  if (payload) {
    const pos = screenPointToFlow({ x: e.clientX, y: e.clientY })
    switch (payload.type) {
      case 'var':
        return dropVar(payload, pos)
      case 'node-spec':
        return dropNodeSpec(payload, pos)
      case 'snippet':
        return dropSnippet(payload, pos)
      case 'library-subgraph':
        return void onPickLibrarySubgraph(payload.id, pos)
      case 'clip':
        return void onPickLibraryClip(payload.id, pos)
    }
  }
}

// Real usage count — sum across all vars (derive from draft, reactive)
const totalVarUsageCount = computed(() => {
  if (!draft.value) return 0
  return (draft.value.vars ?? []).reduce((sum, v) => sum + varMutations.countUsage(v.name), 0)
})

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
// 加新 kind 只需在 pinSpec.ts 里加一条 PIN_SPECS 即可——nodeTypes / NodeExplorerModal / FlowNode 自动响应，避免漏注册。
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

// Inspector 三态 (spec B §3): 选中节点→node / 子图内空选→subgraph / 根图空选→collapsed。
const inspectorMode = computed(() =>
  resolveInspectorMode({
    hasSelectedNode: !!selectedNode.value,
    inSubgraph: editorStore.editorPath.length > 0,
  }),
)
// 显示 = 有内容(非 collapsed 态) 且 用户没手动折叠。
const showInspector = computed(
  () => inspectorMode.value !== 'collapsed' && !sidebarPrefs.value.inspectorCollapsed,
)
// 画布是否空(驱动画布空态「快捷开始」)。
const canvasEmpty = computed(() => (activeGraph.value?.nodes?.length ?? 0) === 0)

const declaredVars = computed<{ name: string; type: VarType }[]>(() =>
  (draft.value?.vars ?? []).map((v) => ({ name: v.name, type: v.type as VarType })),
)

function onDeclareVar(a: { name: string; type: VarType; default: unknown }) {
  applyDraftMutation(() => varMutations.addVar({ name: a.name, type: a.type, default: a.default }))
}

// recording 状态 — 三态在 toolbar 内部判断 (isRecording / countdownSec / idle); 这里只暴露
// recordStore.isRecording 给 RecordingOverlay (countdownSec 通过 useRecording 返回).

// Minimap 节点颜色 — 按 kind 取调色板 border 色对应的 hex (border class 都派生自 PALETTE, 反查即得)
import { KIND_VISUAL } from '@/components/containers/pinSpec'
import { PALETTE } from '@/components/containers/visualRegistry'
const BORDER_CLASS_HEX: Record<string, string> = Object.fromEntries(
  Object.values(PALETTE).map((p) => [p.border, p.hex]),
)
function miniNodeColor(node: any): string {
  const k = node?.data?.kind ?? ''
  const v = KIND_VISUAL[k]
  return BORDER_CLASS_HEX[v?.border ?? ''] ?? PALETTE.zinc.hex
}

// Vue Flow viewport API：屏幕坐标 → canvas 坐标（考虑 zoom/pan）。
const {
  getSelectedNodes,
  removeNodes,
  removeSelectedNodes,
  setCenter,
  getViewport,
  setViewport,
  fitView,
  findNode,
  addSelectedNodes,
} = useVueFlow()
// 节点插入落点统一来源: 视口中心 (录制/库插入/picker) + 指针位置 (拖放/快捷键)。详见 useInsertPoint。
const { viewportCenterForNode, screenPointToFlow } = useInsertPoint()

// 视口缓存: 切图层级时存当前相机、取目标层级相机 (无缓存→fitView)。否则主图↔子图共享一个
// vue-flow 相机, 在内容很远的子图 pan 远了切回主图会停在那 (见 incident: subgraph-viewport-not-cached)。
function graphLevelKey(path: string[]): string {
  return path.length ? path[path.length - 1] : editorStore.MAIN_GRAPH_KEY
}
// 某层级的"起始节点"坐标: 子图 = 入口 marker (sg.entry), 主图 = 唯一的 Start 节点。
// 首次进入 (无缓存视口) 时聚焦到它 —— 内容很大的图 fitView 会缩得很小, 落在入口更可用。
function startNodeOf(path: string[]): { x: number; y: number } | null {
  if (path.length > 0) {
    const e = editorStore.subgraphById(path[path.length - 1])?.entry
    return e?.nodeID
      ? { x: e.x ?? SUBGRAPH_ENTRY_DEFAULT.x, y: e.y ?? SUBGRAPH_ENTRY_DEFAULT.y }
      : null
  }
  const start = activeGraph.value?.nodes.find((n) => n.kind === 'Start')
  return start ? { x: start.x, y: start.y } : null
}
watch(
  () => editorStore.editorPathFor(containerID),
  (newPath, oldPath) => {
    // 旧层级相机同步存下 (此刻相机还停在旧图 — 节点重渲染不动相机)。
    if (oldPath) editorStore.saveViewport(containerID, graphLevelKey(oldPath), getViewport())
    const saved = editorStore.getSavedViewport(containerID, graphLevelKey(newPath))
    // 等新图节点渲染完再定位 (fitView 依赖节点已 mount)。
    nextTick(() => {
      if (saved) {
        setViewport(saved) // 回到上次离开该层级时的相机
        return
      }
      const start = startNodeOf(newPath) // 首次进入 → 聚焦起始节点 (入口 / Start)
      if (start) centerOnNode(setCenter, start, { duration: 0, zoom: 1 })
      else fitView() // 无起始节点兜底
    })
  },
)

// Pin-aware snap engine (PS smart-guides) — 抽到 useSnapEngine.
// 拖拽位置必读 event.node.position, 不读 flowNodes (vmodel shallow sync 下 flowNodes 滞后).
const { snapGuides, onSnapNodeDrag, onSnapNodeDragStop } = useSnapEngine({
  sidebarPrefs,
  activeGraph,
  applyDraftMutation,
})

function onCanvasDragOver(e: DragEvent) {
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'copy'
}

// 折叠选中节点为新子图
const { onFoldSelection } = useFolding({
  draft,
  activeGraph,
  refreshSubgraphStore,
  syncFlowFromDraft,
  getSelectedNodes,
  toast,
})

// 校验问题条状态 (检查按钮 / 保存失败 / 试运行前置校验 共用)。
// problemsExpanded = 底条展开; validationRan = 是否跑过校验 (区分「通过」与「没查」)。
const problemsExpanded = ref(false)
const validationRan = ref(false)
const validationErrors = ref<ValidationError[]>([])

// 保存 (子图带 rev 乐观锁). 提前到 useRecording 之前: 录制完成自动 save 需要 onSave.
const { onSave, saveFlash, staleSubgraphs, reloadStaleSubgraphs, dismissStaleSubgraphs } =
  useEditorSave({
    containerID,
    draft,
    dirty,
    toast,
    // 主图保存因校验失败 → 灌进问题面板 (逐条可跳转 / 一键修复), 取代干巴巴 toast。
    onValidationErrors: (errs) => {
      validationErrors.value = errs
      validationRan.value = true
      problemsExpanded.value = true
    },
  })

// 乐观锁被拒 → "盘上已有更新, 重载?" — 重载取盘上版本丢本地这些子图的修改; 取消保留本地继续改。
watch(staleSubgraphs, async (ids) => {
  if (ids.length === 0) return
  const ok = await confirm({
    title: t('editorSave.stale_title'),
    description: t('editorSave.stale_desc', { n: ids.length }),
    color: 'warning',
    confirmText: t('editorSave.stale_reload'),
    cancelText: t('editorSave.stale_keep'),
  })
  if (ok === true) {
    await reloadStaleSubgraphs()
    syncFlowFromDraft()
  } else {
    dismissStaleSubgraphs()
  }
})

// ⚙ 容器设置 (name/hotkey/description/tags + 输入/截图后端/缩放容差) 改完即落盘 —— 不必等保存整个蓝图。
// 容器热键靠后端 containers.update → emitChange → binder.Refresh 注册到热键中心;
// 只 mutate draft 不落盘 → 热键永远进不了注册中心 (「快捷键」页无容器分组)。
// 只 patch 元数据字段, 不带 graph/vars → Update 的 Unmarshal 只覆盖这几个键, 蓝图 draft 不受影响。
async function onSettingsSave(form: {
  name: string
  hotkey: string
  description: string
  tags: string[]
  category: string
  inputBackend: string
  captureBackend: string
  scaleTolerance: number
}) {
  applyDraftMutation((d) => Object.assign(d, form))
  if (!draft.value) return
  await backend.containers.update(
    draft.value.id,
    JSON.stringify({
      name: form.name,
      hotkey: form.hotkey,
      description: form.description,
      tags: form.tags,
      category: form.category,
      inputBackend: form.inputBackend,
      captureBackend: form.captureBackend,
      scaleTolerance: form.scaleTolerance,
    }),
  )
}

// 全局快捷键: Ctrl+K palette / Ctrl+F search / Ctrl+S save / Ctrl+, settings /
// Ctrl+Z undo / Ctrl+Shift+Z(Y) redo / Tab toggle NodeExplorer.
// dedup 原 5 处 isTypingTarget. composable 内 onMounted/onUnmounted 自挂 keydown listener.
// 放 useEditorSave 之后 — onSave 在那里声明.
useEditorHotkeys({
  commandPaletteOpen,
  nodeSearchOpen,
  settingsOpen,
  dirty,
  onSave,
  undo,
  redo,
  togglePalette: () => toggleDock('vars'),
  toggleInspector: () => {
    sidebarPrefs.value.inspectorCollapsed = !sidebarPrefs.value.inspectorCollapsed
  },
  isNodeLibraryOpen: () => sidebarPrefs.value.leftDrawer === 'nodes',
  toggleNodeLibrary: () => toggleDock('nodes'),
})

// 录制产物落下后选中它 (属性栏 + 画布视觉). syncFlowFromDraft 后 flow 节点要等一拍才在册.
async function selectRecordedNode(id: string) {
  selectedID.value = id
  await nextTick()
  const n = findNode(id)
  if (!n) return
  const cur = getSelectedNodes.value
  if (cur.length) removeSelectedNodes(cur)
  addSelectedNodes([n])
}

// 录制流程: 产物落在当前视口中心、不自动连线、落下即选中 (用户自己接线). 简易产物双击可进编辑.
const {
  startRecording,
  stopRecording,
  countdownSec,
  pendingRecording,
  pendingBusy,
  pendingReplaceMode,
  finalizePending,
  discardPending,
} = useRecording({
  draft,
  activeGraph,
  syncFlowFromDraft,
  refreshSubgraphStore,
  saveDraft: onSave,
  dropPoint: viewportCenterForNode,
  selectNode: selectRecordedNode,
  toast,
})

// 节点剪贴板 (Ctrl+C/V) — 全局化后粘贴 Subgraph 节点 = 引用同一个全局子图
const { onCopySelection, onPasteSelection } = useNodeClipboard({
  draft,
  activeGraph,
  flowNodes,
  syncFlowFromDraft,
  refreshSubgraphStore,
  getSelectedNodes,
  genID: genNodeID,
  toast,
})

// 自动布局 (ELK) + 对齐
const { autoLayout, alignSelected } = useGraphLayout({
  activeGraph,
  getSelectedNodes,
  syncFlowFromDraft,
  dirty,
  toast,
  applyDraftMutation,
})
async function onAutoLayout(direction: 'LR' | 'TB') {
  await autoLayout(direction)
}
function onAlignSelected(mode: AlignMode) {
  alignSelected(mode)
}

// 4 个右键菜单路由 (Node / Multi / Edge / Pin) + action dispatchers + onFindRefsPick
// + onPromoteConfirm. promoteCtx / findRefsState 仍 view 持有 (modal state), 内部写回.
const {
  nodeMenu,
  multiMenu,
  edgeMenu,
  pinMenu,
  onNodeContextMenu,
  onSelectionContextMenu,
  onEdgeContextMenu,
  onCanvasContextMenuCapture,
  onNodeMenuAction,
  onMultiMenuAction,
  onEdgeMenuAction,
  onPinMenuAction,
  onFindRefsPick,
  onPromoteConfirm,
} = useContextMenuRouter({
  draft,
  activeGraph,
  selectedID,
  promoteCtx,
  findRefsState,
  applyDraftMutation,
  varMutations,
  onCopySelection,
  onPasteSelection,
  onFoldSelection,
  onAlignSelected,
  onAutoLayout,
  emitSaveSnippetIntent,
  onSubgraphToScript: convertSubgraphNodeToScript,
  onHardDelete: (node: GraphNode) => hardDeleteNodes([node.id]),
  onDebugFromNode: (node: GraphNode) => {
    void startDebug(node.id, displayNodeLabel(node))
  },
  toast,
})

// 本地剪贴板：含节点 + edges + 被复制 Subgraph 节点绑定的子图 deep copy（1:1 联动用）
// v2：clipboard 在 activeGraph 层级生效（主图 / 子图层级都能 copy/paste）
// clipboard / onCopySelection / onPasteSelection / Ctrl+C/V 监听 由 useNodeClipboard composable 提供（见 setup 顶部）

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

// selectedID → vue-flow 内部选中 的强同步 (清空方向)。
// 属性栏只认 selectedID, 但节点的视觉选中是 vue-flow 自己维护的 node.selected, 两者会漂移:
// 某条路径清了 selectedID 却没清 vue-flow 选中 → 节点仍高亮但属性栏空。此时再点该节点,
// 选择集没变 → 不发 selection-change; node-click 又因微小位移被浏览器吞掉 → selectedID 卡住 →
// 单击不显示属性 (用户被迫"点空白再点")。这里把残留的 vue-flow 选中显式清掉, 让下一次单击
// 永远是一次全新选择 → 必发 selection-change → 一击即显。
watch(selectedID, (id) => {
  if (id !== null) return
  const sel = getSelectedNodes.value
  if (sel.length) removeSelectedNodes(sel)
})

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
const {
  onNodesChange,
  onEdgeDoubleClick,
  onEdgesChange,
  removeSwitchCase,
  onConnect: _onConnectBase,
} = useGraphMutations({
  activeGraph,
  flowEdges,
  syncFlowFromDraft,
  applyDraftMutation,
})

// Wrap onConnect: markConnectSuccess 让 useInlineMenu.onVfConnectEnd 退出 (不开 menu).
function onConnect(c: Parameters<typeof _onConnectBase>[0]) {
  markConnectSuccess()
  _onConnectBase(c)
}

function onConfigUpdate(cfg: Record<string, any>) {
  if (!draft.value || !selectedNode.value) return
  const targetID = selectedNode.value.id
  applyDraftMutation(() => {
    const node = activeGraph.value?.nodes.find((candidate) => candidate.id === targetID)
    if (node) node.config = cfg
  })
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
      const literal = {
        ...(n.config?.literal as Record<string, unknown> | undefined),
        [pin]: value,
      }
      n.config = { ...n.config, literal }
    })
  },
  patchNodeLiteral(nodeId: string, patch: Record<string, unknown>) {
    applyDraftMutation(() => {
      const n = activeGraph.value?.nodes.find((x) => x.id === nodeId)
      if (!n) return
      const literal = { ...(n.config?.literal as Record<string, unknown> | undefined), ...patch }
      n.config = { ...n.config, literal }
    })
  },
  edges: computed(() => activeGraph.value?.edges ?? []),
})

function onLabelUpdate(newLabel: string) {
  if (!selectedNode.value) return
  const targetID = selectedNode.value.id
  applyDraftMutation(() => {
    const g = activeGraph.value
    if (!g) return
    const n = g.nodes.find((x) => x.id === targetID)
    if (!n) return
    const trimmed = newLabel.trim()
    if (trimmed) n.label = trimmed
    else delete n.label
  })
}

function onLogEnabledUpdate(enabled: boolean) {
  if (!selectedNode.value) return
  const targetID = selectedNode.value.id
  applyDraftMutation(() => {
    const g = activeGraph.value
    if (!g) return
    const n = g.nodes.find((x) => x.id === targetID)
    if (!n) return
    if (enabled) n.logEnabled = true
    else delete n.logEnabled
  })
}

// Expr fusion — NodeInspector 通过 editorBus.requestExprFusion 触发, 这里 watch 处理.
import { useExprFusion } from '@/composables/containerEditor/useExprFusion'
import { useEditorBusStore } from '@/stores/editorBus'
const { fuse: fuseExpr } = useExprFusion({ activeGraph, syncFlowFromDraft })
const editorBus = useEditorBusStore()
watch(
  () => editorBus.pendingExprFusion,
  (req) => {
    if (!req) return
    const ok = fuseExpr(req.sourceID, req.targetID, req.targetPin)
    editorBus.clearExprFusion()
    if (ok) {
      selectedID.value = null
    } else {
      toast.add({ title: t('toast.expr_fuse_failed'), color: 'warning' })
    }
  },
)
// tplStore.containerId 是全局单指针, capture()/openScreenPicker 靠它定位本容器目标窗口。
// keep-alive 缓存多个容器编辑器时, 切到已缓存容器只走 onActivated(onMounted 不再触发),
// 漏掉这里指针就停在上一个容器 → WaitTemplate 截图/校验拿错容器的 Win32WindowTarget(「没有异环窗口」)。
// 故 mount + 每次激活都重指, 跟 editorStore.markActive 同构(见 incident keepalive-singleton-subgraph-store-stale)。
onMounted(() => {
  tplStore.setContainer(containerID)
  void resyncDebugState()
  // clip:changed 全局订阅 (listen 幂等) + 首刷. clips 是全局资产、单例 store — 订阅一次即可,
  // 不在 onUnmounted 退订: keep-alive 缓存多个编辑器时, 某个 unmount 退订会害到其他缓存编辑器的刷新。
  clipsStore.listen()
  void clipsStore.refresh()
})
onActivated(() => {
  tplStore.setContainer(containerID)
  void resyncDebugState()
})

function onDeleteSelected() {
  if (!draft.value || !selectedID.value) return
  // 走 vue-flow removeNodes → 触发 onNodesChange(remove) → 统一处理 Subgraph cascade
  removeNodes([selectedID.value])
  selectedID.value = null
}

// 浮动上下文条「删除」：删全部选中节点 (多选)，走同一 removeNodes → cascade。
function onDeleteSelectedNodes() {
  if (!draft.value) return
  const ids = getSelectedNodes.value.map((n) => n.id)
  if (!ids.length) return
  removeNodes(ids)
  selectedID.value = null
}

// 彻底删除指定节点 + 其底层定义 (Subgraph→子图, PlayClip→clip), 带确认 + 引用计数.
// 由右键菜单「彻底删除」触发 (见 onNodeMenuAction 'hard-delete'). 普通删除走 vue-flow 只删引用.
// 无底层定义的节点退化成普通删除. 子图/clip 是全局共享资产, 删定义前查别处引用, 在确认框里写明.
async function hardDeleteNodes(ids: string[]) {
  const g = activeGraph.value
  if (!draft.value || !g || !ids.length) return

  // 分类: 有底层定义的 (子图/clip) vs 无 (普通节点, 退化成只删引用)
  const defs: { nodeID: string; kind: 'subgraph' | 'clip'; defID: string }[] = []
  const plainIDs: string[] = []
  for (const id of ids) {
    const node = (g.nodes as GraphNode[]).find((n) => n.id === id)
    if (!node) continue
    const sgID =
      node.kind === 'Subgraph' ? (node.config?.SubgraphID as string | undefined) : undefined
    const clipID =
      node.kind === 'PlayClip' ? (node.config?.ClipID as string | undefined) : undefined
    if (sgID) defs.push({ nodeID: id, kind: 'subgraph', defID: sgID })
    else if (clipID) defs.push({ nodeID: id, kind: 'clip', defID: clipID })
    else plainIDs.push(id)
  }

  // 没有可彻底删的定义 → 退化成普通删除 (跟裸 Delete 一致)
  if (!defs.length) {
    if (plainIDs.length) {
      removeNodes(plainIDs)
      selectedID.value = null
    }
    return
  }

  // 统计被"本次删除之外"的节点引用数 — 引用扫描会把当前这些节点也算进去, 按 nodeID 排除自身.
  const deleting = new Set(ids)
  let externalRefs = 0
  for (const d of defs) {
    try {
      const refs =
        d.kind === 'subgraph'
          ? await backend.subgraphs.referrers(d.defID)
          : await backend.assets.referrers(d.defID)
      for (const r of refs ?? []) {
        if (!deleting.has(r.nodeID)) externalRefs++
      }
    } catch {
      /* 查不到引用就当 0, 不阻断删除 */
    }
  }

  const yes = await confirm({
    title: t('editor.hard_delete.title'),
    description:
      externalRefs > 0
        ? t('editor.hard_delete.confirm_referenced', { n: defs.length, refs: externalRefs })
        : t('editor.hard_delete.confirm', { n: defs.length }),
    color: 'error',
    confirmText: t('common.delete'),
  })
  if (yes !== true) return

  // 1) 删图上节点 (含无定义的) — 走 removeNodes → onNodesChange + 自动保存
  removeNodes([...defs.map((d) => d.nodeID), ...plainIDs])
  selectedID.value = null

  // 2) 删底层定义
  let failed = 0
  for (const d of defs) {
    try {
      if (d.kind === 'subgraph') {
        const rev = editorStore.subgraphById(d.defID)?.rev ?? 0
        await backend.subgraphs.delete_(d.defID, rev)
      } else {
        await clipsStore.remove(d.defID)
      }
    } catch (e) {
      failed++
      console.warn('hard delete definition failed', d, e)
    }
  }
  try {
    await refreshSubgraphStore()
  } catch {
    /* ignore */
  }

  if (failed > 0) {
    toast.add({ title: t('editor.hard_delete.failed', { n: failed }), color: 'error' })
  } else {
    toast.add({ title: t('editor.hard_delete.done', { n: defs.length }), color: 'success' })
  }
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
      validationRan.value = true
      problemsExpanded.value = true
      return
    }
  } catch (e) {
    toast.add({ title: t('toast.validate_failed'), description: errorMessage(e), color: 'error' })
    return
  }
  await backend.containers.run(draft.value.id)
}

// "检查" 按钮: 主动跑 validate, 始终弹 panel (即使全通过也告知用户)
async function onValidate() {
  if (!draft.value || dirty.value) return
  try {
    const errs = (await backend.containers.validate(draft.value.id)) as ValidationError[]
    validationErrors.value = errs ?? []
    validationRan.value = true
    problemsExpanded.value = true
  } catch (e) {
    toast.add({
      title: t('toast.validate_call_failed'),
      description: errorMessage(e),
      color: 'error',
    })
  }
}

// 命令面板 — 所有 action 已声明, 安全调用.
const jsConsoleOpen = ref(false)
const ytConsole = useYtConsole({
  draft,
  applyBulkMutation,
  selectedIds: () => getSelectedNodes.value.map((n) => n.id),
  subgraphs: () => editorStore.subgraphList,
  subgraphById: (id) => editorStore.subgraphById(id),
  specPinsOf: (k) => PIN_SPECS[k]?.dataIn ?? {},
  defaultsOf: (k) => KIND_DEFAULTS[k] ?? {},
})
const { commands } = useCommandPalette({
  canUndo,
  canRedo,
  dirty,
  sidebarPrefs,
  settingsOpen,
  nodeSearchOpen,
  jsConsoleOpen,
  undo,
  redo,
  onCopySelection,
  onPasteSelection,
  onFoldSelection,
  onAlignSelected,
  onAutoLayout,
  onSave,
  onValidate,
  onTryRun,
  onStopRun,
  onAddVar,
})

async function onValidationPanelRun() {
  problemsExpanded.value = false
  if (!draft.value) return
  await backend.containers.run(draft.value.id)
}

// 一键修复 MISSING_WIN32_WINDOW_TARGET: 往主图 push 一个空 Win32WindowTarget 节点 (用户后续在
// NodeInspector 里填 match/runtime 或点 "捕获前台窗口"). 必须落主图, 不是 activeGraph.
function onFixMissingWin32WindowTarget() {
  if (!draft.value) return
  const mainGraph = draft.value.graph
  const defaults = KIND_DEFAULTS.Win32WindowTarget ?? {}
  const newNode: GraphNode = {
    id: newNodeID('Win32WindowTarget'),
    kind: 'Win32WindowTarget',
    x: 40,
    y: 40,
    config: JSON.parse(JSON.stringify(defaults)),
    createdAt: new Date().toISOString(),
  }
  mainGraph.nodes.push(newNode)
  syncFlowFromDraft()
  problemsExpanded.value = false
  toast.add({
    title: t('toast.win32_window_target_added_title'),
    description: t('toast.win32_window_target_added_desc'),
    color: 'success',
    icon: 'i-tabler-check',
  })
}

// locateNode: 按 id 在主图 + 所有子图里找节点, 返回 live 节点对象 (可改) + 所在子图 id (null=主图)。
// 不用 walkAllGraphs —— 它内部调 useI18n(), 在事件回调 (非 setup/渲染) 里调会抛, 跳转会静默挂掉。
function locateNode(id: string): { node: GraphNode; sgID: string | null } | null {
  if (!draft.value) return null
  const mainHit = draft.value.graph.nodes.find((n) => n.id === id)
  if (mainHit) return { node: mainHit, sgID: null }
  for (const sg of editorStore.subgraphList) {
    const n = sg.graph?.nodes.find((x) => x.id === id)
    if (n) return { node: n, sgID: sg.id }
  }
  return null
}

// 问题面板「跳转」: 进对应 (子) 图 + 选中 + 居中出错节点 (套 useNodeSearch.onPick 同款导航)。
async function onJumpToErrorNode(e: ValidationError) {
  if (!e.nodeId) return
  const loc = locateNode(e.nodeId)
  if (!loc) return
  problemsExpanded.value = false
  const curSg = editorStore.editorPath[editorStore.editorPath.length - 1] ?? null
  if (curSg !== loc.sgID) {
    editorStore.setPath(loc.sgID ? [loc.sgID] : [])
    await nextTick()
  }
  selectedID.value = e.nodeId
  const target = activeGraph.value?.nodes.find((n) => n.id === e.nodeId)
  if (target) {
    try {
      centerOnNode(setCenter, target as { x: number; y: number })
    } catch {}
  }
}

// 问题面板「修复」: 仅 LITERAL_TYPE_MISMATCH 且能安全 coerce (干净数字串→number / 真假串→bool 等) 时生效。
// 改 live literal + 标 dirty/touched, 乐观摘掉该条; 真正确认靠用户再保存/检查。含糊值不在此列 (按钮不显)。
function onFixError(e: ValidationError) {
  if (e.code !== 'LITERAL_TYPE_MISMATCH' || !e.nodeId) return
  const pin = e.params?.pin as string | undefined
  const expected = e.params?.expected as string | undefined
  if (!pin || !expected) return
  const loc = locateNode(e.nodeId)
  const lit = loc?.node.config?.literal as Record<string, unknown> | undefined
  if (!loc || !lit) return
  const fixed = safeCoerceForFix(lit[pin], expected)
  if (fixed === undefined) return
  lit[pin] = fixed
  if (loc.sgID) editorStore.touchSubgraph(containerID, loc.sgID)
  syncFlowFromDraft()
  validationErrors.value = validationErrors.value.filter((x) => x !== e)
}

// currentSubgraph 由 useEditorPath 提供

const allSubgraphTags = computed(() => {
  const set = new Set<string>()
  for (const sg of editorStore.visibleSubgraphs) {
    for (const t of sg.tags ?? []) set.add(t)
  }
  return [...set]
})

const allSubgraphCategories = computed(() => {
  const set = new Set<string>()
  for (const sg of editorStore.visibleSubgraphs) {
    if (sg.category) set.add(sg.category)
  }
  return [...set].sort()
})

const allContainerCategories = computed(() => {
  const set = new Set<string>()
  for (const c of containersStore.list) {
    if (c.category) set.add(c.category)
  }
  return [...set].sort()
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
  // 属性不在 sg.graph 里, activeGraph 深 watch 看不见 — 显式记改动归属本容器。
  editorStore.touchSubgraph(containerID, currentSubgraph.value.id)
  dirty.value = true
}

// ── Dirty 守卫: 单例确认弹窗 (保存/丢弃/取消), Promise 化供路由守卫用 ──
const confirmCloseOpen = ref(false)
let dirtyResolver: ((v: 'save' | 'discard' | 'cancel') => void) | null = null

function askDirty(): Promise<'save' | 'discard' | 'cancel'> {
  return new Promise((resolve) => {
    dirtyResolver = resolve
    confirmCloseOpen.value = true
  })
}
function resolveDirty(v: 'save' | 'discard' | 'cancel') {
  confirmCloseOpen.value = false
  dirtyResolver?.(v)
  dirtyResolver = null
}

// guardDirty: 干净(无未保存)直接放行; 否则弹窗等用户选。
// 返回 true=可继续 (已存/已弃), false=取消。reloadOnDiscard: 嵌入态 keep-alive 实例丢弃需回盘。
async function guardDirty({ reloadOnDiscard }: { reloadOnDiscard: boolean }): Promise<boolean> {
  if (!dirty.value) return true
  const choice = await askDirty()
  if (choice === 'cancel') return false
  if (choice === 'save') {
    await onSave()
    return !dirty.value // 保存失败 (校验错, dirty 仍 true) → 不放行
  }
  // discard
  if (reloadOnDiscard) {
    try {
      await reload()
    } catch {
      dirty.value = false
    }
  } else {
    dirty.value = false
  }
  return true
}
</script>

<style scoped>
/* 自定义节点容器（ContainerFlowNode 自己有 bg/border，这里不再覆盖） */

.editor-workspace {
  position: relative;
  isolation: isolate;
  container: editor-workspace / inline-size;
}

.editor-canvas-shell {
  container: editor-canvas / inline-size;
}

/* ---- Canvas 背景: 安静的翠绿倾向工作台，网点承担空间定位。 ---- */
.canvas-bg {
  background:
    radial-gradient(
      ellipse 70% 42% at 50% 0%,
      color-mix(in oklch, var(--ui-primary) 5%, transparent) 0%,
      transparent 72%
    ),
    linear-gradient(
      180deg,
      color-mix(in oklch, var(--ui-bg) 96%, var(--ui-primary)) 0%,
      color-mix(in oklch, var(--ui-bg) 98%, oklch(0.08 0.006 160)) 100%
    );
}
.canvas-bg::after {
  /* 轻微边缘收束，避免高亮或霓虹发光。 */
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: radial-gradient(ellipse at center, transparent 58%, rgba(0, 0, 0, 0.24) 100%);
  z-index: 0;
}
/* vue-flow 内部 viewport / pane / nodes 都 z-index > 0, vignette 不挡 */

/* ---- Controls (左下角缩放/fit) 深色 ---- */
:deep(.vue-flow__controls) {
  display: flex;
  flex-direction: column;
  background: color-mix(in oklab, var(--ui-bg) 85%, transparent);
  border: 1px solid color-mix(in oklab, var(--ui-border-accented) 80%, transparent);
  border-radius: 6px;
  overflow: hidden;
  box-shadow: 0 4px 10px rgba(0, 0, 0, 0.4);
}
:deep(.vue-flow__controls-button) {
  background: transparent;
  border: none;
  border-bottom: 1px solid color-mix(in oklab, var(--ui-border-accented) 60%, transparent);
  color: var(--ui-text-toned);
  width: 26px;
  height: 26px;
  fill: currentColor;
}
:deep(.vue-flow__controls-button:last-child) {
  border-bottom: none;
}
:deep(.vue-flow__controls-button:hover) {
  background: color-mix(in oklab, var(--ui-bg-accented) 60%, transparent);
  color: var(--ui-text-highlighted);
}
:deep(.vue-flow__controls-button svg) {
  fill: currentColor;
  max-width: 14px;
  max-height: 14px;
}

/* ---- MiniMap (右下角) 深色 ---- */
:deep(.vue-flow__minimap) {
  background: color-mix(in oklab, var(--ui-bg) 85%, transparent);
  border: 1px solid color-mix(in oklab, var(--ui-border-accented) 80%, transparent);
  border-radius: 6px;
  overflow: hidden;
}
:deep(.vue-flow__minimap-mask) {
  fill: rgba(9, 9, 11, 0.55); /* 画布景深专用 (有意非 token), 与 .canvas-bg 渐变同族 */
}

/* ---- Attribution 左下角 "Vue Flow" 水印淡化 ---- */
:deep(.vue-flow__attribution) {
  background: transparent;
  color: color-mix(in oklab, var(--ui-text-muted) 50%, transparent);
  font-size: 9px;
}

/* ---- Edge selected: 边的 stroke 色是 useContainerDraft 内联 style 设的 (data蓝/exec灰),
   内联样式优先级高于类选择器 → 必须 !important 才压得过, 否则点了边看不出高亮.
   只换 primary 色 + 取消虚线, 不改 stroke-width — 加粗会让箭头(markerUnits=strokeWidth)
   跟着放大, 用户只要高亮. ---- */
:deep(.vue-flow__edge.selected .vue-flow__edge-path),
:deep(.vue-flow__edge:focus .vue-flow__edge-path),
:deep(.vue-flow__edge:focus-visible .vue-flow__edge-path) {
  stroke: var(--ui-primary) !important;
  stroke-dasharray: none !important;
  filter: none;
}
</style>
