<template>
  <div
    data-testid="assets-view"
    class="workspace-page workspace-canvas flex h-full min-h-0 w-full flex-col overflow-hidden"
  >
    <header
      class="workspace-page__header flex min-h-[72px] shrink-0 items-center justify-between gap-6 px-8 py-4 max-[900px]:flex-col max-[900px]:items-start max-[900px]:px-6"
    >
      <div class="min-w-0">
        <div class="flex items-center gap-3">
          <span
            class="workspace-page__mark flex size-10 shrink-0 items-center justify-center rounded-[10px] border border-primary/25 bg-primary/10 text-primary"
          >
            <UIcon name="i-tabler-library" class="size-5" />
          </span>
          <div class="min-w-0">
            <div class="flex min-w-0 items-center gap-2">
              <h1
                class="workspace-page__title truncate text-xl leading-tight font-semibold tracking-[-0.02em] text-highlighted"
              >
                {{ t('assets.title') }}
              </h1>
              <UBadge color="neutral" variant="soft" size="sm">{{ total }}</UBadge>
            </div>
          </div>
        </div>
      </div>
      <div class="flex shrink-0 items-center justify-end gap-2">
        <UDropdownMenu :items="libraryMenuItems">
          <UButton
            icon="i-tabler-dots-vertical"
            color="neutral"
            variant="ghost"
            :aria-label="t('assets.library_actions')"
          />
        </UDropdownMenu>
      </div>
    </header>

    <div class="flex min-h-0 flex-1" data-testid="asset-library" data-mode="manage">
      <aside class="workspace-surface flex w-52 shrink-0 flex-col border-r border-default p-2">
        <p class="px-2 pb-2 pt-1 text-[10px] font-semibold uppercase tracking-wide text-dimmed">
          {{ t('assets.asset_types') }}
        </p>
        <UButton
          color="neutral"
          :variant="activeTab === 'macros' ? 'soft' : 'ghost'"
          icon="i-tabler-list-details"
          class="h-auto w-full justify-start px-2.5 py-2 text-left"
          @click="activeTab = 'macros'"
        >
          <span class="min-w-0 flex-1">
            <span class="block text-xs font-medium">{{ t('assets.tabs.macros') }}</span>
            <span class="mt-0.5 block truncate text-[10px] text-dimmed">{{
              t('assets.macros.nav_hint')
            }}</span>
          </span>
          <UBadge v-if="activeTab === 'macros'" color="neutral" variant="soft" size="xs">{{
            total
          }}</UBadge>
        </UButton>
        <UButton
          color="neutral"
          :variant="activeTab === 'clips' ? 'soft' : 'ghost'"
          icon="i-tabler-route-alt-left"
          class="mt-1 h-auto w-full justify-start px-2.5 py-2 text-left"
          @click="activeTab = 'clips'"
        >
          <span class="min-w-0 flex-1">
            <span class="block text-xs font-medium">{{ t('assets.tabs.clips') }}</span>
            <span class="mt-0.5 block truncate text-[10px] text-dimmed">{{
              t('assets.clips.nav_hint')
            }}</span>
          </span>
          <UBadge v-if="activeTab === 'clips'" color="neutral" variant="soft" size="xs">{{
            total
          }}</UBadge>
        </UButton>
        <UButton
          color="neutral"
          :variant="activeTab === 'templates' ? 'soft' : 'ghost'"
          icon="i-tabler-photo"
          class="mt-1 h-auto w-full justify-start px-2.5 py-2 text-left"
          @click="activeTab = 'templates'"
        >
          <span class="min-w-0 flex-1">
            <span class="block text-xs font-medium">{{ t('assets.tabs.templates') }}</span>
            <span class="mt-0.5 block truncate text-[10px] text-dimmed">{{
              t('assets.templates.nav_hint')
            }}</span>
          </span>
          <UBadge v-if="activeTab === 'templates'" color="neutral" variant="soft" size="xs">{{
            total
          }}</UBadge>
        </UButton>
      </aside>

      <main class="flex min-h-0 min-w-0 flex-1 flex-col">
        <div class="workspace-surface shrink-0 border-b border-default">
          <div class="flex min-h-14 items-center gap-3 border-b border-default px-3 py-2">
            <span
              class="flex size-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary"
            >
              <UIcon :name="activeResourceIcon" class="size-4" />
            </span>
            <div class="min-w-0 flex-1">
              <h2 class="text-xs font-semibold text-highlighted">{{ activeResourceTitle }}</h2>
              <p class="mt-0.5 truncate text-[10px] text-dimmed">{{ activeResourceHint }}</p>
            </div>
            <template
              v-if="
                (activeTab === 'macros' || activeTab === 'clips') &&
                recording.state.phase === 'idle'
              "
            >
              <UButton
                data-testid="assets-recording-start"
                :icon="activeTab === 'macros' ? 'i-tabler-list-details' : 'i-tabler-route-alt-left'"
                :label="
                  activeTab === 'macros'
                    ? t('assets.recording.record_macro')
                    : t('assets.recording.record_precise')
                "
                :loading="recordingStarting"
                @click="openResourceAction(activeTab === 'macros' ? 'macro' : 'precise')"
              />
              <UButton
                v-if="activeTab === 'macros'"
                color="neutral"
                variant="soft"
                icon="i-tabler-file-plus"
                :label="t('assets.macros.create_blank')"
                @click="openResourceAction('blank-macro')"
              />
            </template>
            <UButton
              v-else-if="activeTab === 'templates'"
              icon="i-tabler-camera-plus"
              :label="t('assets.templates.capture')"
              :loading="captureBusy"
              @click="openResourceAction('template')"
            />
          </div>
          <form
            class="flex items-center gap-2 border-b border-default p-3"
            role="search"
            @submit.prevent="applyQuery"
          >
            <UInput
              v-model="queryInput"
              icon="i-tabler-search"
              class="min-w-56 flex-1"
              :placeholder="t('assets.search_all_placeholder')"
              :aria-label="t('assets.search_all_placeholder')"
            />
            <UButton
              type="submit"
              color="neutral"
              variant="soft"
              icon="i-tabler-search"
              :label="t('assets.search_action')"
            />
          </form>
          <LibrarySelectionToolbar
            v-if="selectedRows.length"
            :label="t('assets.selected_count', { n: selectedRows.length })"
            :hint="t('batchMetadata.selection_hint')"
            :clear-label="t('assets.clear_selection')"
            @clear="clearSelection"
          >
            <UButton
              data-testid="asset-batch-metadata"
              size="sm"
              variant="soft"
              icon="i-tabler-category-plus"
              :disabled="batchBusy"
              @click="openBatchEdit"
            >
              {{ t('assets.batch_edit') }}
            </UButton>
            <template #destructive>
              <UButton
                size="sm"
                color="error"
                variant="ghost"
                icon="i-tabler-trash"
                :loading="batchBusy"
                @click="deleteSelected"
              >
                {{ t('assets.batch_delete') }}
              </UButton>
            </template>
          </LibrarySelectionToolbar>
          <div v-else class="flex flex-wrap items-center gap-2 p-3">
            <AdaptiveSelect
              v-model="categoryFilter"
              :items="categoryFilterItems"
              icon="i-tabler-category"
              @update:model-value="changeQuery"
            />
            <UInputMenu
              v-model="tagFilters"
              :items="tagOptions"
              multiple
              icon="i-tabler-tags"
              class="min-w-56 max-w-md flex-1"
              :placeholder="t('assets.all_tags')"
              @update:model-value="changeQuery"
            />
            <AdaptiveSelect
              v-model="createdRange"
              :items="createdRangeItems"
              icon="i-tabler-calendar-plus"
              @update:model-value="changeQuery"
            />
            <AdaptiveSelect
              v-model="sort"
              :items="sortItems"
              icon="i-tabler-arrows-sort"
              @update:model-value="changeQuery"
            />
            <UDropdownMenu :items="columnMenuItems">
              <UButton
                color="neutral"
                variant="soft"
                icon="i-tabler-columns-3"
                trailing-icon="i-tabler-chevron-down"
                :label="t('assets.columns_action')"
              />
            </UDropdownMenu>
            <UButton
              v-if="hasLibraryFilters"
              color="neutral"
              variant="ghost"
              icon="i-tabler-filter-x"
              :label="t('assets.reset_filters')"
              @click="resetLibraryFilters"
            />
          </div>
        </div>

        <section
          v-if="recording.state.phase !== 'idle' && recordingTab === activeTab"
          data-testid="assets-recording-controls"
          class="flex shrink-0 items-center gap-3 border-b border-default bg-primary/5 px-3 py-2"
        >
          <span class="size-2 rounded-full bg-error" aria-hidden="true" />
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <p class="text-xs font-medium text-highlighted">{{ t('assets.recording.title') }}</p>
              <UBadge :color="recordingBadge.color" variant="soft" size="xs">{{
                recordingBadge.label
              }}</UBadge>
            </div>
            <p class="truncate text-[10px] text-dimmed">{{ recordingHint }}</p>
          </div>
          <UButton
            v-if="recording.state.phase === 'recording'"
            size="xs"
            color="neutral"
            variant="soft"
            icon="i-tabler-player-pause"
            :label="t('recordingHud.pause')"
            @click="pauseRecording"
          />
          <UButton
            v-if="recording.state.phase === 'paused'"
            size="xs"
            color="neutral"
            variant="soft"
            icon="i-tabler-player-play"
            :label="t('recordingHud.resume')"
            @click="resumeRecording"
          />
          <UButton
            v-if="recording.state.phase === 'armed' || recording.state.phase === 'countdown'"
            size="xs"
            color="error"
            variant="ghost"
            icon="i-tabler-x"
            :label="t('common.cancel')"
            @click="cancelRecordingPreparation"
          />
          <UButton
            v-if="recording.state.phase === 'recording' || recording.state.phase === 'paused'"
            size="xs"
            color="error"
            variant="soft"
            icon="i-tabler-square"
            :label="t('recordingHud.stop')"
            @click="stopRecording"
          />
        </section>

        <div
          v-if="libraryFeedback"
          class="shrink-0 border-b px-4 py-2 text-sm"
          :class="
            libraryFeedback.tone === 'success'
              ? 'border-success/30 bg-success/10 text-success'
              : libraryFeedback.tone === 'warning'
                ? 'border-warning/30 bg-warning/10 text-warning'
                : 'border-error/30 bg-error/10 text-error'
          "
          role="status"
        >
          {{ libraryFeedback.message }}
        </div>

        <div class="workspace-surface min-h-0 flex-1 overflow-y-auto">
          <div v-if="loading" class="space-y-px p-2">
            <USkeleton v-for="index in 10" :key="index" class="h-14 rounded-md" />
          </div>
          <AssetLibraryList
            v-else-if="visibleItems.length"
            :items="visibleItems"
            :visible-columns="visibleColumns"
            :grid-template-columns="assetGridTemplate"
            @preview-state="setPreviewState"
          >
            <template #select-all>
              <UCheckbox
                :model-value="allCurrentPageSelected"
                :aria-label="t('assets.select_page')"
                @update:model-value="toggleCurrentPage(Boolean($event))"
              />
            </template>
            <template #select="{ item }">
              <UCheckbox
                :model-value="Boolean(selected[item.id])"
                :aria-label="t('assets.select_named', { name: item.name })"
                @update:model-value="toggleAsset(assetItem(item.id).source, Boolean($event))"
              />
            </template>
            <template #actions="{ item }">
              <UDropdownMenu :items="assetMenu(assetItem(item.id))">
                <UButton
                  icon="i-tabler-dots"
                  color="neutral"
                  variant="ghost"
                  size="xs"
                  :aria-label="t('assets.asset_actions', { name: item.name })"
                />
              </UDropdownMenu>
            </template>
          </AssetLibraryList>
          <EmptyState
            v-else
            inset
            :icon="
              hasLibraryFilters
                ? 'i-tabler-search-off'
                : activeTab === 'macros'
                  ? 'i-tabler-list-details'
                  : activeTab === 'clips'
                    ? 'i-tabler-route-alt-left'
                    : 'i-tabler-photo-off'
            "
            :title="hasLibraryFilters ? t('assets.no_results') : t(`assets.${activeTab}.empty`)"
            :description="
              hasLibraryFilters ? t('assets.no_results_hint') : t(`assets.${activeTab}.empty_hint`)
            "
          >
            <template #action>
              <UButton
                v-if="hasLibraryFilters"
                color="neutral"
                variant="soft"
                icon="i-tabler-filter-x"
                :label="t('assets.reset_filters')"
                @click="resetLibraryFilters"
              />
              <UButton
                v-else-if="activeTab === 'templates'"
                icon="i-tabler-camera-plus"
                :label="t('assets.templates.capture')"
                @click="openResourceAction('template')"
              />
            </template>
          </EmptyState>
        </div>
        <footer
          v-if="!loading && total > 0"
          class="workspace-surface flex h-11 shrink-0 items-center gap-3 border-t border-default px-3"
        >
          <span class="mr-auto text-xs text-dimmed">
            {{ t('assets.result_range', { start: resultStart, end: resultEnd, total }) }}
          </span>
          <UPagination
            :page="page"
            :total="total"
            :items-per-page="pageSize"
            :sibling-count="1"
            active-variant="subtle"
            show-edges
            @update:page="goToPage"
          />
          <span class="text-xs text-dimmed">{{ t('assets.per_page') }}</span>
          <AdaptiveSelect
            v-model="pageSize"
            :items="pageSizeItems"
            class="w-24"
            width-mode="fixed"
            @update:model-value="changeQuery"
          />
        </footer>
      </main>
    </div>
  </div>

  <BaseModal
    v-model:open="resourceActionOpen"
    :title="resourceActionTitle"
    :icon="resourceActionIcon"
    size="md"
  >
    <div class="space-y-3">
      <p class="text-xs leading-5 text-muted">{{ t('assets.action_target_hint') }}</p>
      <UFormField :label="t('workflow.recording.target')" required>
        <AdaptiveSelect
          v-model="selectedTargetSlot"
          :items="targetItems"
          value-key="value"
          label-key="label"
          :placeholder="t('assets.target_placeholder')"
        />
      </UFormField>
    </div>
    <template #footer>
      <UButton color="neutral" variant="ghost" @click="resourceActionOpen = false">
        {{ t('common.cancel') }}
      </UButton>
      <UButton
        :icon="resourceActionIcon"
        :disabled="!selectedTargetSlot || !selectedTargetSupportsRecording"
        :loading="recordingStarting || captureBusy"
        @click="confirmResourceAction"
      >
        {{ t('common.continue') }}
      </UButton>
    </template>
  </BaseModal>

  <BaseModal
    :open="batchEditing"
    :title="t('assets.batch_edit_title', { n: selectedRows.length })"
    icon="i-tabler-tags"
    size="lg"
    @update:open="(open) => (batchEditing = open)"
  >
    <div class="space-y-5">
      <p class="text-sm text-muted">{{ t('batchMetadata.description') }}</p>
      <UFormField :label="t('common.category')">
        <div class="flex items-center gap-2">
          <AdaptiveSelect
            v-model="batchDraft.categoryMode"
            :items="categoryModeItems"
            class="shrink-0"
          />
          <UInputMenu
            v-if="batchDraft.categoryMode === 'set'"
            v-model="batchDraft.category"
            :items="batchCategoryOptions"
            :create-item="'always'"
            class="min-w-0 flex-1"
            @create="createBatchCategory"
          />
          <span v-else class="text-xs text-dimmed">{{ categoryModeHint }}</span>
        </div>
      </UFormField>
      <UFormField :label="t('common.tags')">
        <div class="flex items-start gap-2">
          <AdaptiveSelect v-model="batchDraft.tagMode" :items="tagModeItems" class="shrink-0" />
          <UInputMenu
            v-if="tagModeNeedsValues"
            v-model="batchDraft.tags"
            :items="batchTagOptions"
            :create-item="'always'"
            multiple
            class="min-w-0 flex-1"
            @create="createBatchTag"
          />
          <span v-else class="pt-2 text-xs text-dimmed">{{ tagModeHint }}</span>
        </div>
      </UFormField>
    </div>
    <template #footer>
      <UButton color="neutral" variant="ghost" @click="batchEditing = false">
        {{ t('common.cancel') }}
      </UButton>
      <UButton
        icon="i-tabler-check"
        :label="t('batchMetadata.apply')"
        :loading="batchBusy"
        :disabled="!batchDraftValid"
        @click="saveBatchMeta"
      />
    </template>
  </BaseModal>

  <BaseModal
    :open="!!preciseViewing"
    :title="preciseViewing?.label ?? t('preciseWorkbench.title')"
    icon="i-tabler-route-alt-left"
    size="5xl"
    tall
    @update:open="(open) => !open && (preciseViewing = null)"
  >
    <PreciseRecordingWorkbench
      v-if="preciseViewing && preciseViewingPreview"
      :preview="preciseViewingPreview"
      :environment="{
        baseResolution: preciseViewing.meta.baseResolution,
        mouseMode: preciseViewing.meta.mouseMode,
        mouseCounts360: preciseViewing.meta.mouseCounts360,
      }"
      :duration-us="preciseViewing.durationUs"
      :trim-start-us="0"
      :trim-end-us="preciseViewing.durationUs"
      :clip-id="preciseViewing.id"
    />
    <template #footer>
      <UButton color="neutral" variant="ghost" @click="preciseViewing = null">{{
        t('common.close')
      }}</UButton>
    </template>
  </BaseModal>

  <BaseModal
    v-model:open="macroCreateOpen"
    :title="t('assets.macros.create_title')"
    icon="i-tabler-file-plus"
    size="lg"
  >
    <div class="space-y-4">
      <p class="text-sm leading-6 text-muted">{{ t('assets.macros.create_hint') }}</p>
      <UFormField :label="t('common.name')" required>
        <UInput v-model="macroCreateDraft.name" autofocus maxlength="80" />
      </UFormField>
      <UFormField :label="t('common.description')" :hint="t('common.optional')">
        <UTextarea v-model="macroCreateDraft.description" :rows="2" />
      </UFormField>
      <div class="grid grid-cols-2 gap-3">
        <UFormField :label="t('common.category')" :hint="t('common.optional')">
          <UInputMenu
            v-model="macroCreateDraft.category"
            :items="metadataCategoryOptions"
            :create-item="'always'"
          />
        </UFormField>
        <UFormField :label="t('common.tags')" :hint="t('common.optional')">
          <UInputMenu
            v-model="macroCreateDraft.tags"
            :items="metadataTagOptions"
            :create-item="'always'"
            multiple
          />
        </UFormField>
      </div>
    </div>
    <template #footer>
      <UButton color="neutral" variant="ghost" @click="macroCreateOpen = false">{{
        t('common.cancel')
      }}</UButton>
      <UButton
        icon="i-tabler-check"
        :loading="macroCreateBusy"
        :disabled="!macroCreateDraft.name.trim()"
        @click="createBlankMacro"
      >
        {{ t('common.create') }}
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
      <div
        class="grid shrink-0 gap-3 rounded-lg border border-default bg-elevated/20 p-3 md:grid-cols-2"
      >
        <UFormField :label="t('common.name')" required>
          <UInput v-model="macroEditing.label" maxlength="80" />
        </UFormField>
        <UFormField :label="t('common.category')" :hint="t('common.optional')">
          <UInputMenu
            v-model="macroEditing.category"
            :items="metadataCategoryOptions"
            :create-item="'always'"
            :placeholder="t('recordingSave.category_placeholder')"
            @create="createMacroCategory"
          />
        </UFormField>
        <UFormField :label="t('common.description')" :hint="t('common.optional')">
          <UTextarea v-model="macroEditing.description" :rows="2" />
        </UFormField>
        <UFormField :label="t('common.tags')" :hint="t('common.optional')">
          <UInputMenu
            v-model="macroEditing.tags"
            :items="metadataTagOptions"
            :create-item="'always'"
            multiple
            @create="createMacroTag"
          />
        </UFormField>
      </div>
      <div
        class="flex shrink-0 items-center gap-3 rounded-lg border border-default bg-elevated/25 px-3 py-2 text-xs text-muted"
      >
        <span>{{ t('assets.macros.base_resolution') }}</span>
        <strong class="font-mono text-toned"
          >{{ macroEditing.document.baseResolution[0] }}×{{
            macroEditing.document.baseResolution[1]
          }}</strong
        >
        <span class="ml-auto font-mono text-[10px] text-dimmed">{{ macroEditing.id }}</span>
      </div>
      <div class="flex shrink-0 items-start gap-2 text-xs leading-5 text-dimmed">
        <UIcon name="i-tabler-route-alt-left" class="mt-0.5 size-4 shrink-0" />
        <span>{{ t('assets.macros.trajectory_hint') }}</span>
      </div>
      <MacroActionEditor
        v-model="macroEditing.document"
        class="min-h-0 flex-1"
        @validity="macroEditValid = $event"
      />
    </div>
    <template #footer>
      <UButton color="neutral" variant="ghost" @click="macroEditing = null">{{
        t('common.cancel')
      }}</UButton>
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

  <ImageVariantManagerModal
    :open="!!variantAsset"
    :title="
      variantAsset
        ? t('assets.templates.manage_title', { name: variantAsset.name })
        : t('assets.templates.manage_variants')
    "
    :variants="variantAsset?.variants ?? []"
    :busy="variantBusy"
    :add-disabled="!selectedTargetSlot"
    @update:open="(open) => !open && (variantAsset = null)"
    @add="recaptureVariant"
    @recapture="recaptureVariant"
    @remove="(variant) => variantAsset && removeVariant(variantAsset, variant.resolution)"
    @remove-blocked="
      libraryFeedback = { tone: 'warning', message: t('assets.templates.last_variant_blocked') }
    "
  />

  <BaseModal
    :open="!!editingItem"
    :title="t('assets.edit_title')"
    icon="i-tabler-edit"
    size="lg"
    @update:open="(open) => !open && (editingItem = null)"
  >
    <div class="space-y-4">
      <UFormField :label="t('common.name')" required>
        <UInput v-model="editDraft.name" maxlength="80" />
      </UFormField>
      <UFormField :label="t('common.description')" :hint="t('common.optional')">
        <UTextarea v-model="editDraft.description" :rows="3" />
      </UFormField>
      <UFormField :label="t('common.category')" :hint="t('common.optional')">
        <UInputMenu
          v-model="editDraft.category"
          :items="metadataCategoryOptions"
          :create-item="'always'"
          @create="createEditCategory"
        />
      </UFormField>
      <UFormField :label="t('common.tags')" :hint="t('common.optional')">
        <UInputMenu
          v-model="editDraft.tags"
          :items="metadataTagOptions"
          :create-item="'always'"
          multiple
          @create="createEditTag"
        />
      </UFormField>
    </div>
    <template #footer>
      <UButton color="neutral" variant="ghost" @click="editingItem = null">{{
        t('common.cancel')
      }}</UButton>
      <UButton :disabled="!editDraft.name.trim()" :loading="editBusy" @click="saveAssetMeta">{{
        t('common.save')
      }}</UButton>
    </template>
  </BaseModal>

  <BaseModal
    :open="!!pendingRecording"
    :title="t('recordingSave.title')"
    icon="i-tabler-device-floppy"
    size="lg"
    :show-close="false"
    :dismissible="false"
  >
    <div v-if="pendingRecording" class="space-y-4">
      <div
        v-if="pendingRecording.mode === 'simple'"
        class="rounded-lg border border-default bg-elevated/35 px-4 py-3"
      >
        <div class="flex items-center justify-between gap-3">
          <p class="text-sm font-medium text-highlighted">
            {{
              pendingRecording.mode === 'simple'
                ? t('recordingSave.macro_type')
                : t('recordingSave.clip_type')
            }}
          </p>
          <UBadge color="neutral" variant="soft">
            {{ t(`recordingSave.mode_${pendingRecording.mode}`) }}
          </UBadge>
        </div>
        <p class="mt-1 text-xs text-muted">
          {{
            t('recordingSave.summary', {
              duration: formatDuration(pendingRecording.durationUs),
              count: pendingRecording.eventCount,
            })
          }}
        </p>
      </div>
      <MacroActionEditor
        v-if="pendingRecording.mode === 'simple' && recordingDocument"
        v-model="recordingDocument"
        @validity="recordingActionsValid = $event"
      />
      <PreciseRecordingWorkbench
        v-else-if="pendingRecording.mode === 'precise'"
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
      <p v-else class="rounded-lg border border-default bg-sunken px-3 py-2 text-xs text-muted">
        {{ t('recordingEditor.editing_unavailable') }}
      </p>
      <UFormField :label="t('recordingSave.name')" required>
        <UInput v-model="recordingDraft.name" autofocus maxlength="80" />
      </UFormField>
      <UFormField :label="t('common.description')" :hint="t('common.optional')">
        <UTextarea v-model="recordingDraft.description" :rows="2" />
      </UFormField>
      <div class="grid grid-cols-2 gap-3">
        <UFormField :label="t('common.category')" :hint="t('common.optional')">
          <UInputMenu
            v-model="recordingDraft.category"
            :items="metadataCategoryOptions"
            :create-item="'always'"
            :placeholder="t('recordingSave.category_placeholder')"
            @create="createRecordingCategory"
          />
        </UFormField>
        <UFormField :label="t('common.tags')" :hint="t('recordingSave.tags_hint')">
          <UInputMenu
            v-model="recordingDraft.tags"
            :items="metadataTagOptions"
            :create-item="'always'"
            :placeholder="t('recordingSave.tags_placeholder')"
            multiple
            @create="createRecordingTag"
          />
        </UFormField>
      </div>
    </div>
    <template #footer>
      <UButton
        color="error"
        variant="ghost"
        :disabled="recordingSaveBusy"
        @click="discardRecording"
      >
        {{ t('recordingSave.discard') }}
      </UButton>
      <UButton
        :loading="recordingSaveBusy"
        :disabled="
          !recordingDraft.name.trim() ||
          (pendingRecording?.mode === 'simple' && !recordingActionsValid)
        "
        @click="saveRecording"
      >
        {{ t('assets.recording.save_to_library') }}
      </UButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import {
  backend,
  type AssetSummary,
  type BlobRef,
  type InputClipSummary,
  type MacroAsset,
  type MacroDocument,
} from '@/lib/backend'
import { errorMessage } from '@/lib/invoke'
import {
  applyBatchMetadata,
  createBatchMetadataDraft,
  hasBatchMetadataChange,
} from '@/lib/batchMetadata'
import {
  useRecordingStore,
  type RecordingMode,
  type RecordingPreview,
  type RecordingStopPayload,
} from '@/stores/recording'
import { useSettingsStore } from '@/stores/settings'
import { useAssetsStore } from '@/stores/assets'
import { useConfirm } from '@/composables/useConfirm'
import { useAutoDismissFeedback } from '@/composables/useAutoDismissFeedback'
import { awaitWailsEvent } from '@/composables/useWailsEvent'
import { useRecordingStart } from '@/composables/useRecordingStart'
import { useRecordingStartFeedback } from '@/composables/useRecordingStartFeedback'
import BaseModal from '@/components/common/BaseModal.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import AssetLibraryList from '@/components/assets/AssetLibraryList.vue'
import ImageVariantManagerModal from '@/components/assets/ImageVariantManagerModal.vue'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'
import MacroActionEditor from '@/components/recording/MacroActionEditor.vue'
import PreciseRecordingWorkbench from '@/components/recording/PreciseRecordingWorkbench.vue'
import LibrarySelectionToolbar from '@/components/library/LibrarySelectionToolbar.vue'

type AssetTab = 'macros' | 'clips' | 'templates'
type ResourceAction = 'macro' | 'precise' | 'template' | 'blank-macro' | 'recapture'
type AssetColumn = 'category' | 'tags' | 'details' | 'createdAt'
type DateRange = 'all' | 'today' | '7d' | '30d' | '90d'
type AssetItem = {
  id: string
  kind: AssetTab
  name: string
  description: string
  category: string
  tags: string[]
  meta: string
  icon: string
  previewBlob?: BlobRef
  createdAt?: string
  source: AssetSummary
}
type AssetMetadataDraft = { category: string; tags: string[] }
const allCategories = '__all__'
const defaultColumns: AssetColumn[] = ['category', 'tags', 'details', 'createdAt']

const { t } = useI18n()
const toast = useToast()
const { confirm } = useConfirm()
const settings = useSettingsStore()
const assets = useAssetsStore()
const recording = useRecordingStore()
const { starting: recordingStarting, start: beginRecording } = useRecordingStart()
const { show: showRecordingStartError } = useRecordingStartFeedback()
const activeTab = ref<AssetTab>('macros')
const queryInput = ref('')
const query = ref('')
const categoryFilter = ref(allCategories)
const tagFilters = ref<string[]>([])
const createdRange = ref<DateRange>('all')
const categories = ref<Array<{ value: string; count: number }>>([])
const tags = ref<Array<{ value: string; count: number }>>([])
const sort = ref('recent_desc')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const visibleColumns = ref<AssetColumn[]>(loadColumns())
const assetPage = ref<AssetSummary[]>([])
const loading = ref(false)
const selected = ref<Record<string, AssetSummary>>({})
const batchEditing = ref(false)
const batchBusy = ref(false)
const batchDraft = reactive(createBatchMetadataDraft())
const libraryFeedback = ref<{ tone: 'success' | 'warning' | 'error'; message: string } | null>(null)
useAutoDismissFeedback(libraryFeedback)
const variantAsset = ref<AssetSummary | null>(null)
const variantBusy = ref(false)
const cleanupBusy = ref(false)
const selectedTargetSlot = ref('')
const resourceActionOpen = ref(false)
const resourceAction = ref<ResourceAction>('macro')
const captureBusy = ref(false)
const editingItem = ref<AssetItem | null>(null)
const editBusy = ref(false)
const macroEditing = ref<MacroAsset | null>(null)
const macroEditBusy = ref(false)
const macroEditValid = ref(true)
const macroCreateOpen = ref(false)
const macroCreateBusy = ref(false)
const macroCreateDraft = reactive({
  name: '',
  description: '',
  category: '',
  tags: [] as string[],
})
const preciseViewing = ref<InputClipSummary | null>(null)
const preciseViewingPreview = computed<RecordingPreview | null>(() => {
  const clip = preciseViewing.value
  if (!clip) return null
  const counts = Object.fromEntries(clip.tracks.map((track) => [track.kind, track.count]))
  return {
    mode: 'precise',
    durationUs: clip.durationUs,
    eventCount: clip.eventCount,
    keyActions: counts.keyboard ?? 0,
    clickActions: counts['mouse-buttons'] ?? 0,
    pointerMoves: counts['absolute-motion'] ?? 0,
    rawDeltas: counts['relative-motion'] ?? 0,
    scrollActions: counts.scroll ?? 0,
    steps: [],
    tracks: clip.tracks,
  }
})
const pendingRecording = ref<RecordingStopPayload | null>(null)
const recordingSaveBusy = ref(false)
const recordingDocument = ref<MacroDocument | null>(null)
const recordingActionsValid = ref(true)
const recordingTrimStartUs = ref(0)
const recordingTrimEndUs = ref(0)
const editDraft = reactive({ name: '', description: '', category: '', tags: [] as string[] })
const recordingDraft = reactive({ name: '', description: '', category: '', tags: [] as string[] })
const createdCategories = ref<string[]>([])
const createdTags = ref<string[]>([])
const previewStates = reactive<Record<string, 'loading' | 'ready' | 'unavailable'>>({})
const selectedRows = computed(() => Object.values(selected.value))
const hasLibraryFilters = computed(() =>
  Boolean(
    query.value ||
    categoryFilter.value !== allCategories ||
    tagFilters.value.length ||
    createdRange.value !== 'all',
  ),
)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))
const resultStart = computed(() => (total.value ? (page.value - 1) * pageSize.value + 1 : 0))
const resultEnd = computed(() => Math.min(page.value * pageSize.value, total.value))
const allCurrentPageSelected = computed(
  () => assetPage.value.length > 0 && assetPage.value.every((asset) => selected.value[asset.guid]),
)
const sortItems = computed(() => [
  { label: t('assets.sort_recent_desc'), value: 'recent_desc' },
  { label: t('assets.sort_name_asc'), value: 'name_asc' },
  { label: t('assets.sort_name_desc'), value: 'name_desc' },
  { label: t('assets.sort_created_desc'), value: 'created_desc' },
])
const createdRangeItems = computed(() => [
  { label: t('assets.created_any'), value: 'all' },
  { label: t('assets.created_today'), value: 'today' },
  { label: t('assets.created_days', { n: 7 }), value: '7d' },
  { label: t('assets.created_days', { n: 30 }), value: '30d' },
  { label: t('assets.created_days', { n: 90 }), value: '90d' },
])
const pageSizeItems = [
  { label: '20', value: 20 },
  { label: '50', value: 50 },
  { label: '100', value: 100 },
]
const categoryFilterItems = computed(() => [
  { label: t('assets.all_categories'), value: allCategories },
  ...categories.value.map((item) => ({
    label: `${item.value} (${item.count})`,
    value: item.value,
  })),
])
const tagOptions = computed(() => tags.value.map((item) => item.value))
const columnOptions = computed<Array<{ key: AssetColumn; label: string }>>(() => [
  { key: 'category', label: t('common.category') },
  { key: 'tags', label: t('common.tags') },
  { key: 'details', label: t('assets.columns.details') },
  { key: 'createdAt', label: t('assets.columns.created') },
])
const visibleColumnSet = computed(() => new Set(visibleColumns.value))
const columnMenuItems = computed(() => [
  columnOptions.value.map((column) => ({
    label: column.label,
    type: 'checkbox' as const,
    checked: visibleColumnSet.value.has(column.key),
    onUpdateChecked: (checked: boolean) => setColumnVisible(column.key, checked),
  })),
  [
    {
      label: t('assets.reset_columns'),
      icon: 'i-tabler-restore',
      onSelect: () => {
        visibleColumns.value = [...defaultColumns]
      },
    },
  ],
])
const assetGridTemplate = computed(() => {
  const columns = ['2.25rem', 'minmax(18rem, 2fr)']
  if (visibleColumnSet.value.has('category')) columns.push('10rem')
  if (visibleColumnSet.value.has('tags')) columns.push('minmax(12rem, 1.2fr)')
  if (visibleColumnSet.value.has('details')) columns.push('9rem')
  if (visibleColumnSet.value.has('createdAt')) columns.push('9rem')
  columns.push('2.5rem')
  return columns.join(' ')
})
const metadataCategoryOptions = computed(() =>
  uniqueStrings([
    ...categories.value.map((item) => item.value),
    ...createdCategories.value,
    editDraft.category,
    recordingDraft.category,
    macroEditing.value?.category ?? '',
  ]),
)
const metadataTagOptions = computed(() =>
  uniqueStrings([
    ...tags.value.map((item) => item.value),
    ...createdTags.value,
    ...editDraft.tags,
    ...recordingDraft.tags,
    ...(macroEditing.value?.tags ?? []),
  ]),
)
const batchCategoryOptions = computed(() =>
  uniqueStrings([
    ...categories.value.map((item) => item.value),
    ...createdCategories.value,
    batchDraft.category,
  ]),
)
const batchTagOptions = computed(() =>
  uniqueStrings([
    ...tags.value.map((item) => item.value),
    ...createdTags.value,
    ...batchDraft.tags,
  ]),
)
const categoryModeItems = computed(() => [
  { label: t('batchMetadata.keep'), value: 'keep' },
  { label: t('batchMetadata.set'), value: 'set' },
  { label: t('batchMetadata.clear'), value: 'clear' },
])
const tagModeItems = computed(() => [
  { label: t('batchMetadata.keep'), value: 'keep' },
  { label: t('batchMetadata.add'), value: 'add' },
  { label: t('batchMetadata.remove'), value: 'remove' },
  { label: t('batchMetadata.replace'), value: 'replace' },
  { label: t('batchMetadata.clear'), value: 'clear' },
])
const tagModeNeedsValues = computed(() => ['add', 'remove', 'replace'].includes(batchDraft.tagMode))
const batchDraftValid = computed(() => hasBatchMetadataChange(batchDraft))
const categoryModeHint = computed(() =>
  t(
    batchDraft.categoryMode === 'clear'
      ? 'batchMetadata.category_clear_hint'
      : 'batchMetadata.keep_hint',
  ),
)
const tagModeHint = computed(() =>
  t(batchDraft.tagMode === 'clear' ? 'batchMetadata.tags_clear_hint' : 'batchMetadata.keep_hint'),
)
const libraryMenuItems = computed(() => [
  [
    {
      label: t('common.refresh'),
      icon: 'i-tabler-refresh',
      onSelect: () => void refreshAssets(),
    },
    {
      label: t('assets.cleanup_action'),
      icon: 'i-tabler-recycle',
      disabled: cleanupBusy.value,
      onSelect: () => void cleanupLibrary(),
    },
  ],
])

const targetItems = computed(() =>
  (settings.data?.automation.targets ?? []).map((target) => ({
    label: `${target.label} · ${target.slot}`,
    value: target.slot,
  })),
)
const activeResourceIcon = computed(() =>
  activeTab.value === 'macros'
    ? 'i-tabler-list-details'
    : activeTab.value === 'clips'
      ? 'i-tabler-route-alt-left'
      : 'i-tabler-photo',
)
const activeResourceTitle = computed(() => t(`assets.tabs.${activeTab.value}`))
const activeResourceHint = computed(() => t(`assets.${activeTab.value}.nav_hint`))
const resourceActionTitle = computed(() => {
  if (resourceAction.value === 'macro') return t('assets.recording.record_macro')
  if (resourceAction.value === 'precise') return t('assets.recording.record_precise')
  if (resourceAction.value === 'template') return t('assets.templates.capture')
  if (resourceAction.value === 'recapture') return t('assets.templates.recapture')
  return t('assets.macros.create_blank')
})
const resourceActionIcon = computed(() => {
  if (resourceAction.value === 'macro') return 'i-tabler-list-details'
  if (resourceAction.value === 'precise') return 'i-tabler-route-alt-left'
  if (resourceAction.value === 'template' || resourceAction.value === 'recapture')
    return 'i-tabler-camera-plus'
  return 'i-tabler-file-plus'
})
const selectedTargetSupportsRecording = computed(() =>
  (settings.data?.automation.targets ?? []).some(
    (target) => target.slot === selectedTargetSlot.value && target.targetKind === 'desktop-window',
  ),
)
const recordingTab = computed<AssetTab>(() =>
  recording.state.mode === 'precise' ? 'clips' : 'macros',
)
const items = computed<AssetItem[]>(() => {
  return assetPage.value.map((asset) => {
    if (asset.kind === 'macro')
      return {
        id: asset.guid,
        kind: 'macros',
        name: asset.name || asset.guid,
        description: asset.description ?? '',
        category: asset.category ?? '',
        tags: asset.tags ?? [],
        meta: t('assets.macros.library_meta', { bytes: asset.blob?.size ?? 0 }),
        icon: 'i-tabler-list-details',
        createdAt: formatAssetDate(asset.createdAt),
        source: asset,
      }
    if (asset.kind === 'clip')
      return {
        id: asset.guid,
        kind: 'clips',
        name: asset.name || asset.guid,
        description: asset.description ?? '',
        category: asset.category ?? '',
        tags: asset.tags ?? [],
        meta: t('assets.clips.library_meta', { bytes: asset.blob?.size ?? 0 }),
        icon: 'i-tabler-movie',
        createdAt: formatAssetDate(asset.createdAt),
        source: asset,
      }
    return {
      id: asset.guid,
      kind: 'templates',
      name: asset.name || asset.guid,
      description: asset.description ?? '',
      category: asset.category ?? '',
      tags: asset.tags ?? [],
      meta: t('assets.templates.meta', { count: asset.variantCount }),
      icon: 'i-tabler-photo',
      previewBlob: asset.thumbnail,
      createdAt: formatAssetDate(asset.createdAt),
      source: asset,
    }
  })
})
const visibleItems = computed(() => items.value)

function assetItem(id: string): AssetItem {
  const item = visibleItems.value.find((candidate) => candidate.id === id)
  if (!item) throw new Error(`asset ${id} is not on the current page`)
  return item
}

function setPreviewState(item: { id: string }, state: 'loading' | 'ready' | 'unavailable'): void {
  previewStates[item.id] = state
}
const recordingBadge = computed(() => {
  switch (recording.state.phase) {
    case 'armed':
      return { label: t('recordingHud.waiting'), color: 'primary' as const }
    case 'countdown':
      return { label: t('recordingHud.countdown'), color: 'primary' as const }
    case 'recording':
      return { label: t('recordingHud.recording'), color: 'error' as const }
    case 'paused':
      return { label: t('recordingHud.paused'), color: 'warning' as const }
    case 'finalizing':
      return { label: t('assets.recording.finalizing'), color: 'warning' as const }
    case 'pending':
      return { label: t('recordingSave.pending'), color: 'warning' as const }
    default:
      return { label: t('assets.recording.ready'), color: 'neutral' as const }
  }
})
const recordingHint = computed(() => {
  if (recording.state.phase === 'armed') return t('assets.recording.waiting_hint')
  if (recording.state.phase === 'countdown') return t('assets.recording.countdown_hint')
  if (recording.state.phase === 'recording' || recording.state.phase === 'paused')
    return t('assets.recording.active_hint', { target: recording.state.targetSlot })
  return t('assets.recording.hint')
})

onMounted(async () => {
  selectedTargetSlot.value = targetItems.value[0]?.value ?? ''
  await Promise.all([refreshAssets(), recording.reconcile()])
})

watch(
  visibleColumns,
  (value) => localStorage.setItem('yotta.asset.columns', JSON.stringify(value)),
  { deep: true },
)

watch(activeTab, async () => {
  clearSelection()
  page.value = 1
  await refreshAssets()
})

watch(
  () => recording.state.pending,
  (pending) => {
    if (!pending) {
      if (recording.state.phase === 'idle') pendingRecording.value = null
      return
    }
    if (recording.invocation && recording.invocation !== 'library') return
    recording.claimInvocation('library')
    openRecordingSave(pending)
  },
  { immediate: true },
)

watch(
  () => recording.completionFailure,
  (failure) => {
    if (failure && recording.invocation === 'library')
      showError(t('recordingSave.save_failed'), failure.message)
  },
)

async function refreshAssets(): Promise<void> {
  loading.value = true
  try {
    const result = await assets.query({
      search: query.value,
      kind:
        activeTab.value === 'macros' ? 'macro' : activeTab.value === 'clips' ? 'clip' : 'template',
      category: categoryFilter.value === allCategories ? '' : categoryFilter.value.trim(),
      tags: tagFilters.value,
      createdSince: rangeStart(createdRange.value),
      sort: sort.value,
      page: page.value,
      pageSize: pageSize.value,
      thumbnailBudget: pageSize.value,
      recentGUIDs: assets.recentGUIDs,
    })
    assetPage.value = result?.items ?? []
    total.value = result?.total ?? 0
    categories.value = result?.categories ?? []
    tags.value = result?.tags ?? []
    if (page.value > pageCount.value) {
      page.value = pageCount.value
      await refreshAssets()
    }
  } catch (error) {
    showError(t('assets.load_failed'), error)
  } finally {
    loading.value = false
  }
}

async function applyQuery(): Promise<void> {
  query.value = queryInput.value.trim()
  page.value = 1
  await refreshAssets()
}

async function changeQuery(): Promise<void> {
  page.value = 1
  await refreshAssets()
}

async function resetLibraryFilters(): Promise<void> {
  queryInput.value = ''
  query.value = ''
  categoryFilter.value = allCategories
  tagFilters.value = []
  createdRange.value = 'all'
  await changeQuery()
}

async function goToPage(next: number): Promise<void> {
  if (next < 1 || next > pageCount.value || next === page.value) return
  page.value = next
  await refreshAssets()
}

function toggleAsset(asset: AssetSummary, checked: boolean): void {
  const next = { ...selected.value }
  if (checked) next[asset.guid] = asset
  else delete next[asset.guid]
  selected.value = next
}

function toggleCurrentPage(checked: boolean): void {
  const next = { ...selected.value }
  for (const asset of assetPage.value) {
    if (checked) next[asset.guid] = asset
    else delete next[asset.guid]
  }
  selected.value = next
}

function clearSelection(): void {
  selected.value = {}
}

function setColumnVisible(column: AssetColumn, visible: boolean): void {
  const current = new Set(visibleColumns.value)
  if (visible) current.add(column)
  else current.delete(column)
  visibleColumns.value = columnOptions.value
    .map((item) => item.key)
    .filter((key) => current.has(key))
}

function loadColumns(): AssetColumn[] {
  try {
    const raw = JSON.parse(localStorage.getItem('yotta.asset.columns') ?? 'null')
    if (!Array.isArray(raw)) return [...defaultColumns]
    const allowed = new Set<AssetColumn>(['category', 'tags', 'details', 'createdAt'])
    const values = raw.filter((value): value is AssetColumn => allowed.has(value))
    return values.length ? values : [...defaultColumns]
  } catch {
    return [...defaultColumns]
  }
}

function rangeStart(range: DateRange): string {
  if (range === 'all') return ''
  const start = new Date()
  if (range === 'today') start.setHours(0, 0, 0, 0)
  else start.setDate(start.getDate() - Number.parseInt(range, 10))
  return start.toISOString()
}

function openBatchEdit(): void {
  Object.assign(batchDraft, createBatchMetadataDraft())
  batchEditing.value = true
}

async function saveBatchMeta(): Promise<void> {
  if (!selectedRows.value.length || !batchDraftValid.value) return
  batchBusy.value = true
  try {
    const results =
      (await backend.assets.batchUpdateMeta(
        selectedRows.value.map((asset) => {
          const metadata = applyBatchMetadata(
            { category: asset.category ?? '', tags: asset.tags ?? [] },
            batchDraft,
          )
          return { guid: asset.guid, category: metadata.category, tags: metadata.tags }
        }),
      )) ?? []
    retainFailedSelection(results.filter((result) => !result.updated).map((result) => result.guid))
    libraryFeedback.value = {
      tone: results.some((result) => !result.updated) ? 'warning' : 'success',
      message: t('assets.batch_update_result', {
        updated: results.filter((result) => result.updated).length,
        failed: results.filter((result) => !result.updated).length,
      }),
    }
    batchEditing.value = false
    await refreshAssets()
  } catch (error) {
    showError(t('assets.save_failed'), error)
  } finally {
    batchBusy.value = false
  }
}

async function deleteSelected(): Promise<void> {
  if (!selectedRows.value.length) return
  const accepted = await confirm({
    title: t('assets.batch_delete_title', { n: selectedRows.value.length }),
    description: t('assets.delete_description'),
    confirmText: t('common.delete'),
    color: 'error',
  })
  if (accepted !== true) return
  batchBusy.value = true
  try {
    const results =
      (await backend.assets.batchDelete(selectedRows.value.map((asset) => asset.guid))) ?? []
    retainFailedSelection(results.filter((result) => !result.deleted).map((result) => result.guid))
    libraryFeedback.value = {
      tone: results.some((result) => !result.deleted) ? 'warning' : 'success',
      message: t('assets.batch_delete_result', {
        deleted: results.filter((result) => result.deleted).length,
        failed: results.filter((result) => !result.deleted).length,
      }),
    }
    await refreshAssets()
  } catch (error) {
    showError(t('assets.delete_failed'), error)
  } finally {
    batchBusy.value = false
  }
}

function retainFailedSelection(guids: string[]): void {
  const failed = new Set(guids)
  selected.value = Object.fromEntries(
    selectedRows.value
      .filter((asset) => failed.has(asset.guid))
      .map((asset) => [asset.guid, asset]),
  )
}

async function startRecording(mode: RecordingMode): Promise<void> {
  try {
    await beginRecording(mode, selectedTargetSlot.value, 'library')
  } catch (error) {
    showRecordingStartError(t('assets.recording.start_failed'), error)
  }
}

function openResourceAction(action: ResourceAction): void {
  resourceAction.value = action
  if (!selectedTargetSlot.value) selectedTargetSlot.value = targetItems.value[0]?.value ?? ''
  resourceActionOpen.value = true
}

async function confirmResourceAction(): Promise<void> {
  if (!selectedTargetSlot.value || !selectedTargetSupportsRecording.value) return
  resourceActionOpen.value = false
  if (resourceAction.value === 'macro') {
    await startRecording('simple')
    return
  }
  if (resourceAction.value === 'precise') {
    await startRecording('precise')
    return
  }
  if (resourceAction.value === 'template') {
    await captureTemplate()
    return
  }
  if (resourceAction.value === 'recapture') {
    await performRecaptureVariant()
    return
  }
  openBlankMacro()
}

async function pauseRecording(): Promise<void> {
  try {
    await recording.pause()
  } catch (error) {
    showError(t('assets.recording.control_failed'), error)
  }
}

async function resumeRecording(): Promise<void> {
  try {
    await recording.resume()
  } catch (error) {
    showError(t('assets.recording.control_failed'), error)
  }
}

async function cancelRecordingPreparation(): Promise<void> {
  try {
    await recording.cancel()
  } catch (error) {
    showError(t('assets.recording.control_failed'), error)
  }
}

async function stopRecording(): Promise<void> {
  try {
    await recording.stop()
  } catch (error) {
    showError(t('assets.recording.control_failed'), error)
  }
}

function openRecordingSave(payload: RecordingStopPayload): void {
  if (pendingRecording.value?.pendingID === payload.pendingID) return
  pendingRecording.value = payload
  recordingDocument.value = payload.document ? cloneMacroDocument(payload.document) : null
  recordingActionsValid.value = true
  recordingTrimStartUs.value = 0
  recordingTrimEndUs.value = payload.durationUs
  recordingDraft.name = ''
  recordingDraft.description = ''
  recordingDraft.category = ''
  recordingDraft.tags = []
}

async function saveRecording(): Promise<void> {
  const pending = pendingRecording.value
  if (!pending) return
  recordingSaveBusy.value = true
  try {
    await recording.finalize({
      pendingID: pending.pendingID,
      destination: 'global-asset',
      label: recordingDraft.name.trim(),
      description: recordingDraft.description.trim(),
      category: recordingDraft.category.trim(),
      tags: uniqueStrings(recordingDraft.tags),
      document:
        pending.document && recordingDocument.value
          ? cloneMacroDocument(recordingDocument.value)
          : undefined,
      trimStartUs: pending.mode === 'precise' ? recordingTrimStartUs.value : undefined,
      trimEndUs: pending.mode === 'precise' ? recordingTrimEndUs.value : undefined,
    })
    pendingRecording.value = null
    recordingDocument.value = null
    await refreshAssets()
  } catch (error) {
    showError(t('recordingSave.save_failed'), error)
  } finally {
    recordingSaveBusy.value = false
  }
}

async function discardRecording(): Promise<void> {
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
    recordingDocument.value = null
  } catch (error) {
    showError(t('recordingSave.discard_failed'), error)
  } finally {
    recordingSaveBusy.value = false
  }
}

function cloneMacroDocument(document: MacroDocument): MacroDocument {
  return {
    ...document,
    baseResolution: [...document.baseResolution] as [number, number],
    meta: { autoMove: { ...document.meta.autoMove } },
    actions: document.actions.map((action) => ({
      ...action,
      from: action.from ? { ...action.from } : undefined,
      point: action.point ? { ...action.point } : undefined,
    })),
  }
}

async function captureTemplate(): Promise<void> {
  if (!selectedTargetSlot.value) return
  captureBusy.value = true
  const id = `template-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
  try {
    const resultPromise = awaitWailsEvent<{ id: string; payload?: { cancelled?: boolean } }>(
      'tools:picker-result',
      (payload) => payload?.id === id,
    )
    await backend.tools.openScreenPicker('template_save', id, selectedTargetSlot.value)
    const result = await resultPromise
    if (!result.payload?.cancelled) await refreshAssets()
  } catch (error) {
    showError(t('assets.templates.capture_failed'), error)
  } finally {
    captureBusy.value = false
  }
}

function assetMenu(item: AssetItem) {
  if (item.kind === 'macros') {
    return [
      [
        {
          label: t('common.edit'),
          icon: 'i-tabler-edit',
          onSelect: () => void openMacroEditor(item.source),
        },
      ],
      [
        {
          label: t('common.delete'),
          icon: 'i-tabler-trash',
          color: 'error' as const,
          onSelect: () => void deleteAsset(item),
        },
      ],
    ]
  }
  const details =
    item.kind === 'templates'
      ? [
          {
            label: t('assets.templates.manage_variants'),
            icon: 'i-tabler-photo-cog',
            onSelect: () => (variantAsset.value = item.source),
          },
        ]
      : item.kind === 'clips'
        ? [
            {
              label: t('assets.clips.open_workbench'),
              icon: 'i-tabler-route-alt-left',
              onSelect: () => void openPreciseWorkbench(item.source),
            },
          ]
        : []
  return [
    [
      ...details,
      {
        label: t('common.edit'),
        icon: 'i-tabler-edit',
        onSelect: () => openEdit(item),
      },
    ],
    [
      {
        label: t('common.delete'),
        icon: 'i-tabler-trash',
        color: 'error' as const,
        onSelect: () => void deleteAsset(item),
      },
    ],
  ]
}

async function openPreciseWorkbench(asset: AssetSummary): Promise<void> {
  try {
    preciseViewing.value = await backend.clips.summary(asset.guid)
  } catch (error) {
    showError(t('assets.clips.load_failed'), error)
  }
}

function openBlankMacro(): void {
  macroCreateDraft.name = ''
  macroCreateDraft.description = ''
  macroCreateDraft.category = ''
  macroCreateDraft.tags = []
  macroCreateOpen.value = true
}

async function createBlankMacro(): Promise<void> {
  if (!macroCreateDraft.name.trim() || !selectedTargetSlot.value) return
  macroCreateBusy.value = true
  try {
    const resolution = await backend.assets.currentResolution(selectedTargetSlot.value)
    if (!resolution) throw new Error(t('assets.macros.resolution_unavailable'))
    const saved = await backend.macros.save({
      label: macroCreateDraft.name.trim(),
      description: macroCreateDraft.description.trim(),
      category: macroCreateDraft.category.trim(),
      tags: uniqueStrings(macroCreateDraft.tags),
      document: {
        schemaVersion: 2,
        baseResolution: resolution,
        meta: { autoMove: { enabled: true, mode: 'linear', durationMs: 300 } },
        actions: [],
      },
    })
    macroCreateOpen.value = false
    await refreshAssets()
    await openMacroEditor({
      guid: saved.id,
      kind: 'macro',
      name: saved.label,
      description: saved.description,
      category: saved.category,
      tags: saved.tags,
      variantCount: 0,
      variants: [],
      blob: saved.blob,
      createdAt: saved.createdAt,
    })
  } catch (error) {
    showError(t('assets.macros.save_failed'), error)
  } finally {
    macroCreateBusy.value = false
  }
}

async function openMacroEditor(asset: AssetSummary): Promise<void> {
  try {
    const value = await backend.macros.get(asset.guid)
    if (!value) throw new Error(`macro ${asset.guid} not found`)
    macroEditing.value = {
      ...value,
      description: value.description ?? '',
      category: value.category ?? '',
      tags: [...(value.tags ?? [])],
      document: cloneMacroDocument(value.document),
      blob: { ...value.blob },
    }
    macroEditValid.value = true
  } catch (error) {
    showError(t('assets.macros.load_failed'), error)
  }
}

async function saveMacro(): Promise<void> {
  if (!macroEditing.value || !macroEditValid.value) return
  macroEditBusy.value = true
  try {
    await backend.macros.save({
      ...macroEditing.value,
      document: cloneMacroDocument(macroEditing.value.document),
    })
    macroEditing.value = null
    await refreshAssets()
  } catch (error) {
    showError(t('assets.macros.save_failed'), error)
  } finally {
    macroEditBusy.value = false
  }
}

function recaptureVariant(): void {
  if (!variantAsset.value) return
  openResourceAction('recapture')
}

async function performRecaptureVariant(): Promise<void> {
  const asset = variantAsset.value
  if (!asset || !selectedTargetSlot.value || variantBusy.value) return
  variantBusy.value = true
  const id = `template-recapture-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
  try {
    const resultPromise = awaitWailsEvent<{ id: string; payload?: { cancelled?: boolean } }>(
      'tools:picker-result',
      (payload) => payload?.id === id,
    )
    await backend.tools.openScreenPicker(
      'template_recapture',
      id,
      selectedTargetSlot.value,
      '',
      asset.guid,
    )
    const result = await resultPromise
    if (!result.payload?.cancelled) {
      await refreshAssets()
      variantAsset.value =
        assetPage.value.find((candidate) => candidate.guid === asset.guid) ?? null
    }
  } catch (error) {
    showError(t('assets.templates.capture_failed'), error)
  } finally {
    variantBusy.value = false
  }
}

async function removeVariant(asset: AssetSummary, resolution: [number, number]): Promise<void> {
  if (asset.variantCount <= 1 || variantBusy.value) return
  variantBusy.value = true
  try {
    await backend.assets.removeVariant(asset.guid, resolution[0], resolution[1])
    await refreshAssets()
    variantAsset.value = assetPage.value.find((candidate) => candidate.guid === asset.guid) ?? null
  } catch (error) {
    showError(t('assets.delete_failed'), error)
  } finally {
    variantBusy.value = false
  }
}

async function cleanupLibrary(): Promise<void> {
  if (cleanupBusy.value) return
  cleanupBusy.value = true
  try {
    const preview = await backend.assets.previewCleanup()
    if (!preview) return
    if (preview.candidateCount === 0) {
      libraryFeedback.value = { tone: 'success', message: t('assets.cleanup_none') }
      return
    }
    const accepted = await confirm({
      title: t('assets.cleanup_title'),
      description: t('assets.cleanup_description', {
        count: preview.candidateCount,
        bytes: preview.candidateBytes,
      }),
      confirmText: t('assets.cleanup_action'),
      color: 'warning',
    })
    if (accepted !== true) return
    const result = (await backend.assets.commitCleanup(preview.token)) as
      | { reclaimed?: number }
      | undefined
    libraryFeedback.value = {
      tone: 'success',
      message: t('assets.cleanup_result', { count: result?.reclaimed ?? 0 }),
    }
  } catch (error) {
    showError(t('assets.cleanup_failed'), error)
  } finally {
    cleanupBusy.value = false
  }
}

function openEdit(item: AssetItem): void {
  editingItem.value = item
  editDraft.name = item.name
  editDraft.description = item.description
  editDraft.category = item.category
  editDraft.tags = [...item.tags]
}

async function saveAssetMeta(): Promise<void> {
  const item = editingItem.value
  if (!item || !editDraft.name.trim()) return
  editBusy.value = true
  try {
    const patch = {
      label: editDraft.name.trim(),
      description: editDraft.description.trim(),
      category: editDraft.category.trim(),
      tags: uniqueStrings(editDraft.tags),
    }
    await backend.assets.updateMeta(
      item.id,
      patch.label,
      patch.description,
      patch.category,
      patch.tags,
    )
    editingItem.value = null
    await refreshAssets()
  } catch (error) {
    showError(t('assets.save_failed'), error)
  } finally {
    editBusy.value = false
  }
}

async function deleteAsset(item: AssetItem): Promise<void> {
  const accepted = await confirm({
    title: t('assets.delete_title', { name: item.name }),
    description: t('assets.delete_description'),
    confirmText: t('common.delete'),
    color: 'error',
  })
  if (accepted !== true) return
  try {
    await backend.assets.delete_(item.id)
    const next = { ...selected.value }
    delete next[item.id]
    selected.value = next
    await refreshAssets()
  } catch (error) {
    showError(t('assets.delete_failed'), error)
  }
}

function createAssetCategory(value: string, draft: AssetMetadataDraft): void {
  const category = value.trim()
  if (!category) return
  createdCategories.value = uniqueStrings([...createdCategories.value, category])
  draft.category = category
}

function createBatchCategory(value: string): void {
  createAssetCategory(value, batchDraft)
}

function createEditCategory(value: string): void {
  createAssetCategory(value, editDraft)
}

function createRecordingCategory(value: string): void {
  createAssetCategory(value, recordingDraft)
}

function createMacroCategory(value: string): void {
  const category = value.trim()
  if (!macroEditing.value || !category) return
  createdCategories.value = uniqueStrings([...createdCategories.value, category])
  macroEditing.value.category = category
}

function createAssetTag(value: string, draft: AssetMetadataDraft): void {
  const tag = value.trim()
  if (!tag) return
  createdTags.value = uniqueStrings([...createdTags.value, tag])
  draft.tags = uniqueStrings([...draft.tags, tag])
}

function createBatchTag(value: string): void {
  createAssetTag(value, batchDraft)
}

function createEditTag(value: string): void {
  createAssetTag(value, editDraft)
}

function createRecordingTag(value: string): void {
  createAssetTag(value, recordingDraft)
}

function createMacroTag(value: string): void {
  const tag = value.trim()
  if (!macroEditing.value || !tag) return
  createdTags.value = uniqueStrings([...createdTags.value, tag])
  macroEditing.value.tags = uniqueStrings([...(macroEditing.value.tags ?? []), tag])
}

function uniqueStrings(values: string[]): string[] {
  const seen = new Set<string>()
  return values
    .map((value) => value.trim())
    .filter((value) => {
      const key = value.toLocaleLowerCase()
      if (!key || seen.has(key)) return false
      seen.add(key)
      return true
    })
}

function formatDuration(durationUs: number): string {
  const seconds = Math.max(0, Math.round(durationUs / 1_000_000))
  return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')}`
}

function formatAssetDate(value?: string): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return new Intl.DateTimeFormat(undefined, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(date)
}

function showError(title: string, error: unknown): void {
  toast.add({
    title,
    description: errorMessage(error),
    color: 'error',
  })
}
</script>
