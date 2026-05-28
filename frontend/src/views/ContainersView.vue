<template>
  <div class="px-8 py-6 space-y-5">
    <header class="flex items-center justify-center">
      <UTabs v-model="tab" :items="tabs" class="w-[360px]" />
    </header>

    <ContainersTab v-if="tab === 'containers'" />
    <TemplatesTab v-else-if="tab === 'templates'" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ContainersTab from '@/components/tasks/ContainersTab.vue'
import TemplatesTab from '@/components/tasks/TemplatesTab.vue'
import { useContainerEditorStore } from '@/stores/containerEditor'

const { t } = useI18n()

const tab = ref<string>('containers')
const tabs = computed(() => [
  { label: t('containers.tab.containers'), value: 'containers', icon: 'i-tabler-schema' },
  { label: t('containers.tab.templates'), value: 'templates', icon: 'i-tabler-photo' },
])

// 进列表 = 用户主动放手 "正在编辑某容器" 状态. 侧栏 '容器' 跳法跟着切回列表.
useContainerEditorStore().clearLastEditing()
</script>
