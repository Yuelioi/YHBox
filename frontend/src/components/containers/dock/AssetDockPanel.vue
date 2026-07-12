<!-- 资产停靠面板宿主: UTabs 收 模板 / 子图库 / Clip 三类资产 (替原 3 个 5xl modal).
     宽态停靠 (~600px), 挤画布不盖。模板支持 pick 模式 (从节点字段唤起, 实时回写)。 -->
<template>
  <div class="flex flex-col h-full min-h-0">
    <div class="flex items-center gap-2 px-2 pt-2 shrink-0">
      <UTabs
        :model-value="tab"
        :items="tabItems"
        :content="false"
        size="sm"
        class="min-w-0 flex-1"
        @update:model-value="(v: string | number) => emit('update:tab', v as AssetTab)"
      />
      <UButton
        v-if="tab !== 'templates'"
        size="xs"
        color="neutral"
        variant="outline"
        icon="i-tabler-broom"
        :aria-label="t('recordingCleanup.title')"
        @click="openCleanup"
      >
        {{ t('recordingCleanup.action') }}
      </UButton>
    </div>
    <div class="flex-1 min-h-0 overflow-hidden">
      <TemplateAssetPanel
        v-if="tab === 'templates'"
        :pick-mode="templatePickMode"
        :model-value="templateSelected"
        @update:model-value="(v: string[]) => emit('update:template-selected', v)"
      />
      <LibraryAssetPanel
        v-else-if="tab === 'library'"
        @pick-subgraph="(id: string) => emit('pick-subgraph', id)"
      />
      <ClipAssetPanel
        v-else-if="tab === 'clips'"
        @pick-clip="(id: string) => emit('pick-clip', id)"
      />
    </div>
    <RecordingCleanupModal
      v-model:open="cleanupOpen"
      :preview="cleanupPreview"
      :loading="cleanupLoading"
      :busy="cleanupBusy"
      :error="cleanupError"
      @confirm="cleanup"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import { backend } from '@/lib/backend'
import { errorMessage } from '@/lib/invoke'
import TemplateAssetPanel from './TemplateAssetPanel.vue'
import LibraryAssetPanel from './LibraryAssetPanel.vue'
import ClipAssetPanel from './ClipAssetPanel.vue'
import RecordingCleanupModal from '@/components/containers/RecordingCleanupModal.vue'

interface RecordingCleanupPreview {
  unused: Array<{ id: string; label: string; kind: string; references: number }>
  referenced: Array<{ id: string; label: string; kind: string; references: number }>
}

type AssetTab = 'templates' | 'library' | 'clips'

defineProps<{
  tab: AssetTab
  templatePickMode: boolean
  templateSelected: string[]
}>()
const emit = defineEmits<{
  'update:tab': [v: AssetTab]
  'update:template-selected': [v: string[]]
  'pick-subgraph': [id: string]
  'pick-clip': [id: string]
}>()

const { t } = useI18n()
const toast = useToast()
const cleanupOpen = ref(false)
const cleanupLoading = ref(false)
const cleanupBusy = ref(false)
const cleanupError = ref('')
const cleanupPreview = ref<RecordingCleanupPreview>({ unused: [], referenced: [] })
const tabItems = computed(() => [
  { value: 'templates', label: t('template.manager.title'), icon: 'i-tabler-photo' },
  { value: 'library', label: t('editor.toolbar.library_explorer'), icon: 'i-tabler-books' },
  { value: 'clips', label: t('clip.manager.title'), icon: 'i-tabler-movie' },
])

async function openCleanup() {
  cleanupOpen.value = true
  cleanupLoading.value = true
  cleanupError.value = ''
  try {
    cleanupPreview.value = (await backend.recording.previewCleanup()) as RecordingCleanupPreview
  } catch (error) {
    cleanupError.value = errorMessage(error)
  } finally {
    cleanupLoading.value = false
  }
}

async function cleanup(ids: string[]) {
  cleanupBusy.value = true
  try {
    const result = (await backend.recording.cleanupUnused(ids)) as {
      deleted: string[]
      skipped: unknown[]
      failed: string[]
    }
    if (result.failed.length > 0) {
      toast.add({
        title: t('recordingCleanup.partial_failed', { n: result.failed.length }),
        color: 'error',
      })
      await openCleanup()
      return
    }
    if (result.skipped.length > 0) {
      toast.add({
        title: t('recordingCleanup.changed_refs', { n: result.skipped.length }),
        color: 'warning',
      })
    }
    cleanupOpen.value = false
  } catch (error) {
    toast.add({
      title: t('recordingCleanup.delete_failed'),
      description: errorMessage(error),
      color: 'error',
    })
  } finally {
    cleanupBusy.value = false
  }
}
</script>
