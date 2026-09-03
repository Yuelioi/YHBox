<template>
  <div class="settings-page">
    <SettingsSection
      :title="t('settings.general.appearance_title')"
      icon="i-tabler-layout-dashboard"
    >
      <SettingsRow :label="t('settings.language')" :hint="t('settings.language_restart_hint')">
        <template #meta><SettingsRestartBadge /></template>
        <AdaptiveSelect
          :model-value="currentLocale"
          :items="localeItems"
          :aria-label="t('settings.language')"
          @update:model-value="onLocaleChange"
        />
      </SettingsRow>
    </SettingsSection>

    <SettingsSection :title="t('settings.startup.section_title')" icon="i-tabler-power">
      <SettingsRow :label="t('settings.startup.autostart_label')">
        <USwitch
          :model-value="autostart"
          :aria-label="t('settings.startup.autostart_label')"
          @update:model-value="onToggleAutostart"
        />
      </SettingsRow>

      <div class="border-t border-default/60" />

      <SettingsRow :label="t('settings.startup.tray_label')">
        <USwitch
          :model-value="minimizeToTray"
          :aria-label="t('settings.startup.tray_label')"
          @update:model-value="onToggleMinimizeToTray"
        />
      </SettingsRow>
    </SettingsSection>

    <SettingsSection
      :title="t('settings.general.capture_diagnostics_title')"
      icon="i-tabler-activity-heartbeat"
    >
      <SettingsRow :label="t('settings.capture.section_title')" :hint="captureMethodHint">
        <template #meta><SettingsRestartBadge /></template>
        <AdaptiveSelect
          :model-value="currentCapture"
          :items="captureItems"
          :aria-label="t('settings.capture.section_title')"
          @update:model-value="onCaptureChange"
        />
      </SettingsRow>

      <div class="border-t border-default/60" />

      <SettingsRow
        :label="t('settings.capture.dump_debug_label')"
        :hint="t('settings.capture.dump_debug_hint')"
      >
        <USwitch
          :model-value="dumpDebug"
          :aria-label="t('settings.capture.dump_debug_label')"
          @update:model-value="onDumpDebugChange"
        />
      </SettingsRow>

      <div class="border-t border-default/60" />

      <SettingsRow :label="t('settings.log.enabled_label')">
        <USwitch
          :model-value="loggerEnabled"
          :aria-label="t('settings.log.enabled_label')"
          @update:model-value="(value: boolean) => patchLogger('enabled', value)"
        />
      </SettingsRow>

      <div class="border-t border-default/60" />

      <SettingsRow :label="t('settings.log.level_label')" :hint="t('settings.log.level_hint')">
        <AdaptiveSelect
          :model-value="loggerLevel"
          :items="logLevelItems"
          :aria-label="t('settings.log.level_label')"
          @update:model-value="(value: string) => patchLogger('level', value)"
        />
      </SettingsRow>

      <div class="border-t border-default/60" />

      <SettingsRow :label="t('settings.log.live_label')">
        <USwitch
          :model-value="loggerLiveView"
          :aria-label="t('settings.log.live_label')"
          @update:model-value="(value: boolean) => patchLogger('liveView', value)"
        />
      </SettingsRow>
    </SettingsSection>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@/composables/useAppToast'
import { useSettingsStore } from '@/stores/settings'
import { setLocale, type Locale } from '@/i18n'
import SettingsRestartBadge from '@/components/settings/SettingsRestartBadge.vue'
import SettingsRow from '@/components/settings/SettingsRow.vue'
import SettingsSection from '@/components/settings/SettingsSection.vue'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'

const { t } = useI18n()
const settingsStore = useSettingsStore()
const toast = useToast()

const currentLocale = computed(() => (settingsStore.data?.locale ?? 'zh') as Locale)
const localeItems = computed(() => [
  { label: t('settings.language_zh'), value: 'zh' },
  { label: t('settings.language_en'), value: 'en' },
])

async function onLocaleChange(value: string) {
  const ok = await settingsStore.patch({ locale: value })
  if (!ok) return
  setLocale(value as Locale)
  if (value === 'en') {
    toast.add({
      title: t('toast.lang_en_warn_title'),
      description: t('toast.lang_en_warn_desc'),
      icon: 'i-tabler-alert-triangle',
      color: 'warning',
    })
  }
}

const currentCapture = computed(() => settingsStore.data?.capture?.method ?? 'auto')
const captureItems = computed(() => [
  { label: t('settings.capture.method.auto'), value: 'auto' },
  { label: t('settings.capture.method.gdi'), value: 'gdi' },
  { label: t('settings.capture.method.wgc'), value: 'wgc' },
  { label: t('settings.capture.method.mock'), value: 'mock' },
])
const captureMethodHint = computed(() => t(`settings.capture.method_hint.${currentCapture.value}`))

function onCaptureChange(value: string) {
  void settingsStore.patch({ capture: { method: value } })
}

const dumpDebug = computed(() => settingsStore.data?.capture?.dumpDebug ?? false)
function onDumpDebugChange(value: boolean) {
  void settingsStore.patch({ capture: { dumpDebug: value } })
}

const autostart = computed(() => settingsStore.data?.ui.autostart ?? false)
const minimizeToTray = computed(() => settingsStore.data?.ui.minimizeToTray ?? false)
function onToggleAutostart(value: boolean) {
  void settingsStore.patch({ ui: { autostart: value } })
}
function onToggleMinimizeToTray(value: boolean) {
  void settingsStore.patch({ ui: { minimizeToTray: value } })
}

const loggerEnabled = computed(() => settingsStore.data?.ui.logger.enabled ?? true)
const loggerLiveView = computed(() => settingsStore.data?.ui.logger.liveView ?? true)
const loggerLevel = computed(() => settingsStore.data?.ui.logger.level ?? 'info')
const logLevelItems = computed(() => [
  { label: 'DEBUG', value: 'debug' },
  { label: 'INFO', value: 'info' },
  { label: 'WARN', value: 'warn' },
  { label: 'ERROR', value: 'error' },
])
function patchLogger(field: string, value: string | boolean) {
  void settingsStore.patch({ ui: { logger: { [field]: value } } })
}
</script>
