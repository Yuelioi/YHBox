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
        :title="collapsed ? t('sidebar.expand_tip') : t('sidebar.collapse_tip')"
        class="w-full relative flex items-center h-9 rounded-md text-sm transition-colors duration-150 text-muted hover:bg-elevated/40 hover:text-highlighted"
        :class="collapsed ? 'justify-center px-0' : 'gap-2.5 px-3'"
        @click="toggle"
      >
        <UIcon
          :name="collapsed ? 'i-tabler-chevrons-right' : 'i-tabler-chevrons-left'"
          class="size-4 shrink-0"
        />
        <span v-if="!collapsed" class="flex-1 truncate text-left">{{ t('sidebar.collapse') }}</span>
      </button>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { useSidebarCollapsed } from '@/composables/useSidebarCollapsed'
import { useContainerEditorStore } from '@/stores/containerEditor'

const { t } = useI18n()
const route = useRoute()
const editorStore = useContainerEditorStore()

const { collapsed, toggle } = useSidebarCollapsed()

const activeClass = 'bg-elevated/60 text-highlighted font-medium'
const inactiveClass = 'text-muted hover:bg-elevated/40 hover:text-highlighted'

// '容器' tab — 有 lastEditingContainerID 就跳回编辑器路由 (keep-alive cache 命中, draft 不丢).
// 否则跳列表. ContainersView mount 时 clearLastEditing → 之后切回又落列表.
const containersTo = computed(() =>
  editorStore.lastEditingContainerID
    ? `/containers/${editorStore.lastEditingContainerID}/edit`
    : '/containers',
)

const automationItems = computed(() => [
  { label: t('sidebar.containers'), to: containersTo.value, icon: 'i-tabler-package' },
  { label: t('sidebar.schedules'), to: '/schedules', icon: 'i-tabler-clock' },
])

const toolItems = computed(() => [
  { label: t('sidebar.settings'), to: '/settings', icon: 'i-tabler-settings' },
  { label: t('sidebar.about'), to: '/about', icon: 'i-tabler-info-circle' },
])

function isActive(to: string): boolean {
  return to === '/' ? route.path === '/' : route.path.startsWith(to)
}
</script>
