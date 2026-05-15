<template>
  <nav
    class="flex flex-col shrink-0 h-full bg-default border-r border-default transition-[width] duration-200 ease-out"
    :class="collapsed ? 'w-14' : 'w-40'"
  >
    <!-- Nav sections -->
    <div class="flex-1 overflow-y-auto overflow-x-hidden py-2">
      <!-- AUTOMATION section -->
      <div v-if="!collapsed" class="px-3 pt-3 pb-1">
        <span class="text-[10px] font-semibold uppercase tracking-widest text-dimmed"
          >Automation</span
        >
      </div>
      <div v-else class="pt-3 pb-1 flex justify-center">
        <span class="h-px w-6 bg-muted rounded-full" />
      </div>

      <div class="px-2 space-y-0.5">
        <RouterLink
          v-for="item in automationItems"
          :key="item.to"
          :to="item.to"
          :title="collapsed ? item.label : undefined"
          class="relative flex items-center h-9 rounded-md text-sm transition-colors duration-150"
          :class="[
            isActive(item.to) ? activeClass : inactiveClass,
            collapsed ? 'justify-center px-0' : 'gap-2.5 px-3',
          ]"
        >
          <span
            v-if="isActive(item.to)"
            class="absolute left-0 top-1.5 bottom-1.5 w-0.5 bg-primary rounded-r"
          />
          <UIcon :name="item.icon" class="size-4 shrink-0" />
          <span v-if="!collapsed" class="flex-1 truncate">{{ item.label }}</span>
          <span
            v-if="item.store"
            class="size-1.5 rounded-full shrink-0"
            :class="[botDotClass(item.store), collapsed ? 'absolute top-1 right-1' : '']"
          />
        </RouterLink>
      </div>

      <!-- TOOLS section -->
      <div v-if="!collapsed" class="px-3 pt-4 pb-1">
        <span class="text-[10px] font-semibold uppercase tracking-widest text-dimmed">Tools</span>
      </div>
      <div v-else class="pt-4 pb-1 flex justify-center">
        <span class="h-px w-6 bg-muted rounded-full" />
      </div>

      <div class="px-2 space-y-0.5">
        <RouterLink
          v-for="item in toolItems"
          :key="item.to"
          :to="item.to"
          :title="collapsed ? item.label : undefined"
          class="relative flex items-center h-9 rounded-md text-sm transition-colors duration-150"
          :class="[
            isActive(item.to) ? activeClass : inactiveClass,
            collapsed ? 'justify-center px-0' : 'gap-2.5 px-3',
          ]"
        >
          <span
            v-if="isActive(item.to)"
            class="absolute left-0 top-1.5 bottom-1.5 w-0.5 bg-primary rounded-r"
          />
          <UIcon :name="item.icon" class="size-4 shrink-0" />
          <span v-if="!collapsed" class="truncate">{{ item.label }}</span>
        </RouterLink>
      </div>
    </div>

    <!-- Collapse toggle: 直接做成跟上方 menu items 同 RouterLink 样式
         (无 border-t、相同 hover/text/size)。视觉上跟设置/帮助/关于无缝衔接。 -->
    <div class="px-2 pb-2 shrink-0">
      <button
        type="button"
        :title="collapsed ? '展开侧栏' : '收起侧栏'"
        class="w-full relative flex items-center h-9 rounded-md text-sm transition-colors duration-150 text-muted hover:bg-elevated/40 hover:text-highlighted"
        :class="collapsed ? 'justify-center px-0' : 'gap-2.5 px-3'"
        @click="toggle"
      >
        <UIcon
          :name="collapsed ? 'i-tabler-chevrons-right' : 'i-tabler-chevrons-left'"
          class="size-4 shrink-0"
        />
        <span v-if="!collapsed" class="flex-1 truncate text-left">收起</span>
      </button>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router'
import { useSidebarCollapsed } from '@/composables/useSidebarCollapsed'
import { BOTS, type BotState } from '@/lib/bot-registry'
import { useBattleStore } from '@/stores/battle'

const route = useRoute()
const battleStore = useBattleStore()

const { collapsed, toggle } = useSidebarCollapsed()

const activeClass = 'bg-elevated/60 text-highlighted font-medium'
const inactiveClass = 'text-muted hover:bg-elevated/40 hover:text-highlighted'

// automation items：4 个长跑 bot 来自 registry + 战斗（hotkey-driven 单独 append）+
// 任务（containers/actions/templates 三合一）+ 计划（schedule trigger 列表）
// 注：原 /actions 入口删除，所有 Action 操作走"任务 → 动作 tab"统一入口
const automationItems = [
  ...BOTS.map((b) => ({ label: b.label, to: b.route, icon: b.icon, store: b.kind })),
  { label: '战斗', to: '/battle', icon: 'i-tabler-swords', store: 'battle' },
  { label: '任务', to: '/tasks', icon: 'i-tabler-package', store: '' },
  { label: '计划', to: '/schedules', icon: 'i-tabler-clock', store: '' },
]

const toolItems = [
  { label: '设置', to: '/settings', icon: 'i-tabler-settings' },
  { label: '帮助', to: '/help', icon: 'i-tabler-help-circle' },
  { label: '关于', to: '/about', icon: 'i-tabler-info-circle' },
]

function isActive(to: string): boolean {
  return to === '/' ? route.path === '/' : route.path.startsWith(to)
}

function getBotState(storeName: string): BotState {
  if (storeName === 'battle') {
    return battleStore.status.enabled ? 'running' : 'idle'
  }
  const bot = BOTS.find((b) => b.kind === storeName)
  return bot ? bot.useStore().state : 'idle'
}

function botDotClass(storeName: string): string {
  const state = getBotState(storeName)
  if (state === 'running') return 'bg-primary animate-pulse'
  if (state === 'paused') return 'bg-warning'
  return 'bg-accented'
}
</script>
