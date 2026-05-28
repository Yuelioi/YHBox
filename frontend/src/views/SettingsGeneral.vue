<template>
  <div class="px-8 py-6 space-y-6">
    <!-- Startup & Close section -->
    <section class="rounded-xl bg-default border border-default p-5 space-y-4">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-power" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">
          {{ t('settings.startup.section_title') }}
        </h2>
      </div>

      <div class="flex items-center justify-between gap-6">
        <div>
          <div class="text-sm text-default">{{ t('settings.startup.autostart_label') }}</div>
          <p class="text-xs text-dimmed mt-0.5">{{ t('settings.startup.autostart_hint') }}</p>
        </div>
        <USwitch :model-value="autostart" @update:model-value="onToggleAutostart" />
      </div>

      <div class="border-t border-default/60" />

      <div class="flex items-center justify-between gap-6">
        <div>
          <div class="text-sm text-default">{{ t('settings.startup.tray_label') }}</div>
          <p class="text-xs text-dimmed mt-0.5">{{ t('settings.startup.tray_hint') }}</p>
        </div>
        <USwitch :model-value="minimizeToTray" @update:model-value="onToggleMinimizeToTray" />
      </div>
    </section>

    <!-- Language section -->
    <section class="rounded-xl bg-default border border-default p-5 space-y-4">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-language" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">{{ t('settings.language') }}</h2>
      </div>

      <div class="flex items-center justify-between gap-6">
        <p class="text-xs text-dimmed">{{ t('settings.language_restart_hint') }}</p>
        <USelect
          :model-value="currentLocale"
          :items="localeItems"
          class="w-32"
          @update:model-value="onLocaleChange"
        />
      </div>
    </section>

    <!-- Capture backend section -->
    <section class="rounded-xl bg-default border border-default p-5 space-y-4">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-camera" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">
          {{ t('settings.capture.section_title') }}
        </h2>
      </div>

      <div class="flex items-start justify-between gap-6">
        <div class="text-xs text-dimmed space-y-1 max-w-md">
          <p>{{ t('settings.capture.hint_auto') }}</p>
          <p>{{ t('settings.capture.hint_gdi') }}</p>
          <p>{{ t('settings.capture.hint_wgc') }}</p>
          <p>{{ t('settings.capture.hint_mock') }}</p>
          <p class="text-dimmed">{{ t('settings.capture.restart_hint') }}</p>
        </div>
        <USelect
          :model-value="currentCapture"
          :items="captureItems"
          class="w-32"
          @update:model-value="onCaptureChange"
        />
      </div>

      <div class="border-t border-default/60" />

      <div class="flex items-center justify-between gap-6">
        <div>
          <div class="text-sm text-default">{{ t('settings.capture.dump_debug_label') }}</div>
          <p class="text-xs text-dimmed mt-0.5">{{ t('settings.capture.dump_debug_hint') }}</p>
        </div>
        <USwitch :model-value="dumpDebug" @update:model-value="onDumpDebugChange" />
      </div>
    </section>

    <!-- Log: 所有日志设置统一在底部日志面板 header 的设置图标里 -->
    <section class="rounded-xl bg-default border border-default p-5">
      <div class="flex items-start gap-3">
        <UIcon name="i-tabler-terminal" class="size-4 text-dimmed mt-0.5 shrink-0" />
        <div class="space-y-1">
          <h2 class="text-sm font-medium text-highlighted">{{ t('settings.log.section_title') }}</h2>
          <p class="text-xs text-dimmed">{{ t('settings.log.hint') }}</p>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import { useSettingsStore } from '@/stores/settings'
import { setLocale, type Locale } from '@/i18n'

const { t } = useI18n()
const settingsStore = useSettingsStore()
const toast = useToast()

const currentLocale = computed(() => (settingsStore.data?.locale ?? 'zh') as Locale)

const localeItems = computed(() => [
  { label: t('settings.language_zh'), value: 'zh' },
  { label: t('settings.language_en'), value: 'en' },
])

async function onLocaleChange(v: string) {
  await settingsStore.patch({ locale: v })
  setLocale(v as Locale)
  if (v === 'en') {
    toast.add({
      title: t('toast.lang_en_warn_title'),
      description: t('toast.lang_en_warn_desc'),
      icon: 'i-tabler-alert-triangle',
      color: 'warning',
    })
  } else {
    toast.add({
      title: t('settings.language_changed_title'),
      description: t('settings.language_changed_desc'),
      icon: 'i-tabler-info-circle',
      color: 'neutral',
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
async function onCaptureChange(v: string) {
  await settingsStore.patch({ capture: { method: v } })
  toast.add({
    title: t('settings.capture.method_changed_title', { method: v.toUpperCase() }),
    description: t('settings.capture.method_changed_desc'),
    icon: 'i-tabler-info-circle',
    color: 'neutral',
  })
}

const dumpDebug = computed(() => settingsStore.data?.capture?.dumpDebug ?? false)
async function onDumpDebugChange(v: boolean) {
  await settingsStore.patch({ capture: { dumpDebug: v } })
}

const autostart = computed(() => settingsStore.data?.ui.autostart ?? false)
const minimizeToTray = computed(() => settingsStore.data?.ui.minimizeToTray ?? false)

async function onToggleAutostart(v: boolean) {
  await settingsStore.patch({ ui: { autostart: v } })
  toast.add({
    title: v ? t('toast.autostart_on') : t('toast.autostart_off'),
    icon: 'i-tabler-check',
    color: 'neutral',
  })
}

async function onToggleMinimizeToTray(v: boolean) {
  await settingsStore.patch({ ui: { minimizeToTray: v } })
}
</script>
