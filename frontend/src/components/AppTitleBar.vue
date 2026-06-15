<template>
  <!--
    Frameless 窗口的自定义 title bar —— 同时是全局导航 (侧栏已删, 省空间)。
    - 左: 品牌 + 主导航 (容器 / 计划), 图标+字, 当前视图底部下划线高亮
    - 中: drag region (--wails-draggable: drag) + 当前视图标题, 用户拖这里移窗
    - 右: 工具图标 (悬浮启动器 / 设置 / 关于) + minimize / maximize-restore / close
  -->
  <div class="h-14 shrink-0 flex items-stretch bg-default border-b border-default select-none">
    <!-- LEFT: brand + primary nav -->
    <div class="shrink-0 flex items-stretch">
      <div class="flex items-center gap-2 px-4 border-r border-default">
        <div
          class="size-7 rounded-md bg-primary/10 border border-primary/20 flex items-center justify-center shrink-0"
        >
          <UIcon name="i-tabler-device-gamepad-2" class="size-4 text-primary" />
        </div>
        <span class="text-sm font-semibold tracking-tight text-highlighted">Yotta</span>
      </div>

      <nav class="flex items-stretch" style="--wails-draggable: no-drag">
        <RouterLink
          v-for="item in navItems"
          :key="item.key"
          :to="item.to"
          :title="item.label"
          class="relative flex items-center gap-2 px-4 text-sm transition-colors duration-150"
          :class="item.active ? 'text-highlighted' : 'text-muted hover:bg-elevated/60 hover:text-highlighted'"
        >
          <span
            v-if="item.active"
            class="absolute left-2 right-2 bottom-0 h-0.5 bg-primary rounded-t"
          />
          <UIcon :name="item.icon" class="size-4 shrink-0" />
          <span class="truncate">{{ item.label }}</span>
        </RouterLink>
      </nav>
    </div>

    <!-- CENTER: drag region with current view title -->
    <div class="flex-1 flex items-center px-6 min-w-0" style="--wails-draggable: drag">
      <UIcon v-if="currentIcon" :name="currentIcon" class="size-4 text-muted shrink-0 mr-2" />
      <span class="text-sm font-medium text-highlighted truncate">{{ currentTitle }}</span>
    </div>

    <!-- RIGHT: utility icons + window controls -->
    <div class="shrink-0 flex items-stretch" style="--wails-draggable: no-drag">
      <!-- 悬浮启动器 / 设置 / 关于：任何路由 1 步可达 -->
      <button
        type="button"
        class="w-10 flex items-center justify-center text-muted hover:bg-elevated/60 hover:text-highlighted transition-colors duration-150"
        :title="t('sidebar.launcher')"
        @click="openLauncher"
      >
        <UIcon name="i-tabler-rocket" class="size-4" />
      </button>
      <RouterLink
        to="/settings"
        class="w-10 flex items-center justify-center hover:bg-elevated/60 hover:text-highlighted transition-colors duration-150"
        :class="route.name === 'settings' ? 'text-primary' : 'text-muted'"
        :title="t('sidebar.settings')"
      >
        <UIcon name="i-tabler-settings" class="size-4" />
      </RouterLink>
      <RouterLink
        to="/about"
        class="w-10 flex items-center justify-center hover:bg-elevated/60 hover:text-highlighted transition-colors duration-150"
        :class="route.name === 'about' ? 'text-primary' : 'text-muted'"
        :title="t('sidebar.about')"
      >
        <UIcon name="i-tabler-info-circle" class="size-4" />
      </RouterLink>
      <span class="w-px bg-default/60 my-3" />

      <button
        type="button"
        class="w-12 flex items-center justify-center text-muted hover:bg-elevated/60 hover:text-highlighted transition-colors duration-150"
        :title="t('editor.window.minimize')"
        @click="onMinimise"
      >
        <UIcon name="i-tabler-minus" class="size-4" />
      </button>
      <button
        type="button"
        class="w-12 flex items-center justify-center text-muted hover:bg-elevated/60 hover:text-highlighted transition-colors duration-150"
        :title="isMaximised ? t('editor.window.restore') : t('editor.window.maximize')"
        @click="onToggleMaximise"
      >
        <UIcon :name="isMaximised ? 'i-tabler-copy' : 'i-tabler-square'" class="size-3.5" />
      </button>
      <button
        type="button"
        class="w-12 flex items-center justify-center text-muted hover:bg-error hover:text-highlighted transition-colors duration-150"
        :title="t('editor.window.close')"
        @click="onClose"
      >
        <UIcon name="i-tabler-x" class="size-4" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { useWindowControls } from '@/composables/useWindowControls'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { backend } from '@/lib/backend'

const { t } = useI18n()
const route = useRoute()
const editorStore = useContainerEditorStore()
const { isMaximised, onMinimise, onToggleMaximise, closeImmediate: onClose } = useWindowControls()

// '容器' 主导航 — 有 lastEditingContainerID 就跳回编辑器路由 (keep-alive cache 命中, draft 不丢),
// 否则跳列表 (从侧栏迁来的逻辑)。
const containersTo = computed(() =>
  editorStore.lastEditingContainerID
    ? `/containers/${editorStore.lastEditingContainerID}/edit`
    : '/containers',
)

const navItems = computed(() => [
  {
    key: 'containers',
    to: containersTo.value,
    icon: 'i-tabler-package',
    label: t('sidebar.containers'),
    active: route.name === 'containers' || route.name === 'container-edit',
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
  containers: { titleKey: 'sidebar.containers', icon: 'i-tabler-package' },
  'container-edit': { titleKey: 'sidebar.container_edit', icon: 'i-tabler-schema' },
  schedules: { titleKey: 'sidebar.schedules', icon: 'i-tabler-clock' },
  settings: { titleKey: 'sidebar.settings', icon: 'i-tabler-settings' },
  about: { titleKey: 'sidebar.about', icon: 'i-tabler-info-circle' },
}
const currentTitle = computed(() => {
  const meta = VIEW_META[route.name as string]
  return meta ? t(meta.titleKey) : ''
})
const currentIcon = computed(() => VIEW_META[route.name as string]?.icon ?? '')

function openLauncher() {
  void backend.tools.openLauncher()
}
// 窗口控件 (isMaximised + onMinimise / onToggleMaximise / onClose) 全由 useWindowControls 提供
</script>
