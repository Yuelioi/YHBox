<template>
  <div class="flex h-full">
    <!-- 左侧 vertical tab -->
    <aside class="w-32 shrink-0 border-r border-default p-3 space-y-1">
      <button
        v-for="t in tabs"
        :key="t.key"
        :class="[
          'w-full text-left px-3 py-2 rounded text-sm transition-colors',
          activeTab === t.key
            ? 'bg-elevated text-default'
            : 'text-muted hover:text-default hover:bg-elevated/50',
        ]"
        @click="activeTab = t.key"
      >
        {{ t.label }}
      </button>
    </aside>
    <!-- 右侧 content -->
    <div class="flex-1 overflow-auto">
      <SettingsGeneral v-if="activeTab === 'general'" />
      <SettingsHotkeys v-if="activeTab === 'hotkeys'" />
      <SettingsInput v-if="activeTab === 'input'" />
      <SettingsLauncher v-if="activeTab === 'launcher'" />
      <SettingsAI v-if="activeTab === 'ai'" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import SettingsGeneral from './SettingsGeneral.vue'
import SettingsHotkeys from './SettingsHotkeys.vue'
import SettingsInput from './SettingsInput.vue'
import SettingsLauncher from './SettingsLauncher.vue'
import SettingsAI from './SettingsAI.vue'

const { t } = useI18n()

type TabKey = 'general' | 'hotkeys' | 'input' | 'launcher' | 'ai'

const tabs = computed<{ key: TabKey; label: string }[]>(() => [
  { key: 'general', label: t('settingsTab.general') },
  { key: 'hotkeys', label: t('settingsTab.hotkeys') },
  { key: 'input', label: t('settingsTab.input_calibration') },
  { key: 'launcher', label: t('settingsTab.launcher') },
  { key: 'ai', label: t('settingsTab.ai') },
])
const activeTab = ref<TabKey>('general')
</script>
