<template>
  <div class="settings-page">
    <SettingsSection
      :title="t('settings.general.appearance_title')"
      :description="t('settings.general.appearance_hint')"
      icon="i-tabler-layout-dashboard"
    >
      <SettingsRow
        :label="t('settings.editor_display.detail_label')"
        :hint="t('settings.editor_display.detail_hint')"
      >
        <USelect
          :model-value="sidebarPrefs.experienceMode"
          :items="editorDetailItems"
          class="w-40"
          :aria-label="t('settings.editor_display.detail_label')"
          @update:model-value="onEditorDetailChange"
        />
      </SettingsRow>

      <div class="border-t border-default/60" />

      <SettingsRow :label="t('settings.language')" :hint="t('settings.language_restart_hint')">
        <template #meta><SettingsRestartBadge /></template>
        <USelect
          :model-value="currentLocale"
          :items="localeItems"
          class="w-40"
          :aria-label="t('settings.language')"
          @update:model-value="onLocaleChange"
        />
      </SettingsRow>
    </SettingsSection>

    <SettingsSection
      :title="t('settings.startup.section_title')"
      :description="t('settings.general.behavior_hint')"
      icon="i-tabler-power"
    >
      <SettingsRow
        :label="t('settings.startup.autostart_label')"
        :hint="t('settings.startup.autostart_hint')"
      >
        <USwitch
          :model-value="autostart"
          :aria-label="t('settings.startup.autostart_label')"
          @update:model-value="onToggleAutostart"
        />
      </SettingsRow>

      <div class="border-t border-default/60" />

      <SettingsRow
        :label="t('settings.startup.tray_label')"
        :hint="t('settings.startup.tray_hint')"
      >
        <USwitch
          :model-value="minimizeToTray"
          :aria-label="t('settings.startup.tray_label')"
          @update:model-value="onToggleMinimizeToTray"
        />
      </SettingsRow>
    </SettingsSection>

    <SettingsSection
      :title="t('settings.general.capture_diagnostics_title')"
      :description="t('settings.general.capture_diagnostics_hint')"
      icon="i-tabler-activity-heartbeat"
    >
      <SettingsRow :label="t('settings.capture.section_title')" :hint="captureMethodHint">
        <template #meta><SettingsRestartBadge /></template>
        <USelect
          :model-value="currentCapture"
          :items="captureItems"
          class="w-40"
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

      <SettingsRow :label="t('settings.log.enabled_label')" :hint="t('settings.log.enabled_hint')">
        <USwitch
          :model-value="loggerEnabled"
          :aria-label="t('settings.log.enabled_label')"
          @update:model-value="(value: boolean) => patchLogger('enabled', value)"
        />
      </SettingsRow>

      <div class="border-t border-default/60" />

      <SettingsRow :label="t('settings.log.level_label')" :hint="t('settings.log.level_hint')">
        <USelect
          :model-value="loggerLevel"
          :items="logLevelItems"
          class="w-40"
          :aria-label="t('settings.log.level_label')"
          @update:model-value="(value: string) => patchLogger('level', value)"
        />
      </SettingsRow>

      <div class="border-t border-default/60" />

      <SettingsRow :label="t('settings.log.live_label')" :hint="t('settings.log.live_hint')">
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
import { useToast } from '@nuxt/ui/composables'
import { useSettingsStore } from '@/stores/settings'
import { setLocale, type Locale } from '@/i18n'
import { useSidebarPrefs } from '@/composables/editor/useSidebarPrefs'
import SettingsRestartBadge from '@/components/settings/SettingsRestartBadge.vue'
import SettingsRow from '@/components/settings/SettingsRow.vue'
import SettingsSection from '@/components/settings/SettingsSection.vue'

const { t } = useI18n()
const settingsStore = useSettingsStore()
const toast = useToast()
const { prefs: sidebarPrefs } = useSidebarPrefs()

const editorDetailItems = computed(() => [
  { label: t('editor.experience.basic'), value: 'basic' },
  { label: t('editor.experience.pro'), value: 'pro' },
])

function onEditorDetailChange(value: string) {
  if (value === 'basic' || value === 'pro') sidebarPrefs.value.experienceMode = value
}

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
