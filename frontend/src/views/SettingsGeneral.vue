<template>
  <div class="px-8 py-6 space-y-6">
    <!-- Startup & Close section -->
    <section class="rounded-xl bg-default border border-default p-5 space-y-4">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-power" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">启动与关闭</h2>
      </div>

      <div class="flex items-center justify-between gap-6">
        <div>
          <div class="text-sm text-default">开机自启</div>
          <p class="text-xs text-dimmed mt-0.5">
            登录 Windows 后自动启动 YHBox。写入注册表
            <code class="text-toned bg-elevated/60 px-1 py-0.5 rounded">HKCU\...\Run\YHBox</code
            >，不需要管理员权限。
          </p>
        </div>
        <USwitch :model-value="autostart" @update:model-value="onToggleAutostart" />
      </div>

      <div class="border-t border-default/60" />

      <div class="flex items-center justify-between gap-6">
        <div>
          <div class="text-sm text-default">关闭最小化到托盘</div>
          <p class="text-xs text-dimmed mt-0.5">
            点关闭按钮（×）时不退出，而是收到右下角系统托盘。右键托盘图标可强制退出。
          </p>
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
        <h2 class="text-sm font-medium text-highlighted">截屏方式</h2>
      </div>

      <div class="flex items-start justify-between gap-6">
        <div class="text-xs text-dimmed space-y-1 max-w-md">
          <p>Auto: Win11/Server 2022 走 WGC（关黄框 + 后台稳），其它走 GDI。新装默认。</p>
          <p>GDI（PrintWindow）兼容性最好。</p>
          <p>WGC（Windows Graphics Capture）后台抓帧稳，但 Win10 上有黄框关不掉。</p>
          <p>Mock 从 bin/mock-frames/ 读 PNG 序列回放，调试用，无需开游戏。</p>
          <p class="text-dimmed">改完需重启 exe 生效。</p>
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
          <div class="text-sm text-default">落盘 detect 标注图</div>
          <p class="text-xs text-dimmed mt-0.5">
            bot 识别关键路径异步把带框的 PNG 写到
            <code class="text-toned bg-elevated/60 px-1 py-0.5 rounded">debug/captures/</code>，
            调参/排查识别问题用。立即生效。
          </p>
        </div>
        <USwitch :model-value="dumpDebug" @update:model-value="onDumpDebugChange" />
      </div>
    </section>

    <!-- Log: 所有日志设置统一在底部日志面板 header 的设置图标里 -->
    <section class="rounded-xl bg-default border border-default p-5">
      <div class="flex items-start gap-3">
        <UIcon name="i-tabler-terminal" class="size-4 text-dimmed mt-0.5 shrink-0" />
        <div class="space-y-1">
          <h2 class="text-sm font-medium text-highlighted">日志</h2>
          <p class="text-xs text-dimmed">
            折叠/展开、写入文件、时间戳、折行、自动滚动等设置在
            <span class="text-toned">底部日志面板 header</span> 的设置图标里调整。
          </p>
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
      title: 'Language switched to English',
      description:
        '尚未支持英文模版：fish / cook / battle 的视觉模板未采集 EN 版本，这几个功能会显示为不可用。UI 字符串已切换。',
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
const captureItems = [
  { label: 'Auto (按 OS 选)', value: 'auto' },
  { label: 'GDI', value: 'gdi' },
  { label: 'WGC', value: 'wgc' },
  { label: 'Mock (离线回放)', value: 'mock' },
]
async function onCaptureChange(v: string) {
  await settingsStore.patch({ capture: { method: v } })
  toast.add({
    title: `截屏方式已切到 ${v.toUpperCase()}`,
    description: '重启程序生效',
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
    title: v ? '已开启开机自启' : '已关闭开机自启',
    icon: 'i-tabler-check',
    color: 'neutral',
  })
}

async function onToggleMinimizeToTray(v: boolean) {
  await settingsStore.patch({ ui: { minimizeToTray: v } })
}
</script>
