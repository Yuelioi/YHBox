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
        :recording-phase="recording.state.phase"
        @back="router.push('/workflows')"
        @rename="renameWorkflow"
        @undo="session.undo()"
        @redo="session.redo()"
        @find-node="openNodeSearch"
        @toggle-ai="toggleAIReview"
        @toggle-state="toggleStatePanel"
        @compile="compile"
        @toggle-diagnostics="diagnosticsOpen = !diagnosticsOpen"
        @toggle-timeline="toggleRuntimeWorkbench('timeline')"
        @toggle-debugger="toggleRuntimeWorkbench('debug')"
        @start-debug="startDebug"
        @start-recording="openRecordingStart"
        @pause-recording="pauseRecording"
        @resume-recording="resumeRecording"
        @stop-recording="stopRecording"
        @run="startRun"
        @stop="cancelRun"
        @save="save"
      />

      <div class="flex items-center gap-2 border-b border-default bg-default px-3 py-1.5">
        <div class="flex min-w-0 flex-1 items-center gap-1">
          <template v-for="(graphId, index) in session.graphPath" :key="`${index}:${graphId}`">
            <UIcon v-if="index" name="i-tabler-chevron-right" class="size-3 text-dimmed" />
            <UButton
              :data-testid="`workflow-graph-breadcrumb-${graphId}`"
              color="neutral"
              variant="ghost"
              size="xs"
              :label="graphLabel(graphId)"
              @click="openGraphAt(index)"
            />
          </template>
        </div>
        <div class="flex shrink-0 items-center gap-2 border-l border-default pl-3">
          <span class="hidden whitespace-nowrap text-xs text-muted xl:inline">{{
            t('workflow.target_default.label')
          }}</span>
          <AdaptiveSelect
            :model-value="workflowDefaultTargetSlot"
            :items="workflowAutomationTargetItems"
            value-key="value"
            label-key="label"
            class="min-w-64"
            :placeholder="t('workflow.target_default.placeholder')"
            @update:model-value="setWorkflowDefaultTarget"
          />
          <UButton
            v-if="workflowDefaultTargetSlot"
            icon="i-tabler-x"
            color="neutral"
            variant="ghost"
            size="xs"
            :aria-label="t('workflow.target_default.clear')"
            @click="session.setTargetDefault('target', '')"
          />
        </div>
        <UDropdownMenu :items="graphMenuItems">
          <UButton
            icon="i-tabler-folders"
            color="neutral"
            variant="ghost"
            size="xs"
            :label="t('workflow.graphs.all')"
          />
        </UDropdownMenu>
        <UDropdownMenu :items="callMenuItems">
          <UButton
            data-testid="workflow-graph-add-call"
            icon="i-tabler-library-plus"
            color="neutral"
            variant="ghost"
            size="xs"
            :label="t('workflow.graphs.add_call')"
            :disabled="callableGraphs.length === 0"
          />
        </UDropdownMenu>
        <UButton
          data-testid="workflow-annotation-add"
          icon="i-tabler-note"
          color="neutral"
          variant="ghost"
          size="xs"
          :aria-label="t('workflow.graphs.add_comment')"
          @click="addComment"
        />
        <UButton
          v-if="session.currentGraph?.kind === 'subgraph'"
          data-testid="workflow-graph-infer-interface"
          icon="i-tabler-plug-connected"
          color="neutral"
          variant="ghost"
          size="xs"
          :label="t('workflow.graphs.infer_interface')"
          @click="inferGraphInterface"
        />
        <UButton
          data-testid="workflow-graph-rename"
          icon="i-tabler-pencil"
          color="neutral"
          variant="ghost"
          size="xs"
          :aria-label="t('common.rename')"
          @click="openGraphDialog('rename')"
        />
        <UButton
          v-if="session.currentGraph?.kind === 'subgraph'"
          icon="i-tabler-trash"
          color="error"
          variant="ghost"
          size="xs"
          :aria-label="t('common.delete')"
          @click="deleteCurrentGraph"
        />
        <UButton
          data-testid="workflow-graph-new"
          icon="i-tabler-plus"
          color="neutral"
          variant="soft"
          size="xs"
          :label="t('workflow.graphs.new')"
          @click="openGraphDialog('create')"
        />
      </div>

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
        v-if="session.saveConflict"
        data-testid="workflow-save-error"
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
        <nav
          class="flex w-11 shrink-0 flex-col items-center gap-1 border-r border-default bg-elevated/20 py-2"
          :aria-label="t('workflow.workspace_tools')"
        >
          <UButton
            data-testid="workflow-workspace-nodes"
            icon="i-tabler-topology-star-3"
            color="neutral"
            :variant="workspacePanel === 'nodes' ? 'soft' : 'ghost'"
            size="sm"
            :aria-label="t('workflow.editor.node_catalog')"
            :aria-pressed="workspacePanel === 'nodes'"
            @click="workspacePanel = 'nodes'"
          />
          <UButton
            data-testid="workflow-workspace-resources"
            icon="i-tabler-library"
            color="neutral"
            :variant="workspacePanel === 'resources' ? 'soft' : 'ghost'"
            size="sm"
            :aria-label="t('workflow.resources.title')"
            :aria-pressed="workspacePanel === 'resources'"
            @click="workspacePanel = 'resources'"
          />
          <UButton
            data-testid="workflow-workspace-snippets"
            icon="i-tabler-bookmarks"
            color="neutral"
            :variant="workspacePanel === 'snippets' ? 'soft' : 'ghost'"
            size="sm"
            :aria-label="t('workflow.snippets.title')"
            :aria-pressed="workspacePanel === 'snippets'"
            @click="workspacePanel = 'snippets'"
          />
        </nav>

        <aside
          class="flex shrink-0 flex-col border-r border-default bg-default"
          :class="workspacePanel === 'nodes' ? 'w-56' : 'w-80'"
        >
          <template v-if="workspacePanel === 'nodes'">
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
              <div v-if="!catalogGroups.length" class="px-3 py-10 text-center">
                <UIcon name="i-tabler-search-off" class="mx-auto mb-2 size-5 text-dimmed" />
                <p class="text-xs text-muted">{{ t('workflow.catalog.no_results') }}</p>
              </div>
            </div>
            <div class="border-t border-default px-3 py-2 font-mono text-[10px] text-dimmed">
              {{ session.authoring.projectionDigest.slice(0, 24) }}
            </div>
          </template>
          <WorkflowResourceDock
            v-else-if="workspacePanel === 'resources'"
            :recording-phase="recording.state.phase"
            @start-recording="openRecordingStart"
            @capture-template="openTemplateCapture"
            @open-library="router.push('/assets')"
            @use="useWorkspaceResource"
          />
          <WorkflowSnippetDock
            v-else
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
                :debug-current="
                  isDebugCurrent(session.currentGraph?.id ?? '', slotProps.data.node.id)
                "
                :connected-input-ids="connectedInputIDs(slotProps.data.node.id)"
                :target-slot="targetSlotForNode(slotProps.data.node, slotProps.data.projection)"
                @command="applyCommand"
                @toggle-breakpoint="
                  toggleBreakpoint(session.currentGraph?.id ?? '', slotProps.data.node.id)
                "
                @save-snippet="openSnippetForNode(slotProps.data.node)"
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
                <UIcon
                  :name="
                    session.currentGraph?.kind === 'subgraph'
                      ? 'i-tabler-folders'
                      : 'i-tabler-player-play'
                  "
                  class="size-5"
                />
              </div>
              <h2 class="text-sm font-semibold text-highlighted">
                {{
                  t(
                    session.currentGraph?.kind === 'subgraph'
                      ? 'workflow.empty_canvas.subgraph_title'
                      : 'workflow.empty_canvas.title',
                  )
                }}
              </h2>
              <p class="mt-2 text-xs leading-5 text-muted">
                {{
                  t(
                    session.currentGraph?.kind === 'subgraph'
                      ? 'workflow.empty_canvas.subgraph_description'
                      : 'workflow.empty_canvas.description',
                  )
                }}
              </p>
              <UButton
                v-if="session.currentGraph?.kind === 'main'"
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
          @update="applyCommand({ kind: 'update-graph-call', call: $event })"
          @open="openCalledGraph(selectedCallGraph.id)"
          @remove="applyCommand({ kind: 'remove-graph-call', callId: selectedCall.id })"
        />
        <WorkflowGraphInterfacePanel
          v-else-if="session.currentGraph?.kind === 'subgraph' && !selectedNodeId"
          :graph="session.currentGraph"
          @infer="inferGraphInterface"
        />
        <WorkflowInspector
          v-else
          :node="selectedNode"
          :projection="selectedProjection"
          :variables="session.source?.variables ?? []"
          :target-defaults="session.source?.targetDefaults ?? []"
          :types="session.authoring?.body.types ?? []"
          :connected-input-ids="selectedConnectedInputIDs"
          @command="applyCommand"
        />
      </div>

      <WorkflowRuntimeWorkbench
        v-model:open="runtimeWorkbenchOpen"
        v-model:tab="runtimeWorkbenchTab"
        :run="session.activeRun"
        :snapshot="session.debugSnapshot"
        :debug-busy="debugControlBusy"
        :node-labels="debugNodeLabels"
        @cancel="cancelRun"
        @refresh="refreshRun"
        @page="loadTimelinePage"
        @focus-node="focusNode"
        @continue="controlDebug('continue')"
        @pause="controlDebug('pause')"
        @step="controlDebug('step')"
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
      v-model:open="recordingStartOpen"
      :title="
        t(
          recordingMode === 'simple'
            ? 'workflow.recording.macro_title'
            : 'workflow.recording.precise_title',
        )
      "
      :icon="recordingMode === 'simple' ? 'i-tabler-list-details' : 'i-tabler-route-alt-left'"
      size="md"
    >
      <div class="space-y-3">
        <div class="flex items-start gap-3 rounded-lg border border-default bg-elevated/30 p-3">
          <UIcon
            :name="recordingMode === 'simple' ? 'i-tabler-list-details' : 'i-tabler-route-alt-left'"
            class="mt-0.5 size-5 shrink-0 text-primary"
          />
          <p class="text-xs leading-5 text-muted">
            {{
              t(
                recordingMode === 'simple'
                  ? 'workflow.recording.macro_hint'
                  : 'workflow.recording.precise_hint',
              )
            }}
          </p>
        </div>
        <UFormField :label="t('workflow.recording.target')" required>
          <AdaptiveSelect
            v-model="recordingTargetSlot"
            :items="recordingTargetItems"
            value-key="value"
            label-key="label"
            :placeholder="t('assets.target_placeholder')"
          />
        </UFormField>
      </div>
      <p class="mt-3 text-xs leading-5 text-muted">{{ t('workflow.recording.start_hint') }}</p>
      <template #footer>
        <UButton color="neutral" variant="ghost" @click="recordingStartOpen = false">
          {{ t('common.cancel') }}
        </UButton>
        <UButton
          :disabled="!recordingTargetSlot"
          :loading="recordingControlBusy"
          @click="startRecording"
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
      :open="!!pendingRecording"
      :title="t('workflow.recording.preview_title')"
      icon="i-tabler-list-check"
      size="5xl"
      tall
      :show-close="false"
      :dismissible="false"
    >
      <div v-if="pendingRecording" class="space-y-4">
        <div class="grid grid-cols-2 gap-3">
          <div class="rounded-lg border border-default bg-elevated/35 px-4 py-3">
            <p class="text-xs text-muted">{{ t('workflow.recording.result_mode') }}</p>
            <div class="mt-1 flex items-center gap-2">
              <UBadge
                :color="pendingRecording.preview.mode === 'simple' ? 'primary' : 'warning'"
                variant="soft"
              >
                {{ t(`workflow.recording.mode_${pendingRecording.preview.mode}`) }}
              </UBadge>
              <span class="text-xs text-toned">
                {{
                  t('recordingSave.summary', {
                    duration: formatRecordingDuration(pendingRecording.durationUs),
                    count: pendingRecording.eventCount,
                  })
                }}
              </span>
            </div>
          </div>
          <div class="rounded-lg border border-default bg-elevated/35 px-4 py-3 text-xs text-muted">
            {{
              t('workflow.recording.action_summary', {
                keys: pendingRecording.preview.keyActions,
                clicks: pendingRecording.preview.clickActions,
                moves: pendingRecording.preview.pointerMoves + pendingRecording.preview.rawDeltas,
                scrolls: pendingRecording.preview.scrollActions,
              })
            }}
          </div>
        </div>
        <div
          v-if="pendingRecording.mode === 'simple' && pendingRecording.preview.steps.length"
          class="max-h-48 space-y-1 overflow-y-auto rounded-lg border border-default bg-sunken p-2"
        >
          <div
            v-for="(step, index) in pendingRecording.preview.steps"
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
          v-if="pendingRecording.mode === 'precise'"
          :preview="pendingRecording.preview"
          :environment="pendingRecording.environment"
          :duration-us="pendingRecording.durationUs"
          :trim-start-us="recordingTrimStartUs"
          :trim-end-us="recordingTrimEndUs"
          :pending-id="pendingRecording.pendingID"
          editable-trim
          @update:trim-start-us="recordingTrimStartUs = $event"
          @update:trim-end-us="recordingTrimEndUs = $event"
        />
        <MacroActionEditor
          v-if="pendingRecording.mode === 'simple' && pendingRecording.actions"
          v-model="recordingActions"
          @validity="recordingActionsValid = $event"
        />
        <p
          v-else-if="pendingRecording.mode === 'simple'"
          class="rounded-lg border border-default bg-sunken px-3 py-2 text-xs text-muted"
        >
          {{ t('recordingEditor.editing_unavailable') }}
        </p>
        <UFormField :label="t('recordingSave.name')" required>
          <UInput v-model="recordingDraft.name" maxlength="80" autofocus />
        </UFormField>
        <UFormField :label="t('common.description')" :hint="t('common.optional')">
          <UTextarea v-model="recordingDraft.description" :rows="2" />
        </UFormField>
        <div class="grid grid-cols-2 gap-3">
          <UFormField :label="t('common.category')" :hint="t('common.optional')">
            <UInput v-model="recordingDraft.category" />
          </UFormField>
          <UFormField :label="t('common.tags')" :hint="t('assets.tags_hint')">
            <UInput v-model="recordingDraft.tags" />
          </UFormField>
        </div>
      </div>
      <template #footer>
        <UButton
          color="error"
          variant="ghost"
          :disabled="recordingSaveBusy"
          @click="discardPendingRecording"
        >
          {{ t('recordingSave.discard') }}
        </UButton>
        <UButton
          :loading="recordingSaveBusy"
          :disabled="
            !recordingDraft.name.trim() ||
            (pendingRecording?.mode === 'simple' && !recordingActionsValid)
          "
          @click="saveRecordingResource"
        >
          {{ t('assets.recording.save_to_library') }}
        </UButton>
      </template>
    </BaseModal>
    <WorkflowSnippetModal
      :open="snippetModalOpen"
      :snippet-id="snippetDraft?.id ?? ''"
      :node-type-id="snippetDraft?.payload.nodeRef.nodeTypeId ?? ''"
      :initial="snippetModalInitial"
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
  onBeforeUnmount,
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
import {
  type Edge,
  type EditorCommand,
  type Node,
  type NodeProjection,
  type StateReferenceMode,
} from '@/app/editor/EditorSession'
import type { Annotation, GraphCall } from '../../../contracts/workflow/3.1/workflow-source'
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
import WorkflowNode from '@/app/editor/WorkflowNode.vue'
import { effectiveTargetSlot } from '@/app/editor/authoringSurface'
import WorkflowInspector from '@/app/editor/WorkflowInspector.vue'
import AIWorkflowReviewPanel from '@/app/editor/AIWorkflowReviewPanel.vue'
import WorkflowDiagnosticsPanel from '@/app/editor/WorkflowDiagnosticsPanel.vue'
import WorkflowEditorToolbar from '@/app/editor/WorkflowEditorToolbar.vue'
import WorkflowResourceDock from '@/app/editor/WorkflowResourceDock.vue'
import WorkflowSnippetDock from '@/app/editor/WorkflowSnippetDock.vue'
import WorkflowSnippetModal from '@/app/editor/WorkflowSnippetModal.vue'
import WorkflowRuntimeWorkbench from '@/app/editor/WorkflowRuntimeWorkbench.vue'
import WorkflowStatePanel from '@/app/editor/WorkflowStatePanel.vue'
import WorkflowConnectionMenu, {
  type WorkflowConnectionCandidate,
} from '@/app/editor/WorkflowConnectionMenu.vue'
import WorkflowSelectionToolbar from '@/app/editor/WorkflowSelectionToolbar.vue'
import WorkflowGraphCall from '@/app/editor/WorkflowGraphCall.vue'
import WorkflowGraphCallInspector from '@/app/editor/WorkflowGraphCallInspector.vue'
import WorkflowGraphBoundary from '@/app/editor/WorkflowGraphBoundary.vue'
import WorkflowGraphInterfacePanel from '@/app/editor/WorkflowGraphInterfacePanel.vue'
import WorkflowAnnotation from '@/app/editor/WorkflowAnnotation.vue'
import WorkflowRerouteEdge from '@/app/editor/WorkflowRerouteEdge.vue'
import BaseModal from '@/components/common/BaseModal.vue'
import {
  useRecordingStore,
  type MacroAction,
  type RecordingMode,
  type RecordingStopPayload,
} from '@/stores/recording'
import { useSettingsStore } from '@/stores/settings'
import { useAssetsStore, type AssetPickerSelection } from '@/stores/assets'
import { backend, type WorkflowSnippet } from '@/lib/backend'
import { useSnippetsStore } from '@/stores/snippets'
import { errorMessage } from '@/lib/invoke'
import { awaitWailsEvent } from '@/composables/useWailsEvent'
import { nodeRunStatuses } from '@/app/editor/runTrace'
import { nodeDiagnosticSeverities, type WorkflowDiagnostic } from '@/app/editor/workflowDiagnostics'
import {
  compatibleCandidatePorts,
  type ConversionCandidatePlan,
  type ConnectionIssue,
} from '@/app/editor/connectionCompatibility'
import {
  createWorkflowNodeGestureState,
  projectWorkflowFlowNodes,
} from '@/app/editor/workflowFlowProjection'
import {
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

const MacroActionEditor = defineAsyncComponent(
  () => import('@/components/recording/MacroActionEditor.vue'),
)
const PreciseRecordingWorkbench = defineAsyncComponent(
  () => import('@/components/recording/PreciseRecordingWorkbench.vue'),
)

const route = useRoute()
const router = useRouter()
const toast = useToast()
const { confirm } = useConfirm()
const { t, te } = useI18n()
const session = createEditorSession(workflowTransport)
const recording = useRecordingStore()
const { start: beginRecording } = useRecordingStart()
const settings = useSettingsStore()
const assets = useAssetsStore()
const snippets = useSnippetsStore()
const selectedNodeId = ref('')
const selectedNodeIds = ref(new Set<string>())
const selectedEdgeId = ref('')
const nodeDragActive = ref(false)
const aiPanelOpen = ref(false)
const statePanelOpen = ref(false)
const catalogQuery = ref('')
const workspacePanel = ref<'nodes' | 'resources' | 'snippets'>('nodes')
const graphDialogOpen = ref(false)
const graphDialogMode = ref<'create' | 'rename'>('create')
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
const compileSucceeded = ref(false)
const saveSucceeded = ref(false)
const canvasElement = ref<HTMLElement | null>(null)
const connectionStart = ref<ConnectionAnchor | null>(null)
const connectionMenu = ref<ConnectionMenuState | null>(null)
const pendingConversion = ref<PendingConversion | null>(null)
const pendingStatePromotion = ref<PendingStatePromotion | null>(null)
const statePromotionName = ref('')
const connectionHint = ref('')
const connectionError = ref('')
const snapGuides = ref<{ x?: number; y?: number }>({})
const layouting = ref(false)
const minimapOpen = ref(false)
const diagnosticsOpen = ref(false)
type RuntimeWorkbenchTab = 'logs' | 'timeline' | 'debug'
const runtimeWorkbenchOpen = ref(false)
const runtimeWorkbenchTab = ref<RuntimeWorkbenchTab>('logs')
const debugControlBusy = ref(false)
const runTimelineOpen = computed(
  () => runtimeWorkbenchOpen.value && runtimeWorkbenchTab.value === 'timeline',
)
const debuggerOpen = computed(
  () => runtimeWorkbenchOpen.value && runtimeWorkbenchTab.value === 'debug',
)
const breakpointKeys = ref(new Set<string>())
const recordingStartOpen = ref(false)
const recordingTargetSlot = ref('')
const recordingMode = ref<RecordingMode>('simple')
const recordingControlBusy = ref(false)
const pendingRecording = ref<RecordingStopPayload | null>(null)
const recordingSaveBusy = ref(false)
const recordingActions = ref<MacroAction[]>([])
const recordingActionsValid = ref(true)
const recordingTrimStartUs = ref(0)
const recordingTrimEndUs = ref(0)
const recordingDraft = reactive({ name: '', description: '', category: '', tags: '' })
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
let marqueeSelectionBase = new Set<string>()

const NODE_TYPE_DRAG_FORMAT = 'application/x-yotta-node-type'
const STATE_REFERENCE_DRAG_FORMAT = 'application/x-yotta-state-reference'
const SNIPPET_DRAG_FORMAT = 'application/x-yotta-snippet'
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

interface WorkflowSelectionClipboard {
  format: 'yotta.workflow-selection'
  version: 2
  nodes: Node[]
  calls: GraphCall[]
  annotations: Annotation[]
  edges: Edge[]
}

interface WorkflowNodeSearchResult {
  graphId: string
  nodeId: string
  label: string
  icon: string
  searchText: string
}

const graphMenuItems = computed(() => [
  (session.source?.graphs ?? []).map((graph) => ({
    label: graphLabel(graph.id),
    icon: graph.kind === 'main' ? 'i-tabler-home' : 'i-tabler-folders',
    onSelect: () => openCalledGraph(graph.id),
  })),
])

const callableGraphs = computed(() =>
  (session.source?.graphs ?? []).filter(
    (graph) =>
      graph.kind === 'subgraph' &&
      graph.id !== session.currentGraph?.id &&
      !graphReaches(graph.id, session.currentGraph?.id ?? ''),
  ),
)

const callMenuItems = computed(() => [
  callableGraphs.value.map((graph) => ({
    label: graphLabel(graph.id),
    icon: 'i-tabler-folders',
    onSelect: () => addGraphCall(graph.id),
  })),
])

const catalogGroups = computed(() => {
  const query = catalogQuery.value.trim().toLocaleLowerCase()
  const grouped = new Map<string, NodeProjection[]>()
  for (const projection of session.authoring?.body.nodes ?? []) {
    if (session.currentGraph?.kind === 'subgraph' && projection.instruction.kind === 'run-root')
      continue
    if (!visibleForCreationTemplate(projection)) continue
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
  const automationCapabilities = projection.capabilities.filter((capability) =>
    capability.capability.capabilityId.includes('/capabilities/automation/'),
  )
  return automationCapabilities.every((capability) => capability.targetKinds.includes(targetKind))
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

const flowEdges = computed<FlowEdge[]>(() => [
  ...(session.currentGraph?.edges ?? []).map((edge) => {
    const visual = workflowEdgeVisualState(edge, nodeRunStatusById.value)
    return {
      id: edgeId(edge),
      source: edge.from.nodeId,
      target: edge.to.nodeId,
      sourceHandle: graphHandle(edge.channel, 'output', edge.from.portId),
      targetHandle: graphHandle(edge.channel, 'input', edge.to.portId),
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
const recordingTargetItems = computed(() =>
  (settings.data?.automation.targets ?? [])
    .filter((target) => target.targetKind === 'desktop-window')
    .map((target) => ({
      label: `${target.label} · ${target.slot}`,
      value: target.slot,
    })),
)
const workflowDefaultTargetSlot = computed(
  () => session.source?.targetDefaults?.find((item) => item.target === 'target')?.slot ?? '',
)
const workflowAutomationTargetItems = computed(() =>
  (settings.data?.automation.targets ?? []).map((target) => ({
    label: `${target.label} · ${target.slot}`,
    value: target.slot,
  })),
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
  (pending) => {
    if (
      !pending ||
      route.name !== 'workflow-edit' ||
      (recording.invocation && recording.invocation !== 'editor')
    )
      return
    recording.claimInvocation('editor')
    openRecordingPreview(pending)
  },
  { immediate: true },
)

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
  ])
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
  clearTimeout(compileFlashTimer)
  clearTimeout(saveFlashTimer)
  clearTimeout(connectionEndTimer)
})
onBeforeRouteLeave(async () => {
  if (recording.state.phase === 'recording' || recording.state.phase === 'paused') {
    const leaveRecording = await confirm({
      title: t('workflow.recording.leave_title'),
      description: t('workflow.recording.leave_hint'),
      confirmText: t('workflow.recording.leave_action'),
      color: 'warning',
    })
    if (leaveRecording !== true) return false
    await recording.cancel()
  }
  if (pendingRecording.value) {
    const discard = await confirm({
      title: t('recordingSave.discard'),
      description: t('recordingSave.discard_confirm_hint'),
      confirmText: t('common.delete'),
      color: 'error',
    })
    if (discard !== true) return false
    await recording.discard(pendingRecording.value.pendingID)
    pendingRecording.value = null
  }
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

function openRecordingStart(mode: RecordingMode): void {
  if (recording.state.phase !== 'idle') {
    if (recording.state.pending) openRecordingPreview(recording.state.pending)
    return
  }
  const targets = recordingTargetItems.value
  if (!targets.length) {
    showError(t('workflow.recording.start_failed'), t('workflow.inspector.no_installed_target'))
    return
  }
  const selectedSlot = selectedNode.value?.config.slot
  recordingMode.value = mode
  recordingTargetSlot.value =
    typeof selectedSlot === 'string' && targets.some((item) => item.value === selectedSlot)
      ? selectedSlot
      : recordingTargetSlot.value || targets[0]?.value || ''
  recordingStartOpen.value = true
}

async function startRecording(): Promise<void> {
  if (!recordingTargetSlot.value) return
  recordingControlBusy.value = true
  try {
    if (await beginRecording(recordingMode.value, recordingTargetSlot.value, 'editor')) {
      recordingStartOpen.value = false
    }
  } catch (error) {
    showError(t('workflow.recording.start_failed'), error)
    return
  } finally {
    recordingControlBusy.value = false
  }
}

async function pauseRecording(): Promise<void> {
  try {
    await recording.pause()
  } catch (error) {
    showError(t('workflow.recording.control_failed'), error)
  }
}

async function resumeRecording(): Promise<void> {
  try {
    await recording.resume()
  } catch (error) {
    showError(t('workflow.recording.control_failed'), error)
  }
}

async function stopRecording(): Promise<void> {
  try {
    const payload = await recording.stop()
    if (payload) openRecordingPreview(payload)
  } catch (error) {
    showError(t('workflow.recording.control_failed'), error)
  }
}

function openRecordingPreview(payload: RecordingStopPayload): void {
  if (pendingRecording.value?.pendingID === payload.pendingID) return
  pendingRecording.value = payload
  recordingActions.value = cloneRecordingActions(payload.actions ?? [])
  recordingActionsValid.value = true
  recordingTrimStartUs.value = 0
  recordingTrimEndUs.value = payload.durationUs
  recordingDraft.name = ''
  recordingDraft.description = ''
  recordingDraft.category = ''
  recordingDraft.tags = ''
}

async function saveRecordingResource(): Promise<void> {
  const pending = pendingRecording.value
  if (!pending || !recordingDraft.name.trim()) return
  recordingSaveBusy.value = true
  try {
    const saved = await recording.finalize({
      pendingID: pending.pendingID,
      label: recordingDraft.name.trim(),
      description: recordingDraft.description.trim(),
      category: recordingDraft.category.trim(),
      tags: splitRecordingTags(recordingDraft.tags),
      actions: pending.actions ? cloneRecordingActions(recordingActions.value) : undefined,
      trimStartUs: pending.mode === 'precise' ? recordingTrimStartUs.value : undefined,
      trimEndUs: pending.mode === 'precise' ? recordingTrimEndUs.value : undefined,
    })
    pendingRecording.value = null
    useWorkspaceResource({
      guid: saved.assetID,
      kind: saved.assetKind,
      name: saved.label,
      blob: { ...saved.blob },
    })
    assets.invalidate()
  } catch (error) {
    showError(t('recordingSave.save_failed'), error)
  } finally {
    recordingSaveBusy.value = false
  }
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

async function captureWorkspaceTemplate(): Promise<void> {
  if (!captureTargetSlot.value) return
  templateCaptureBusy.value = true
  const id = `workflow-template-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
  try {
    const resultPromise = awaitWailsEvent<{
      id: string
      payload?: { cancelled?: boolean; guid?: string }
    }>('tools:picker-result', (payload) => payload?.id === id)
    await backend.tools.openScreenPicker('template_save', id, captureTargetSlot.value)
    templateCaptureOpen.value = false
    const result = await resultPromise
    const guid = result.payload?.guid
    if (!guid || result.payload?.cancelled) return
    assets.invalidate()
    const asset = await backend.assets.get(guid)
    const variant = asset.variants?.[0]
    if (!variant) return
    useWorkspaceResource({
      guid,
      kind: 'template',
      name: asset.name,
      resolution:
        variant.resolution.length === 2
          ? [variant.resolution[0], variant.resolution[1]]
          : undefined,
      blob: { ...variant.blob },
    })
  } catch (error) {
    showError(t('assets.templates.capture_failed'), error)
  } finally {
    templateCaptureBusy.value = false
  }
}

function useWorkspaceResource(selection: AssetPickerSelection): void {
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
  const position = rect
    ? screenToFlowCoordinate({ x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 })
    : { x: 160, y: 160 }
  const targetSlot =
    workflowDefaultTargetSlot.value ||
    recordingTargetSlot.value ||
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
}): Promise<void> {
  if (!snippetDraft.value || snippetSaveBusy.value) return
  snippetSaveBusy.value = true
  try {
    await snippets.save({ ...plainCopy(snippetDraft.value), ...metadata })
    snippetModalOpen.value = false
    workspacePanel.value = 'snippets'
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
  const pending = pendingRecording.value
  if (!pending) return
  const accepted = await confirm({
    title: t('recordingSave.discard'),
    description: t('recordingSave.discard_confirm_hint'),
    confirmText: t('common.delete'),
    color: 'error',
  })
  if (accepted !== true) return
  recordingSaveBusy.value = true
  try {
    await recording.discard(pending.pendingID)
    pendingRecording.value = null
  } catch (error) {
    showError(t('recordingSave.discard_failed'), error)
  } finally {
    recordingSaveBusy.value = false
  }
}

function cloneRecordingActions(actions: MacroAction[]): MacroAction[] {
  return actions.map((action) => ({
    ...action,
    point: action.point ? { ...action.point } : undefined,
  }))
}

function splitRecordingTags(value: string): string[] {
  return [
    ...new Set(
      value
        .split(/[,，]/)
        .map((tag) => tag.trim())
        .filter(Boolean),
    ),
  ]
}

function formatRecordingDuration(durationUs: number): string {
  const seconds = Math.max(0, Math.round(durationUs / 1_000_000))
  return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')}`
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

function inferGraphInterface(): void {
  try {
    session.inferCurrentGraphInterface()
    void fitCurrentGraph()
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

function addGraphCall(graphId: string): void {
  const rect = canvasElement.value?.getBoundingClientRect()
  const position = rect
    ? screenToFlowCoordinate({ x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 })
    : { x: 160, y: 160 }
  try {
    const callId = session.insertGraphCall(graphId, position)
    selectedNodeIds.value = new Set([callId])
    selectedNodeId.value = callId
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

function openGraphDialog(mode: 'create' | 'rename'): void {
  graphDialogMode.value = mode
  graphName.value = mode === 'rename' ? graphLabel(session.currentGraph?.id ?? '') : ''
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
    } else if (session.currentGraph) session.renameGraph(session.currentGraph.id, name)
    graphDialogOpen.value = false
  } catch (error) {
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

async function deleteCurrentGraph(): Promise<void> {
  const graph = session.currentGraph
  if (!graph || graph.kind !== 'subgraph') return
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
    showError(t('workflow.toast.edit_rejected'), error)
  }
}

function toggleAIReview(): void {
  aiPanelOpen.value = !aiPanelOpen.value
  if (aiPanelOpen.value) statePanelOpen.value = false
}

function toggleStatePanel(): void {
  statePanelOpen.value = !statePanelOpen.value
  if (statePanelOpen.value) aiPanelOpen.value = false
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
  if (event.key === 'Escape') {
    if (selectedNodeIds.value.size || selectedNodeId.value || selectedEdgeId.value) {
      event.preventDefault()
      clearEditorSelection()
    }
    return
  }
  const modifier = event.ctrlKey || event.metaKey
  if (modifier && !event.altKey) {
    const key = event.key.toLocaleLowerCase()
    if (key === 'f') {
      event.preventDefault()
      openNodeSearch()
      return
    }
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
  if (!selectedNodeIds.value.size && !selectedNodeId.value && !selectedEdgeId.value) return
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
  if (
    !event.dataTransfer?.types.includes(NODE_TYPE_DRAG_FORMAT) &&
    !event.dataTransfer?.types.includes(STATE_REFERENCE_DRAG_FORMAT) &&
    !event.dataTransfer?.types.includes(SNIPPET_DRAG_FORMAT)
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
  if (nodeTypeId || stateReference || snippetID) event.preventDefault()
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
  if (!stateReference) return
  const parsed = parseStateReferenceDrop(stateReference)
  if (!isStateReferenceDrop(parsed)) return
  insertStateReference(parsed.name, parsed.mode, position)
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

function stateTypeChangeImpact(name: string, typeId: string) {
  const type = session.authoring?.body.types.find(
    (candidate) => candidate.typeRef.typeId === typeId,
  )
  return type
    ? session.stateTypeChangeImpact(name, { kind: 'ref', ref: { ...type.typeRef } })
    : { references: [], issues: [] }
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

function captureMarqueeSelection(event: PointerEvent): void {
  const target = event.target as HTMLElement | null
  marqueeSelectionBase =
    event.button === 0 && event.shiftKey && target?.classList.contains('vue-flow__pane')
      ? new Set(selectedNodeIds.value)
      : new Set()
}

function handleCanvasWheel(event: WheelEvent): void {
  const canvas = canvasElement.value
  if (!canvas) return
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
  if (!marqueeSelectionBase.size) return
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
  removeSelectedNodes(getSelectedNodes.value)
  selectedNodeId.value = ''
  selectedNodeIds.value = new Set()
  selectedEdgeId.value = ''
  marqueeSelectionBase = new Set()
}

function clearRunTrace(): void {
  session.clearRunTrace()
}

async function fitCurrentGraph(): Promise<void> {
  await nextTick()
  await new Promise<void>((resolve) =>
    requestAnimationFrame(() => requestAnimationFrame(() => setTimeout(resolve, 50))),
  )
  await fitView({ padding: 0.24, duration: 180 })
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
    const graph = session.currentGraph
    const position =
      graph?.nodes.find((candidate) => candidate.id === nodeId)?.position ??
      graph?.calls?.find((candidate) => candidate.id === nodeId)?.position ??
      graph?.annotations?.find((candidate) => candidate.id === nodeId)?.position
    return position ? [sizedFlowNode(nodeId, position)] : []
  })
}

function applyCanvasPositions(
  positions: Array<{ nodeId: string; position: { x: number; y: number } }>,
): boolean {
  const graph = session.currentGraph
  if (!graph) return false
  let applied = false
  const nodes = positions.filter((item) => graph.nodes.some((node) => node.id === item.nodeId))
  if (nodes.length) applied = applyCommand({ kind: 'move-nodes', positions: nodes }) || applied
  for (const item of positions) {
    const call = graph.calls?.find((candidate) => candidate.id === item.nodeId)
    if (call)
      applied =
        applyCommand({ kind: 'update-graph-call', call: { ...call, position: item.position } }) ||
        applied
    const annotation = graph.annotations?.find((candidate) => candidate.id === item.nodeId)
    if (annotation)
      applied =
        applyCommand({
          kind: 'update-annotation',
          annotation: { ...annotation, position: item.position },
        }) || applied
  }
  return applied
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
  if (positions.length) applyCanvasPositions(positions)
}

function distributeSelection(mode: DistributeMode): void {
  const positions = distributeNodePositions(selectedSizedNodes(), mode)
  if (positions.length) applyCanvasPositions(positions)
}

async function autoLayout(direction: 'LR' | 'TB'): Promise<void> {
  if (layouting.value) return
  const graph = session.currentGraph
  const source = session.source
  if (!graph || !source || graph.nodes.length + (graph.calls?.length ?? 0) === 0) return
  const nodes = [
    ...graph.nodes.map((node) => sizedFlowNode(node.id, node.position)),
    ...(graph.calls ?? []).map((call) => sizedFlowNode(call.id, call.position)),
  ]
  layouting.value = true
  try {
    const positions = await autoLayoutNodePositions(nodes, graph.edges, direction)
    if (session.source !== source || session.currentGraph?.id !== graph.id) return
    if (applyCanvasPositions(positions)) {
      await fitCurrentGraph()
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
  const graph = session.currentGraph
  const nodeIds = ids.filter((id) => graph?.nodes.some((node) => node.id === id))
  const callIds = ids.filter((id) => graph?.calls?.some((call) => call.id === id))
  const annotationIds = ids.filter((id) =>
    graph?.annotations?.some((annotation) => annotation.id === id),
  )
  if (nodeIds.length) applyCommand({ kind: 'remove-nodes', nodeIds })
  for (const callId of callIds) applyCommand({ kind: 'remove-graph-call', callId })
  for (const annotationId of annotationIds)
    applyCommand({ kind: 'remove-annotation', annotationId })
  if (ids.length) {
    selectedNodeId.value = ''
    selectedNodeIds.value = new Set()
    return
  }
  if (selectedEdgeId.value) disconnectEdge(selectedEdgeId.value)
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
  if (!snapshot.nodes.length && !snapshot.calls.length && !snapshot.annotations.length) return
  workflowClipboard = {
    format: 'yotta.workflow-selection',
    version: 2,
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
    parsed.version !== 2 ||
    !Array.isArray(parsed.nodes) ||
    !Array.isArray(parsed.calls) ||
    !Array.isArray(parsed.annotations) ||
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
  const source = event.event as MouseEvent | undefined
  if (!source?.shiftKey && !source?.ctrlKey && !source?.metaKey) {
    selectedNodeIds.value = new Set([event.node.id])
  }
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
  const positions = dragPositions(event)
  for (const item of positions) {
    nodeGestures.track(item.nodeId, item.position)
    updateNode(item.nodeId, { position: item.position })
  }
}

function moveNode(event: NodeDragEvent): void {
  const positions = dragPositions(event)
  for (const item of positions) nodeGestures.track(item.nodeId, item.position)
  applyCanvasPositions(positions)
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
    await session.run()
    diagnosticsOpen.value = session.diagnostics.length > 0
  } catch (error) {
    showError(t('workflow.toast.run_failed'), error)
  }
}

async function startDebug(): Promise<void> {
  try {
    const run = await session.startDebug(debugBreakpoints())
    diagnosticsOpen.value = session.diagnostics.length > 0
    if (!run) return
    openRuntimeWorkbench('debug')
    const snapshot = session.debugSnapshot
    if (snapshot?.status === 'paused' && snapshot.nodeId) {
      await focusNode(
        snapshot.graphPath ?? (snapshot.graphId ? [snapshot.graphId] : []),
        snapshot.nodeId,
      )
    }
  } catch (error) {
    showError(t('workflow.toast.debug_failed'), error)
  }
}

async function controlDebug(action: 'continue' | 'pause' | 'step'): Promise<void> {
  if (debugControlBusy.value) return
  debugControlBusy.value = true
  try {
    await session.controlDebug(action)
  } catch (error) {
    showError(t('workflow.toast.debug_failed'), error)
  } finally {
    debugControlBusy.value = false
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

async function cancelRun(): Promise<void> {
  try {
    await session.cancelRun()
  } catch (error) {
    showError(t('workflow.toast.stop_failed'), error)
  }
}

async function refreshRun(): Promise<void> {
  try {
    const run = await session.refreshRun()
    if (run?.failure) openRuntimeWorkbench('logs')
  } catch (error) {
    showError(t('workflow.toast.refresh_failed'), error)
  }
}

async function loadTimelinePage(page: number): Promise<void> {
  try {
    await session.loadTimelinePage(page)
  } catch (error) {
    showError(t('workflow.toast.refresh_failed'), error)
  }
}

function openRuntimeWorkbench(tab: RuntimeWorkbenchTab): void {
  runtimeWorkbenchTab.value = tab
  runtimeWorkbenchOpen.value = true
}

function toggleRuntimeWorkbench(tab: RuntimeWorkbenchTab): void {
  if (runtimeWorkbenchOpen.value && runtimeWorkbenchTab.value === tab) {
    runtimeWorkbenchOpen.value = false
    return
  }
  openRuntimeWorkbench(tab)
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
    description: errorMessage(error),
    color: 'error',
  })
}

function plainCopy<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
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
