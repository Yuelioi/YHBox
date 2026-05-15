<template>
  <UApp>
    <!-- Standalone 子窗口（独立编辑器等）：跳过整个主壳，直接渲染 router-view -->
    <div v-if="isStandalone" class="h-screen overflow-hidden bg-default">
      <router-view />
    </div>

    <div v-else class="h-screen flex flex-col bg-default overflow-hidden">
      <!-- Custom title bar (frameless window) -->
      <AppTitleBar />

      <!-- Main two-column area.
           main 加 pr-3 (12px right padding) + scrollbar-gutter:stable:
           wails3 frameless 在 Windows 给窗口边缘留 ~8px NC area 做系统 resize
           zone。webview 占满 client area 时 native scrollbar 出现会盖住那 8px
           让鼠标 cursor 变 scrollbar arrow 没机会到 NC area。
             - pr-3 把内容右边距拉到 12px，鼠标可在最右 12px 干净穿过去命中 resize
             - scrollbar-gutter:stable 保证有无滚动条 main 宽度都一样，
               避免内容跳一下 -->
      <div class="flex flex-1 overflow-hidden">
        <AppSidebar />
        <main class="flex-1 overflow-auto pr-3" style="scrollbar-gutter: stable">
          <router-view />
        </main>
      </div>

      <!-- Log panel: slide transition, only on bot routes -->
      <Transition
        enter-active-class="transition-all duration-200 ease-out"
        leave-active-class="transition-all duration-200 ease-in"
        enter-from-class="!h-0 opacity-0"
        enter-to-class="opacity-100"
        leave-from-class="opacity-100"
        leave-to-class="!h-0 opacity-0"
      >
        <div
          v-if="showLog"
          class="shrink-0 overflow-hidden transition-[height] duration-200 ease-out"
          :style="{ height: logStore.lines.length === 0 ? '100px' : '220px' }"
        >
          <LogPanel />
        </div>
      </Transition>

      <!-- Global status bar -->
      <AppStatusBar />

      <!-- 全局浮窗：action 在跑时右下角显示，任何 tab 都可见 + 一键停 -->
      <ActionRunningPill />
    </div>

    <!-- 全局 RecorderDialog：录制/倒计时任何路由都可见（包括 standalone 子窗） -->
    <RecorderDialog v-if="actionsStore.recording || actionsStore.countdown !== null" />
  </UApp>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import AppTitleBar from './components/AppTitleBar.vue'
import AppSidebar from './components/AppSidebar.vue'
import LogPanel from './components/LogPanel.vue'
import AppStatusBar from './components/AppStatusBar.vue'
import ActionRunningPill from './components/actions/ActionRunningPill.vue'
import RecorderDialog from './components/actions/RecorderDialog.vue'
import { useSettingsStore } from './stores/settings'
import { useLogStore } from './stores/log'
import { useActionsStore } from './stores/actions'
import { isBotRoute } from './router'
import { setLocale, type Locale } from './i18n'

const route = useRoute()
const settingsStore = useSettingsStore()
const logStore = useLogStore()
const actionsStore = useActionsStore()

// route.meta.standalone === true → 子窗口模式（不包主壳）
const isStandalone = computed(() => !!route.meta.standalone)

const showLog = computed(() => {
  if (!isBotRoute(route.name)) return false
  return settingsStore.data?.ui.logger.show ?? true
})

watch(
  () => settingsStore.data?.locale,
  (v) => {
    if (v === 'zh' || v === 'en') setLocale(v)
  },
  { immediate: true },
)
</script>
