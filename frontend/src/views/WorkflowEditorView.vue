<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden bg-default">
    <div v-if="session.phase === 'loading'" class="flex flex-1 items-center justify-center px-8">
      <div class="w-full max-w-xl space-y-3" :aria-label="t('workflow.editor.loading')">
        <USkeleton class="h-10 w-2/3 rounded-lg" />
        <USkeleton class="h-72 w-full rounded-lg" />
      </div>
    </div>

    <div
      v-else-if="session.openFailure && !session.source"
      class="flex flex-1 items-center justify-center p-8"
    >
      <div class="max-w-lg rounded-lg border border-error/35 bg-error/10 p-5" role="alert">
        <h1 class="text-sm font-semibold text-error">
          {{ t('workflow.editor.open_failed') }}
        </h1>
        <p class="mt-2 text-xs leading-5 text-muted">{{ session.openFailure }}</p>
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
        :context="editorToolbarContext"
        @back="router.push('/workflows')"
        @command="handleEditorToolbarCommand"
      >
        <template #breadcrumbs>
          <template v-for="(graphId, index) in session.graphPath" :key="`${index}:${graphId}`">
            <UIcon name="i-tabler-chevron-right" class="size-3 shrink-0 text-dimmed" />
            <UButton
              :data-testid="`workflow-graph-breadcrumb-${graphId}`"
              class="max-w-36"
              color="neutral"
              variant="ghost"
              size="xs"
              :label="graphLabel(graphId)"
              @click="openGraphAt(index)"
            />
          </template>
        </template>
        <template #target>
          <UPopover mode="click" :ui="{ content: 'w-80 p-3' }">
            <UButton
              data-testid="workflow-target-default"
              icon="i-tabler-device-desktop"
              color="neutral"
              variant="ghost"
              size="xs"
              :aria-label="workflowDefaultTargetLabel"
              :title="workflowDefaultTargetLabel"
            >
              <span class="hidden max-w-40 truncate min-[1180px]:inline">
                {{ workflowDefaultTargetLabel }}
              </span>
            </UButton>
            <template #content>
              <div class="space-y-2">
                <p class="text-xs font-medium text-highlighted">
                  {{ t('workflow.target_default.label') }}
                </p>
                <AdaptiveSelect
                  :model-value="workflowDefaultTargetSlot"
                  :items="workflowAutomationTargetItems"
                  value-key="value"
                  label-key="label"
                  width-mode="fill"
                  :placeholder="t('workflow.target_default.placeholder')"
                  @update:model-value="setWorkflowDefaultTarget"
                />
                <UButton
                  v-if="workflowDefaultTargetSlot"
                  icon="i-tabler-x"
                  color="neutral"
                  variant="ghost"
                  size="xs"
                  :label="t('workflow.target_default.clear')"
                  @click="session.setTargetDefault('target', '')"
                />
              </div>
            </template>
          </UPopover>
        </template>
      </WorkflowEditorToolbar>

      <WorkflowMetadataDialog
        v-model:open="workflowSettingsOpen"
        :name="workflowMetadata.name"
        :description="workflowMetadata.description"
        :category="workflowMetadata.category"
        :tags="workflowMetadata.tags"
        :busy="workflowSettingsBusy"
        :error="workflowSettingsError"
        @submit="saveWorkflowSettings"
      />

      <div
        v-if="creationTemplate"
        class="flex flex-wrap items-center gap-2 border-b border-primary/25 bg-primary/5 px-4 py-2 text-xs text-muted"
        role="status"
      >
        <UIcon :name="creationTemplateIcon" class="size-4 text-primary" aria-hidden="true" />
        <span class="min-w-0 flex-1">
          {{ t(`workflow.template.${creationTemplate}.hint`) }}
        </span>
        <RouterLink
          class="font-medium text-primary hover:underline"
          to="/settings?section=automation"
        >
          {{ t('workflow.template.configure_targets') }}
        </RouterLink>
      </div>

      <div
        v-if="session.saveError"
        data-testid="workflow-save-error"
        class="flex items-center gap-2 border-b border-error/35 bg-error/10 px-4 py-2 text-xs text-error"
        role="alert"
      >
        <span class="min-w-0 flex-1">
          {{
            isRevisionConflict
              ? t('workflow.editor.revision_conflict')
              : t('workflow.editor.save_failed', { message: session.saveError })
          }}
        </span>
        <UButton
          v-if="isRevisionConflict"
          size="xs"
          color="error"
          variant="soft"
          :label="t('workflow.editor.reload_latest')"
          @click="reloadWorkflow"
        />
        <UButton
          v-else-if="session.saveErrorTarget"
          size="xs"
          color="error"
          variant="soft"
          :label="t('workflow.editor.locate_save_error')"
          @click="locateSaveError"
        />
        <UButton
          size="xs"
          color="error"
          variant="ghost"
          icon="i-tabler-x"
          :aria-label="t('common.close')"
          @click="session.dismissSaveError()"
        />
      </div>
      <div class="flex min-h-0 flex-1">
        <WorkflowWorkspaceRail
          :active-panel="workspacePanel"
          :open="workspaceSidebarOpen"
          @select="toggleWorkspacePanel"
        />

        <aside
          v-if="workspaceSidebarOpen"
          data-testid="workflow-workspace-sidebar"
          class="relative flex shrink-0 flex-col border-r border-default bg-default"
          :style="{ width: `${workspaceSidebarWidth}px` }"
        >
          <div
            role="separator"
            tabindex="0"
            :aria-label="t('workflow.sidebar.resize_workspace')"
            aria-orientation="vertical"
            class="absolute inset-y-0 right-0 z-30 w-1 cursor-col-resize transition-colors hover:bg-primary/50 focus-visible:bg-primary/70 focus-visible:outline-none"
            @pointerdown="startSidebarResize('workspace', $event)"
            @keydown.left.prevent="
              workspaceSidebarWidth = resizeWorkspaceSidebar(workspaceSidebarWidth, -16)
            "
            @keydown.right.prevent="
              workspaceSidebarWidth = resizeWorkspaceSidebar(workspaceSidebarWidth, 16)
            "
          />
          <WorkflowGraphManager
            v-if="workspacePanel === 'graphs'"
            :source="session.source"
            :current-graph-id="session.currentGraph?.id"
            :callable-graph-ids="callableGraphIds"
            :drag-format="GRAPH_CALL_DRAG_FORMAT"
            @open="openCalledGraph"
            @insert="addGraphCall"
            @create="openGraphDialog('create')"
            @rename="openGraphDialog('rename', $event)"
            @duplicate="duplicateGraphDefinition"
            @delete="deleteGraphDefinition"
            @delete-cascade="deleteGraphDefinitionCascade"
            @locate="locateGraphCall"
          />
          <WorkflowResourceDock
            v-else-if="workspaceResourcePanel"
            :kind="workspaceResourceKind"
            :source="session.source"
            :recording-phase="recording.state.phase"
            :locate-request="resourceLocateRequest"
            @start-recording="openRecordingStart"
            @capture-template="openTemplateCapture"
            @open-library="router.push('/assets')"
            @edit="openMacroEditor"
            @edit-workflow-resource="openWorkflowResourceEditor"
            @duplicate-workflow-resource="duplicateWorkflowResource"
            @use="useWorkspaceResource"
            @use-workflow="useWorkflowResource"
            @import-workflow-resource="importWorkflowResource"
            @update-workflow-resources="updateWorkflowResources"
            @remove-workflow-resources="removeWorkflowResources"
          />
          <WorkflowSnippetDock
            v-else-if="workspacePanel === 'snippets'"
            :drag-format="SNIPPET_DRAG_FORMAT"
            @use="useSnippet"
            @edit="editSnippet"
            @delete="deleteSnippet"
          />
        </aside>

        <div
          ref="canvasElement"
          data-testid="workflow-canvas"
          :data-graph-id="session.currentGraph?.id"
          class="relative min-w-0 flex-1 bg-elevated/15 transition-shadow"
          :class="nodeDragActive ? 'ring-1 ring-inset ring-primary/60' : ''"
          @pointerdown.capture="captureMarqueeSelection"
          @pointerenter="canvasPointerInside = true"
          @pointerleave="canvasPointerInside = false"
          @pointermove.capture="trackCanvasPointer"
          @wheel.capture="handleCanvasWheel"
          @dragover="continueNodeDrag"
          @dragleave.self="finishNodeDrag"
          @drop="dropNode"
        >
          <VueFlow
            :nodes="flowNodes"
            :edges="flowEdges"
            :delete-key-code="null"
            :selection-key-code="WORKFLOW_CANVAS_INTERACTION.selectionKeyCode"
            :multi-selection-key-code="WORKFLOW_CANVAS_INTERACTION.multiSelectionKeyCode"
            :pan-activation-key-code="WORKFLOW_CANVAS_INTERACTION.panActivationKeyCode"
            :pan-on-drag="WORKFLOW_CANVAS_INTERACTION.panOnDrag"
            :select-nodes-on-drag="WORKFLOW_CANVAS_INTERACTION.selectNodesOnDrag"
            :is-valid-connection="isValidConnection"
            fit-view-on-init
            :min-zoom="0.2"
            :max-zoom="2"
            class="workflow-flow"
            @connect="connect"
            @connect-start="startConnection"
            @connect-end="endConnection"
            @node-click="selectNode"
            @edge-click="selectEdge"
            @pane-click="handlePaneClick"
            @selection-end="finishMarqueeSelection"
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
                :diagnostic-severity="nodeDiagnosticSeverityById.get(slotProps.data.node.id)"
                :breakpoint="hasBreakpoint(session.currentGraph?.id ?? '', slotProps.data.node.id)"
                :debug-mode="debugModeActive"
                :debug-current="
                  isDebugCurrent(session.currentGraph?.id ?? '', slotProps.data.node.id)
                "
                :connected-input-ids="connectedInputIDs(slotProps.data.node.id)"
                :target-slot="targetSlotForNode(slotProps.data.node, slotProps.data.projection)"
                :selection-count="selectedNodeIds.size"
                @command="applyCommand"
                @context-open="selectNodeForContextMenu(slotProps.data.node.id)"
                @copy="copySelection"
                @cut="cutSelection"
                @duplicate="duplicateSelection"
                @collapse="collapseSelection"
                @toggle-disabled="
                  applyCommand({
                    kind: 'set-node-disabled',
                    nodeId: slotProps.data.node.id,
                    disabled: !slotProps.data.node.disabled,
                  })
                "
                @toggle-breakpoint="
                  toggleBreakpoint(session.currentGraph?.id ?? '', slotProps.data.node.id)
                "
                @save-snippet="openSnippetForNode(slotProps.data.node)"
                @remove="removeSelection"
              />
            </template>
            <template #node-graph-call="slotProps">
              <WorkflowGraphCall
                :call="slotProps.data.call"
                :graph="slotProps.data.graph"
                :selected="slotProps.selected"
                @open="openCalledGraph(slotProps.data.call.graphId)"
              />
            </template>
            <template #node-graph-boundary="slotProps">
              <WorkflowGraphBoundary :boundary="slotProps.data" />
            </template>
            <template #node-annotation="slotProps">
              <WorkflowAnnotation
                :annotation="slotProps.data.annotation"
                :selected="slotProps.selected"
                @update="updateAnnotation"
              />
            </template>
            <template #edge-reroute="slotProps">
              <WorkflowRerouteEdge
                :id="slotProps.id"
                :source-x="slotProps.sourceX"
                :source-y="slotProps.sourceY"
                :target-x="slotProps.targetX"
                :target-y="slotProps.targetY"
                :style="slotProps.style"
                :edge="slotProps.data.edge"
                @update="setEdgeReroutes(slotProps.data.edge, $event)"
              />
            </template>
            <Background :gap="20" :size="1" pattern-color="rgb(113 113 122 / 0.26)" />
            <Controls position="bottom-left" />
            <MiniMap
              v-if="minimapOpen"
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
            @collapse="collapseSelection"
            @remove="removeSelection"
          />
          <div
            data-testid="workflow-canvas-add-node-entry"
            class="absolute left-3 top-3 z-20 flex gap-0.5 rounded-lg border border-default bg-default/95 p-1 shadow-lg"
          >
            <UButton
              data-testid="workflow-canvas-add-node"
              icon="i-tabler-plus"
              color="neutral"
              variant="ghost"
              size="xs"
              :label="t('workflow.canvas.add_node')"
              @click="openQuickAddFromButton"
            />
            <UTooltip :text="t('workflow.graphs.add_comment')">
              <UButton
                data-testid="workflow-annotation-add"
                icon="i-tabler-note"
                color="neutral"
                variant="ghost"
                size="xs"
                :aria-label="t('workflow.graphs.add_comment')"
                @click="addComment"
              />
            </UTooltip>
          </div>
          <div
            class="absolute right-3 top-3 z-20 flex gap-1 rounded-lg border border-default bg-default/95 p-1 shadow-lg"
          >
            <template v-if="!selectedNodeIds.size && selectedEdgeId">
              <template v-if="selectedSourceEdge()">
                <UButton
                  data-testid="workflow-reroute-add"
                  icon="i-tabler-point"
                  color="neutral"
                  variant="ghost"
                  size="xs"
                  :label="t('workflow.reroute.add')"
                  @click="addEdgeReroute"
                />
                <UButton
                  icon="i-tabler-eraser"
                  color="neutral"
                  variant="ghost"
                  size="xs"
                  :aria-label="t('workflow.reroute.clear')"
                  @click="clearEdgeReroutes"
                />
              </template>
              <UButton
                icon="i-tabler-trash"
                color="error"
                variant="ghost"
                size="xs"
                :aria-label="t('common.delete')"
                @click="disconnectEdge(selectedEdgeId)"
              />
            </template>
            <template v-if="!selectedNodeIds.size">
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
            </template>
            <UButton
              v-if="session.activeRun"
              data-testid="workflow-clear-run-trace"
              icon="i-tabler-route-off"
              color="neutral"
              variant="ghost"
              size="xs"
              :disabled="runActive"
              :aria-label="t('workflow.canvas.clear_run_trace')"
              @click="clearRunTrace"
            />
            <UButton
              data-testid="workflow-minimap-toggle"
              icon="i-tabler-map-2"
              color="neutral"
              :variant="minimapOpen ? 'soft' : 'ghost'"
              size="xs"
              :aria-label="
                t(minimapOpen ? 'workflow.canvas.hide_minimap' : 'workflow.canvas.show_minimap')
              "
              :aria-pressed="minimapOpen"
              @click="minimapOpen = !minimapOpen"
            />
            <UPopover mode="click" :ui="{ content: 'w-72 p-3' }">
              <UButton
                data-testid="workflow-canvas-help"
                icon="i-tabler-help-circle"
                color="neutral"
                variant="ghost"
                size="xs"
                :aria-label="t('workflow.canvas.help')"
              />
              <template #content>
                <div class="space-y-2 text-xs">
                  <p class="font-medium text-highlighted">{{ t('workflow.canvas.help') }}</p>
                  <dl class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1.5 text-muted">
                    <dt class="font-mono text-toned">{{ t('workflow.canvas.marquee_key') }}</dt>
                    <dd>{{ t('workflow.canvas.marquee') }}</dd>
                    <dt class="font-mono text-toned">Shift</dt>
                    <dd>{{ t('workflow.canvas.add_selection') }}</dd>
                    <dt class="font-mono text-toned">Ctrl</dt>
                    <dd>{{ t('workflow.canvas.toggle_selection') }}</dd>
                    <dt class="font-mono text-toned">Space / MMB</dt>
                    <dd>{{ t('workflow.canvas.pan') }}</dd>
                    <dt class="font-mono text-toned">Delete / Esc</dt>
                    <dd>{{ t('workflow.canvas.delete_clear') }}</dd>
                  </dl>
                </div>
              </template>
            </UPopover>
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
            :error="connectionError"
            @select="selectConnectionCandidate"
            @close="closeConnectionMenu"
          />
          <template v-if="session.currentGraph?.kind === 'main'">
            <div
              v-if="currentGraphElementCount === 0"
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
          </template>
          <div
            v-if="currentGraphElementCount === 0 && session.currentGraph?.kind === 'subgraph'"
            data-testid="workflow-subgraph-empty-hint"
            class="pointer-events-none absolute bottom-4 left-1/2 z-10 -translate-x-1/2 rounded-lg border border-default bg-default/90 px-3 py-2 text-center text-[11px] text-muted shadow-sm"
            role="status"
          >
            {{ t('workflow.empty_canvas.subgraph_description') }}
          </div>
        </div>

        <aside
          v-if="inspectorSidebarOpen"
          data-testid="workflow-inspector-sidebar"
          class="relative flex h-full shrink-0 [&>aside]:!w-full"
          :style="{ width: `${inspectorSidebarWidth}px` }"
        >
          <div
            role="separator"
            tabindex="0"
            :aria-label="t('workflow.sidebar.resize_inspector')"
            aria-orientation="vertical"
            class="absolute inset-y-0 left-0 z-30 w-1 cursor-col-resize transition-colors hover:bg-primary/50 focus-visible:bg-primary/70 focus-visible:outline-none"
            @pointerdown="startSidebarResize('inspector', $event)"
            @keydown.left.prevent="
              inspectorSidebarWidth = resizeInspectorSidebar(inspectorSidebarWidth, 16)
            "
            @keydown.right.prevent="
              inspectorSidebarWidth = resizeInspectorSidebar(inspectorSidebarWidth, -16)
            "
          />
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
            :references="stateReferenceLocations"
            :type-change-impact="stateTypeChangeImpact"
            @command="applyCommand"
            @insert="insertStateReferenceAtCenter"
            @locate="locateStateReference"
            @locate-reference="locateStateReferenceAt"
            @close="statePanelOpen = false"
          />
          <WorkflowGraphCallInspector
            v-else-if="selectedCall && selectedCallGraph"
            :call="selectedCall"
            :graph="selectedCallGraph"
            :ports="selectedCallPorts"
            :resources="session.source?.resources ?? []"
            @update="applyCommand({ kind: 'update-graph-call', call: $event })"
            @open="openCalledGraph(selectedCallGraph.id)"
            @duplicate="duplicateSelectedGraphCall"
            @fork="forkSelectedGraphCall"
            @expand="expandSelectedGraphCall"
            @remove="applyCommand({ kind: 'remove-graph-call', callId: selectedCall.id })"
            @locate-resource="locateBoundResource"
          />
          <WorkflowGraphInterfacePanel
            v-else-if="session.currentGraph?.kind === 'subgraph' && !selectedNodeId"
            :graph="session.currentGraph"
            :candidates="graphInterfaceCandidates"
            :reference-counts="graphInterfaceReferenceCounts"
            :infer-disabled="!canInferGraphInterface.valid"
            :infer-hint="
              canInferGraphInterface.valid
                ? t('workflow.graphs.infer_interface_hint')
                : canInferGraphInterface.message
            "
            @infer="inferGraphInterface"
            @add="addGraphInterfaceCandidate"
            @rename="renameGraphInterfaceItem"
            @move="moveGraphInterfaceItem"
            @remove="removeGraphInterfaceItem"
          />
          <WorkflowInspector
            v-else
            :node="selectedNode"
            :projection="selectedProjection"
            :variables="session.source?.variables ?? []"
            :target-defaults="session.source?.targetDefaults ?? []"
            :types="session.authoring?.body.types ?? []"
            :connected-input-ids="selectedConnectedInputIDs"
            :resources="session.source?.resources ?? []"
            @command="applyCommand"
            @capture-template="selectedNode && captureTemplateForNode(selectedNode.id)"
            @locate-resource="locateBoundResource"
          />
        </aside>
      </div>

      <WorkflowRuntimeWorkbench
        v-model:open="runtimeWorkbenchOpen"
        v-model:tab="runtimeWorkbenchTab"
        :run="session.activeRun"
        :snapshot="session.debugSnapshot"
        :debug-busy="debugControlBusy"
        :node-labels="debugNodeLabels"
        :unhandled-routes="unhandledRunRoutes"
        :diagnostics="session.diagnostics"
        :timeline-exporting="timelineExporting"
        @cancel="editorRuns.execute({ kind: 'cancel' })"
        @refresh="editorRuns.execute({ kind: 'refresh' })"
        @export-timeline="exportRunTimeline"
        @page="(page) => editorRuns.execute({ kind: 'load-timeline-page', page })"
        @focus-node="focusNode"
        @focus="focusDiagnostic"
        @continue="editorRuns.execute({ kind: 'control-debug', action: 'continue' })"
        @pause="editorRuns.execute({ kind: 'control-debug', action: 'pause' })"
        @step="editorRuns.execute({ kind: 'control-debug', action: 'step' })"
      />

      <WorkflowQuickAddMenu
        v-model:open="quickAddOpen"
        :items="quickAddItems"
        :anchor="quickAddAnchor"
        @choose="selectQuickAddItem"
      />

      <BaseModal
        v-model:open="graphDialogOpen"
        :title="
          graphDialogMode === 'create' ? t('workflow.graphs.new') : t('workflow.graphs.rename')
        "
        icon="i-tabler-folders"
        size="md"
      >
        <UFormField :label="t('common.name')" required>
          <UInput
            v-model="graphName"
            data-testid="workflow-graph-name"
            autofocus
            maxlength="256"
            @keydown.enter="commitGraphDialog"
          />
        </UFormField>
        <template #footer>
          <UButton
            data-testid="workflow-graph-cancel"
            color="neutral"
            variant="ghost"
            :label="t('common.cancel')"
            @click="graphDialogOpen = false"
          />
          <UButton
            data-testid="workflow-graph-confirm"
            :disabled="!graphName.trim()"
            :label="t('common.confirm')"
            @click="commitGraphDialog"
          />
        </template>
      </BaseModal>
    </template>

    <BaseModal
      :open="Boolean(pendingConversion)"
      :title="t('workflow.connection.conversion_title')"
      icon="i-tabler-arrows-transfer-down"
      size="md"
      @update:open="(open) => !open && cancelConversion()"
    >
      <p class="text-xs leading-5 text-muted">
        {{
          t('workflow.connection.conversion_hint', {
            source: pendingConversion?.sourceType,
            target: pendingConversion?.targetType,
          })
        }}
      </p>
      <div class="mt-3 space-y-2">
        <button
          v-for="candidate in pendingConversion?.candidates ?? []"
          :key="candidate.nodeTypeId"
          type="button"
          data-testid="workflow-conversion-candidate"
          class="flex w-full items-center gap-3 rounded-lg border border-default px-3 py-3 text-left hover:border-primary/50 hover:bg-elevated focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
          @click="applyConversion(candidate)"
        >
          <UIcon name="i-tabler-arrows-transfer-down" class="size-4 shrink-0 text-primary" />
          <span class="min-w-0 flex-1">
            <span class="block truncate text-xs font-semibold text-highlighted">
              {{ conversionTitle(candidate) }}
            </span>
            <span class="mt-0.5 block text-[10px] text-muted">
              {{ t('workflow.connection.conversion_cost', { cost: candidate.cost }) }}
            </span>
          </span>
          <UBadge color="warning" variant="soft" size="sm">
            {{ t(`workflow.connection.conversion_${candidate.kind}`) }}
          </UBadge>
        </button>
      </div>
      <template #footer>
        <UButton
          color="neutral"
          variant="ghost"
          :label="t('common.cancel')"
          @click="cancelConversion"
        />
      </template>
    </BaseModal>

    <BaseModal
      :open="Boolean(pendingStatePromotion)"
      :title="t('workflow.state_panel.promote_title')"
      icon="i-tabler-database-plus"
      size="md"
      @update:open="(open) => !open && cancelStatePromotion()"
    >
      <p class="text-xs leading-5 text-muted">
        {{
          t('workflow.state_panel.promote_hint', {
            type: pendingStatePromotion?.typeLabel,
          })
        }}
      </p>
      <UFormField class="mt-3" :label="t('workflow.inspector.state_name_placeholder')" required>
        <UInput
          v-model="statePromotionName"
          data-testid="workflow-state-promotion-name"
          autofocus
          maxlength="128"
          @keydown.enter.prevent="commitStatePromotion"
        />
      </UFormField>
      <p v-if="statePromotionError" class="mt-2 text-[11px] text-error">
        {{ statePromotionError }}
      </p>
      <template #footer>
        <UButton
          color="neutral"
          variant="ghost"
          :label="t('common.cancel')"
          @click="cancelStatePromotion"
        />
        <UButton
          data-testid="workflow-state-promotion-confirm"
          :disabled="Boolean(statePromotionError)"
          :label="t('workflow.state_panel.promote_action')"
          @click="commitStatePromotion"
        />
      </template>
    </BaseModal>

    <BaseModal
      v-model:open="nodeSearchOpen"
      :title="t('workflow.node_search.title')"
      icon="i-tabler-search"
      size="lg"
    >
      <UInput
        v-model="nodeSearchQuery"
        data-testid="workflow-node-search-input"
        icon="i-tabler-search"
        autofocus
        :placeholder="t('workflow.node_search.placeholder')"
        @keydown.enter.prevent="focusFirstNodeSearchResult"
      />
      <div class="mt-3 max-h-96 space-y-1 overflow-y-auto">
        <button
          v-for="result in nodeSearchResults"
          :key="`${result.graphId}:${result.nodeId}`"
          type="button"
          class="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left hover:bg-elevated focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
          @click="selectNodeSearchResult(result)"
        >
          <UIcon :name="`i-tabler-${result.icon || 'box'}`" class="size-4 text-primary" />
          <span class="min-w-0 flex-1">
            <span class="block truncate text-xs font-medium text-highlighted">{{
              result.label
            }}</span>
            <span class="block truncate font-mono text-[10px] text-dimmed">{{
              result.nodeId
            }}</span>
          </span>
          <span class="shrink-0 text-[10px] text-muted">{{ result.graphId }}</span>
        </button>
        <div v-if="!nodeSearchResults.length" class="px-3 py-8 text-center text-xs text-muted">
          {{
            nodeSearchQuery.trim()
              ? t('workflow.node_search.no_results')
              : t('workflow.node_search.empty')
          }}
        </div>
      </div>
      <template #footer>
        <span class="mr-auto text-[11px] text-muted">
          {{ t('workflow.node_search.result_count', { n: nodeSearchResults.length }) }}
        </span>
        <UButton color="neutral" variant="ghost" @click="nodeSearchOpen = false">
          {{ t('common.close') }}
        </UButton>
      </template>
    </BaseModal>

    <BaseModal
      v-model:open="recordingEditor.startOpen"
      :title="
        t(
          recordingEditor.mode === 'simple'
            ? 'workflow.recording.macro_title'
            : 'workflow.recording.precise_title',
        )
      "
      :icon="
        recordingEditor.mode === 'simple' ? 'i-tabler-list-details' : 'i-tabler-route-alt-left'
      "
      size="md"
    >
      <div class="space-y-3">
        <div class="flex items-start gap-3 rounded-lg border border-default bg-elevated/30 p-3">
          <UIcon
            :name="
              recordingEditor.mode === 'simple'
                ? 'i-tabler-list-details'
                : 'i-tabler-route-alt-left'
            "
            class="mt-0.5 size-5 shrink-0 text-primary"
          />
          <p class="text-xs leading-5 text-muted">
            {{
              t(
                recordingEditor.mode === 'simple'
                  ? 'workflow.recording.macro_hint'
                  : 'workflow.recording.precise_hint',
              )
            }}
          </p>
        </div>
        <UFormField :label="t('workflow.recording.target')" required>
          <AdaptiveSelect
            v-model="recordingEditor.targetSlot"
            :items="recordingTargetItems"
            value-key="value"
            label-key="label"
            :placeholder="t('assets.target_placeholder')"
          />
        </UFormField>
      </div>
      <p class="mt-3 text-xs leading-5 text-muted">{{ t('workflow.recording.start_hint') }}</p>
      <template #footer>
        <UButton color="neutral" variant="ghost" @click="recordingEditor.startOpen = false">
          {{ t('common.cancel') }}
        </UButton>
        <UButton
          :disabled="!recordingEditor.targetSlot"
          :loading="recordingEditor.controlBusy"
          @click="editorRecording.execute({ kind: 'start' })"
        >
          {{ t('workflow.recording.start') }}
        </UButton>
      </template>
    </BaseModal>

    <BaseModal
      v-model:open="templateCaptureOpen"
      :title="t('assets.templates.capture')"
      icon="i-tabler-camera-plus"
      size="md"
    >
      <div class="space-y-3">
        <p class="text-xs leading-5 text-muted">{{ t('workflow.resources.capture_hint') }}</p>
        <UFormField :label="t('workflow.recording.target')" required>
          <AdaptiveSelect
            v-model="captureTargetSlot"
            :items="recordingTargetItems"
            value-key="value"
            label-key="label"
            :placeholder="t('assets.target_placeholder')"
          />
        </UFormField>
      </div>
      <template #footer>
        <UButton color="neutral" variant="ghost" @click="templateCaptureOpen = false">
          {{ t('common.cancel') }}
        </UButton>
        <UButton
          icon="i-tabler-camera-plus"
          :disabled="!captureTargetSlot"
          :loading="templateCaptureBusy"
          @click="captureWorkspaceTemplate"
        >
          {{ t('assets.templates.capture') }}
        </UButton>
      </template>
    </BaseModal>

    <BaseModal
      :open="!!recordingEditor.pending"
      :title="t('recordingSave.title')"
      icon="i-tabler-list-check"
      size="3xl"
      :show-close="false"
      :dismissible="false"
    >
      <div v-if="recordingEditor.pending" class="space-y-4">
        <div v-if="recordingEditor.pending.mode === 'simple'" class="grid grid-cols-2 gap-3">
          <div class="rounded-lg border border-default bg-elevated/35 px-4 py-3">
            <p class="text-xs text-muted">{{ t('workflow.recording.result_mode') }}</p>
            <div class="mt-1 flex items-center gap-2">
              <UBadge
                :color="recordingEditor.pending.preview.mode === 'simple' ? 'primary' : 'warning'"
                variant="soft"
              >
                {{ t(`workflow.recording.mode_${recordingEditor.pending.preview.mode}`) }}
              </UBadge>
              <span class="text-xs text-toned">
                {{
                  t('recordingSave.summary', {
                    duration: formatRecordingDuration(recordingEditor.pending.durationUs),
                    count: recordingEditor.pending.eventCount,
                  })
                }}
              </span>
            </div>
          </div>
          <div class="rounded-lg border border-default bg-elevated/35 px-4 py-3 text-xs text-muted">
            {{
              t('workflow.recording.action_summary', {
                keys: recordingEditor.pending.preview.keyActions,
                clicks: recordingEditor.pending.preview.clickActions,
                moves:
                  recordingEditor.pending.preview.pointerMoves +
                  recordingEditor.pending.preview.rawDeltas,
                scrolls: recordingEditor.pending.preview.scrollActions,
              })
            }}
          </div>
        </div>
        <div
          v-if="
            recordingEditor.pending.mode === 'simple' &&
            recordingEditor.pending.preview.steps.length
          "
          class="max-h-48 space-y-1 overflow-y-auto rounded-lg border border-default bg-sunken p-2"
        >
          <div
            v-for="(step, index) in recordingEditor.pending.preview.steps"
            :key="`${step.atUs}:${index}`"
            class="flex items-center gap-2 rounded-md px-2 py-1.5 text-xs text-muted"
          >
            <span class="w-12 shrink-0 font-mono text-[10px] text-dimmed">
              {{ (step.atUs / 1_000_000).toFixed(2) }}s
            </span>
            <UIcon
              :name="
                step.kind.startsWith('key-')
                  ? 'i-tabler-keyboard'
                  : step.kind === 'sleep'
                    ? 'i-tabler-clock-pause'
                    : 'i-tabler-pointer'
              "
              class="size-4 shrink-0 text-primary"
            />
            <span class="truncate text-toned">
              {{
                step.kind.startsWith('key-')
                  ? `${step.kind === 'key-down' ? '↓' : '↑'} ${step.key}`
                  : step.kind === 'sleep'
                    ? `${Math.round(step.durationUs / 1000)} ms`
                    : `${step.button ?? step.kind} · ${Math.round((step.point?.x ?? 0) * 100)}%, ${Math.round((step.point?.y ?? 0) * 100)}%`
              }}
            </span>
          </div>
        </div>
        <PreciseRecordingWorkbench
          v-if="recordingEditor.pending.mode === 'precise'"
          :preview="recordingEditor.pending.preview"
          :environment="recordingEditor.pending.environment"
          :duration-us="recordingEditor.pending.durationUs"
          :trim-start-us="recordingEditor.trimStartUs"
          :trim-end-us="recordingEditor.trimEndUs"
          :pending-id="recordingEditor.pending.pendingID"
          editable-trim
          @update:trim-start-us="recordingEditor.trimStartUs = $event"
          @update:trim-end-us="recordingEditor.trimEndUs = $event"
        />
        <MacroActionEditor
          v-if="recordingEditor.pending.mode === 'simple' && recordingEditor.document"
          v-model="recordingEditor.document"
          @validity="recordingEditor.actionsValid = $event"
        />
        <p
          v-else-if="recordingEditor.pending.mode === 'simple'"
          class="rounded-lg border border-default bg-sunken px-3 py-2 text-xs text-muted"
        >
          {{ t('recordingEditor.editing_unavailable') }}
        </p>
        <RecordingMetadataFields
          v-model:name="recordingEditor.draft.name"
          v-model:description="recordingEditor.draft.description"
          v-model:category="recordingEditor.draft.category"
          v-model:tags="recordingEditor.draft.tags"
          :categories="recordingEditor.facetCategories"
          :tag-suggestions="recordingEditor.facetTags"
        />
      </div>
      <template #footer>
        <UButton
          color="error"
          variant="ghost"
          :disabled="recordingEditor.saveBusy"
          @click="discardPendingRecording"
        >
          {{ t('recordingSave.discard') }}
        </UButton>
        <UButton
          :loading="recordingEditor.saveBusy"
          :disabled="
            !recordingEditor.draft.name.trim() ||
            (recordingEditor.pending?.mode === 'simple' && !recordingEditor.actionsValid)
          "
          @click="editorRecording.execute({ kind: 'finalize' })"
        >
          {{ t('assets.recording.save_to_library') }}
        </UButton>
      </template>
    </BaseModal>
    <BaseModal
      :open="!!macroEditing"
      :title="macroEditing?.label ?? t('macroEditor.title')"
      icon="i-tabler-list-details"
      size="5xl"
      tall
      @update:open="(open) => !open && (macroEditing = null)"
    >
      <div v-if="macroEditing" class="flex h-full min-h-0 flex-col gap-3">
        <div class="shrink-0 space-y-3 rounded-lg border border-default bg-elevated/20 p-3">
          <RecordingMetadataFields
            v-model:name="macroEditing.label"
            v-model:description="macroEditing.description"
            v-model:category="macroEditing.category"
            v-model:tags="macroEditing.tags"
            :categories="macroMetadataCategories"
            :tag-suggestions="macroMetadataTags"
          />
        </div>
        <div
          class="flex shrink-0 items-center gap-3 rounded-lg border border-default bg-elevated/25 px-3 py-2 text-xs text-muted"
        >
          <span>{{ t('assets.macros.base_resolution') }}</span>
          <strong class="font-mono text-toned">
            {{ macroEditing.document.baseResolution[0] }}×{{
              macroEditing.document.baseResolution[1]
            }}
          </strong>
          <span class="ml-auto truncate font-mono text-[10px] text-dimmed">{{
            macroEditing.id
          }}</span>
        </div>
        <MacroActionEditor
          v-model="macroEditing.document"
          class="min-h-0 flex-1"
          @validity="macroEditValid = $event"
        />
      </div>
      <template #footer>
        <UButton color="neutral" variant="ghost" @click="macroEditing = null">
          {{ t('common.cancel') }}
        </UButton>
        <UButton
          icon="i-tabler-device-floppy"
          :loading="macroEditBusy"
          :disabled="!macroEditValid || !macroEditing?.label.trim()"
          @click="saveMacro"
        >
          {{ t('common.save') }}
        </UButton>
      </template>
    </BaseModal>
    <BaseModal
      :open="!!workflowMacroEditing"
      :title="workflowMacroEditing?.resource.name ?? t('macroEditor.title')"
      icon="i-tabler-list-details"
      size="5xl"
      tall
      @update:open="(open) => !open && (workflowMacroEditing = null)"
    >
      <div v-if="workflowMacroEditing" class="flex h-full min-h-0 flex-col gap-3">
        <div class="shrink-0 space-y-3 rounded-lg border border-default bg-elevated/20 p-3">
          <RecordingMetadataFields
            v-model:name="workflowMacroEditing.resource.name"
            v-model:description="workflowMacroEditing.resource.description"
            v-model:category="workflowMacroEditing.resource.category"
            v-model:tags="workflowMacroEditing.resource.tags"
            :categories="macroMetadataCategories"
            :tag-suggestions="macroMetadataTags"
          />
        </div>
        <div
          class="flex shrink-0 items-center gap-3 rounded-lg border border-default bg-elevated/25 px-3 py-2 text-xs text-muted"
        >
          <span>{{ t('assets.macros.base_resolution') }}</span>
          <strong class="font-mono text-toned">
            {{ workflowMacroEditing.document.baseResolution[0] }}×{{
              workflowMacroEditing.document.baseResolution[1]
            }}
          </strong>
          <span class="ml-auto truncate font-mono text-[10px] text-dimmed">{{
            workflowMacroEditing.resource.id
          }}</span>
        </div>
        <MacroActionEditor
          v-model="workflowMacroEditing.document"
          class="min-h-0 flex-1"
          @validity="workflowMacroEditValid = $event"
        />
      </div>
      <template #footer>
        <UButton color="neutral" variant="ghost" @click="workflowMacroEditing = null">
          {{ t('common.cancel') }}
        </UButton>
        <UButton
          icon="i-tabler-device-floppy"
          :loading="workflowResourceEditBusy"
          :disabled="!workflowMacroEditValid || !workflowMacroEditing?.resource.name.trim()"
          @click="saveWorkflowMacro"
        >
          {{ t('common.save') }}
        </UButton>
      </template>
    </BaseModal>
    <BaseModal
      :open="!!workflowClipEditing"
      :title="workflowClipEditing?.resource.name ?? t('preciseWorkbench.title')"
      icon="i-tabler-route-alt-left"
      size="5xl"
      tall
      @update:open="(open) => !open && (workflowClipEditing = null)"
    >
      <PreciseRecordingWorkbench
        v-if="workflowClipEditing && workflowClipPreview"
        :preview="workflowClipPreview"
        :environment="{
          baseResolution: workflowClipEditing.content.baseResolution,
          mouseMode: workflowClipEditing.content.mouseMode,
          mouseCounts360: workflowClipEditing.content.mouseCounts360,
        }"
        :duration-us="workflowClipEditing.content.durationUs"
        :trim-start-us="workflowClipTrimStartUs"
        :trim-end-us="workflowClipTrimEndUs"
        :workflow-resource="workflowClipEditing.resource"
        editable-trim
        @update:trim-start-us="workflowClipTrimStartUs = $event"
        @update:trim-end-us="workflowClipTrimEndUs = $event"
      />
      <template #footer>
        <UButton color="neutral" variant="ghost" @click="workflowClipEditing = null">
          {{ t('common.cancel') }}
        </UButton>
        <UButton
          icon="i-tabler-cut"
          :loading="workflowResourceEditBusy"
          :disabled="!workflowClipTrimChanged"
          @click="saveWorkflowClipTrim"
        >
          {{ t('common.save') }}
        </UButton>
      </template>
    </BaseModal>
    <WorkflowSnippetModal
      :open="snippetModalOpen"
      :snippet-id="snippetDraft?.id ?? ''"
      :node-type-id="snippetDraft?.payload.nodeRef.nodeTypeId ?? ''"
      :initial="snippetModalInitial"
      :existing="snippets.items"
      :busy="snippetSaveBusy"
      @update:open="snippetModalOpen = $event"
      @save="saveSnippet"
    />
  </div>
</template>

<script setup lang="ts">
import {
  computed,
  defineAsyncComponent,
  nextTick,
  onActivated,
  onBeforeUnmount,
  onDeactivated,
  onMounted,
  reactive,
  ref,
  watch,
} from 'vue'
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
  type Node as FlowNode,
  type OnConnectStartParams,
} from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'
import '@vue-flow/controls/dist/style.css'
import '@vue-flow/minimap/dist/style.css'
import { useI18n } from 'vue-i18n'
import { useLocalStorage } from '@vueuse/core'
import {
  type Edge,
  type EditorCommand,
  type Node,
  type NodeProjection,
  type StateReferenceMode,
} from '@/app/editor/EditorSession'
import type {
  Annotation,
  TypeExpression,
  WorkflowResource,
} from '../../../contracts/workflow/current/workflow-source'
import { createEditorSession } from '@/app/editor/createEditorSession'
import { graphHandle, parseGraphHandle, type ParsedHandle } from '@/app/editor/graphHandles'
import {
  onDebugChanged,
  onRunChanged,
  workflowTransport,
  type DebugBreakpoint,
} from '@/app/transport/workflow'
import { useConfirm } from '@/composables/useConfirm'
import { useRecordingStart } from '@/composables/useRecordingStart'
import { useRecordingStartFeedback } from '@/composables/useRecordingStartFeedback'
import WorkflowNode from '@/app/editor/WorkflowNode.vue'
import { effectiveTargetSlot } from '@/app/editor/authoringSurface'
import WorkflowEditorToolbar from '@/app/editor/WorkflowEditorToolbar.vue'
import WorkflowWorkspaceRail from '@/app/editor/WorkflowWorkspaceRail.vue'
import {
  createEditorRunController,
  type EditorRuntimeWorkbenchTab,
} from '@/app/editor/EditorRunController'
import { createEditorResourceController } from '@/app/editor/EditorResourceController'
import {
  createEditorRecordingController,
  formatRecordingDuration,
} from '@/app/editor/EditorRecordingController'
import { createEditorCanvasLayoutController } from '@/app/editor/EditorCanvasLayoutController'
import { createEditorSelectionController } from '@/app/editor/EditorSelectionController'
import type { EditorToolbarCommand, EditorToolbarContext } from '@/app/editor/editorToolbarModel'
import type { WorkflowWorkspacePanel } from '@/app/editor/workspacePanel'
import type { WorkflowMetadataDraft } from '@/app/editor/WorkflowMetadataDialog.vue'
import { parseWorkspaceResource, RESOURCE_DRAG_FORMAT } from '@/app/editor/resourceDrag'
import { snapshotGlobalAssetByID } from '@/app/editor/workflowResourceSnapshot'
import type {
  ResourceLocateRequest,
  ResourceLocation,
  WorkspaceResourceKind,
} from '@/app/editor/resourceLocator'
import WorkflowSnippetDock from '@/app/editor/WorkflowSnippetDock.vue'
import WorkflowConnectionMenu, {
  type WorkflowConnectionCandidate,
} from '@/app/editor/WorkflowConnectionMenu.vue'
import WorkflowSelectionToolbar from '@/app/editor/WorkflowSelectionToolbar.vue'
import WorkflowGraphCall from '@/app/editor/WorkflowGraphCall.vue'
import WorkflowGraphBoundary from '@/app/editor/WorkflowGraphBoundary.vue'
import WorkflowAnnotation from '@/app/editor/WorkflowAnnotation.vue'
import WorkflowRerouteEdge from '@/app/editor/WorkflowRerouteEdge.vue'
import BaseModal from '@/components/common/BaseModal.vue'
import { useRecordingStore, type RecordingMode } from '@/stores/recording'
import { useSettingsStore } from '@/stores/settings'
import { useAssetsStore, type AssetPickerSelection } from '@/stores/assets'
import { backend, type AssetSummary, type WorkflowSnippet } from '@/lib/backend'
import { useSnippetsStore } from '@/stores/snippets'
import { shortcutFromKeyboardEvent } from '@/app/editor/snippetShortcut'
import { resolveEditorKeyboardAction } from '@/app/editor/editorKeyboard'
import type { WorkflowQuickAddItem } from '@/app/editor/workflowQuickAdd'
import { errorMessage } from '@/lib/invoke'
import { awaitWailsEvent } from '@/composables/useWailsEvent'
import { nodeRunStatuses, unhandledExecRouteKeys } from '@/app/editor/runTrace'
import { nodeDiagnosticSeverities, type WorkflowDiagnostic } from '@/app/editor/workflowDiagnostics'
import {
  compatibleCandidatePorts,
  projectedTargetHandleChannel,
  type ConversionCandidatePlan,
  type ConnectionIssue,
} from '@/app/editor/connectionCompatibility'
import {
  createWorkflowNodeGestureState,
  projectWorkflowFlowNodes,
} from '@/app/editor/workflowFlowProjection'
import {
  canvasOwnsWheelTarget,
  mergeMarqueeSelection,
  WORKFLOW_CANVAS_INTERACTION,
  zoomViewportAtPoint,
} from '@/app/editor/workflowCanvasInteraction'
import { workflowEdgeVisualState } from '@/app/editor/workflowEdgeVisualState'
import {
  analyzeCollapseBoundary,
  graphBoundaryBindingFromConnection,
  graphBoundaryKeyFromEdge,
  isGraphBoundaryNodeId,
  projectGraphBoundaries,
} from '@/app/editor/workflowGraphBoundary'
import { collapseSelectionErrorReason } from '@/app/editor/collapseSelectionError'
import { projectGraphDefinitions } from '@/app/editor/subgraphManagement'
import type {
  GraphInterfaceCandidateKind,
  GraphInterfaceItemKind,
} from '@/app/editor/subgraphInterface'
import type { AlignMode, DistributeMode } from '@/app/editor/workflowLayout'

defineOptions({ name: 'WorkflowEditorView' })

const MacroActionEditor = defineAsyncComponent(
  () => import('@/components/recording/MacroActionEditor.vue'),
)
const WorkflowSnippetModal = defineAsyncComponent(
  () => import('@/app/editor/WorkflowSnippetModal.vue'),
)
const WorkflowQuickAddMenu = defineAsyncComponent(
  () => import('@/app/editor/WorkflowQuickAddMenu.vue'),
)
const WorkflowGraphManager = defineAsyncComponent(
  () => import('@/app/editor/WorkflowGraphManager.vue'),
)
const WorkflowGraphCallInspector = defineAsyncComponent(
  () => import('@/app/editor/WorkflowGraphCallInspector.vue'),
)
const WorkflowGraphInterfacePanel = defineAsyncComponent(
  () => import('@/app/editor/WorkflowGraphInterfacePanel.vue'),
)
const WorkflowResourceDock = defineAsyncComponent(
  () => import('@/app/editor/WorkflowResourceDock.vue'),
)
const WorkflowInspector = defineAsyncComponent(() => import('@/app/editor/WorkflowInspector.vue'))
const WorkflowRuntimeWorkbench = defineAsyncComponent(
  () => import('@/app/editor/WorkflowRuntimeWorkbench.vue'),
)
const WorkflowStatePanel = defineAsyncComponent(() => import('@/app/editor/WorkflowStatePanel.vue'))
const PreciseRecordingWorkbench = defineAsyncComponent(
  () => import('@/components/recording/PreciseRecordingWorkbench.vue'),
)
const RecordingMetadataFields = defineAsyncComponent(
  () => import('@/components/recording/RecordingMetadataFields.vue'),
)
const WorkflowMetadataDialog = defineAsyncComponent(
  () => import('@/app/editor/WorkflowMetadataDialog.vue'),
)
const AIWorkflowReviewPanel = defineAsyncComponent(
  () => import('@/app/editor/AIWorkflowReviewPanel.vue'),
)

const route = useRoute()
const router = useRouter()
const toast = useToast()
const { confirm } = useConfirm()
const { t, te } = useI18n()
const session = createEditorSession(workflowTransport)
const recording = useRecordingStore()
const editorViewActive = ref(true)
const { start: beginRecording } = useRecordingStart()
const { show: showRecordingStartError } = useRecordingStartFeedback()
const settings = useSettingsStore()
const assets = useAssetsStore()
const snippets = useSnippetsStore()
const editorResources = createEditorResourceController({
  port: {
    openWorkflow: (resource) => backend.workflowResources.open(resource),
    rewriteWorkflow: (resource, edit) => backend.workflowResources.rewrite(resource, edit),
    getMacro: (id) => backend.macros.get(id),
    saveMacro: (asset) => backend.macros.save(asset),
  },
  replaceWorkflowResource: (resourceId, resource) =>
    session.apply({ kind: 'replace-resource', resourceId, resource }),
  invalidateAssets: () => assets.invalidate(),
  translate: (key) => t(key),
  showError,
})
const selectedNodeId = ref('')
const selectedNodeIds = ref(new Set<string>())
const selectedEdgeId = ref('')
const nodeDragActive = ref(false)
const aiPanelOpen = ref(false)
const statePanelOpen = ref(false)
const quickAddOpen = ref(false)
const quickAddPosition = ref({ x: 160, y: 160 })
const quickAddAnchor = ref({ x: 160, y: 160 })
const canvasPointerInside = ref(false)
const lastCanvasPointer = ref<{ x: number; y: number } | null>(null)
const workspacePanel = ref<WorkflowWorkspacePanel>('graphs')
const workspaceSidebarOpen = ref(true)
const workspaceSidebarWidth = ref(320)
const resourceLocateRequest = ref<ResourceLocateRequest | null>(null)
let resourceLocateSequence = 0
const inspectorAutoOpen = useLocalStorage('yotta.workflow.inspector.auto-open', true)
const inspectorSidebarOpen = ref(inspectorAutoOpen.value)
const inspectorSidebarWidth = ref(360)
const workflowSettingsOpen = ref(false)
const workflowSettingsBusy = ref(false)
const workflowSettingsError = ref('')
const workflowMetadata = reactive<WorkflowMetadataDraft>({
  name: '',
  description: '',
  category: '',
  tags: [],
})
const workspaceResourcePanel = computed(
  () =>
    workspacePanel.value === 'macro' ||
    workspacePanel.value === 'clip' ||
    workspacePanel.value === 'template',
)
const workspaceResourceKind = computed<WorkspaceResourceKind>(() => {
  const panel = workspacePanel.value
  return panel === 'macro' || panel === 'clip' || panel === 'template' ? panel : 'macro'
})
const graphDialogOpen = ref(false)
const graphDialogMode = ref<'create' | 'rename'>('create')
const graphDialogTargetId = ref('')
const graphName = ref('')
const nodeSearchOpen = ref(false)
const nodeSearchQuery = ref('')
const creationTemplate = computed(() => {
  const value = route.query.template
  return value === 'windows' ||
    value === 'android' ||
    value === 'browser' ||
    value === 'cross-target'
    ? value
    : ''
})
const creationTemplateIcon = computed(() =>
  creationTemplate.value === 'android'
    ? 'i-tabler-brand-android'
    : creationTemplate.value === 'browser'
      ? 'i-tabler-brand-chrome'
      : creationTemplate.value === 'windows'
        ? 'i-tabler-brand-windows'
        : 'i-tabler-devices',
)
const canvasElement = ref<HTMLElement | null>(null)
const connectionStart = ref<ConnectionAnchor | null>(null)
const connectionMenu = ref<ConnectionMenuState | null>(null)
const pendingConversion = ref<PendingConversion | null>(null)
const pendingStatePromotion = ref<PendingStatePromotion | null>(null)
const statePromotionName = ref('')
const connectionHint = ref('')
const connectionError = ref('')
const minimapOpen = ref(false)
const runtimeWorkbenchOpen = ref(false)
const runtimeWorkbenchTab = ref<EditorRuntimeWorkbenchTab>('logs')
const timelineExporting = ref(false)
const diagnosticsOpen = computed(
  () => runtimeWorkbenchOpen.value && runtimeWorkbenchTab.value === 'diagnostics',
)
const runTimelineOpen = computed(
  () => runtimeWorkbenchOpen.value && runtimeWorkbenchTab.value === 'timeline',
)
const debuggerOpen = computed(
  () => runtimeWorkbenchOpen.value && runtimeWorkbenchTab.value === 'debug',
)
const debugModeActive = computed(
  () =>
    debuggerOpen.value ||
    Boolean(session.debugSnapshot && session.debugSnapshot.status !== 'completed'),
)
const editorRuns = createEditorRunController({
  session,
  translate: (key) => t(key),
  showError,
  showSuccess,
  openWorkbench: openRuntimeWorkbench,
  focusDebugNode: focusNode,
})
const { saveSucceeded, debugControlBusy } = editorRuns
const editorToolbarContext = computed<Omit<EditorToolbarContext, 'dirty'>>(() => ({
  canUndo: session.canUndo,
  canRedo: session.canRedo,
  aiPanelOpen: aiPanelOpen.value,
  statePanelOpen: statePanelOpen.value,
  inspectorOpen: inspectorSidebarOpen.value,
  runActive: runActive.value,
  saving: session.phase === 'saving',
  saveSucceeded: saveSucceeded.value,
  diagnosticCount: session.diagnostics.length,
  diagnosticsOpen: diagnosticsOpen.value,
  hasRunTimeline: Boolean(session.activeRun),
  runTimelineOpen: runTimelineOpen.value,
  debugModeActive: debugModeActive.value,
  debuggerOpen: debuggerOpen.value,
  recordingPhase: recording.state.phase,
}))
const isRevisionConflict = computed(() => session.saveErrorKind === 'revision')
const breakpointKeys = ref(new Set<string>())
const {
  macroEditing,
  macroEditBusy,
  macroEditValid,
  workflowMacroEditing,
  workflowMacroEditValid,
  workflowClipEditing,
  workflowClipTrimStartUs,
  workflowClipTrimEndUs,
  workflowResourceEditBusy,
  workflowClipPreview,
  workflowClipTrimChanged,
} = editorResources
const macroMetadataCategories = computed(() =>
  [
    ...new Set(
      [
        ...recordingEditor.facetCategories,
        ...(session.source?.resources ?? []).map((resource) => resource.category),
      ]
        .map((value) => value?.trim() ?? '')
        .filter(Boolean),
    ),
  ].sort((left, right) => left.localeCompare(right)),
)
const macroMetadataTags = computed(() =>
  [
    ...new Set([
      ...recordingEditor.facetTags,
      ...(session.source?.resources ?? []).flatMap((resource) => resource.tags ?? []),
    ]),
  ]
    .map((value) => value?.trim() ?? '')
    .filter(Boolean)
    .sort((left, right) => left.localeCompare(right)),
)
const templateCaptureOpen = ref(false)
const captureTargetSlot = ref('')
const templateCaptureBusy = ref(false)
const snippetModalOpen = ref(false)
const snippetSaveBusy = ref(false)
const snippetDraft = ref<WorkflowSnippet | null>(null)
const snippetModalInitial = computed(() =>
  snippetDraft.value
    ? {
        name: snippetDraft.value.name,
        description: snippetDraft.value.description,
        category: snippetDraft.value.category,
        tags: snippetDraft.value.tags,
        shortcut: snippetDraft.value.shortcut,
      }
    : undefined,
)
const {
  addSelectedNodes,
  findNode,
  fitView,
  flowToScreenCoordinate,
  getViewport,
  getSelectedNodes,
  removeSelectedNodes,
  screenToFlowCoordinate,
  setCenter,
  setViewport,
  updateNode,
} = useVueFlow()
const editorSelection = createEditorSelectionController({
  session,
  selectedNodeId,
  selectedNodeIds,
  selectedEdgeId,
  selectedFlowNodes: () => getSelectedNodes.value,
  findNode,
  addSelectedNodes,
  removeSelectedNodes,
  applyCommand,
  disconnectEdge,
  clipboard: navigator.clipboard,
  translate: (key) => t(key),
  showError,
})
const editorCanvasLayout = createEditorCanvasLayoutController({
  session,
  canvasElement,
  selectedNodeIds,
  findNode,
  fitView,
  flowToScreenCoordinate,
  applyCommand,
  layoutErrorTitle: () => t('workflow.selection.layout_failed'),
  showError,
})
const { snapGuides, layouting, fitCurrentGraph } = editorCanvasLayout
const nodeGestures = createWorkflowNodeGestureState()
let unsubscribeRun: (() => void) | undefined
let unsubscribeDebug: (() => void) | undefined
let stopSidebarResize: (() => void) | undefined
let connectionEndTimer: ReturnType<typeof setTimeout> | undefined
let connectionMadeThisGesture = false
let nextPosition = 0
let marqueeSelectionBase = new Set<string>()
let marqueeSelectionActive = false

const NODE_TYPE_DRAG_FORMAT = 'application/x-yotta-node-type'
const STATE_REFERENCE_DRAG_FORMAT = 'application/x-yotta-state-reference'
const SNIPPET_DRAG_FORMAT = 'application/x-yotta-snippet'
const GRAPH_CALL_DRAG_FORMAT = 'application/x-yotta-graph-call'
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

interface PendingConversion {
  edge: Edge
  candidates: ConversionCandidatePlan[]
  sourceType: string
  targetType: string
  position: { x: number; y: number }
}

interface PendingStatePromotion {
  nodeId: string
  portId: string
  position: { x: number; y: number }
  typeLabel: string
}

interface WorkflowNodeSearchResult {
  graphId: string
  nodeId: string
  label: string
  icon: string
  searchText: string
}

const callableGraphs = computed(() =>
  (session.source?.graphs ?? []).filter(
    (graph) =>
      graph.kind === 'subgraph' &&
      graph.id !== session.currentGraph?.id &&
      !graphReaches(graph.id, session.currentGraph?.id ?? ''),
  ),
)
const callableGraphIds = computed(() => callableGraphs.value.map((graph) => graph.id))

const catalogNodes = computed(() =>
  (session.authoring?.body.nodes ?? []).filter((projection) => {
    if (session.currentGraph?.kind === 'subgraph' && projection.instruction.kind === 'run-root')
      return false
    return visibleForCreationTemplate(projection)
  }),
)
const quickAddItems = computed<WorkflowQuickAddItem[]>(() => [
  ...catalogNodes.value.map((projection) => {
    const description =
      projection.descriptionKey && te(projection.descriptionKey)
        ? t(projection.descriptionKey)
        : projection.nodeRef.nodeTypeId
    const category = `node:${projection.category || 'other'}`
    return {
      id: projection.nodeRef.nodeTypeId,
      kind: 'node' as const,
      title: projectionTitle(projection),
      description,
      category,
      categoryLabel: categoryLabel(projection.category || 'other'),
      icon: `i-tabler-${projection.icon || 'box'}`,
      searchText: catalogSearchText(projection),
    }
  }),
  ...snippets.items.map((snippet) => ({
    id: snippet.id,
    kind: 'snippet' as const,
    title: snippet.name,
    description: snippet.description || snippet.nodeTypeId,
    category: 'snippet:all',
    categoryLabel: t('workflow.snippets.title'),
    icon: 'i-tabler-bookmark',
    shortcut: snippet.shortcut,
    searchText: [
      snippet.name,
      snippet.description,
      snippet.category,
      snippet.tags.join(' '),
      snippet.nodeTypeId,
      snippet.shortcut,
    ]
      .filter(Boolean)
      .join(' ')
      .toLocaleLowerCase(),
  })),
])
const nodeSearchResults = computed<WorkflowNodeSearchResult[]>(() => {
  const query = nodeSearchQuery.value.trim().toLocaleLowerCase()
  if (!query) return []
  return (session.source?.graphs ?? [])
    .flatMap((graph) =>
      graph.nodes.map((node) => {
        const projection = session.nodeProjection(node.nodeRef.nodeTypeId)
        const typeTitle = projection ? projectionTitle(projection) : node.nodeRef.nodeTypeId
        const label = node.label || typeTitle
        return {
          graphId: graph.id,
          nodeId: node.id,
          label,
          icon: projection?.icon ?? 'box',
          searchText: [label, typeTitle, node.id, node.nodeRef.nodeTypeId, graph.id]
            .join(' ')
            .toLocaleLowerCase(),
        }
      }),
    )
    .filter((result) => result.searchText.includes(query))
    .sort(
      (left, right) =>
        left.graphId.localeCompare(right.graphId) || left.label.localeCompare(right.label),
    )
    .slice(0, 200)
})

function visibleForCreationTemplate(projection: NodeProjection): boolean {
  const template = creationTemplate.value
  if (!template || template === 'cross-target') return true
  const targetKind =
    template === 'android'
      ? 'android-device'
      : template === 'browser'
        ? 'browser-cdp'
        : 'desktop-window'
  const automationTargets = (projection.configuredTargets ?? []).filter((target) =>
    target.targetKinds.some((kind) =>
      ['desktop-window', 'android-device', 'browser-cdp'].includes(kind),
    ),
  )
  return automationTargets.every((target) => target.targetKinds.includes(targetKind))
}

const flowNodes = computed<FlowNode[]>(() => {
  const graph = session.currentGraph
  if (!graph) return []
  return [
    ...projectWorkflowFlowNodes(
      graph.nodes,
      session.nodeInstanceProjection.bind(session),
      nodeGestures.positions,
    ),
    ...(graph.calls ?? []).flatMap((call) => {
      const callee = session.calleeGraph(call)
      return callee
        ? [
            {
              id: call.id,
              type: 'graph-call',
              position: nodeGestures.positions.get(call.id) ?? call.position,
              data: { call, graph: callee },
              dragHandle: '.workflow-node-drag-handle',
            },
          ]
        : []
    }),
    ...(graph.annotations ?? []).map((annotation) => ({
      id: annotation.id,
      type: 'annotation',
      position: nodeGestures.positions.get(annotation.id) ?? annotation.position,
      data: { annotation },
      dragHandle: '.workflow-node-drag-handle',
    })),
    ...projectGraphBoundaries(graph).nodes,
  ] as FlowNode[]
})

const graphBoundaryProjection = computed(() =>
  session.currentGraph ? projectGraphBoundaries(session.currentGraph) : { nodes: [], edges: [] },
)
const currentGraphElementCount = computed(() => {
  const graph = session.currentGraph
  return graph
    ? graph.nodes.length + (graph.calls?.length ?? 0) + (graph.annotations?.length ?? 0)
    : 0
})
const canInferGraphInterface = computed(() => {
  const readiness = session.currentGraphInterfaceReadiness()
  if (readiness.valid) return { valid: true, message: '' }
  const key =
    readiness.reason === 'multiple-entry'
      ? 'workflow.graphs.infer_multiple_entries'
      : readiness.reason === 'missing-entry-or-exit'
        ? 'workflow.graphs.infer_missing_endpoints'
        : 'workflow.graphs.infer_not_subgraph'
  return { valid: false, message: t(key) }
})
const graphInterfaceCandidates = computed(() =>
  session.currentGraph?.kind === 'subgraph' ? session.currentGraphInterfaceCandidates() : [],
)
const graphInterfaceReferenceCounts = computed<Record<string, number>>(() => {
  const graph = session.currentGraph
  if (!graph || graph.kind !== 'subgraph') return {}
  return Object.fromEntries([
    ...graph.inputs.map((port) => [
      `input:${port.id}`,
      session.currentGraphInterfaceReferences('input', port.id).length,
    ]),
    ...graph.outputs.map((port) => [
      `output:${port.id}`,
      session.currentGraphInterfaceReferences('output', port.id).length,
    ]),
    ...(graph.exits ?? []).map((exit) => [
      `exit:${exit.id}`,
      session.currentGraphInterfaceReferences('exit', exit.id).length,
    ]),
  ])
})

const flowEdges = computed<FlowEdge[]>(() => [
  ...(session.currentGraph?.edges ?? []).map((edge) => {
    const visual = workflowEdgeVisualState(edge, nodeRunStatusById.value)
    return {
      id: edgeId(edge),
      source: edge.from.nodeId,
      target: edge.to.nodeId,
      sourceHandle: graphHandle(edge.channel, 'output', edge.from.portId),
      targetHandle: edgeTargetHandle(edge),
      selected: selectedEdgeId.value === edgeId(edge),
      type: edge.presentation?.reroutes?.length ? 'reroute' : undefined,
      data: { edge },
      animated: visual.animated,
      style: { stroke: visual.stroke, strokeWidth: visual.strokeWidth },
    }
  }),
  ...graphBoundaryProjection.value.edges.map((edge) => ({
    ...edge,
    selected: selectedEdgeId.value === edge.id,
  })),
])

const compatibleConnectionCandidates = computed<WorkflowConnectionCandidate[]>(() => {
  const menu = connectionMenu.value
  if (!menu) return []
  const anchorNode = session.currentGraph?.nodes.find((node) => node.id === menu.anchor.nodeId)
  if (!anchorNode) return []
  const anchorProjection = session.nodeInstanceProjection(anchorNode)
  if (!anchorProjection) return []
  const candidates: WorkflowConnectionCandidate[] = (session.authoring?.body.nodes ?? [])
    .flatMap((projection) =>
      compatibleCandidatePorts(
        anchorProjection,
        menu.anchor.handle,
        projection,
        new Map((session.authoring?.body.types ?? []).map((type) => [type.typeRef.typeId, type])),
      ).map((port) => ({
        key: `${projection.nodeRef.nodeTypeId}:${port.handle.channel}:${port.handle.portId}`,
        nodeTypeId: projection.nodeRef.nodeTypeId,
        title: projectionTitle(projection),
        icon: projection.icon,
        searchText: catalogSearchText(projection),
        handle: port.handle,
        match: port.match,
        conversionKind: projection.conversion?.kind,
      })),
    )
    .sort(
      (left, right) =>
        connectionMatchRank(left.match) - connectionMatchRank(right.match) ||
        left.title.localeCompare(right.title) ||
        left.key.localeCompare(right.key),
    )
  const output =
    menu.anchor.handle.direction === 'output' && menu.anchor.handle.channel === 'data'
      ? anchorProjection.dataOutputs.find((port) => port.id === menu.anchor.handle.portId)
      : undefined
  const outputExpression = output?.type.expression
  const stateType =
    output?.carrier === 'durable' && outputExpression?.kind === 'ref'
      ? session.authoring?.body.types.find(
          (type) =>
            type.typeRef.typeId === outputExpression.ref.typeId &&
            type.typeRef.semanticDigest === outputExpression.ref.semanticDigest &&
            type.traits.includes('durable') &&
            (type.examples.length > 0 || type.control !== 'object'),
        )
      : undefined
  if (stateType) {
    candidates.unshift({
      key: '__promote-output-to-state__',
      nodeTypeId: '__promote-output-to-state__',
      title: t('workflow.state_panel.promote_action'),
      icon: 'database-plus',
      searchText:
        `${t('workflow.state_panel.promote_action')} ${stateType.titleKey && te(stateType.titleKey) ? t(stateType.titleKey) : stateType.typeRef.typeId}`.toLocaleLowerCase(),
      promoteState: true,
      actionHint: t('workflow.state_panel.promote_candidate_hint'),
    })
  }
  return candidates
})

function connectionMatchRank(match: WorkflowConnectionCandidate['match']): number {
  if (match === 'exact') return 0
  if (match === 'generic-bind') return 1
  if (match === 'assignable') return 2
  return 3
}

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
const stateReferenceLocations = computed<
  Record<string, Array<{ graphId: string; nodeId: string; mode: 'read' | 'write' }>>
>(() => {
  const result: Record<
    string,
    Array<{ graphId: string; nodeId: string; mode: 'read' | 'write' }>
  > = {}
  for (const graph of session.source?.graphs ?? []) {
    for (const node of graph.nodes) {
      if (!node.nodeRef.nodeTypeId.includes('/nodes/state/')) continue
      const variable = node.config.variable
      if (typeof variable !== 'string') continue
      const references = (result[variable] ??= [])
      references.push({
        graphId: graph.id,
        nodeId: node.id,
        mode: node.nodeRef.nodeTypeId.endsWith('/write') ? 'write' : 'read',
      })
    }
  }
  return result
})
const statePromotionError = computed(() => {
  const name = statePromotionName.value.trim()
  if (!/^[A-Za-z0-9_][A-Za-z0-9._-]*$/.test(name))
    return t('workflow.state_panel.promote_invalid_name')
  if (session.source?.variables.some((variable) => variable.name === name))
    return t('workflow.state_panel.promote_duplicate_name')
  return ''
})
const selectedCall = computed(
  () => session.currentGraph?.calls?.find((call) => call.id === selectedNodeId.value) ?? null,
)
const selectedCallGraph = computed(() =>
  selectedCall.value ? (session.calleeGraph(selectedCall.value) ?? null) : null,
)
const selectedCallPorts = computed(() =>
  (selectedCallGraph.value?.inputs ?? []).flatMap((port) => {
    const projection = session.graphInputProjection(selectedCallGraph.value!.id, port.id)
    return projection ? [projection] : []
  }),
)
const selectedProjection = computed(() =>
  selectedNode.value ? (session.nodeInstanceProjection(selectedNode.value) ?? null) : null,
)
const runActive = computed(() =>
  session.activeRun
    ? ['QUEUED', 'RUNNING'].includes(session.activeRun.status.toUpperCase())
    : false,
)
const nodeRunStatusById = computed(() =>
  nodeRunStatuses(session.activeRun, session.currentGraph?.id ?? ''),
)
const nodeDiagnosticSeverityById = computed(() =>
  nodeDiagnosticSeverities(session.diagnostics, session.currentGraph?.id ?? ''),
)
const debugNodeLabels = computed<Record<string, string>>(() =>
  Object.fromEntries(
    (session.source?.graphs ?? []).flatMap((graph) =>
      graph.nodes.map((node) => {
        const projection = session.nodeProjection(node.nodeRef.nodeTypeId)
        return [node.id, node.label || (projection ? projectionTitle(projection) : node.id)]
      }),
    ),
  ),
)
const unhandledRunRoutes = computed(() =>
  session.source ? [...unhandledExecRouteKeys(session.source)] : [],
)
const recordingTargetItems = computed(() =>
  (settings.data?.automation.targets ?? [])
    .filter((target) => target.targetKind === 'desktop-window')
    .map((target) => ({
      label: `${target.label} · ${target.slot}`,
      value: target.slot,
    })),
)
const editorRecording = createEditorRecordingController({
  port: {
    start: (mode, targetSlot) => beginRecording(mode, targetSlot, 'editor'),
    pause: () => recording.pause(),
    resume: () => recording.resume(),
    stop: () => recording.stop(),
    cancel: () => recording.cancel(),
    discard: (pendingID) => recording.discard(pendingID),
    finalize: (input) => recording.finalize(input),
    claimInvocation: (origin) => recording.claimInvocation(origin),
    queryFacets: async (kind) => {
      const page = await assets.query({
        search: '',
        kind,
        category: '',
        tags: [],
        sort: 'created_desc',
        page: 1,
        pageSize: 1,
        thumbnailBudget: 0,
        recentGUIDs: [],
      })
      return {
        categories: page.categories.map((item) => item.value),
        tags: page.tags.map((item) => item.value),
      }
    },
  },
  snapshot: () => ({
    phase: recording.state.phase,
    pending: recording.state.pending,
    invocation: recording.invocation,
  }),
  targets: () => recordingTargetItems.value,
  selectedTargetSlot: () => selectedNode.value?.config.slot,
  importResource: (resource) => importWorkflowResource(resource),
  translate: (key) => t(key),
  showError,
  showStartError: (title, error) => showRecordingStartError(title, error),
})
const recordingEditor = editorRecording.state
const workflowDefaultTargetSlot = computed(
  () => session.source?.targetDefaults?.find((item) => item.target === 'target')?.slot ?? '',
)
const workflowAutomationTargetItems = computed(() =>
  (settings.data?.automation.targets ?? []).map((target) => ({
    label: `${target.label} · ${target.slot}`,
    value: target.slot,
  })),
)
const workflowDefaultTargetLabel = computed(
  () =>
    workflowAutomationTargetItems.value.find(
      (target) => target.value === workflowDefaultTargetSlot.value,
    )?.label ?? t('workflow.target_default.automatic'),
)

function connectedInputIDs(nodeID: string): ReadonlySet<string> {
  return new Set(
    (session.currentGraph?.edges ?? [])
      .filter((edge) => edge.to.nodeId === nodeID && edge.channel === 'data')
      .map((edge) => edge.to.portId),
  )
}

const selectedConnectedInputIDs = computed<ReadonlySet<string>>(() =>
  selectedNode.value ? connectedInputIDs(selectedNode.value.id) : new Set(),
)

function targetSlotForNode(node: Node, projection: NodeProjection): string {
  return effectiveTargetSlot(projection, node, session.source?.targetDefaults ?? [])
}

function setWorkflowDefaultTarget(value: unknown): void {
  session.setTargetDefault('target', typeof value === 'string' ? value : '')
}

watch(
  () => recording.state.pending,
  () =>
    void editorRecording.execute({
      kind: 'sync-pending',
      editorActive: editorViewActive.value,
      editorRoute: route.name === 'workflow-edit',
    }),
  { immediate: true },
)

onActivated(() => {
  editorViewActive.value = true
  if (session.source && !session.dirty) {
    void session
      .refreshIfClean()
      .catch((error) => showError(t('workflow.toast.refresh_failed'), error))
  }
  void editorRecording.execute({
    kind: 'sync-pending',
    editorActive: true,
    editorRoute: route.name === 'workflow-edit',
  })
})
onDeactivated(() => {
  editorViewActive.value = false
})

watch(
  () => recording.completionFailure,
  (failure) => {
    if (failure && recording.invocation === 'editor')
      showError(t('recordingSave.save_failed'), failure.message)
  },
)

onMounted(async () => {
  document.addEventListener('keydown', handleEditorKeydown)
  await Promise.allSettled([
    settings.loaded ? Promise.resolve() : settings.load(),
    recording.reconcile(),
    snippets.load(),
  ])
  const workflowId = String(route.params.id ?? '')
  try {
    await session.load(workflowId)
  } catch {
    return
  }
  unsubscribeRun = onRunChanged((event) => {
    if (event.runId === session.activeRun?.runId) void editorRuns.execute({ kind: 'refresh' })
  })
  unsubscribeDebug = onDebugChanged((event) => {
    if (!session.acceptDebugSnapshot(event.runId, event.snapshot)) return
    if (event.snapshot.status === 'paused') openRuntimeWorkbench('debug')
    if (event.snapshot.status === 'paused' && event.snapshot.nodeId) {
      void focusNode(event.snapshot.graphId ? [event.snapshot.graphId] : [], event.snapshot.nodeId)
    }
  })
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleEditorKeydown)
  unsubscribeRun?.()
  unsubscribeDebug?.()
  editorRuns.dispose()
  clearTimeout(connectionEndTimer)
  stopSidebarResize?.()
})
onBeforeRouteLeave(async () => {
  if (
    recording.state.phase === 'armed' ||
    recording.state.phase === 'countdown' ||
    recording.state.phase === 'recording' ||
    recording.state.phase === 'paused'
  ) {
    const leaveRecording = await confirm({
      title: t('workflow.recording.leave_title'),
      description: t('workflow.recording.leave_hint'),
      confirmText: t('workflow.recording.leave_action'),
      color: 'warning',
    })
    if (leaveRecording !== true) return false
    if (!(await editorRecording.execute({ kind: 'cancel' }))) return false
  }
  if (recordingEditor.pending) {
    const discard = await confirm({
      title: t('recordingSave.discard'),
      description: t('recordingSave.discard_confirm_hint'),
      confirmText: t('common.delete'),
      color: 'error',
    })
    if (discard !== true) return false
    if (!(await editorRecording.execute({ kind: 'discard' }))) return false
  }
  if (!session.dirty) return true
  const decision = await confirm({
    title: t('workflow.editor.leave_title'),
    description: t('workflow.editor.leave_confirm'),
    confirmText: t('workflow.editor.save_and_exit'),
    color: 'primary',
    alternateText: t('workflow.editor.discard_action'),
    alternateValue: 'discard',
    alternateColor: 'error',
  })
  if (decision === true) return (await editorRuns.execute({ kind: 'save' })).ok
  return decision === 'discard'
})

function openRecordingStart(mode: RecordingMode): void {
  void editorRecording.execute({ kind: 'open-start', mode })
}

function openTemplateCapture(): void {
  const targets = recordingTargetItems.value
  if (!targets.length) {
    showError(t('assets.templates.capture_failed'), t('workflow.inspector.no_installed_target'))
    return
  }
  const selectedSlot = selectedNode.value?.config.slot
  captureTargetSlot.value =
    typeof selectedSlot === 'string' && targets.some((item) => item.value === selectedSlot)
      ? selectedSlot
      : workflowDefaultTargetSlot.value || captureTargetSlot.value || targets[0]?.value || ''
  templateCaptureOpen.value = true
}

function captureTemplateForNode(nodeId: string): void {
  selectNodeForContextMenu(nodeId)
  openTemplateCapture()
}

async function captureWorkspaceTemplate(): Promise<void> {
  if (!captureTargetSlot.value) return
  templateCaptureBusy.value = true
  const id = `workflow-template-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
  try {
    const resultPromise = awaitWailsEvent<{
      id: string
      payload?: { cancelled?: boolean; resource?: WorkflowResource }
    }>('tools:picker-result', (payload) => payload?.id === id)
    await backend.tools.openScreenPicker('workflow_resource', id, captureTargetSlot.value)
    templateCaptureOpen.value = false
    const result = await resultPromise
    const resource = result.payload?.resource
    if (!resource || result.payload?.cancelled) return
    importWorkflowResource(resource)
  } catch (error) {
    showError(t('assets.templates.capture_failed'), error)
  } finally {
    templateCaptureBusy.value = false
  }
}

function useWorkspaceResource(
  selection: AssetPickerSelection,
  dropPosition?: { x: number; y: number },
): void {
  const portId =
    selection.kind === 'macro' ? 'macro' : selection.kind === 'clip' ? 'clip' : 'template'
  const current = selectedNode.value
  const projection = current ? session.nodeProjection(current.nodeRef.nodeTypeId) : undefined
  if (
    current &&
    projection?.dataInputs.some(
      (port) =>
        port.id === portId &&
        port.type.representations.some((representation) => representation.kind === 'blob-ref'),
    ) &&
    applyCommand({
      kind: 'bind-blob',
      nodeId: current.id,
      portId,
      blob: { ...selection.blob },
    })
  ) {
    assets.markUsed(selection.guid)
    return
  }

  const nodeTypeId =
    selection.kind === 'macro'
      ? 'https://schemas.yotta.dev/nodes/automation/play-macro'
      : selection.kind === 'clip'
        ? 'https://schemas.yotta.dev/nodes/automation/play-input-clip'
        : 'https://schemas.yotta.dev/nodes/automation/click-template'
  const rect = canvasElement.value?.getBoundingClientRect()
  const position =
    dropPosition ??
    (rect
      ? screenToFlowCoordinate({ x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 })
      : { x: 160, y: 160 })
  const targetSlot =
    workflowDefaultTargetSlot.value ||
    recordingEditor.targetSlot ||
    captureTargetSlot.value ||
    recordingTargetItems.value[0]?.value ||
    ''
  try {
    const ids = session.insertLinearDraft(
      [
        {
          nodeTypeID: nodeTypeId,
          config: targetSlot ? { slot: targetSlot } : {},
          values: {},
          blobs: { [portId]: { ...selection.blob } },
          execInput: 'in',
          execOutput: 'completed',
        },
      ],
      position,
    )
    selectedNodeIds.value = new Set(ids)
    selectedNodeId.value = ids[0] ?? ''
    assets.markUsed(selection.guid)
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

function useWorkflowResource(resource: WorkflowResource, variantId: string): void {
  placeWorkflowResource(resource, variantId, false)
}

function locateBoundResource(location: ResourceLocation): void {
  workspaceSidebarOpen.value = true
  workspacePanel.value = location.kind
  resourceLocateSequence += 1
  resourceLocateRequest.value = {
    ...location,
    requestId: resourceLocateSequence,
  }
}

function importWorkflowResource(
  resource: WorkflowResource,
  position?: { x: number; y: number },
): void {
  const source = session.source
  if (!source) return
  const baseID = resource.id
  let id = baseID
  let suffix = 2
  while (source.resources.some((candidate) => candidate.id === id)) {
    id = `${baseID}-${suffix}`
    suffix++
  }
  const snapshot = { ...plainCopy(resource), id }
  const variantId = snapshot.kind === 'image' ? (snapshot.image?.variants[0]?.id ?? '') : ''
  placeWorkflowResource(snapshot, variantId, true, position)
}

function placeWorkflowResource(
  resource: WorkflowResource,
  variantId: string,
  addResource: boolean,
  requestedPosition?: { x: number; y: number },
): void {
  const portId =
    resource.kind === 'macro' ? 'macro' : resource.kind === 'input-clip' ? 'clip' : 'template'
  const binding = {
    resourceId: resource.id,
    ...(variantId ? { variantId } : {}),
  }
  const current = selectedNode.value
  const projection = current ? session.nodeProjection(current.nodeRef.nodeTypeId) : undefined
  if (current && projection?.dataInputs.some((port) => port.id === portId)) {
    try {
      if (addResource) {
        session.applyBatch([
          { kind: 'add-resource', resource },
          { kind: 'bind-resource', nodeId: current.id, portId, resource: binding },
        ])
      } else {
        session.apply({
          kind: 'bind-resource',
          nodeId: current.id,
          portId,
          resource: binding,
        })
      }
      return
    } catch (error) {
      showError(t('workflow.toast.edit_rejected'), error)
      return
    }
  }

  const nodeTypeId =
    resource.kind === 'macro'
      ? 'https://schemas.yotta.dev/nodes/automation/play-macro'
      : resource.kind === 'input-clip'
        ? 'https://schemas.yotta.dev/nodes/automation/play-input-clip'
        : 'https://schemas.yotta.dev/nodes/automation/click-template'
  const rect = canvasElement.value?.getBoundingClientRect()
  const position =
    requestedPosition ??
    (rect
      ? screenToFlowCoordinate({ x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 })
      : { x: 160, y: 160 })
  const targetSlot =
    workflowDefaultTargetSlot.value ||
    recordingEditor.targetSlot ||
    captureTargetSlot.value ||
    recordingTargetItems.value[0]?.value ||
    ''
  try {
    const ids = session.insertLinearDraft(
      [
        {
          nodeTypeID: nodeTypeId,
          config: targetSlot ? { slot: targetSlot } : {},
          values: {},
          blobs: {},
          resources: { [portId]: binding },
          execInput: 'in',
          execOutput: 'completed',
        },
      ],
      position,
      addResource ? [resource] : [],
    )
    selectedNodeIds.value = new Set(ids)
    selectedNodeId.value = ids[0] ?? ''
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

function updateWorkflowResources(
  payloads: Array<{
    resourceId: string
    name: string
    description: string
    category: string
    tags: string[]
  }>,
): void {
  try {
    session.applyBatch(
      payloads.map((payload) => ({
        kind: 'update-resource-metadata' as const,
        ...payload,
      })),
    )
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

function removeWorkflowResources(resourceIds: string[]): void {
  try {
    session.applyBatch(
      resourceIds.map((resourceId) => ({
        kind: 'remove-resource' as const,
        resourceId,
      })),
    )
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

function openSnippetForNode(node: Node): void {
  const projection = session.nodeProjection(node.nodeRef.nodeTypeId)
  snippetDraft.value = {
    schemaVersion: '1',
    id: '',
    name: node.label || (projection ? projectionTitle(projection) : node.nodeRef.nodeTypeId),
    description: '',
    category: projection?.category ?? '',
    tags: [],
    createdAt: new Date(0).toISOString(),
    updatedAt: new Date(0).toISOString(),
    usageCount: 0,
    payload: {
      nodeRef: plainCopy(node.nodeRef),
      label: node.label,
      config: plainCopy(node.config),
      bindings: plainCopy(node.bindings),
      disabled: node.disabled,
    },
  }
  snippetModalOpen.value = true
}

async function editSnippet(id: string): Promise<void> {
  try {
    snippetDraft.value = await snippets.get(id)
    snippetModalOpen.value = true
  } catch (error) {
    showError(t('workflow.snippets.load_failed'), error)
  }
}

async function saveSnippet(metadata: {
  name: string
  description: string
  category: string
  tags: string[]
  shortcut: string
}): Promise<void> {
  if (!snippetDraft.value || snippetSaveBusy.value) return
  snippetSaveBusy.value = true
  try {
    await snippets.save({ ...plainCopy(snippetDraft.value), ...metadata })
    snippetModalOpen.value = false
    workspacePanel.value = 'snippets'
    workspaceSidebarOpen.value = true
  } catch (error) {
    showError(t('workflow.snippets.save_failed'), error)
  } finally {
    snippetSaveBusy.value = false
  }
}

async function deleteSnippet(id: string): Promise<void> {
  const item = snippets.items.find((candidate) => candidate.id === id)
  const accepted = await confirm({
    title: t('workflow.snippets.delete_title'),
    description: t('workflow.snippets.delete_hint', { name: item?.name ?? id }),
    confirmText: t('workflow.snippets.delete'),
    cancelText: t('common.cancel'),
    color: 'error',
  })
  if (!accepted) return
  try {
    await snippets.remove(id)
  } catch (error) {
    showError(t('workflow.snippets.delete_failed'), error)
  }
}

async function useSnippet(id: string, position?: { x: number; y: number }): Promise<void> {
  try {
    const snippet = await snippets.get(id)
    const current = session.nodeProjection(snippet.payload.nodeRef.nodeTypeId)
    if (!current) throw new Error(t('workflow.snippets.node_unavailable'))
    if (
      current.nodeRef.semanticDigest !== snippet.payload.nodeRef.semanticDigest ||
      current.nodeRef.version !== snippet.payload.nodeRef.version
    ) {
      throw new Error(t('workflow.snippets.contract_changed'))
    }
    const rect = canvasElement.value?.getBoundingClientRect()
    const origin =
      position ??
      (rect
        ? screenToFlowCoordinate({ x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 })
        : { x: 160, y: 160 })
    const [nodeID] = session.insertNodeSelection(
      {
        nodes: [
          {
            id: 'snippet-template',
            nodeRef: plainCopy(snippet.payload.nodeRef),
            label: snippet.payload.label,
            position: { x: 0, y: 0 },
            config: plainCopy(snippet.payload.config),
            bindings: plainCopy(snippet.payload.bindings) as Node['bindings'],
            disabled: snippet.payload.disabled,
          },
        ],
        edges: [],
      },
      origin,
    )
    if (!nodeID) return
    await selectInsertedNodes([nodeID])
    try {
      await snippets.markUsed(id)
    } catch (error) {
      showError(t('workflow.snippets.usage_failed'), error)
    }
  } catch (error) {
    showError(t('workflow.snippets.insert_failed'), error)
  }
}

async function discardPendingRecording(): Promise<void> {
  if (!recordingEditor.pending) return
  const accepted = await confirm({
    title: t('recordingSave.discard'),
    description: t('recordingSave.discard_confirm_hint'),
    confirmText: t('common.delete'),
    color: 'error',
  })
  if (accepted !== true) return
  await editorRecording.execute({ kind: 'discard' })
}

async function openWorkflowResourceEditor(resource: WorkflowResource): Promise<void> {
  await editorResources.execute({ kind: 'open-workflow', resource })
}

async function saveWorkflowMacro(): Promise<void> {
  await editorResources.execute({ kind: 'save-workflow-macro' })
}

async function saveWorkflowClipTrim(): Promise<void> {
  await editorResources.execute({ kind: 'save-workflow-clip' })
}

function duplicateWorkflowResource(resource: WorkflowResource): void {
  try {
    session.apply({ kind: 'add-resource', resource: plainCopy(resource) })
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

async function openMacroEditor(asset: AssetSummary): Promise<void> {
  await editorResources.execute({ kind: 'open-global-macro', asset })
}

async function saveMacro(): Promise<void> {
  await editorResources.execute({ kind: 'save-global-macro' })
}

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
  if (
    session.currentGraph?.kind === 'subgraph' &&
    session.nodeProjection(nodeTypeId)?.instruction.kind === 'run-root'
  )
    return
  const offset = position ? 0 : nextPosition++ * 28
  applyCommand({
    kind: 'add-node',
    nodeTypeId,
    position: position ?? { x: 100 + offset, y: 100 + offset },
  })
}

async function inferGraphInterface(): Promise<void> {
  try {
    if (!canInferGraphInterface.value.valid) {
      toast.add({
        title: t('workflow.graphs.infer_interface_blocked'),
        description: canInferGraphInterface.value.message,
        color: 'warning',
      })
      return
    }
    const preview = session.previewCurrentGraphInterfaceInference()
    const referenced = preview.removed.flatMap((item) =>
      item.kind === 'entry' ? [] : session.currentGraphInterfaceReferences(item.kind, item.id),
    )
    if (referenced.length) {
      toast.add({
        title: t('workflow.graphs.infer_interface_blocked'),
        description: t('workflow.graphs.infer_interface_blocked_hint', {
          count: referenced.length,
        }),
        color: 'warning',
      })
      return
    }
    const accepted = await confirm({
      title: t('workflow.graphs.infer_interface_title'),
      description: t('workflow.graphs.infer_interface_preview', {
        added: preview.added.length,
        removed: preview.removed.length,
      }),
      confirmText: t('workflow.graphs.infer_interface_confirm'),
      cancelText: t('common.cancel'),
      color: 'primary',
    })
    if (!accepted) return
    session.applyCurrentGraphInterfaceInference(preview)
    void fitCurrentGraph()
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

function addGraphInterfaceCandidate(candidateKey: string): void {
  try {
    session.addCurrentGraphInterfaceCandidate(candidateKey)
    void fitCurrentGraph()
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

function renameGraphInterfaceItem(kind: GraphInterfaceItemKind, id: string, name: string): void {
  try {
    session.renameCurrentGraphInterfaceItem(kind, id, name)
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

function moveGraphInterfaceItem(kind: GraphInterfaceItemKind, id: string, direction: -1 | 1): void {
  try {
    session.moveCurrentGraphInterfaceItem(kind, id, direction)
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

function removeGraphInterfaceItem(kind: GraphInterfaceCandidateKind, id: string): void {
  try {
    if (kind !== 'entry') {
      const references = session.currentGraphInterfaceReferences(kind, id)
      if (references.length) {
        toast.add({
          title: t('workflow.graphs.remove_interface_blocked'),
          description: t('workflow.graphs.interface_referenced', {
            count: references.length,
          }),
          color: 'warning',
        })
        return
      }
    }
    session.removeCurrentGraphInterfaceItem(kind, id)
    void fitCurrentGraph()
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

function addGraphCall(graphId: string, requestedPosition?: { x: number; y: number }): void {
  const rect = canvasElement.value?.getBoundingClientRect()
  const position =
    requestedPosition ??
    (rect
      ? screenToFlowCoordinate({ x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 })
      : { x: 160, y: 160 })
  try {
    const callId = session.insertGraphCall(graphId, position)
    selectedNodeIds.value = new Set([callId])
    selectedNodeId.value = callId
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

function duplicateSelectedGraphCall(): void {
  const call = selectedCall.value
  if (!call) return
  try {
    const callId = session.duplicateCurrentGraphCall(call.id)
    selectedNodeIds.value = new Set([callId])
    selectedNodeId.value = callId
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

function forkSelectedGraphCall(): void {
  const call = selectedCall.value
  if (!call) return
  try {
    session.forkCurrentGraphCall(call.id)
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

async function expandSelectedGraphCall(): Promise<void> {
  const call = selectedCall.value
  if (!call) return
  const accepted = await confirm({
    title: t('workflow.graphs.expand_call_title'),
    description: t('workflow.graphs.expand_call_hint'),
    confirmText: t('workflow.graphs.expand_call'),
    cancelText: t('common.cancel'),
    color: 'primary',
  })
  if (!accepted) return
  try {
    const elementIds = session.expandCurrentGraphCall(call.id)
    selectedNodeIds.value = new Set(elementIds)
    selectedNodeId.value = elementIds[0] ?? ''
    void fitCurrentGraph()
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

function graphReaches(graphId: string, targetId: string, visited = new Set<string>()): boolean {
  if (!targetId || visited.has(graphId)) return false
  if (graphId === targetId) return true
  visited.add(graphId)
  const graph = session.source?.graphs.find((candidate) => candidate.id === graphId)
  return Boolean(
    graph?.calls?.some((call) => graphReaches(call.graphId, targetId, new Set(visited))),
  )
}

function graphLabel(graphId: string): string {
  const graph = session.source?.graphs.find((candidate) => candidate.id === graphId)
  return graph?.name || (graph?.kind === 'main' ? t('workflow.graphs.main') : graphId)
}

function openCalledGraph(graphId: string): void {
  const entry = session.source?.entryGraph
  if (!entry) return
  session.openGraphPath(graphId === entry ? [entry] : [entry, graphId])
  clearEditorSelection()
  void fitCurrentGraph()
}

function openGraphAt(index: number): void {
  session.openGraphPath(session.graphPath.slice(0, index + 1))
  clearEditorSelection()
  void fitCurrentGraph()
}

function openGraphDialog(
  mode: 'create' | 'rename',
  graphId = session.currentGraph?.id ?? '',
): void {
  graphDialogMode.value = mode
  graphDialogTargetId.value = mode === 'rename' ? graphId : ''
  graphName.value = mode === 'rename' ? graphLabel(graphId) : ''
  graphDialogOpen.value = true
}

function commitGraphDialog(): void {
  const name = graphName.value.trim()
  if (!name) return
  try {
    if (graphDialogMode.value === 'create') {
      session.createSubgraph(name)
      clearEditorSelection()
      void fitCurrentGraph()
    } else if (graphDialogTargetId.value) session.renameGraph(graphDialogTargetId.value, name)
    graphDialogOpen.value = false
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

async function deleteGraphDefinition(graphId: string): Promise<void> {
  const source = session.source
  const graph = source?.graphs.find((candidate) => candidate.id === graphId)
  if (!source || !graph || graph.kind !== 'subgraph') return
  const definition = projectGraphDefinitions(source).find((candidate) => candidate.id === graphId)
  if (definition?.callCount) {
    toast.add({
      title: t('workflow.graphs.delete_definition'),
      description: t('workflow.graphs.delete_definition_referenced', {
        count: definition.callCount,
      }),
      color: 'warning',
    })
    return
  }
  const accepted = await confirm({
    title: t('workflow.graphs.delete_title'),
    description: t('workflow.graphs.delete_hint', { name: graphLabel(graph.id) }),
    confirmText: t('common.delete'),
    color: 'error',
  })
  if (accepted !== true) return
  try {
    session.removeGraph(graph.id)
    clearEditorSelection()
    void fitCurrentGraph()
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

function duplicateGraphDefinition(graphId: string): void {
  try {
    const copyId = session.duplicateGraphDefinition(graphId)
    openCalledGraph(copyId)
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

async function deleteGraphDefinitionCascade(graphId: string): Promise<void> {
  const source = session.source
  const graph = source?.graphs.find((candidate) => candidate.id === graphId)
  const definition = source
    ? projectGraphDefinitions(source).find((candidate) => candidate.id === graphId)
    : undefined
  if (!source || !graph || graph.kind !== 'subgraph' || !definition?.callCount) return
  const accepted = await confirm({
    title: t('workflow.graphs.delete_definition_cascade_title'),
    description: t('workflow.graphs.delete_definition_cascade_confirm', {
      name: graphLabel(graphId),
      count: definition.callCount,
    }),
    confirmText: t('workflow.graphs.delete_definition_cascade'),
    cancelText: t('common.cancel'),
    color: 'error',
  })
  if (!accepted) return
  try {
    session.removeGraphCascade(graphId)
    clearEditorSelection()
    void fitCurrentGraph()
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

async function locateGraphCall(parentGraphId: string, callId: string): Promise<void> {
  const entry = session.source?.entryGraph
  if (!entry) return
  await focusNode(parentGraphId === entry ? [entry] : [entry, parentGraphId], callId)
}

function addComment(): void {
  const rect = canvasElement.value?.getBoundingClientRect()
  const center = rect
    ? screenToFlowCoordinate({ x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 })
    : { x: 160, y: 160 }
  const position = { x: center.x + 180, y: center.y - 140 }
  const id = session.addAnnotation(position)
  selectedNodeIds.value = new Set([id])
  selectedNodeId.value = id
}

function updateAnnotation(annotation: Annotation): void {
  applyCommand({ kind: 'update-annotation', annotation })
}

function collapseSelection(): void {
  const graph = session.currentGraph
  if (graph) {
    const issue = analyzeCollapseBoundary(graph, selectedNodeIds.value)
    if (issue) {
      const edge =
        issue.kind === 'multiple-entry' ? (issue.edges[1] ?? issue.edges[0]) : issue.edges[0]
      if (edge) selectedEdgeId.value = edgeId(edge)
      const message = t(`workflow.selection.collapse_${issue.kind.replace('-', '_')}`, {
        count: issue.edges.length,
      })
      showError(t('workflow.selection.collapse_rejected'), new Error(message))
      return
    }
  }
  try {
    const callId = session.collapseSelection(
      [...selectedNodeIds.value],
      t('workflow.graphs.default_name'),
    )
    selectedNodeIds.value = new Set([callId])
    selectedNodeId.value = callId
  } catch (error) {
    const reason = collapseSelectionErrorReason(error)
    showError(
      t('workflow.selection.collapse_rejected'),
      new Error(t(`workflow.selection.collapse_${reason}`)),
    )
  }
}

function toggleAIReview(): void {
  aiPanelOpen.value = !aiPanelOpen.value
  if (aiPanelOpen.value) {
    statePanelOpen.value = false
    inspectorSidebarOpen.value = true
  }
}

function toggleStatePanel(): void {
  statePanelOpen.value = !statePanelOpen.value
  if (statePanelOpen.value) {
    aiPanelOpen.value = false
    inspectorSidebarOpen.value = true
  }
}

function setInspectorVisibility(open: boolean): void {
  inspectorAutoOpen.value = open
  inspectorSidebarOpen.value = open
}

function toggleWorkspacePanel(panel: WorkflowWorkspacePanel): void {
  if (workspaceSidebarOpen.value && workspacePanel.value === panel) {
    workspaceSidebarOpen.value = false
    return
  }
  workspacePanel.value = panel
  workspaceSidebarOpen.value = true
  if (workspaceSidebarWidth.value < 280) workspaceSidebarWidth.value = 320
}

function resizeWorkspaceSidebar(startWidth: number, deltaX: number): number {
  return Math.min(480, Math.max(240, startWidth + deltaX))
}

function resizeInspectorSidebar(startWidth: number, deltaX: number): number {
  return Math.min(560, Math.max(280, startWidth + deltaX))
}

function startSidebarResize(side: 'workspace' | 'inspector', event: PointerEvent): void {
  event.preventDefault()
  stopSidebarResize?.()
  const startX = event.clientX
  const startWidth =
    side === 'workspace' ? workspaceSidebarWidth.value : inspectorSidebarWidth.value
  const move = (moveEvent: PointerEvent) => {
    if (side === 'workspace') {
      workspaceSidebarWidth.value = resizeWorkspaceSidebar(startWidth, moveEvent.clientX - startX)
    } else {
      inspectorSidebarWidth.value = resizeInspectorSidebar(startWidth, startX - moveEvent.clientX)
    }
  }
  const stop = () => {
    window.removeEventListener('pointermove', move)
    window.removeEventListener('pointerup', stop)
    stopSidebarResize = undefined
  }
  stopSidebarResize = stop
  window.addEventListener('pointermove', move)
  window.addEventListener('pointerup', stop, { once: true })
}

function openNodeSearch(): void {
  nodeSearchQuery.value = ''
  nodeSearchOpen.value = true
}

function focusFirstNodeSearchResult(): void {
  const first = nodeSearchResults.value[0]
  if (first) void selectNodeSearchResult(first)
}

async function selectNodeSearchResult(result: WorkflowNodeSearchResult): Promise<void> {
  nodeSearchOpen.value = false
  await focusNode([result.graphId], result.nodeId)
}

function handleEditorKeydown(event: KeyboardEvent): void {
  const snippetShortcut = shortcutFromKeyboardEvent(event)
  const snippet =
    canvasPointerInside.value && snippetShortcut
      ? snippets.items.find(
          (item) => item.shortcut?.toLocaleLowerCase() === snippetShortcut.toLocaleLowerCase(),
        )
      : undefined

  const action = resolveEditorKeyboardAction(event, {
    connectionMenuOpen: !!connectionMenu.value,
    canvasPointerInside: canvasPointerInside.value,
    hasNodeSelection: selectedNodeIds.value.size > 0,
    hasSelection: !!(selectedNodeIds.value.size || selectedNodeId.value || selectedEdgeId.value),
    matchedSnippetID: snippet?.id,
  })
  if (!action) return

  event.preventDefault()
  switch (action.kind) {
    case 'close-connection-menu':
      closeConnectionMenu()
      return
    case 'open-quick-add':
      openQuickAdd()
      return
    case 'use-snippet':
      void useSnippet(action.snippetID, canvasInsertionPosition())
      return
    case 'clear-selection':
      clearEditorSelection()
      return
    case 'find-node':
      openNodeSearch()
      return
    case 'copy-selection':
      void copySelection()
      return
    case 'cut-selection':
      void cutSelection()
      return
    case 'paste-selection':
      void pasteSelection()
      return
    case 'duplicate-selection':
      duplicateSelection()
      return
    case 'undo':
      session.undo()
      return
    case 'redo':
      session.redo()
      return
    case 'remove-selection':
      removeSelection()
      return
  }
}

function continueNodeDrag(event: DragEvent): void {
  if (
    !event.dataTransfer?.types.includes(NODE_TYPE_DRAG_FORMAT) &&
    !event.dataTransfer?.types.includes(STATE_REFERENCE_DRAG_FORMAT) &&
    !event.dataTransfer?.types.includes(SNIPPET_DRAG_FORMAT) &&
    !event.dataTransfer?.types.includes(GRAPH_CALL_DRAG_FORMAT) &&
    !event.dataTransfer?.types.includes(RESOURCE_DRAG_FORMAT)
  )
    return
  event.preventDefault()
  event.dataTransfer.dropEffect = 'copy'
  nodeDragActive.value = true
}

function finishNodeDrag(): void {
  nodeDragActive.value = false
}

function dropNode(event: DragEvent): void {
  const nodeTypeId = event.dataTransfer?.getData(NODE_TYPE_DRAG_FORMAT)
  const stateReference = event.dataTransfer?.getData(STATE_REFERENCE_DRAG_FORMAT)
  const snippetID = event.dataTransfer?.getData(SNIPPET_DRAG_FORMAT)
  const graphCallID = event.dataTransfer?.getData(GRAPH_CALL_DRAG_FORMAT)
  const workspaceResource = event.dataTransfer?.getData(RESOURCE_DRAG_FORMAT)
  if (nodeTypeId || stateReference || snippetID || graphCallID || workspaceResource)
    event.preventDefault()
  finishNodeDrag()
  const position = screenToFlowCoordinate({ x: event.clientX, y: event.clientY })
  if (nodeTypeId) {
    addNode(nodeTypeId, position)
    return
  }
  if (snippetID) {
    void useSnippet(snippetID, position)
    return
  }
  if (graphCallID) {
    addGraphCall(graphCallID, position)
    return
  }
  if (workspaceResource) {
    void dropWorkspaceResource(workspaceResource, position)
    return
  }
  if (!stateReference) return
  const parsed = parseStateReferenceDrop(stateReference)
  if (!isStateReferenceDrop(parsed)) return
  insertStateReference(parsed.name, parsed.mode, position)
}

async function dropWorkspaceResource(
  raw: string,
  position: { x: number; y: number },
): Promise<void> {
  const guid = parseWorkspaceResource(raw)
  if (!guid) return
  try {
    importWorkflowResource(await snapshotGlobalAssetByID(guid), position)
    assets.markUsed(guid)
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

function parseStateReferenceDrop(raw: string): unknown {
  try {
    return JSON.parse(raw)
  } catch {
    return null
  }
}

function insertStateReferenceAtCenter(name: string, mode: StateReferenceMode): void {
  const rect = canvasElement.value?.getBoundingClientRect()
  const position = rect
    ? screenToFlowCoordinate({ x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 })
    : { x: 160, y: 160 }
  insertStateReference(name, mode, position)
}

async function locateStateReference(name: string): Promise<void> {
  const reference = stateReferenceLocations.value[name]?.[0]
  if (reference) await locateStateReferenceAt(reference.graphId, reference.nodeId)
}

function stateTypeChangeImpact(name: string, type: TypeExpression) {
  return session.stateTypeChangeImpact(name, structuredClone(type))
}

async function locateStateReferenceAt(graphId: string, nodeId: string): Promise<void> {
  const source = session.source
  if (!source?.graphs.some((graph) => graph.id === graphId)) return
  await focusNode([source.entryGraph, graphId], nodeId)
}

function insertStateReference(
  name: string,
  mode: StateReferenceMode,
  position: { x: number; y: number },
): void {
  try {
    const nodeId = session.insertStateReference(name, mode, position)
    selectedNodeIds.value = new Set([nodeId])
    selectedNodeId.value = nodeId
  } catch (error) {
    showError(t('workflow.state_panel.insert_failed'), error)
  }
}

function isStateReferenceDrop(value: unknown): value is { name: string; mode: 'read' | 'write' } {
  if (!value || typeof value !== 'object') return false
  const candidate = value as Record<string, unknown>
  return (
    typeof candidate.name === 'string' && (candidate.mode === 'read' || candidate.mode === 'write')
  )
}

function connect(connection: Connection): void {
  const graph = session.currentGraph
  const boundary = graph ? graphBoundaryBindingFromConnection(connection, graph) : null
  if (boundary) {
    const compatibility = session.graphBoundaryCompatibility(boundary)
    if (!compatibility.valid) {
      connectionHint.value = connectionIssueText(compatibility)
      return
    }
    try {
      session.bindGraphBoundary(boundary)
      connectionMadeThisGesture = true
      connectionHint.value = ''
    } catch (error) {
      showError(t('workflow.toast.edit_rejected'), error)
    }
    return
  }
  const edge = connectionEdge(connection)
  if (!edge) return
  const compatibility = session.connectionCompatibility(edge)
  if (!compatibility.valid) {
    if (compatibility.conversions?.length && compatibility.sourceType && compatibility.targetType) {
      pendingConversion.value = {
        edge,
        candidates: compatibility.conversions,
        sourceType: compatibility.sourceType,
        targetType: compatibility.targetType,
        position: conversionNodePosition(edge),
      }
      connectionMadeThisGesture = true
      connectionHint.value = ''
      return
    }
    connectionHint.value = connectionIssueText(compatibility)
    return
  }
  if (applyCommand({ kind: 'connect', edge })) {
    connectionMadeThisGesture = true
    connectionHint.value = ''
  }
}

function isValidConnection(connection: Connection): boolean {
  const graph = session.currentGraph
  const boundary = graph ? graphBoundaryBindingFromConnection(connection, graph) : null
  if (boundary) {
    const compatibility = session.graphBoundaryCompatibility(boundary)
    connectionHint.value = compatibility.valid ? '' : connectionIssueText(compatibility)
    return compatibility.valid
  }
  const edge = connectionEdge(connection)
  if (!edge) return false
  const compatibility = session.connectionCompatibility(edge)
  const conversionAvailable = Boolean(compatibility.conversions?.length)
  connectionHint.value =
    compatibility.valid || conversionAvailable ? '' : connectionIssueText(compatibility)
  return compatibility.valid || conversionAvailable
}

function conversionNodePosition(edge: Edge): { x: number; y: number } {
  const graph = session.currentGraph
  const position = (nodeId: string) =>
    graph?.nodes.find((node) => node.id === nodeId)?.position ??
    graph?.calls?.find((call) => call.id === nodeId)?.position
  const source = position(edge.from.nodeId)
  const target = position(edge.to.nodeId)
  if (!source || !target) return { x: 160, y: 160 }
  return { x: (source.x + target.x) / 2, y: (source.y + target.y) / 2 }
}

function conversionTitle(candidate: ConversionCandidatePlan): string {
  if (candidate.titleKey && te(candidate.titleKey)) return t(candidate.titleKey)
  return candidate.nodeTypeId.split('/').at(-1) ?? candidate.nodeTypeId
}

function applyConversion(candidate: ConversionCandidatePlan): void {
  const pending = pendingConversion.value
  if (!pending) return
  try {
    const nodeId = session.insertConversionBridge(pending.edge, candidate, pending.position)
    selectedNodeIds.value = new Set([nodeId])
    selectedNodeId.value = nodeId
    pendingConversion.value = null
    connectionMadeThisGesture = true
  } catch (error) {
    showError(t('workflow.connection.conversion_failed'), error)
  }
}

function cancelConversion(): void {
  pendingConversion.value = null
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
    if (anchor && point && !isGraphBoundaryNodeId(anchor.nodeId)) openConnectionMenu(anchor, point)
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
  connectionError.value = ''
}

function closeConnectionMenu(): void {
  connectionMenu.value = null
  connectionError.value = ''
}

function selectConnectionCandidate(candidate: WorkflowConnectionCandidate): void {
  const menu = connectionMenu.value
  if (!menu) return
  connectionError.value = ''
  const position = { x: menu.flowPosition.x, y: menu.flowPosition.y }
  if (candidate.promoteState) {
    const anchorNode = session.currentGraph?.nodes.find((node) => node.id === menu.anchor.nodeId)
    const projection = anchorNode ? session.nodeInstanceProjection(anchorNode) : undefined
    const output = projection?.dataOutputs.find(
      (port) => menu.anchor.handle.channel === 'data' && port.id === menu.anchor.handle.portId,
    )
    if (!output) return
    pendingStatePromotion.value = {
      nodeId: menu.anchor.nodeId,
      portId: menu.anchor.handle.portId,
      position,
      typeLabel: output.type.label,
    }
    statePromotionName.value = uniqueStateName(menu.anchor.handle.portId)
    closeConnectionMenu()
    return
  }
  if (!candidate.handle) {
    addNode(candidate.nodeTypeId, position)
    closeConnectionMenu()
    return
  }
  try {
    selectedNodeId.value = session.insertConnectedNode(
      menu.anchor.nodeId,
      menu.anchor.handle,
      candidate.nodeTypeId,
      candidate.handle,
      position,
    )
    closeConnectionMenu()
  } catch (error) {
    connectionError.value = error instanceof Error ? error.message : String(error)
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

function uniqueStateName(base: string): string {
  const normalized = base.replace(/[^A-Za-z0-9._-]+/g, '_').replace(/^[^A-Za-z0-9_]+/, '')
  const prefix = normalized || 'value'
  const existing = new Set((session.source?.variables ?? []).map((variable) => variable.name))
  if (!existing.has(prefix)) return prefix
  for (let index = 2; index <= 4096; index++) {
    const candidate = `${prefix}_${index}`
    if (!existing.has(candidate)) return candidate
  }
  return `${prefix}_${Date.now()}`
}

function commitStatePromotion(): void {
  const pending = pendingStatePromotion.value
  if (!pending || statePromotionError.value) return
  try {
    const nodeId = session.promoteOutputToState(
      pending.nodeId,
      pending.portId,
      statePromotionName.value.trim(),
      pending.position,
    )
    selectedNodeIds.value = new Set([nodeId])
    selectedNodeId.value = nodeId
    statePanelOpen.value = true
    cancelStatePromotion()
  } catch (error) {
    showError(t('workflow.state_panel.promote_failed'), error)
  }
}

function cancelStatePromotion(): void {
  pendingStatePromotion.value = null
  statePromotionName.value = ''
}

function handlePaneClick(): void {
  clearEditorSelection()
  closeConnectionMenu()
}

function trackCanvasPointer(event: PointerEvent): void {
  lastCanvasPointer.value = { x: event.clientX, y: event.clientY }
}

function canvasClientPointToFlowPosition(point: { x: number; y: number }): {
  x: number
  y: number
} {
  const rect = canvasElement.value?.getBoundingClientRect()
  const viewport = getViewport()
  const zoom = Number.isFinite(viewport.zoom) && viewport.zoom > 0 ? viewport.zoom : 1
  const x = (point.x - (rect?.left ?? 0) - viewport.x) / zoom
  const y = (point.y - (rect?.top ?? 0) - viewport.y) / zoom
  return { x: Number.isFinite(x) ? x : 160, y: Number.isFinite(y) ? y : 160 }
}

function canvasInsertionPosition(): { x: number; y: number } {
  const rect = canvasElement.value?.getBoundingClientRect()
  return canvasClientPointToFlowPosition(
    lastCanvasPointer.value ?? {
      x: (rect?.left ?? 0) + (rect?.width ?? 320) / 2,
      y: (rect?.top ?? 0) + (rect?.height ?? 320) / 2,
    },
  )
}

function openQuickAdd(): void {
  quickAddPosition.value = canvasInsertionPosition()
  const rect = canvasElement.value?.getBoundingClientRect()
  quickAddAnchor.value = lastCanvasPointer.value ?? {
    x: (rect?.left ?? 0) + (rect?.width ?? 320) / 2,
    y: (rect?.top ?? 0) + (rect?.height ?? 320) / 2,
  }
  quickAddOpen.value = true
}

function openQuickAddFromButton(event: MouseEvent): void {
  const canvasRect = canvasElement.value?.getBoundingClientRect()
  const triggerRect =
    event.currentTarget instanceof HTMLElement ? event.currentTarget.getBoundingClientRect() : null
  quickAddPosition.value = canvasClientPointToFlowPosition({
    x: (canvasRect?.left ?? 0) + (canvasRect?.width ?? 320) / 2,
    y: (canvasRect?.top ?? 0) + (canvasRect?.height ?? 320) / 2,
  })
  quickAddAnchor.value = {
    x: triggerRect?.left ?? canvasRect?.left ?? 8,
    y: (triggerRect?.bottom ?? canvasRect?.top ?? 8) + 8,
  }
  quickAddOpen.value = true
}

function selectQuickAddItem(item: WorkflowQuickAddItem): void {
  const position = { x: quickAddPosition.value.x, y: quickAddPosition.value.y }
  if (item.kind === 'node') addNode(item.id, position)
  else void useSnippet(item.id, position)
}

function captureMarqueeSelection(event: PointerEvent): void {
  const target = event.target as HTMLElement | null
  marqueeSelectionActive =
    event.button === 0 && event.shiftKey && target?.classList.contains('vue-flow__pane') === true
  marqueeSelectionBase = marqueeSelectionActive ? new Set(selectedNodeIds.value) : new Set()
}

function handleCanvasWheel(event: WheelEvent): void {
  const canvas = canvasElement.value
  if (!canvas || !canvasOwnsWheelTarget(event.target)) return
  const rect = canvas.getBoundingClientRect()
  const next = zoomViewportAtPoint(
    getViewport(),
    { x: event.clientX - rect.left, y: event.clientY - rect.top },
    event.deltaY,
    event.deltaMode,
  )
  if (next.zoom === getViewport().zoom) return
  event.preventDefault()
  event.stopPropagation()
  void setViewport(next)
}

async function finishMarqueeSelection(): Promise<void> {
  if (!marqueeSelectionActive) return
  marqueeSelectionActive = false
  await nextTick()
  const merged = mergeMarqueeSelection(marqueeSelectionBase, selectedNodeIds.value)
  marqueeSelectionBase = new Set()
  const nodes = [...merged].flatMap((nodeId) => {
    const node = findNode(nodeId)
    return node ? [node] : []
  })
  if (nodes.length) addSelectedNodes(nodes)
  selectedNodeIds.value = merged
  selectedNodeId.value = [...merged].at(-1) ?? ''
  selectedEdgeId.value = ''
}

function clearEditorSelection(): void {
  marqueeSelectionActive = false
  marqueeSelectionBase = new Set()
  void editorSelection.execute({ kind: 'clear' })
}

function clearRunTrace(): void {
  session.clearRunTrace()
}

function alignSelection(mode: AlignMode): void {
  void editorCanvasLayout.execute({ kind: 'align', mode })
}

function distributeSelection(mode: DistributeMode): void {
  void editorCanvasLayout.execute({ kind: 'distribute', mode })
}

function autoLayout(direction: 'LR' | 'TB'): void {
  void editorCanvasLayout.execute({ kind: 'auto-layout', direction })
}

function removeSelection(): void {
  void editorSelection.execute({ kind: 'remove' })
}

function duplicateSelection(): void {
  void editorSelection.execute({ kind: 'duplicate' })
}

function copySelection(): Promise<void> {
  return editorSelection.execute({ kind: 'copy' })
}

function cutSelection(): Promise<void> {
  return editorSelection.execute({ kind: 'cut' })
}

function pasteSelection(): Promise<void> {
  return editorSelection.execute({ kind: 'paste' })
}

function selectInsertedNodes(nodeIds: string[]): Promise<void> {
  return editorSelection.execute({ kind: 'select-inserted', nodeIds })
}

function connectionEdge(connection: Connection): Edge | null {
  const source = parseGraphHandle(connection.sourceHandle)
  const target = parseGraphHandle(connection.targetHandle)
  if (!source || !target || source.direction !== 'output' || target.direction !== 'input')
    return null
  return {
    channel: source.channel,
    from: { nodeId: connection.source, portId: source.portId },
    to: { nodeId: connection.target, portId: target.portId },
  }
}

function edgeTargetHandle(edge: Edge): string {
  const node = session.currentGraph?.nodes.find((candidate) => candidate.id === edge.to.nodeId)
  const projection = node ? session.nodeInstanceProjection(node) : undefined
  const channel = projection
    ? projectedTargetHandleChannel(projection, edge.channel, edge.to.portId)
    : edge.channel
  return graphHandle(channel, 'input', edge.to.portId)
}

function connectionIssueText(compatibility: {
  issue?: ConnectionIssue
  sourceType?: string
  targetType?: string
}): string {
  if (compatibility.issue === 'type' && compatibility.sourceType && compatibility.targetType) {
    return t('workflow.connection.issue.type_detail', {
      source: compatibility.sourceType,
      target: compatibility.targetType,
    })
  }
  return t(`workflow.connection.issue.${compatibility.issue ?? 'port'}`)
}

function eventClientPoint(event?: MouseEvent | TouchEvent): { x: number; y: number } | null {
  if (!event) return null
  if (event instanceof MouseEvent) return { x: event.clientX, y: event.clientY }
  const touch = event.changedTouches[0]
  return touch ? { x: touch.clientX, y: touch.clientY } : null
}

function disconnect(event: EdgeMouseEvent): void {
  disconnectEdge(event.edge.id)
}

function setEdgeReroutes(edge: Edge, reroutes: Array<{ x: number; y: number }>): void {
  applyCommand({ kind: 'set-edge-reroutes', edge, reroutes })
}

function selectedSourceEdge(): Edge | undefined {
  return session.currentGraph?.edges.find((edge) => edgeId(edge) === selectedEdgeId.value)
}

function addEdgeReroute(): void {
  const edge = selectedSourceEdge()
  const graph = session.currentGraph
  if (!edge || !graph) return
  const position = (id: string) =>
    graph.nodes.find((node) => node.id === id)?.position ??
    graph.calls?.find((call) => call.id === id)?.position
  const from = position(edge.from.nodeId)
  const to = position(edge.to.nodeId)
  if (!from || !to) return
  const reroutes = [...(edge.presentation?.reroutes ?? [])]
  reroutes.push({ x: (from.x + to.x) / 2 + 115, y: (from.y + to.y) / 2 + 45 })
  setEdgeReroutes(edge, reroutes)
}

function clearEdgeReroutes(): void {
  const edge = selectedSourceEdge()
  if (edge) setEdgeReroutes(edge, [])
}

function disconnectEdge(id: string): void {
  const projected = graphBoundaryProjection.value.edges.find((edge) => edge.id === id)
  const boundary = projected ? graphBoundaryKeyFromEdge(projected) : null
  if (boundary) {
    try {
      session.unbindGraphBoundary(boundary)
      selectedEdgeId.value = ''
    } catch (error) {
      showError(t('workflow.toast.edit_rejected'), error)
    }
    return
  }
  const edge = session.currentGraph?.edges.find((candidate) => edgeId(candidate) === id)
  if (edge && applyCommand({ kind: 'disconnect', edge })) selectedEdgeId.value = ''
}

function selectEdge(event: EdgeMouseEvent): void {
  removeSelectedNodes(getSelectedNodes.value)
  selectedNodeId.value = ''
  selectedNodeIds.value = new Set()
  selectedEdgeId.value = event.edge.id
}

function selectNode(event: NodeMouseEvent): void {
  selectedEdgeId.value = ''
  selectedNodeId.value = event.node.id
  statePanelOpen.value = false
  aiPanelOpen.value = false
  if (inspectorAutoOpen.value) inspectorSidebarOpen.value = true
  const source = event.event as MouseEvent | undefined
  if (!source?.shiftKey && !source?.ctrlKey && !source?.metaKey) {
    selectedNodeIds.value = new Set([event.node.id])
  }
}

function selectNodeForContextMenu(nodeId: string): void {
  selectedEdgeId.value = ''
  selectedNodeId.value = nodeId
  statePanelOpen.value = false
  aiPanelOpen.value = false
  if (inspectorAutoOpen.value) inspectorSidebarOpen.value = true
  if (selectedNodeIds.value.has(nodeId)) return
  removeSelectedNodes(getSelectedNodes.value)
  const node = findNode(nodeId)
  if (node) addSelectedNodes([node])
  selectedNodeIds.value = new Set([nodeId])
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
    if (selected.size) selectedEdgeId.value = ''
  }
}

function trackNodeDrag(event: NodeDragEvent): void {
  const positions = editorCanvasLayout.dragPositions(event)
  for (const item of positions) {
    nodeGestures.track(item.nodeId, item.position)
    updateNode(item.nodeId, { position: item.position })
  }
}

function moveNode(event: NodeDragEvent): void {
  const positions = editorCanvasLayout.dragPositions(event)
  for (const item of positions) nodeGestures.track(item.nodeId, item.position)
  editorCanvasLayout.applyPositions(positions)
  for (const item of positions) nodeGestures.clear(item.nodeId)
  void editorCanvasLayout.execute({ kind: 'clear-guides' })
}

function handleEditorToolbarCommand(command: EditorToolbarCommand): void {
  switch (command) {
    case 'undo':
      session.undo()
      return
    case 'redo':
      session.redo()
      return
    case 'find-node':
      openNodeSearch()
      return
    case 'toggle-ai':
      toggleAIReview()
      return
    case 'toggle-state':
      toggleStatePanel()
      return
    case 'toggle-inspector':
      setInspectorVisibility(!inspectorAutoOpen.value)
      return
    case 'check-workflow':
      void editorRuns.execute({ kind: 'check-workflow' })
      return
    case 'toggle-diagnostics':
      toggleRuntimeWorkbench('diagnostics')
      return
    case 'toggle-timeline':
      toggleRuntimeWorkbench('timeline')
      return
    case 'toggle-debugger':
      toggleRuntimeWorkbench('debug')
      return
    case 'start-debug':
      void editorRuns.execute({ kind: 'start-debug', breakpoints: debugBreakpoints() })
      return
    case 'pause-recording':
      void editorRecording.execute({ kind: 'pause' })
      return
    case 'resume-recording':
      void editorRecording.execute({ kind: 'resume' })
      return
    case 'stop-recording':
      void editorRecording.execute({ kind: 'stop' })
      return
    case 'run':
      void editorRuns.execute({ kind: 'start' })
      return
    case 'stop':
      void editorRuns.execute({ kind: 'cancel' })
      return
    case 'save':
      void editorRuns.execute({ kind: 'save' })
      return
    case 'settings':
      void openWorkflowSettings()
      return
    case 'reload':
      void reloadWorkflow()
  }
}

async function openWorkflowSettings(): Promise<void> {
  const source = session.source
  if (!source) return
  workflowMetadata.name = source.workflow.name
  workflowMetadata.description = ''
  workflowMetadata.category = ''
  workflowMetadata.tags = []
  workflowSettingsError.value = ''
  workflowSettingsOpen.value = true
  workflowSettingsBusy.value = true
  try {
    const metadata = await workflowTransport.getSource(session.workflowId)
    workflowMetadata.name = metadata.name || source.workflow.name
    workflowMetadata.description = metadata.description ?? ''
    workflowMetadata.category = metadata.category ?? ''
    workflowMetadata.tags = [...(metadata.tags ?? [])]
  } catch (error) {
    workflowSettingsError.value = errorMessage(error)
  } finally {
    workflowSettingsBusy.value = false
  }
}

async function saveWorkflowSettings(draft: WorkflowMetadataDraft): Promise<void> {
  if (workflowSettingsBusy.value) return
  workflowSettingsBusy.value = true
  workflowSettingsError.value = ''
  try {
    if (session.dirty && !(await editorRuns.execute({ kind: 'save' })).ok) {
      workflowSettingsError.value = t('workflow.editor.metadata_save_blocked')
      return
    }
    await workflowTransport.updateSourceMetadata(session.workflowId, session.baseRevision, draft)
    await session.load(session.workflowId)
    workflowSettingsOpen.value = false
  } catch (error) {
    workflowSettingsError.value = errorMessage(error)
  } finally {
    workflowSettingsBusy.value = false
  }
}

async function reloadWorkflow(): Promise<void> {
  if (session.dirty) {
    const accepted = await confirm({
      title: t('workflow.editor.discard_title'),
      description: t('workflow.editor.discard_confirm'),
      confirmText: t('workflow.editor.discard_action'),
      color: 'warning',
    })
    if (accepted !== true) return
  }
  try {
    await session.load(session.workflowId)
    selectedNodeId.value = ''
    selectedNodeIds.value = new Set()
    selectedEdgeId.value = ''
  } catch (error) {
    showError(t('workflow.toast.refresh_failed'), error)
  }
}

async function locateSaveError(): Promise<void> {
  const target = session.saveErrorTarget
  if (!target) return
  await focusNode([target.graphId], target.nodeId)
}

async function acceptAIProposal(): Promise<void> {
  selectedNodeId.value = ''
  try {
    await session.load(session.workflowId)
  } catch (error) {
    showError(t('workflow.ai.refresh_failed'), error)
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
    return {
      graphId: key.slice(0, separator),
      nodeId: key.slice(separator + 1),
    } as DebugBreakpoint
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

function openRuntimeWorkbench(tab: EditorRuntimeWorkbenchTab): void {
  runtimeWorkbenchTab.value = tab
  runtimeWorkbenchOpen.value = true
}

function toggleRuntimeWorkbench(tab: EditorRuntimeWorkbenchTab): void {
  if (runtimeWorkbenchOpen.value && runtimeWorkbenchTab.value === tab) {
    runtimeWorkbenchOpen.value = false
    return
  }
  openRuntimeWorkbench(tab)
}

async function exportRunTimeline(): Promise<void> {
  const run = session.activeRun
  if (!run || timelineExporting.value) return
  const destination = await workflowTransport.chooseRunTimelineDestination(
    `yotta-run-${run.runId}.json`,
  )
  if (!destination) return
  timelineExporting.value = true
  try {
    const result = await workflowTransport.exportRunTimeline(run.runId, destination)
    showSuccess(t('workflow.timeline.export_succeeded', { count: result.entries }))
  } catch (error) {
    showError(t('workflow.timeline.export_failed'), error)
  } finally {
    timelineExporting.value = false
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
  statePanelOpen.value = false
  aiPanelOpen.value = false
  if (inspectorAutoOpen.value) inspectorSidebarOpen.value = true
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
    description: errorMessage(error),
    color: 'error',
  })
}

function showSuccess(title: string): void {
  toast.add({ title, color: 'success' })
}

function plainCopy<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}
</script>

<style scoped src="./WorkflowEditorView.css"></style>
