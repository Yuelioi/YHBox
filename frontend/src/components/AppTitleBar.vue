<template>
  <!--
    Frameless 窗口的自定义 title bar —— 同时是全局导航 (侧栏已删, 省空间)。
    - 左: 品牌 + 主导航 (工作流 / 计划), 图标+字, 当前视图底部下划线高亮
    - 中: drag region (--wails-draggable: drag) + 当前视图标题, 用户拖这里移窗
    - 右: 工具图标 (设置 / 关于) + minimize / maximize-restore / close
  -->
  <div class="h-14 shrink-0 flex items-stretch bg-default border-b border-default select-none">
    <!-- LEFT: brand + primary nav -->
    <div class="shrink-0 flex items-stretch">
      <div class="flex items-center gap-2 px-4 border-r border-default">
        <div
          class="size-7 rounded-md bg-primary/10 border border-primary/20 flex items-center justify-center shrink-0"
        >
          <UIcon name="i-tabler-device-gamepad-2" class="size-4 text-primary" aria-hidden="true" />
        </div>
        <span class="text-sm font-semibold tracking-tight text-highlighted">Yotta</span>
        <span v-if="versionLabel" class="font-mono text-[11px] tabular-nums text-dimmed">
          {{ versionLabel }}
        </span>
      </div>

      <nav
        class="flex items-stretch"
        :aria-label="t('sidebar.primary_navigation')"
        style="--wails-draggable: no-drag"
      >
        <RouterLink
          v-for="item in navItems"
          :key="item.key"
          :to="item.to"
          :title="item.label"
          :aria-current="item.active ? 'page' : undefined"
          class="relative flex items-center gap-2 px-4 text-sm transition-colors duration-150"
          :class="
            item.active
              ? 'text-highlighted'
              : 'text-muted hover:bg-elevated/60 hover:text-highlighted'
          "
        >
          <span
            v-if="item.active"
            class="absolute left-2 right-2 bottom-0 h-0.5 bg-primary rounded-t"
            aria-hidden="true"
          />
          <UIcon :name="item.icon" class="size-4 shrink-0" aria-hidden="true" />
          <span class="truncate">{{ item.label }}</span>
        </RouterLink>
      </nav>
    </div>

    <!-- CENTER: drag region with current view title -->
    <div class="flex-1 flex items-center px-6 min-w-0" style="--wails-draggable: drag">
      <UIcon
        v-if="currentIcon"
        :name="currentIcon"
        class="size-4 text-muted shrink-0 mr-2"
        aria-hidden="true"
      />
      <span class="text-sm font-medium text-highlighted truncate">{{ currentTitle }}</span>
    </div>

    <!-- RIGHT: utility icons + window controls (border-l 跟左侧品牌 border-r 对称, 把工具区跟导航/标题分开) -->
    <div
      class="shrink-0 flex items-stretch border-l border-default"
      style="--wails-draggable: no-drag"
    >
      <button
        type="button"
        data-testid="open-launcher"
        class="flex w-10 items-center justify-center text-muted transition-colors duration-150 hover:bg-elevated/60 hover:text-highlighted disabled:opacity-50"
        :title="t('sidebar.open_launcher')"
        :aria-label="t('sidebar.open_launcher')"
        :disabled="launcherOpening"
        @click="openLauncher"
      >
        <UIcon
          :name="launcherOpening ? 'i-tabler-loader-2' : 'i-tabler-rocket'"
          class="size-4"
          :class="launcherOpening ? 'animate-spin' : ''"
          aria-hidden="true"
        />
      </button>
      <RouterLink
        to="/settings"
        class="w-10 flex items-center justify-center hover:bg-elevated/60 hover:text-highlighted transition-colors duration-150"
        :class="route.name === 'settings' ? 'text-primary' : 'text-muted'"
        :title="t('sidebar.settings')"
        :aria-label="t('sidebar.settings')"
        :aria-current="route.name === 'settings' ? 'page' : undefined"
      >
        <UIcon name="i-tabler-settings" class="size-4" aria-hidden="true" />
      </RouterLink>
      <RouterLink
        to="/about"
        class="w-10 flex items-center justify-center hover:bg-elevated/60 hover:text-highlighted transition-colors duration-150"
        :class="route.name === 'about' ? 'text-primary' : 'text-muted'"
        :title="t('sidebar.about')"
        :aria-label="t('sidebar.about')"
        :aria-current="route.name === 'about' ? 'page' : undefined"
      >
        <UIcon name="i-tabler-info-circle" class="size-4" aria-hidden="true" />
      </RouterLink>
      <span class="w-px bg-default/60 my-3" />

      <div class="flex items-stretch" role="group" :aria-label="t('editor.window.controls')">
        <button
          type="button"
          class="w-12 flex items-center justify-center text-muted hover:bg-elevated/60 hover:text-highlighted transition-colors duration-150"
          :title="t('editor.window.minimize')"
          :aria-label="t('editor.window.minimize')"
          @click="onMinimise"
        >
          <UIcon name="i-tabler-minus" class="size-4" aria-hidden="true" />
        </button>
        <button
          type="button"
          class="w-12 flex items-center justify-center text-muted hover:bg-elevated/60 hover:text-highlighted transition-colors duration-150"
          :title="isMaximised ? t('editor.window.restore') : t('editor.window.maximize')"
          :aria-label="isMaximised ? t('editor.window.restore') : t('editor.window.maximize')"
          @click="onToggleMaximise"
        >
          <UIcon
            :name="isMaximised ? 'i-tabler-copy' : 'i-tabler-square'"
            class="size-3.5"
            aria-hidden="true"
          />
        </button>
        <button
          type="button"
          class="w-12 flex items-center justify-center text-muted hover:bg-error hover:text-highlighted transition-colors duration-150"
          :title="t('editor.window.close')"
          :aria-label="t('editor.window.close')"
          @click="onClose"
        >
          <UIcon name="i-tabler-x" class="size-4" aria-hidden="true" />
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useWindowControls } from '@/composables/useWindowControls'
import { backend } from '@/lib/backend'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const { isMaximised, onMinimise, onToggleMaximise, closeImmediate: onClose } = useWindowControls()
const appVersion = ref('')
const launcherOpening = ref(false)
let stopMainNavigate: (() => void) | undefined
const versionLabel = computed(() => (appVersion.value ? `v${appVersion.value}` : ''))

const navItems = computed(() => [
  {
    key: 'workflows',
    to: '/workflows',
    icon: 'i-tabler-route',
    label: t('sidebar.workflows'),
    active: route.name === 'workflows' || route.name === 'workflow-edit',
  },
  {
    key: 'assets',
    to: '/assets',
    icon: 'i-tabler-library',
    label: t('sidebar.assets'),
    active: route.name === 'assets',
  },
  {
    key: 'schedules',
    to: '/schedules',
    icon: 'i-tabler-clock',
    label: t('sidebar.schedules'),
    active: route.name === 'schedules',
  },
])

// route.name → i18n key. 标题文字走 t() (locale 切换刷新), icon 配静态 map.
const VIEW_META: Record<string, { titleKey: string; icon: string }> = {
  workflows: { titleKey: 'sidebar.workflows', icon: 'i-tabler-route' },
  'workflow-edit': { titleKey: 'sidebar.workflow_edit', icon: 'i-tabler-schema' },
  assets: { titleKey: 'sidebar.assets', icon: 'i-tabler-library' },
  schedules: { titleKey: 'sidebar.schedules', icon: 'i-tabler-clock' },
  settings: { titleKey: 'sidebar.settings', icon: 'i-tabler-settings' },
  about: { titleKey: 'sidebar.about', icon: 'i-tabler-info-circle' },
}
// 左侧主导航已高亮工作流/计划，中间不重复显示同名标题。
const SUPPRESS_CENTER = new Set(['workflows', 'assets', 'schedules'])
const currentTitle = computed(() => {
  if (SUPPRESS_CENTER.has(route.name as string)) return ''
  const meta = VIEW_META[route.name as string]
  return meta ? t(meta.titleKey) : ''
})
const currentIcon = computed(() => {
  if (SUPPRESS_CENTER.has(route.name as string)) return ''
  return VIEW_META[route.name as string]?.icon ?? ''
})

onMounted(async () => {
  const info = await backend.appInfo.info()
  appVersion.value = String(info?.version ?? '')
  stopMainNavigate = backend.events.onMainNavigate((target) => {
    void router.push({
      path: target.path,
      query: target.section ? { section: target.section } : {},
    })
  })
})
onUnmounted(() => stopMainNavigate?.())

async function openLauncher(): Promise<void> {
  if (launcherOpening.value) return
  launcherOpening.value = true
  await backend.tools.openLauncher()
  launcherOpening.value = false
}
// 窗口控件 (isMaximised + onMinimise / onToggleMaximise / onClose) 全由 useWindowControls 提供
</script>
