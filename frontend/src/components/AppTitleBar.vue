<template>
  <!--
    Frameless 窗口的自定义 title bar.
    - 整行 h-14（跟以前 sidebar brand 行同高）
    - 中间 drag region 用 CSS --wails-draggable: drag 让用户拖拽窗口；按钮区域是 no-drag
    - 右侧 minimize / maximize-or-restore / close 三个按钮
  -->
  <div class="h-14 shrink-0 flex items-stretch bg-default border-b border-default select-none">
    <!-- LEFT: brand -->
    <div
      class="shrink-0 flex items-center gap-2 px-4 border-r border-default"
      :class="collapsed ? 'w-14 justify-center px-0' : 'w-40'"
    >
      <div
        class="size-7 rounded-md bg-primary/10 border border-primary/20 flex items-center justify-center shrink-0"
      >
        <UIcon name="i-tabler-device-gamepad-2" class="size-4 text-primary" />
      </div>
      <template v-if="!collapsed">
        <span class="text-sm font-semibold tracking-tight text-highlighted">YHBox</span>
      </template>
    </div>

    <!-- CENTER: drag region with current view title -->
    <div class="flex-1 flex items-center px-6 min-w-0" style="--wails-draggable: drag">
      <UIcon v-if="currentIcon" :name="currentIcon" class="size-4 text-muted shrink-0 mr-2" />
      <span class="text-sm font-medium text-highlighted truncate">{{ currentTitle }}</span>
    </div>

    <!-- RIGHT: quick-access icons + window controls -->
    <div class="shrink-0 flex items-stretch" style="--wails-draggable: no-drag">
      <!-- 快捷 icon：设置 / 帮助。任何路由 1 步可达 -->
      <RouterLink
        to="/settings"
        class="w-10 flex items-center justify-center text-muted hover:bg-elevated/60 hover:text-highlighted transition-colors duration-150"
        :class="route.name === 'settings' ? 'text-primary' : ''"
        title="设置"
      >
        <UIcon name="i-tabler-settings" class="size-4" />
      </RouterLink>
      <RouterLink
        to="/help"
        class="w-10 flex items-center justify-center text-muted hover:bg-elevated/60 hover:text-highlighted transition-colors duration-150"
        :class="route.name === 'help' ? 'text-primary' : ''"
        title="帮助"
      >
        <UIcon name="i-tabler-help-circle" class="size-4" />
      </RouterLink>
      <span class="w-px bg-default/60 my-3" />

      <button
        type="button"
        class="w-12 flex items-center justify-center text-muted hover:bg-elevated/60 hover:text-highlighted transition-colors duration-150"
        title="最小化"
        @click="onMinimise"
      >
        <UIcon name="i-tabler-minus" class="size-4" />
      </button>
      <button
        type="button"
        class="w-12 flex items-center justify-center text-muted hover:bg-elevated/60 hover:text-highlighted transition-colors duration-150"
        :title="isMaximised ? '还原' : '最大化'"
        @click="onToggleMaximise"
      >
        <UIcon :name="isMaximised ? 'i-tabler-copy' : 'i-tabler-square'" class="size-3.5" />
      </button>
      <button
        type="button"
        class="w-12 flex items-center justify-center text-muted hover:bg-error hover:text-highlighted transition-colors duration-150"
        title="关闭"
        @click="onClose"
      >
        <UIcon name="i-tabler-x" class="size-4" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useSidebarCollapsed } from '@/composables/useSidebarCollapsed'
import { useWindowControls } from '@/composables/useWindowControls'

const version = '1.1.0'
const route = useRoute()
const { collapsed } = useSidebarCollapsed()
const { isMaximised, onMinimise, onToggleMaximise, closeImmediate: onClose } = useWindowControls()

const VIEW_META: Record<string, { title: string; icon: string }> = {
  fish: { title: '钓鱼', icon: 'i-tabler-fish' },
  cook: { title: '店长', icon: 'i-tabler-tools-kitchen-2' },
  piano: { title: '弹琴', icon: 'i-tabler-music' },
  battle: { title: '战斗', icon: 'i-tabler-swords' },
  settings: { title: '设置', icon: 'i-tabler-settings' },
  help: { title: '帮助', icon: 'i-tabler-help-circle' },
  about: { title: '关于', icon: 'i-tabler-info-circle' },
}
const currentTitle = computed(() => VIEW_META[route.name as string]?.title ?? '')
const currentIcon = computed(() => VIEW_META[route.name as string]?.icon ?? '')
// 窗口控件 (isMaximised + onMinimise / onToggleMaximise / onClose) 全由 useWindowControls 提供
</script>
