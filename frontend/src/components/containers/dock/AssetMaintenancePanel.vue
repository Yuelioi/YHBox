<template>
  <div class="h-full min-h-0 overflow-y-auto">
    <header class="border-b border-default px-4 py-4">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-database-cog" class="size-4 text-primary" aria-hidden="true" />
        <h2 class="text-sm font-semibold text-highlighted">{{ t('assetMaintenance.title') }}</h2>
      </div>
      <p class="mt-1 max-w-lg text-xs leading-relaxed text-dimmed">
        {{ t('assetMaintenance.description') }}
      </p>
    </header>

    <div class="divide-y divide-default">
      <section class="flex items-start gap-3 px-4 py-4">
        <UIcon name="i-tabler-player-record" class="mt-0.5 size-4 shrink-0 text-toned" />
        <div class="min-w-0 flex-1">
          <h3 class="text-xs font-medium text-highlighted">
            {{ t('assetMaintenance.recordings.title') }}
          </h3>
          <p class="mt-1 text-xs leading-relaxed text-dimmed">
            {{ t('assetMaintenance.recordings.description') }}
          </p>
        </div>
        <UButton
          size="xs"
          color="neutral"
          variant="outline"
          icon="i-tabler-broom"
          class="shrink-0"
          @click="openCleanup"
        >
          {{ t('assetMaintenance.recordings.action') }}
        </UButton>
      </section>

      <section class="flex items-start gap-3 px-4 py-4">
        <UIcon name="i-tabler-hierarchy" class="mt-0.5 size-4 shrink-0 text-toned" />
        <div class="min-w-0 flex-1">
          <h3 class="text-xs font-medium text-highlighted">
            {{ t('assetMaintenance.subgraphs.title') }}
          </h3>
          <p class="mt-1 text-xs leading-relaxed text-dimmed">
            {{ t('assetMaintenance.subgraphs.description') }}
          </p>
        </div>
        <UButton
          size="xs"
          color="neutral"
          variant="ghost"
          icon="i-tabler-arrow-right"
          class="shrink-0"
          @click="emit('navigate', 'library')"
        >
          {{ t('assetMaintenance.subgraphs.action') }}
        </UButton>
      </section>

      <section class="flex items-start gap-3 px-4 py-4">
        <UIcon name="i-tabler-photo" class="mt-0.5 size-4 shrink-0 text-toned" />
        <div class="min-w-0 flex-1">
          <h3 class="text-xs font-medium text-highlighted">
            {{ t('assetMaintenance.templates.title') }}
          </h3>
          <p class="mt-1 text-xs leading-relaxed text-dimmed">
            {{ t('assetMaintenance.templates.description') }}
          </p>
        </div>
        <UButton
          size="xs"
          color="neutral"
          variant="ghost"
          icon="i-tabler-arrow-right"
          class="shrink-0"
          @click="emit('navigate', 'templates')"
        >
          {{ t('assetMaintenance.templates.action') }}
        </UButton>
      </section>
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
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import { backend } from '@/lib/backend'
import { errorMessage } from '@/lib/invoke'
import RecordingCleanupModal from '@/components/containers/RecordingCleanupModal.vue'

interface RecordingCleanupPreview {
  unused: Array<{ id: string; label: string; kind: string; references: number }>
  referenced: Array<{ id: string; label: string; kind: string; references: number }>
}

const emit = defineEmits<{
  navigate: [target: 'templates' | 'library']
}>()

const { t } = useI18n()
const toast = useToast()
const cleanupOpen = ref(false)
const cleanupLoading = ref(false)
const cleanupBusy = ref(false)
const cleanupError = ref('')
const cleanupPreview = ref<RecordingCleanupPreview>({ unused: [], referenced: [] })

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
