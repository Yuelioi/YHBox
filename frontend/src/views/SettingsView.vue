<template>
  <div class="flex h-full min-h-0">
    <!-- 左侧 vertical tab -->
    <aside class="w-44 shrink-0 border-r border-default bg-muted/10 p-4">
      <div class="mb-4 flex items-center gap-2 px-2">
        <UIcon name="i-tabler-settings" class="size-4 text-primary" aria-hidden="true" />
        <h1 class="text-sm font-semibold text-highlighted">{{ t('sidebar.settings') }}</h1>
      </div>
      <nav
        class="space-y-1"
        role="tablist"
        aria-orientation="vertical"
        :aria-label="t('sidebar.settings')"
      >
        <button
          v-for="tab in tabs"
          :id="`settings-tab-${tab.key}`"
          :key="tab.key"
          type="button"
          role="tab"
          :aria-selected="activeTab === tab.key"
          aria-controls="settings-tabpanel"
          :class="[
            'flex h-10 w-full items-center gap-2 rounded-md px-3 text-left text-sm transition-colors focus-visible:ring-2 focus-visible:ring-primary',
            activeTab === tab.key
              ? 'bg-elevated text-highlighted'
              : 'text-muted hover:bg-elevated/50 hover:text-default',
          ]"
          @click="activeTab = tab.key"
          @keydown="onTabKeydown"
        >
          <UIcon
            :name="tab.icon"
            class="size-4 shrink-0"
            :class="activeTab === tab.key ? 'text-primary' : 'text-dimmed'"
            aria-hidden="true"
          />
          <span class="truncate">{{ tab.label }}</span>
        </button>
      </nav>
    </aside>
    <!-- 右侧 content -->
    <div
      id="settings-tabpanel"
      role="tabpanel"
      :aria-labelledby="`settings-tab-${activeTab}`"
      tabindex="0"
      class="min-w-0 flex-1 overflow-auto"
    >
      <SettingsGeneral v-if="activeTab === 'general'" />
      <SettingsHotkeys v-if="activeTab === 'hotkeys'" />
      <SettingsInput v-if="activeTab === 'input'" />
      <SettingsLauncher v-if="activeTab === 'launcher'" />
      <SettingsAI v-if="activeTab === 'ai'" />
      <SettingsMCP v-if="activeTab === 'mcp'" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import SettingsGeneral from './SettingsGeneral.vue'
import SettingsHotkeys from './SettingsHotkeys.vue'
import SettingsInput from './SettingsInput.vue'
import SettingsLauncher from './SettingsLauncher.vue'
import SettingsAI from './SettingsAI.vue'
import SettingsMCP from './SettingsMCP.vue'

const { t } = useI18n()

type TabKey = 'general' | 'hotkeys' | 'input' | 'launcher' | 'ai' | 'mcp'

const tabs = computed<{ key: TabKey; label: string; icon: string }[]>(() => [
  { key: 'general', label: t('settingsTab.general'), icon: 'i-tabler-adjustments' },
  { key: 'hotkeys', label: t('settingsTab.hotkeys'), icon: 'i-tabler-keyboard' },
  { key: 'input', label: t('settingsTab.input_calibration'), icon: 'i-tabler-mouse' },
  { key: 'launcher', label: t('settingsTab.launcher'), icon: 'i-tabler-layout-grid' },
  { key: 'ai', label: t('settingsTab.ai'), icon: 'i-tabler-sparkles' },
  { key: 'mcp', label: t('settingsTab.mcp'), icon: 'i-tabler-plug' },
])
const activeTab = ref<TabKey>('general')

function onTabKeydown(event: KeyboardEvent) {
  const keys = tabs.value.map((tab) => tab.key)
  const current = keys.indexOf(activeTab.value)
  let next = current
  if (event.key === 'ArrowDown') next = (current + 1) % keys.length
  else if (event.key === 'ArrowUp') next = (current - 1 + keys.length) % keys.length
  else if (event.key === 'Home') next = 0
  else if (event.key === 'End') next = keys.length - 1
  else return

  event.preventDefault()
  activeTab.value = keys[next]
  void nextTick(() => document.getElementById(`settings-tab-${activeTab.value}`)?.focus())
}
</script>
