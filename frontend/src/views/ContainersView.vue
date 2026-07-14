<template>
  <div class="workspace-page">
    <header class="workspace-page__header">
      <div class="min-w-0">
        <div class="flex items-center gap-3">
          <span class="workspace-page__mark"
            ><UIcon name="i-tabler-box-multiple" class="size-5"
          /></span>
          <div>
            <p class="workspace-page__eyebrow">{{ t('containers.workspace.eyebrow') }}</p>
            <h1 class="workspace-page__title">{{ t('containers.workspace.title') }}</h1>
          </div>
        </div>
        <p class="workspace-page__description">{{ t('containers.workspace.description') }}</p>
      </div>

      <UTabs v-model="tab" :items="tabs" size="sm" :content="false" class="workspace-page__tabs" />
    </header>

    <div class="min-h-0 flex-1 px-6 pb-5 sm:px-8">
      <ContainersTab v-if="tab === 'local'" />
      <EmptyState
        v-else
        icon="i-tabler-cloud"
        :title="t('containers.online.title')"
        :description="t('containers.online.desc')"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ContainersTab from '@/components/tasks/ContainersTab.vue'
import { useContainerEditorStore } from '@/stores/containerEditor'
import EmptyState from '@/components/common/EmptyState.vue'

const { t } = useI18n()
const tab = ref('local')
const tabs = computed(() => [
  { label: t('containers.tab.local'), value: 'local', icon: 'i-tabler-device-desktop' },
  { label: t('containers.tab.online'), value: 'online', icon: 'i-tabler-cloud' },
])

useContainerEditorStore().clearLastEditing()
</script>
