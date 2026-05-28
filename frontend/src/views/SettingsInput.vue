<template>
  <div class="p-6 space-y-6 max-w-2xl">
    <header class="space-y-2">
      <h2 class="text-base font-medium text-highlighted">{{ t('settings.input.title') }}</h2>
      <p class="text-xs text-dimmed">{{ t('settings.input.intro') }}</p>
      <div
        class="rounded-md bg-elevated/30 border border-default/50 px-3 py-2 text-[11px] text-muted leading-relaxed"
      >
        <UIcon
          name="i-tabler-info-circle"
          class="size-3.5 inline-block align-middle mr-1 text-amber-300/80"
        />
        <span class="text-toned">{{ t('settings.input.intro_box.what_label') }}</span
        >: {{ t('settings.input.intro_box.what_desc') }}
        <ul class="list-disc pl-5 mt-1 space-y-0.5">
          <li>{{ t('settings.input.intro_box.item_default_source') }}</li>
          <li>{{ t('settings.input.intro_box.item_sync_action') }}</li>
        </ul>
        <span class="mt-1 block">
          {{ t('settings.input.intro_box.footnote_prefix') }}<span class="text-warning">{{
            t('settings.input.intro_box.footnote_negation')
          }}</span
          >{{ t('settings.input.intro_box.footnote_rest') }}
        </span>
      </div>
    </header>

    <!-- 录制配置 -->
    <section class="rounded-md border border-default bg-elevated/40 p-4 space-y-3">
      <header>
        <h3 class="text-sm font-medium text-highlighted">{{ t('settings.input.record.title') }}</h3>
        <p class="text-[11px] text-dimmed mt-0.5">{{ t('settings.input.record.hint') }}</p>
      </header>

      <!-- 停录热键 -->
      <div class="flex items-center justify-between gap-4">
        <div class="min-w-0">
          <div class="text-sm text-default">{{ t('settings.input.record.stop_hotkey_label') }}</div>
          <div class="text-[11px] text-dimmed">
            {{ t('settings.input.record.stop_hotkey_hint') }}
          </div>
        </div>
        <div class="w-56 shrink-0">
          <HotkeyCaptureInput
            :model-value="settings?.ui.recordingStopHotkey ?? 'F12'"
            @update:model-value="(v: string) => patchRecord({ recordingStopHotkey: v })"
          />
        </div>
      </div>

      <!-- 鼠标语义 -->
      <div class="flex items-center justify-between gap-4">
        <div class="min-w-0">
          <div class="text-sm text-default">{{ t('settings.input.record.mouse_mode_label') }}</div>
          <div class="text-[11px] text-dimmed leading-snug">
            {{ t('settings.input.record.mouse_mode_hint') }}
          </div>
        </div>
        <USelect
          :model-value="settings?.ui.recordingMouseMode ?? 'relative'"
          :items="mouseModeItems"
          class="w-56"
          @update:model-value="(v: string) => patchRecord({ recordingMouseMode: v })"
        />
      </div>
    </section>

    <section class="rounded-md border border-default bg-elevated/40 p-4 space-y-4">
      <div class="flex items-center justify-between gap-4">
        <div class="min-w-0">
          <h3 class="text-sm font-medium text-highlighted">{{ t('settings.input.counts.title') }}</h3>
          <p class="text-[11px] text-dimmed mt-0.5">{{ t('settings.input.counts.hint') }}</p>
        </div>
        <div class="flex items-center gap-2 shrink-0">
          <UInputNumber
            v-model="manualCounts"
            :min="0"
            :max="999999"
            :step="100"
            class="w-32"
            @blur="onCommitManual"
          />
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            icon="i-tabler-check"
            :title="t('settings.input.counts.save_manual')"
            @click="onCommitManual"
          />
        </div>
      </div>

      <div class="flex items-center gap-2 flex-wrap">
        <UButton size="sm" color="primary" icon="i-tabler-target" @click="calibratorOpen = true">
          {{
            (settings?.ui.mouseCounts360 ?? 0) > 0
              ? t('settings.input.counts.recalibrate')
              : t('settings.input.counts.calibrate')
          }}
        </UButton>
        <UButton
          size="sm"
          variant="soft"
          color="neutral"
          icon="i-tabler-pointer"
          @click="openMouseHUD"
        >
          {{ t('settings.input.counts.open_hud') }}
        </UButton>
        <UButton
          v-if="(settings?.ui.mouseCounts360 ?? 0) > 0"
          size="sm"
          variant="soft"
          color="primary"
          icon="i-tabler-refresh"
          @click="onSyncAll"
        >
          {{ t('settings.input.counts.sync_all') }}
        </UButton>
        <span class="ml-auto text-[11px] text-dimmed">
          {{ t('settings.input.counts.share_hint') }}
        </span>
      </div>
    </section>

    <!-- 说明 -->
    <section
      class="rounded-md border border-default/60 bg-default/50 p-4 text-xs text-dimmed space-y-2"
    >
      <h4 class="text-xs uppercase tracking-wider text-toned">
        {{ t('settings.input.howto.title') }}
      </h4>
      <ol class="list-decimal pl-5 space-y-1">
        <li>{{ t('settings.input.howto.step_open') }}</li>
        <li>{{ t('settings.input.howto.step_focus') }}</li>
        <li>{{ t('settings.input.howto.step_start') }}</li>
        <li>{{ t('settings.input.howto.step_spin') }}</li>
        <li>{{ t('settings.input.howto.step_stop') }}</li>
        <li>{{ t('settings.input.howto.step_save') }}</li>
      </ol>
    </section>

    <!-- 校准 Modal -->
    <CalibratorModal v-model:open="calibratorOpen" @save="onCalibratorSaved" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend } from '@/lib/backend'
import { useSettingsStore } from '@/stores/settings'
import CalibratorModal from '@/components/calibration/CalibratorModal.vue'
import HotkeyCaptureInput from '@/components/hotkeys/HotkeyCaptureInput.vue'
import { useToast } from '@nuxt/ui/composables'
import { useConfirm } from '@/composables/useConfirm'

const { t } = useI18n()
const { confirm } = useConfirm()

const settingsStore = useSettingsStore()
const settings = computed(() => settingsStore.data)
const toast = useToast()

const mouseModeItems = computed(() => [
  { label: t('settings.input.record.mouse_mode.relative'), value: 'relative' },
  { label: t('settings.input.record.mouse_mode.absolute'), value: 'absolute' },
])

const manualCounts = ref<number>(0)
watch(
  () => settings.value?.ui.mouseCounts360,
  (v) => {
    manualCounts.value = v ?? 0
  },
  { immediate: true },
)

async function onCommitManual() {
  const v = Number(manualCounts.value)
  if (!Number.isFinite(v) || v < 0) return
  const cur = settings.value?.ui.mouseCounts360 ?? 0
  if (v === cur) return
  await settingsStore.patch({ ui: { mouseCounts360: Math.floor(v) } })
}

async function patchRecord(patch: Record<string, any>) {
  if (!settings.value) return
  await settingsStore.patch({ ui: patch })
}

async function openMouseHUD() {
  await backend.tools.openMouseHUD()
}

async function onSyncAll() {
  const cur = settings.value?.ui.mouseCounts360 ?? 0
  if (cur <= 0) {
    toast.add({ title: t('settings.input.toast.counts_not_set'), color: 'warning' })
    return
  }
  const yes = await confirm({
    title: t('settings.input.confirm.sync_title'),
    description: t('settings.input.confirm.sync_desc', { cur }),
    confirmText: t('settings.input.confirm.sync_confirm'),
    color: 'primary',
  })
  if (yes !== true) return
  const r = (await backend.containers.syncLocalMouseCalibration(cur)) as any
  if (r) {
    toast.add({
      title: t('settings.input.toast.synced_title', { n: r.updated?.length ?? 0 }),
      description: r.skipped?.length
        ? t('settings.input.toast.synced_skipped', { n: r.skipped.length })
        : undefined,
      color: 'success',
    })
  }
}

const calibratorOpen = ref(false)

async function onCalibratorSaved(counts: number) {
  await settingsStore.patch({ ui: { mouseCounts360: counts } })
  const yes = await confirm({
    title: t('settings.input.confirm.sync_title'),
    description: t('settings.input.confirm.calibrator_done_desc', { counts }),
    confirmText: t('settings.input.confirm.sync_confirm'),
    cancelText: t('settings.input.confirm.sync_cancel'),
    color: 'primary',
  })
  if (yes === true) {
    const r = await backend.containers.syncLocalMouseCalibration(counts)
    if (r) {
      toast.add({
        title: t('settings.input.toast.synced_title', {
          n: (r as any).updated?.length ?? 0,
        }),
        color: 'success',
      })
    }
  }
}
</script>
