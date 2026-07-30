<template>
  <div class="settings-page">
    <SettingsSection
      :title="t('settingsLauncher.access_title')"
      :description="t('settingsLauncher.access_hint')"
      icon="i-tabler-rocket"
    >
      <template #actions>
        <UButton
          size="sm"
          color="primary"
          variant="soft"
          icon="i-tabler-window"
          :loading="launcherOpening"
          @click="openLauncher"
        >
          {{ t('settingsLauncher.open_now') }}
        </UButton>
      </template>
      <div class="settings-inset flex flex-col gap-3 sm:flex-row sm:items-center">
        <div class="min-w-0 flex-1">
          <p class="settings-detail__label">{{ t('settingsLauncher.hotkey_title') }}</p>
          <p class="settings-detail__hint">
            {{ t(`settingsLauncher.hotkey_${launcherHotkeyStatus}`) }}
          </p>
          <p v-if="launcherHotkey?.lastError" class="mt-1 break-all text-xs text-error">
            {{ launcherHotkey.lastError }}
          </p>
        </div>
        <UKbd v-if="launcherHotkey?.hotkeyStr" :value="launcherHotkey.hotkeyStr" />
        <UButton
          to="/settings?section=hotkeys"
          size="xs"
          color="neutral"
          variant="ghost"
          icon="i-tabler-keyboard"
        >
          {{ t('settingsLauncher.configure_hotkey') }}
        </UButton>
      </div>
    </SettingsSection>

    <SettingsSection
      :title="t('settingsLauncher.appearance_title')"
      :description="t('settingsLauncher.display_hint')"
      icon="i-tabler-adjustments-horizontal"
    >
      <SettingsRow :label="t('settingsLauncher.display_label')">
        <AdaptiveSelect
          :model-value="display"
          :items="displayItems"
          :aria-label="t('settingsLauncher.display_label')"
          @update:model-value="setDisplay"
        />
      </SettingsRow>
      <SettingsRow :label="t('settingsLauncher.size_label')">
        <div class="flex items-center rounded-lg border border-default p-0.5">
          <UButton
            v-for="item in sizeItems"
            :key="item.value"
            size="xs"
            :color="size === item.value ? 'primary' : 'neutral'"
            :variant="size === item.value ? 'soft' : 'ghost'"
            :aria-pressed="size === item.value"
            @click="setSize(item.value)"
          >
            {{ item.label }}
          </UButton>
        </div>
      </SettingsRow>
    </SettingsSection>

    <SettingsSection
      :title="t('settingsLauncher.health_title')"
      :description="t('settingsLauncher.health_hint')"
      icon="i-tabler-shield-check"
      :badge="healthBadge"
    >
      <div
        v-if="dependenciesLoaded"
        class="launcher-health-card"
        :class="{ 'launcher-health-card--warning': staleCount }"
      >
        <div class="launcher-health-stats">
          <div>
            <strong>{{ resolution.items.length }}</strong>
            <span>{{ t('settingsLauncher.health_available') }}</span>
          </div>
          <div>
            <strong :class="{ 'text-warning': staleCount }">{{ staleCount }}</strong>
            <span>{{ t('settingsLauncher.health_stale') }}</span>
          </div>
        </div>

        <div v-if="staleCount" class="launcher-health-action">
          <span class="launcher-health-action__icon">
            <UIcon name="i-tabler-unlink" class="size-4" />
          </span>
          <div class="min-w-0 flex-1">
            <p class="text-xs font-semibold text-highlighted">
              {{ t('settingsLauncher.stale_title', { n: staleCount }) }}
            </p>
            <ul class="launcher-stale-list" :aria-label="t('settingsLauncher.cleanup_scope')">
              <li v-for="item in resolution.staleBlocks" :key="item.id">
                {{ staleBlockName(item) }}
              </li>
            </ul>
          </div>
          <UButton
            size="xs"
            color="warning"
            variant="soft"
            icon="i-tabler-eraser"
            :loading="cleanupBusy"
            @click="cleanupStale"
          >
            {{ t('settingsLauncher.cleanup_stale', { n: staleCount }) }}
          </UButton>
        </div>
        <div v-else class="launcher-health-action launcher-health-action--ready">
          <span class="launcher-health-action__icon">
            <UIcon name="i-tabler-check" class="size-4" />
          </span>
          <div class="min-w-0 flex-1">
            <p class="text-xs font-semibold text-highlighted">
              {{ t('settingsLauncher.health_ready') }}
            </p>
            <p class="mt-1 text-[11px] leading-4 text-muted">
              {{ t('settingsLauncher.health_ready_hint') }}
            </p>
          </div>
          <UButton
            v-if="cleanupUndo"
            size="xs"
            color="neutral"
            variant="outline"
            icon="i-tabler-arrow-back-up"
            :loading="cleanupBusy"
            @click="undoCleanup"
          >
            {{ t('settingsLauncher.undo_cleanup') }}
          </UButton>
        </div>
      </div>
      <div v-else class="launcher-health-loading" aria-busy="true">
        <USkeleton class="h-14 flex-1" />
        <USkeleton class="h-14 flex-1" />
        <USkeleton class="h-14 flex-1" />
      </div>
    </SettingsSection>

    <SettingsSection
      :title="t('settingsLauncher.layout_title')"
      :description="t('settingsLauncher.layout_hint')"
      icon="i-tabler-layout-list"
      :badge="String(editItems.length)"
    >
      <div class="min-w-0 space-y-3">
        <div class="launcher-library">
          <div class="flex flex-wrap items-center gap-2">
            <UButton
              size="sm"
              color="primary"
              variant="soft"
              icon="i-tabler-layout-grid-add"
              @click="workflowPickerOpen = true"
            >
              {{ t('settingsLauncher.add_workflow') }}
            </UButton>
            <UButton
              size="xs"
              variant="soft"
              color="neutral"
              icon="i-tabler-heading"
              @click="addLabel"
            >
              {{ t('settingsLauncher.label_block') }}
            </UButton>
            <UDropdownMenu :items="separatorMenuItems">
              <UButton
                size="xs"
                variant="soft"
                color="neutral"
                icon="i-tabler-separator-horizontal"
                trailing-icon="i-tabler-chevron-down"
              >
                {{ t('settingsLauncher.separator_block') }}
              </UButton>
            </UDropdownMenu>
          </div>
          <p class="mt-2 text-[11px] leading-4 text-dimmed">
            {{ t('settingsLauncher.insert_hint') }}
          </p>
        </div>

        <div v-if="editItems.length === 0" class="settings-empty-state">
          <UIcon name="i-tabler-layout-off" class="size-6 text-dimmed" />
          <p class="text-sm font-medium text-default">{{ t('settingsLauncher.empty') }}</p>
        </div>
        <VueDraggable
          v-else
          v-model="editItems"
          :animation="150"
          handle=".drag-h"
          class="launcher-block-list"
          @end="persist"
        >
          <article
            v-for="(block, index) in editItems"
            :key="block.id"
            class="launcher-block"
            :class="{ 'launcher-block--selected': selectedBlockId === block.id }"
            @click="selectBlock(block.id)"
            @focusin="selectBlock(block.id)"
          >
            <UIcon
              name="i-tabler-grip-vertical"
              class="drag-h size-4 shrink-0 cursor-grab text-dimmed"
            />
            <template v-if="block.type === 'workflow'">
              <UPopover :ui="{ content: 'w-[300px] p-2' }">
                <UButton
                  size="xs"
                  variant="outline"
                  color="neutral"
                  square
                  :title="t('settingsLauncher.pick_icon')"
                  :aria-label="t('settingsLauncher.pick_icon')"
                >
                  <UIcon :name="block.icon || 'i-tabler-photo-plus'" class="size-4" />
                </UButton>
                <template #content>
                  <div class="space-y-2">
                    <IconPicker
                      :model-value="block.icon"
                      @update:model-value="(value: string) => setIcon(block.id, value)"
                    />
                    <UButton
                      v-if="block.icon"
                      size="xs"
                      variant="ghost"
                      color="neutral"
                      block
                      @click="setIcon(block.id, '')"
                    >
                      {{ t('settingsLauncher.clear_icon') }}
                    </UButton>
                  </div>
                </template>
              </UPopover>
              <div class="min-w-0 flex-1">
                <UInput
                  :model-value="block.label"
                  size="sm"
                  :placeholder="workflowName(block.workflowId)"
                  :aria-label="t('settingsLauncher.label_placeholder')"
                  @update:model-value="
                    (value: string | number) => setLabel(block.id, String(value))
                  "
                  @change="persist"
                />
                <p class="mt-1 truncate text-[11px] text-dimmed">
                  {{
                    t('settingsLauncher.from_workflow', {
                      name: workflowName(block.workflowId),
                    })
                  }}
                  <span v-if="addedCounts[block.workflowId ?? ''] > 1">
                    ·
                    {{
                      t('settingsLauncher.entry_count', {
                        n: addedCounts[block.workflowId ?? ''],
                      })
                    }}
                  </span>
                </p>
              </div>
            </template>
            <template v-else-if="block.type === 'label'">
              <UIcon name="i-tabler-heading" class="size-4 shrink-0 text-dimmed" />
              <UInput
                :model-value="block.label"
                class="min-w-0 flex-1"
                size="sm"
                :placeholder="t('settingsLauncher.label_placeholder')"
                @update:model-value="(value: string | number) => setLabel(block.id, String(value))"
                @change="persist"
              />
            </template>
            <div v-else class="flex min-w-0 flex-1 items-center gap-2 text-xs text-dimmed">
              <UIcon
                :name="
                  block.type === 'hsep'
                    ? 'i-tabler-separator-horizontal'
                    : 'i-tabler-separator-vertical'
                "
                class="size-4"
              />
              {{ t(block.type === 'hsep' ? 'settingsLauncher.hsep' : 'settingsLauncher.vsep') }}
            </div>
            <div class="ml-auto flex shrink-0 items-center">
              <UButton
                size="xs"
                variant="ghost"
                color="neutral"
                icon="i-tabler-arrow-up"
                :disabled="index === 0"
                :aria-label="t('settingsLauncher.move_up')"
                @click="moveBlock(index, index - 1)"
              />
              <UButton
                size="xs"
                variant="ghost"
                color="neutral"
                icon="i-tabler-arrow-down"
                :disabled="index === editItems.length - 1"
                :aria-label="t('settingsLauncher.move_down')"
                @click="moveBlock(index, index + 1)"
              />
              <UButton
                size="xs"
                variant="ghost"
                color="error"
                icon="i-tabler-trash"
                :aria-label="t('settingsLauncher.delete_block')"
                @click="removeBlock(block.id)"
              />
            </div>
          </article>
        </VueDraggable>
      </div>
    </SettingsSection>

    <WorkflowPickerModal
      v-model:open="workflowPickerOpen"
      :added-counts="addedCounts"
      @add="addWorkflows"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { VueDraggable } from 'vue-draggable-plus'
import { useSettingsStore, type LauncherBlock } from '@/stores/settings'
import { useHotkeysStore } from '@/stores/hotkeys'
import { backend } from '@/lib/backend'
import { workflowTransport, type SourceView } from '@/app/transport/workflow'
import IconPicker from '@/components/common/IconPicker.vue'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'
import WorkflowPickerModal from '@/components/launcher/WorkflowPickerModal.vue'
import SettingsRow from '@/components/settings/SettingsRow.vue'
import SettingsSection from '@/components/settings/SettingsSection.vue'
import {
  cleanupStaleLauncherBlocks,
  normalizeLauncherDisplay,
  normalizeLauncherSize,
  resolveLauncher,
  type LauncherDisplay,
  type LauncherSize,
} from '@/components/launcher/launcherModel'

const settingsStore = useSettingsStore()
const hotkeysStore = useHotkeysStore()
const { t } = useI18n()
const workflows = ref<SourceView[]>([])
const editItems = ref<LauncherBlock[]>([])
const cleanupBusy = ref(false)
const launcherOpening = ref(false)
const dependenciesLoaded = ref(false)
const cleanupUndo = ref<LauncherBlock[] | null>(null)
const workflowPickerOpen = ref(false)
const selectedBlockId = ref('')
const launcherHotkey = computed(() =>
  hotkeysStore.list.find((entry) => entry.key === 'system.launcher-toggle'),
)
const launcherHotkeyStatus = computed(() => launcherHotkey.value?.status ?? 'unbound')
const copyItems = (items: LauncherBlock[]) => items.map((block) => ({ ...block }))
const syncFromStore = () =>
  (editItems.value = copyItems(settingsStore.data?.ui.launcherItems ?? []))
watch(() => settingsStore.data?.ui.launcherItems, syncFromStore, { immediate: true })

const display = computed<LauncherDisplay>(() =>
  normalizeLauncherDisplay(settingsStore.data?.ui.launcherDisplay),
)
const size = computed<LauncherSize>(() =>
  normalizeLauncherSize(settingsStore.data?.ui.launcherSize),
)
const displayItems = computed(() => [
  { label: t('settingsLauncher.display_both'), value: 'both' },
  { label: t('settingsLauncher.display_icon'), value: 'icon' },
  { label: t('settingsLauncher.display_text'), value: 'text' },
])
const sizeItems = computed<Array<{ label: string; value: LauncherSize }>>(() => [
  { label: t('settingsLauncher.size_xsmall'), value: 'xsmall' },
  { label: t('settingsLauncher.size_small'), value: 'small' },
  { label: t('settingsLauncher.size_medium'), value: 'medium' },
  { label: t('settingsLauncher.size_large'), value: 'large' },
])
const addedCounts = computed<Record<string, number>>(() => {
  const counts: Record<string, number> = {}
  for (const item of editItems.value) {
    if (item.type !== 'workflow' || !item.workflowId) continue
    counts[item.workflowId] = (counts[item.workflowId] ?? 0) + 1
  }
  return counts
})
const resolution = computed(() => resolveLauncher(editItems.value, workflows.value))
const staleCount = computed(() => resolution.value.staleBlocks.length)
const healthBadge = computed(() =>
  staleCount.value ? t('settingsLauncher.health_attention') : t('settingsLauncher.health_normal'),
)
const persist = () => settingsStore.patch({ ui: { launcherItems: copyItems(editItems.value) } })
const setDisplay = (value: string) => void settingsStore.patch({ ui: { launcherDisplay: value } })
const setSize = (value: LauncherSize) => void settingsStore.patch({ ui: { launcherSize: value } })
const separatorMenuItems = computed(() => [
  [
    {
      label: t('settingsLauncher.hsep'),
      icon: 'i-tabler-separator-horizontal',
      onSelect: addHsep,
    },
    {
      label: t('settingsLauncher.vsep'),
      icon: 'i-tabler-separator-vertical',
      onSelect: addVsep,
    },
  ],
])
const block = (id: string) => editItems.value.find((item) => item.id === id)
const genId = () => `lb_${crypto.randomUUID()}`

function moveBlock(from: number, to: number) {
  if (to < 0 || to >= editItems.value.length) return
  const [item] = editItems.value.splice(from, 1)
  if (!item) return
  editItems.value.splice(to, 0, item)
  persist()
}
function insertBlocks(blocks: LauncherBlock[]) {
  const selectedIndex = editItems.value.findIndex((item) => item.id === selectedBlockId.value)
  const insertAt = selectedIndex < 0 ? editItems.value.length : selectedIndex + 1
  editItems.value.splice(insertAt, 0, ...blocks)
  selectedBlockId.value = blocks.at(-1)?.id ?? selectedBlockId.value
  persist()
}
function addWorkflows(selected: SourceView[]) {
  const blocks = selected.map<LauncherBlock>((workflow) => ({
    id: genId(),
    type: 'workflow',
    workflowId: workflow.workflowId,
    icon: '',
    label: '',
  }))
  if (blocks.length) insertBlocks(blocks)
}
function addLabel() {
  insertBlocks([{ id: genId(), type: 'label', label: '' }])
}
function addHsep() {
  insertBlocks([{ id: genId(), type: 'hsep' }])
}
function addVsep() {
  insertBlocks([{ id: genId(), type: 'vsep' }])
}
function removeBlock(id: string) {
  editItems.value = editItems.value.filter((item) => item.id !== id)
  if (selectedBlockId.value === id) selectedBlockId.value = ''
  persist()
}
function selectBlock(id: string) {
  selectedBlockId.value = id
}
async function cleanupStale() {
  if (!staleCount.value || cleanupBusy.value) return
  cleanupBusy.value = true
  const previousBlocks = copyItems(editItems.value)
  const cleaned = cleanupStaleLauncherBlocks(
    editItems.value,
    new Set(workflows.value.map((workflow) => workflow.workflowId)),
  )
  editItems.value = copyItems(cleaned.blocks)
  const saved = await persist()
  if (!saved) {
    editItems.value = previousBlocks
    cleanupBusy.value = false
    return
  }
  cleanupUndo.value = previousBlocks
  cleanupBusy.value = false
}
async function undoCleanup() {
  const snapshot = cleanupUndo.value
  if (!snapshot || cleanupBusy.value) return
  cleanupBusy.value = true
  const currentBlocks = copyItems(editItems.value)
  editItems.value = copyItems(snapshot)
  const saved = await persist()
  if (saved) {
    cleanupUndo.value = null
  } else {
    editItems.value = currentBlocks
  }
  cleanupBusy.value = false
}
function setIcon(id: string, icon: string) {
  const item = block(id)
  if (item) {
    item.icon = icon
    persist()
  }
}
function setLabel(id: string, label: string) {
  const item = block(id)
  if (item) item.label = label
}
function workflowName(id?: string) {
  return (
    workflows.value.find((workflow) => workflow.workflowId === id)?.name ??
    t('settingsLauncher.deleted_workflow')
  )
}
function staleBlockName(item: LauncherBlock) {
  return item.label?.trim() || item.workflowId || t('settingsLauncher.deleted_workflow')
}

async function openLauncher(): Promise<void> {
  if (launcherOpening.value) return
  launcherOpening.value = true
  await backend.tools.openLauncher()
  launcherOpening.value = false
}

onMounted(async () => {
  const [, listed] = await Promise.all([hotkeysStore.reload(), workflowTransport.listSources()])
  workflows.value = listed
  dependenciesLoaded.value = true
})
</script>

<style scoped>
.launcher-block {
  display: flex;
  min-height: 60px;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border: 0;
  border-radius: 0;
  background: transparent;
  transition:
    border-color 140ms ease,
    background-color 140ms ease;
}

.launcher-block:hover {
  background: var(--settings-row-hover-bg);
}

.launcher-block--selected {
  background: color-mix(in oklab, var(--ui-primary) 7%, var(--ui-bg));
}

.launcher-block-list {
  overflow: hidden;
  border-block: 1px solid var(--ui-border);
}

.launcher-block-list > * + * {
  border-top: 1px solid color-mix(in oklab, var(--ui-border) 76%, transparent);
}

.launcher-library {
  padding: 14px 16px;
  border-block: 1px dashed var(--ui-border);
  background: transparent;
}

.launcher-health-card {
  overflow: hidden;
  border-block: 1px solid var(--ui-border);
  background: transparent;
}

.launcher-health-card--warning {
  border-color: color-mix(in oklab, var(--ui-warning) 28%, var(--ui-border));
}

.launcher-health-loading {
  display: flex;
  gap: 1px;
  overflow: hidden;
  border-block: 1px solid var(--ui-border);
}

.launcher-health-stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  border-bottom: 1px solid var(--ui-border);
}

.launcher-health-stats > div {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
  padding: 14px 16px;
}

.launcher-health-stats > div + div {
  border-left: 1px solid var(--ui-border);
}

.launcher-health-stats strong {
  color: var(--ui-text-highlighted);
  font-size: 20px;
  font-weight: 650;
  line-height: 24px;
  font-variant-numeric: tabular-nums;
}

.launcher-health-stats span {
  overflow: hidden;
  color: var(--ui-text-dimmed);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.launcher-health-action {
  display: flex;
  min-height: 64px;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
}

.launcher-health-action__icon {
  display: inline-flex;
  width: 32px;
  height: 32px;
  flex: none;
  align-items: center;
  justify-content: center;
  border: 1px solid color-mix(in oklab, var(--ui-warning) 30%, var(--ui-border));
  border-radius: 8px;
  color: var(--ui-warning);
  background: color-mix(in oklab, var(--ui-warning) 8%, transparent);
}

.launcher-health-action--ready .launcher-health-action__icon {
  color: var(--ui-success);
  border-color: color-mix(in oklab, var(--ui-success) 28%, var(--ui-border));
  background: color-mix(in oklab, var(--ui-success) 7%, transparent);
}

.launcher-stale-list {
  display: flex;
  max-height: 72px;
  flex-wrap: wrap;
  gap: 4px;
  overflow: auto;
  margin-top: 6px;
  padding-right: 4px;
  list-style: none;
}

.launcher-stale-list li {
  max-width: 100%;
  padding: 2px 6px;
  border: 1px solid color-mix(in oklab, var(--ui-warning) 22%, var(--ui-border));
  border-radius: 5px;
  color: var(--ui-text-muted);
  background: color-mix(in oklab, var(--ui-warning) 5%, transparent);
  font-size: 10px;
  line-height: 14px;
  overflow-wrap: anywhere;
}
</style>
