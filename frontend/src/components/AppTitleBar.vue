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
          v-for="item in navigation.primary"
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
    <div
      class="flex-1 flex items-center px-6 min-w-0"
      data-testid="app-context-title"
      style="--wails-draggable: drag"
    >
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
          class="w-12 flex items-center justify-center text-muted hover:bg-error hover:text-highlighted transition-colors duration-150 disabled:opacity-50"
          :title="t('editor.window.close')"
          :aria-label="t('editor.window.close')"
          :disabled="closeRequestPending"
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
import { buildAppNavigation } from '@/app/navigation/appNavigation'
import { requestMainWindowClose } from '@/app/window/requestMainWindowClose'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const { isMaximised, onMinimise, onToggleMaximise, closeImmediate } = useWindowControls()
const appVersion = ref('')
const launcherOpening = ref(false)
const closeRequestPending = ref(false)
let stopMainNavigate: (() => void) | undefined
const versionLabel = computed(() => (appVersion.value ? `v${appVersion.value}` : ''))

const navigation = computed(() => buildAppNavigation(String(route.name ?? ''), t))
const currentTitle = computed(() => navigation.value.contextTitle)
const currentIcon = computed(() => navigation.value.contextIcon)

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

async function onClose(): Promise<void> {
  if (closeRequestPending.value) return
  closeRequestPending.value = true
  try {
    await requestMainWindowClose(route.name, router, closeImmediate)
  } finally {
    closeRequestPending.value = false
  }
}
</script>
