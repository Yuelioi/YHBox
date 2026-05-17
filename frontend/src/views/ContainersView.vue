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
import { ref } from 'vue'
import ContainersTab from '@/components/tasks/ContainersTab.vue'
import TemplatesTab from '@/components/tasks/TemplatesTab.vue'

const tab = ref<string>('containers')
const tabs = [
  { label: '容器', value: 'containers', icon: 'i-tabler-schema' },
  { label: '模板', value: 'templates', icon: 'i-tabler-photo' },
]
</script>
