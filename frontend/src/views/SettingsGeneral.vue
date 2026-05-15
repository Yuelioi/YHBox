<template>
  <div class="px-8 py-6 space-y-6">
    <!-- Game window section -->
    <section class="rounded-xl bg-default border border-default p-5 space-y-4">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-device-desktop" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">游戏窗口</h2>
      </div>

      <div class="flex items-center justify-between gap-4 pt-1">
        <div class="flex items-center gap-2 min-w-0">
          <span
            class="size-2 rounded-full shrink-0 transition-colors duration-300"
            :class="gameDotClass"
          />
          <span class="text-sm text-toned truncate">{{ gameLabel }}</span>
        </div>
        <UButton
          color="neutral"
          variant="soft"
          icon="i-tabler-refresh"
          size="sm"
          :loading="detecting"
          :disabled="detecting"
          @click="onDetect"
        >
          重新检测
        </UButton>
      </div>
    </section>

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

    <!-- Log section -->
    <section class="rounded-xl bg-default border border-default p-5 space-y-4">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-terminal" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">日志</h2>
      </div>

      <div class="flex items-center justify-between gap-6">
        <div>
          <div class="text-sm text-default">显示日志面板</div>
          <p class="text-xs text-dimmed mt-0.5">控制 bot 页面底部的日志区域是否显示。</p>
        </div>
        <USwitch :model-value="show" @update:model-value="onToggleShow" />
      </div>

      <div class="border-t border-default/60" />

      <div class="flex items-center justify-between gap-6">
        <div>
          <div class="text-sm text-default">写入本地日志文件</div>
          <p class="text-xs text-dimmed mt-0.5">
            把日志同时写到
            <code class="text-toned bg-elevated/60 px-1 py-0.5 rounded">logs/</code> 目录下的 JSON
            文件。下次启动 bot 生效。
          </p>
        </div>
        <USwitch :model-value="writeFile" @update:model-value="onToggleWriteFile" />
      </div>

      <p class="text-xs text-dimmed pt-1 flex items-start gap-1.5">
        <UIcon name="i-tabler-info-circle" class="size-3.5 shrink-0 mt-0.5" />
        <span>时间戳显示 / 标签显示 / 折行 / 自动滚动 在日志面板顶部即可切换。</span>
      </p>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import { useSettingsStore } from '@/stores/settings'
import { useGameStore } from '@/stores/game'
import { setLocale, type Locale } from '@/i18n'

const { t } = useI18n()
const settingsStore = useSettingsStore()
const gameStore = useGameStore()
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

const show = computed(() => settingsStore.data?.ui.logger.show ?? true)
const writeFile = computed(() => settingsStore.data?.ui.logger.writeFile ?? true)

async function onToggleShow(v: boolean) {
  await settingsStore.patch({ ui: { logger: { show: v } } })
}

async function onToggleWriteFile(v: boolean) {
  await settingsStore.patch({ ui: { logger: { writeFile: v } } })
  toast.add({
    title: v ? '已开启日志文件写入' : '已关闭日志文件写入',
    color: 'neutral',
    icon: 'i-tabler-check',
    description: v ? '下次启动 bot 会写入 logs/ 目录。' : undefined,
  })
}

const detecting = ref(false)
const gameLabel = computed(() => {
  const s = gameStore.status
  if (!s) return '正在检测...'
  if (!s.ok) return '未检测到异环窗口（请确认游戏已运行）'
  return `${s.title} · ${s.w}×${s.h}`
})
const gameDotClass = computed(() => {
  const s = gameStore.status
  if (!s) return 'bg-accented'
  if (!s.ok) return 'bg-error'
  return 'bg-primary'
})

async function onDetect() {
  if (detecting.value) return
  detecting.value = true
  try {
    await gameStore.detect()
  } finally {
    setTimeout(() => {
      detecting.value = false
    }, 400)
  }
}
</script>
