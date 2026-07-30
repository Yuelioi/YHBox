<template>
  <div class="settings-page">
    <div class="settings-toolbar">
      <UInput
        v-model="searchText"
        :placeholder="t('hotkeys.search_placeholder')"
        icon="i-tabler-search"
        class="min-w-0 flex-1"
        :aria-label="t('hotkeys.search_placeholder')"
      />
      <AdaptiveSelect
        v-model="statusFilter"
        :items="filterItems"
        class="shrink-0"
        :aria-label="t('hotkeys.filter_label')"
      />
      <UDropdownMenu :items="resetMenuItems">
        <UButton
          color="neutral"
          variant="outline"
          icon="i-tabler-restore"
          trailing-icon="i-tabler-chevron-down"
        >
          {{ t('hotkeys.reset_menu') }}
        </UButton>
      </UDropdownMenu>
    </div>

    <div class="flex flex-wrap items-center gap-2" aria-live="polite">
      <UBadge color="neutral" variant="subtle" icon="i-tabler-keyboard">
        {{ t('hotkeys.summary.total', { n: store.list.length }) }}
      </UBadge>
      <UBadge color="error" variant="subtle" icon="i-tabler-alert-circle">
        {{ t('hotkeys.summary.failed', { n: failedCount }) }}
      </UBadge>
      <UBadge color="neutral" variant="subtle" icon="i-tabler-keyboard-off">
        {{ t('hotkeys.summary.unbound', { n: unboundCount }) }}
      </UBadge>
    </div>

    <SettingsSection
      v-for="group in filteredGrouped"
      :key="group.source"
      :title="groupLabel(group.source)"
      :description="t(`hotkeys.group_hint.${group.source}`)"
      :icon="groupIcon(group.source)"
    >
      <template #badge>
        <UBadge color="neutral" variant="subtle" size="xs">{{ group.entries.length }}</UBadge>
      </template>

      <div class="settings-collection">
        <div v-for="entry in group.entries" :key="entry.key" class="hotkey-row">
          <div class="min-w-0 flex-1">
            <div class="truncate text-sm font-medium text-default">
              {{ t(entry.label, entry.labelParams ?? {}) }}
            </div>
            <div
              v-if="entry.lastError"
              class="mt-1 flex items-start gap-1.5 text-xs leading-relaxed text-error"
              role="alert"
            >
              <UIcon name="i-tabler-alert-circle" class="mt-0.5 size-3.5 shrink-0" />
              <span>{{ t('hotkeys.status.register_failed') }}：{{ entry.lastError }}</span>
            </div>
            <div v-else-if="entry.readonlyReason" class="mt-1 text-xs text-dimmed">
              {{ entry.readonlyReason }}
            </div>
            <div v-else-if="entry.status === 'unbound'" class="mt-1 text-xs text-dimmed">
              {{ t('hotkeys.status.unbound') }}
            </div>
          </div>
          <HotkeyCaptureInput
            class="w-full sm:w-56 sm:shrink-0"
            :model-value="entry.hotkeyStr"
            :disabled="!!entry.readonlyReason"
            :aria-label="
              t('hotkeys.capture_aria', { name: t(entry.label, entry.labelParams ?? {}) })
            "
            @update:model-value="(value: string) => onUpdate(entry.key, value)"
          />
        </div>
      </div>
    </SettingsSection>

    <div v-if="filteredGrouped.length === 0" class="settings-empty-state">
      <UIcon name="i-tabler-search-off" class="size-6 text-dimmed" aria-hidden="true" />
      <div>
        <p class="text-sm font-medium text-default">{{ t('hotkeys.empty') }}</p>
        <p class="mt-1 text-xs text-dimmed">{{ t('hotkeys.empty_hint') }}</p>
      </div>
      <UButton size="xs" variant="soft" color="neutral" @click="clearFilters">
        {{ t('hotkeys.clear_filters') }}
      </UButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import { useHotkeysStore } from '@/stores/hotkeys'
import { errorMessage } from '@/lib/invoke'
import { useConfirm } from '@/composables/useConfirm'
import { backend } from '@/lib/backend'
import HotkeyCaptureInput from '@/components/hotkeys/HotkeyCaptureInput.vue'
import SettingsSection from '@/components/settings/SettingsSection.vue'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'

type StatusFilter = 'all' | 'failed' | 'unbound'

const { t } = useI18n()
const toast = useToast()
const store = useHotkeysStore()
const { confirm } = useConfirm()
const searchText = ref('')
const statusFilter = ref<StatusFilter>('all')

onMounted(() => void store.reload().catch(showError))

const failedCount = computed(() => store.list.filter((entry) => entry.status === 'failed').length)
const unboundCount = computed(() => store.list.filter((entry) => entry.status === 'unbound').length)
const filterItems = computed(() => [
  { label: t('hotkeys.filter.all'), value: 'all' },
  { label: t('hotkeys.filter.failed'), value: 'failed' },
  { label: t('hotkeys.filter.unbound'), value: 'unbound' },
])

const filteredGrouped = computed(() => {
  const query = searchText.value.trim().toLocaleLowerCase()
  const filtered = store.list.filter((entry) => {
    if (statusFilter.value !== 'all' && entry.status !== statusFilter.value) return false
    if (!query) return true
    const label = t(entry.label, entry.labelParams ?? {})
    return (
      label.toLocaleLowerCase().includes(query) ||
      entry.hotkeyStr.toLocaleLowerCase().includes(query)
    )
  })
  const groups: Record<string, typeof filtered> = {
    system: [],
    recording: [],
    action: [],
    schedule: [],
    editor: [],
  }
  for (const entry of filtered) groups[entry.source]?.push(entry)
  for (const key of Object.keys(groups)) {
    groups[key].sort((a, b) =>
      t(a.label, a.labelParams ?? {}).localeCompare(t(b.label, b.labelParams ?? {}), 'zh'),
    )
  }
  return Object.entries(groups)
    .filter(([, entries]) => entries.length > 0)
    .map(([source, entries]) => ({ source, entries }))
})

const resetMenuItems = computed(() => [
  [
    {
      label: t('hotkeys.reset_system'),
      icon: 'i-tabler-restore',
      onSelect: onResetSystem,
    },
  ],
])

function groupIcon(source: string): string {
  return (
    {
      system: 'i-tabler-tool',
      recording: 'i-tabler-player-record',
      action: 'i-tabler-bolt',
      schedule: 'i-tabler-calendar-clock',
      editor: 'i-tabler-edit',
    }[source] ?? 'i-tabler-keyboard'
  )
}

function groupLabel(source: string): string {
  return t(`hotkeys.group.${source}`)
}

function clearFilters() {
  searchText.value = ''
  statusFilter.value = 'all'
}

async function onUpdate(key: string, hotkeyStr: string) {
  try {
    await store.update(key, hotkeyStr)
  } catch (error) {
    showError(error)
  }
}

async function onResetSystem() {
  const ok = await confirm({
    title: t('hotkeys.confirm.reset_title'),
    description: t('hotkeys.confirm.reset_desc'),
    confirmText: t('hotkeys.confirm.reset_ok'),
    cancelText: t('common.cancel'),
    color: 'warning',
  })
  if (ok !== true) return
  try {
    await backend.hotkeys.resetSystemDefaults()
    await store.reload()
  } catch (error) {
    showError(error)
  }
}

function showError(error: unknown): void {
  toast.add({
    title: t('toast.operation_failed'),
    description: errorMessage(error),
    color: 'error',
  })
}
</script>

<style scoped>
.settings-toolbar {
  position: sticky;
  top: -28px;
  z-index: 10;
  display: flex;
  align-items: center;
  gap: 8px;
  padding-block: 12px;
  background: color-mix(in oklab, var(--ui-bg) 94%, transparent);
  backdrop-filter: blur(10px);
}

.hotkey-row {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 12px 0;
}

@media (max-width: 860px) {
  .settings-toolbar,
  .hotkey-row {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
