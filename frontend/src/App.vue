<template>
  <UApp :toaster="{ position: 'top-center', duration: 2500 }">
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

      <!-- Log panel: 全局始终渲染, 折叠/展开由 LogPanel 内部读 settings.UI.Logger.PanelOpen 自管 -->
      <LogPanel />

      <!-- Global status bar -->
      <AppStatusBar />

      </div>

    <!-- 全局 CalibratorModal：供 NodeInspector MouseCalibration 节点触发 -->
    <CalibratorModal v-model:open="globalCalibOpen" @save="onGlobalCalibSave" />

    <!-- 全局 ConfirmDialog：useConfirm() Promise API 触发 -->
    <ConfirmDialog
      v-if="confirmState.opts"
      :open="confirmState.open"
      v-bind="confirmState.opts"
      @update:open="onConfirmDialogUpdateOpen"
      @resolve="resolveActive"
    />
  </UApp>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import AppTitleBar from './components/AppTitleBar.vue'
import AppSidebar from './components/AppSidebar.vue'
import LogPanel from './components/LogPanel.vue'
import AppStatusBar from './components/AppStatusBar.vue'
import CalibratorModal from './components/calibration/CalibratorModal.vue'
import ConfirmDialog from './components/common/ConfirmDialog.vue'
import { useConfirm } from './composables/useConfirm'
import { useSettingsStore } from './stores/settings'
import { setLocale, type Locale } from './i18n'

const route = useRoute()
const settingsStore = useSettingsStore()

const globalCalibOpen = ref(false)
let pendingSave: ((counts: number) => void) | null = null

// 全局 ConfirmDialog 单例 state
const { state: confirmState, resolveActive } = useConfirm()
function onConfirmDialogUpdateOpen(v: boolean) {
  if (!v && confirmState.opts) {
    // 关闭 = 取消（boolean: false / input: ''）
    resolveActive(confirmState.opts.inputDefault !== undefined ? '' : false)
  }
}

function handleOpenEvent(e: Event) {
  const detail = (e as CustomEvent).detail
  pendingSave = detail?.onSave ?? null
  globalCalibOpen.value = true
}

function onGlobalCalibSave(counts: number) {
  if (pendingSave) {
    pendingSave(counts)
    pendingSave = null
  }
}

onMounted(() => {
  window.addEventListener('open-calibrator-modal', handleOpenEvent)
})
onUnmounted(() => {
  window.removeEventListener('open-calibrator-modal', handleOpenEvent)
})

// 子窗口模式（不包主壳）:
//   meta.standalone — MouseHUD / ScreenPicker / RecordingHUD 老路径
//   query.standalone=1 — container-edit 子窗口形态 (Task 5+)
const isStandalone = computed(() => {
  if (route.meta.standalone) return true
  if (route.query.standalone === '1') return true
  return false
})

watch(
  () => settingsStore.data?.locale,
  (v) => {
    if (v === 'zh' || v === 'en') setLocale(v)
  },
  { immediate: true },
)
</script>
