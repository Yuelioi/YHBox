<template>
  <div
    data-testid="assets-view"
    class="mx-auto flex h-full min-h-0 w-full max-w-7xl flex-col px-6 py-5"
  >
    <header class="flex shrink-0 items-start justify-between gap-6 border-b border-default pb-5">
      <div>
        <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-primary">
          {{ t('assets.eyebrow') }}
        </p>
        <h1 class="mt-1 text-xl font-semibold text-highlighted">{{ t('assets.title') }}</h1>
        <p class="mt-1 max-w-2xl text-sm leading-6 text-muted">{{ t('assets.description') }}</p>
      </div>
      <USelect
        v-model="selectedTargetSlot"
        :items="targetItems"
        value-key="value"
        label-key="label"
        class="w-64 shrink-0"
        :placeholder="t('assets.target_placeholder')"
        :aria-label="t('assets.target_placeholder')"
      />
    </header>

    <section class="mt-5 shrink-0 rounded-xl border border-default bg-default p-4">
      <span data-testid="assets-recording-controls" class="sr-only">{{
        t('assets.recording.title')
      }}</span>
      <div class="flex items-center gap-4">
        <div class="flex size-10 shrink-0 items-center justify-center rounded-lg bg-primary/10">
          <UIcon name="i-tabler-player-record" class="size-5 text-primary" />
        </div>
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-2">
            <h2 class="text-sm font-semibold text-highlighted">
              {{ t('assets.recording.title') }}
            </h2>
            <UBadge :color="recordingBadge.color" variant="soft" size="sm">
              {{ recordingBadge.label }}
            </UBadge>
          </div>
          <p class="mt-1 text-xs leading-5 text-muted">{{ recordingHint }}</p>
        </div>
        <div class="flex shrink-0 items-center gap-2">
          <UButton
            v-if="recording.state.phase === 'recording'"
            color="neutral"
            variant="soft"
            icon="i-tabler-player-pause"
            :label="t('recordingHud.pause')"
            @click="pauseRecording"
          />
          <UButton
            v-if="recording.state.phase === 'paused'"
            color="neutral"
            variant="soft"
            icon="i-tabler-player-play"
            :label="t('recordingHud.resume')"
            @click="resumeRecording"
          />
          <UButton
            v-if="recording.state.phase === 'recording' || recording.state.phase === 'paused'"
            color="error"
            variant="soft"
            icon="i-tabler-square"
            :label="t('recordingHud.stop')"
            @click="stopRecording"
          />
          <UButton
            v-else
            icon="i-tabler-player-record"
            :label="t('assets.recording.start')"
            :disabled="!selectedTargetSlot || recording.state.phase === 'finalizing'"
            @click="startRecording"
          />
        </div>
      </div>
    </section>

    <div class="mt-5 flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl border border-default">
      <div class="flex shrink-0 items-center gap-2 border-b border-default bg-default px-4 py-3">
        <UButton
          :label="t('assets.tabs.clips')"
          icon="i-tabler-movie"
          color="neutral"
          :variant="activeTab === 'clips' ? 'soft' : 'ghost'"
          @click="activeTab = 'clips'"
        />
        <UButton
          :label="t('assets.tabs.templates')"
          icon="i-tabler-photo"
          color="neutral"
          :variant="activeTab === 'templates' ? 'soft' : 'ghost'"
          @click="activeTab = 'templates'"
        />
        <div class="mx-2 h-6 w-px bg-default" />
        <UInput
          v-model="query"
          icon="i-tabler-search"
          class="max-w-sm flex-1"
          :placeholder="t('assets.search_placeholder')"
          :aria-label="t('assets.search_placeholder')"
        />
        <div class="flex-1" />
        <UButton
          v-if="activeTab === 'templates'"
          color="primary"
          variant="soft"
          icon="i-tabler-camera-plus"
          :label="t('assets.templates.capture')"
          :disabled="!selectedTargetSlot"
          :loading="captureBusy"
          @click="captureTemplate"
        />
        <UButton
          color="neutral"
          variant="ghost"
          icon="i-tabler-refresh"
          :aria-label="t('common.refresh')"
          @click="refreshAssets"
        />
      </div>

      <div class="min-h-0 flex-1 overflow-y-auto bg-elevated/15 p-4">
        <div
          v-if="visibleItems.length"
          class="grid grid-cols-1 gap-3 lg:grid-cols-2 2xl:grid-cols-3"
        >
          <article
            v-for="item in visibleItems"
            :key="item.id"
            class="flex min-w-0 items-start gap-3 rounded-xl border border-default bg-default p-4 transition-colors hover:border-accented"
          >
            <div
              class="flex size-10 shrink-0 items-center justify-center rounded-lg bg-elevated/70 text-primary"
            >
              <UIcon :name="item.icon" class="size-5" />
            </div>
            <div class="min-w-0 flex-1">
              <div class="flex items-start gap-2">
                <div class="min-w-0 flex-1">
                  <h3 class="truncate text-sm font-medium text-highlighted">{{ item.name }}</h3>
                  <p class="mt-0.5 text-[11px] text-dimmed">{{ item.meta }}</p>
                </div>
                <UDropdownMenu :items="assetMenu(item)">
                  <UButton
                    icon="i-tabler-dots"
                    color="neutral"
                    variant="ghost"
                    size="xs"
                    :aria-label="t('assets.asset_actions', { name: item.name })"
                  />
                </UDropdownMenu>
              </div>
              <p v-if="item.description" class="mt-2 line-clamp-2 text-xs leading-5 text-muted">
                {{ item.description }}
              </p>
              <div v-if="item.category || item.tags.length" class="mt-3 flex flex-wrap gap-1.5">
                <UBadge v-if="item.category" color="neutral" variant="soft" size="sm">
                  {{ item.category }}
                </UBadge>
                <UBadge
                  v-for="tag in item.tags"
                  :key="tag"
                  color="primary"
                  variant="subtle"
                  size="sm"
                >
                  {{ tag }}
                </UBadge>
              </div>
            </div>
          </article>
        </div>
        <EmptyState
          v-else
          :icon="
            query
              ? 'i-tabler-search-off'
              : activeTab === 'clips'
                ? 'i-tabler-movie-off'
                : 'i-tabler-photo-off'
          "
          :title="query ? t('assets.no_results') : t(`assets.${activeTab}.empty`)"
          :description="query ? t('assets.no_results_hint') : t(`assets.${activeTab}.empty_hint`)"
        >
          <template v-if="!query && activeTab === 'templates'" #action>
            <UButton
              icon="i-tabler-camera-plus"
              :label="t('assets.templates.capture')"
              :disabled="!selectedTargetSlot"
              @click="captureTemplate"
            />
          </template>
        </EmptyState>
      </div>
    </div>
  </div>

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
        <UInput v-model="editDraft.category" />
      </UFormField>
      <UFormField :label="t('common.tags')" :hint="t('assets.tags_hint')">
        <UInput v-model="editDraft.tags" />
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
      <div class="rounded-lg border border-default bg-elevated/35 px-4 py-3">
        <p class="text-sm font-medium text-highlighted">{{ t('recordingSave.clip_type') }}</p>
        <p class="mt-1 text-xs text-muted">
          {{
            t('recordingSave.summary', {
              duration: formatDuration(pendingRecording.durationUs),
              count: pendingRecording.eventCount,
            })
          }}
        </p>
      </div>
      <UFormField :label="t('recordingSave.name')" required>
        <UInput v-model="recordingDraft.name" autofocus maxlength="80" />
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
        @click="discardRecording"
      >
        {{ t('recordingSave.discard') }}
      </UButton>
      <UButton
        :loading="recordingSaveBusy"
        :disabled="!recordingDraft.name.trim()"
        @click="saveRecording"
      >
        {{ t('assets.recording.save_to_library') }}
      </UButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import { backend, type AssetSummary } from '@/lib/backend'
import { useClipsStore, type ClipSummary } from '@/stores/clips'
import { useTemplatesStore } from '@/stores/templates'
import { useRecordingStore, type RecordingStopPayload } from '@/stores/recording'
import { useSettingsStore } from '@/stores/settings'
import { useConfirm } from '@/composables/useConfirm'
import { awaitWailsEvent, useWailsEvent } from '@/composables/useWailsEvent'
import BaseModal from '@/components/common/BaseModal.vue'
import EmptyState from '@/components/common/EmptyState.vue'

type AssetTab = 'clips' | 'templates'
type AssetItem = {
  id: string
  kind: AssetTab
  name: string
  description: string
  category: string
  tags: string[]
  meta: string
  icon: string
  source: ClipSummary | AssetSummary
}

const { t } = useI18n()
const toast = useToast()
const { confirm } = useConfirm()
const settings = useSettingsStore()
const clipsStore = useClipsStore()
const templatesStore = useTemplatesStore()
const recording = useRecordingStore()
const activeTab = ref<AssetTab>('clips')
const query = ref('')
const selectedTargetSlot = ref('')
const captureBusy = ref(false)
const editingItem = ref<AssetItem | null>(null)
const editBusy = ref(false)
const pendingRecording = ref<RecordingStopPayload | null>(null)
const recordingSaveBusy = ref(false)
const editDraft = reactive({ name: '', description: '', category: '', tags: '' })
const recordingDraft = reactive({ name: '', description: '', category: '', tags: '' })

const targetItems = computed(() =>
  (settings.data?.automation.win32Targets ?? []).map((target) => ({
    label: `${target.label} · ${target.slot}`,
    value: target.slot,
  })),
)
const items = computed<AssetItem[]>(() => {
  if (activeTab.value === 'clips')
    return clipsStore.clips.map((clip) => ({
      id: clip.id,
      kind: 'clips',
      name: clip.label || clip.id,
      description: clip.description ?? '',
      category: clip.category ?? '',
      tags: clip.tags ?? [],
      meta: t('assets.clips.meta', {
        duration: formatDuration(clip.durationUs),
        count: clip.eventCount,
        mode: clip.meta.mouseMode,
      }),
      icon: 'i-tabler-movie',
      source: clip,
    }))
  return Object.values(templatesStore.map).map((asset) => ({
    id: asset.guid,
    kind: 'templates',
    name: asset.name || asset.guid,
    description: asset.description ?? '',
    category: asset.category ?? '',
    tags: asset.tags ?? [],
    meta: t('assets.templates.meta', { count: asset.variantCount }),
    icon: 'i-tabler-photo',
    source: asset,
  }))
})
const visibleItems = computed(() => {
  const normalized = query.value.trim().toLocaleLowerCase()
  if (!normalized) return items.value
  return items.value.filter((item) =>
    [item.name, item.description, item.category, ...item.tags]
      .join(' ')
      .toLocaleLowerCase()
      .includes(normalized),
  )
})
const recordingBadge = computed(() => {
  switch (recording.state.phase) {
    case 'recording':
      return { label: t('recordingHud.recording'), color: 'error' as const }
    case 'paused':
      return { label: t('recordingHud.paused'), color: 'warning' as const }
    case 'finalizing':
      return { label: t('assets.recording.finalizing'), color: 'warning' as const }
    default:
      return { label: t('assets.recording.ready'), color: 'neutral' as const }
  }
})
const recordingHint = computed(() => {
  if (recording.state.phase === 'recording' || recording.state.phase === 'paused')
    return t('assets.recording.active_hint', { target: recording.state.targetSlot })
  return t('assets.recording.hint')
})

useWailsEvent<Record<string, unknown> | Array<Record<string, unknown>>>(
  'recording:completed',
  (raw) => {
    const payload = Array.isArray(raw) ? raw[0] : raw
    if (typeof payload?.error === 'string') {
      showError(t('recordingSave.save_failed'), payload.error)
      return
    }
    if (typeof payload?.pendingID !== 'string') return
    openRecordingSave({
      pendingID: payload.pendingID,
      targetSlot: String(payload.targetSlot ?? ''),
      durationUs: Number(payload.durationUs ?? 0),
      eventCount: Number(payload.eventCount ?? 0),
    })
  },
)

onMounted(async () => {
  selectedTargetSlot.value = targetItems.value[0]?.value ?? ''
  clipsStore.listen()
  await Promise.all([refreshAssets(), recording.reconcile()])
})

async function refreshAssets(): Promise<void> {
  try {
    await Promise.all([clipsStore.refresh(), templatesStore.reload()])
  } catch (error) {
    showError(t('assets.load_failed'), error)
  }
}

async function startRecording(): Promise<void> {
  try {
    await recording.start(selectedTargetSlot.value)
    await backend.tools.openRecordingHUD()
  } catch (error) {
    showError(t('assets.recording.start_failed'), error)
  }
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

async function stopRecording(): Promise<void> {
  try {
    const payload = await recording.stop()
    if (payload) openRecordingSave(payload)
  } catch (error) {
    showError(t('assets.recording.control_failed'), error)
  }
}

function openRecordingSave(payload: RecordingStopPayload): void {
  pendingRecording.value = payload
  recordingDraft.name = ''
  recordingDraft.description = ''
  recordingDraft.category = ''
  recordingDraft.tags = ''
}

async function saveRecording(): Promise<void> {
  const pending = pendingRecording.value
  if (!pending) return
  recordingSaveBusy.value = true
  try {
    await recording.finalize({
      pendingID: pending.pendingID,
      label: recordingDraft.name.trim(),
      description: recordingDraft.description.trim(),
      category: recordingDraft.category.trim(),
      tags: splitTags(recordingDraft.tags),
    })
    pendingRecording.value = null
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
  } catch (error) {
    showError(t('recordingSave.discard_failed'), error)
  } finally {
    recordingSaveBusy.value = false
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
    if (!result.payload?.cancelled) await templatesStore.reload()
  } catch (error) {
    showError(t('assets.templates.capture_failed'), error)
  } finally {
    captureBusy.value = false
  }
}

function assetMenu(item: AssetItem) {
  return [
    [
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

function openEdit(item: AssetItem): void {
  editingItem.value = item
  editDraft.name = item.name
  editDraft.description = item.description
  editDraft.category = item.category
  editDraft.tags = item.tags.join(', ')
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
      tags: splitTags(editDraft.tags),
    }
    if (item.kind === 'clips') await clipsStore.update(item.id, patch)
    else
      await templatesStore.updateMeta(
        item.id,
        patch.label,
        patch.description,
        patch.category,
        patch.tags,
      )
    editingItem.value = null
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
    if (item.kind === 'clips') await clipsStore.remove(item.id)
    else await templatesStore.remove(item.id)
  } catch (error) {
    showError(t('assets.delete_failed'), error)
  }
}

function splitTags(value: string): string[] {
  return [
    ...new Set(
      value
        .split(/[,，]/)
        .map((tag) => tag.trim())
        .filter(Boolean),
    ),
  ]
}

function formatDuration(durationUs: number): string {
  const seconds = Math.max(0, Math.round(durationUs / 1_000_000))
  return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')}`
}

function showError(title: string, error: unknown): void {
  toast.add({
    title,
    description: error instanceof Error ? error.message : String(error),
    color: 'error',
  })
}
</script>
