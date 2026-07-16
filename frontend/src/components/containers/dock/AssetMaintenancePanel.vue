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
          variant="outline"
          icon="i-tabler-hierarchy"
          class="shrink-0"
          @click="openCleanup"
        >
          {{ t('assetMaintenance.subgraphs.action') }}
        </UButton>
      </section>
    </div>

    <AssetCleanupModal
      v-model:open="cleanupOpen"
      :resource="cleanupResource"
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
import AssetCleanupModal from '@/components/containers/AssetCleanupModal.vue'

interface CleanupPreview {
  unused: Array<{ id: string; label: string; kind: string; references: number }>
  referenced: Array<{ id: string; label: string; kind: string; references: number }>
}

const { t } = useI18n()
const toast = useToast()
const cleanupOpen = ref(false)
const cleanupResource = 'subgraph' as const
const cleanupLoading = ref(false)
const cleanupBusy = ref(false)
const cleanupError = ref('')
const cleanupPreview = ref<CleanupPreview>({ unused: [], referenced: [] })

async function openCleanup() {
  cleanupOpen.value = true
  cleanupLoading.value = true
  cleanupError.value = ''
  try {
    const preview = await backend.subgraphs.previewCleanup()
    cleanupPreview.value = (preview ?? { unused: [], referenced: [] }) as CleanupPreview
  } catch (error) {
    cleanupError.value = errorMessage(error)
  } finally {
    cleanupLoading.value = false
  }
}

async function cleanup(ids: string[]) {
  cleanupBusy.value = true
  try {
    const result = (await backend.subgraphs.cleanupUnused(ids)) as {
      deleted: string[]
      skipped: unknown[]
      failed: string[]
    }
    const copyPrefix = 'subgraphCleanup'
    if (result.failed.length > 0) {
      toast.add({
        title: t(`${copyPrefix}.partial_failed`, { n: result.failed.length }),
        color: 'error',
      })
      await openCleanup()
      return
    }
    if (result.skipped.length > 0) {
      toast.add({
        title: t(`${copyPrefix}.changed_refs`, { n: result.skipped.length }),
        color: 'warning',
      })
    }
    cleanupOpen.value = false
  } catch (error) {
    toast.add({
      title: t('subgraphCleanup.delete_failed'),
      description: errorMessage(error),
      color: 'error',
    })
  } finally {
    cleanupBusy.value = false
  }
}
</script>
