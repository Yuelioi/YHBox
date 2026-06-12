<template>
  <div class="px-8 py-6 space-y-5">
    <header class="flex items-center justify-center">
      <UTabs v-model="tab" :items="tabs" class="w-[360px]" />
    </header>

    <ContainersTab v-if="tab === 'local'" />
    <!-- 在线容器: 占位 (整包容器分享/下载留口, 未实现 — 见 specs/2026-06-13-editor-rail-resources.md ⑥) -->
    <div v-else class="flex flex-col items-center justify-center text-center py-16">
      <UIcon name="i-tabler-cloud" class="size-12 text-dimmed mb-3" />
      <h3 class="text-sm text-toned font-medium">{{ t('containers.online.title') }}</h3>
      <p class="text-xs text-dimmed mt-2 max-w-xs">{{ t('containers.online.desc') }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ContainersTab from '@/components/tasks/ContainersTab.vue'
import { useContainerEditorStore } from '@/stores/containerEditor'

const { t } = useI18n()

const tab = ref<string>('local')
const tabs = computed(() => [
  { label: t('containers.tab.local'), value: 'local', icon: 'i-tabler-schema' },
  { label: t('containers.tab.online'), value: 'online', icon: 'i-tabler-cloud' },
])

// 进列表 = 用户主动放手 "正在编辑某容器" 状态. 侧栏 '容器' 跳法跟着切回列表.
useContainerEditorStore().clearLastEditing()
</script>
