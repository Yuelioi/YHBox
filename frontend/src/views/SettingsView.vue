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
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import SettingsGeneral from './SettingsGeneral.vue'
import SettingsHotkeys from './SettingsHotkeys.vue'
import SettingsInput from './SettingsInput.vue'

type TabKey = 'general' | 'hotkeys' | 'input'

const tabs: { key: TabKey; label: string }[] = [
  { key: 'general', label: '通用' },
  { key: 'hotkeys', label: '快捷键' },
  { key: 'input', label: '输入校准' },
]
const activeTab = ref<TabKey>('general')
</script>
