<template>
  <UContextMenu :items="ctxMenuItems">
    <div
      :class="[
        'rounded-md border bg-elevated/30 transition-colors cursor-pointer relative group',
        viewMode === 'list' ? 'p-2 flex items-center gap-3' : 'p-3',
        selected ? 'border-primary ring-2 ring-primary/40' : 'border-default hover:border-primary/50',
      ]"
      @click="$emit('select', sgID)"
    >
      <div
        v-if="viewMode === 'list'"
        class="size-7 rounded-md flex items-center justify-center shrink-0 bg-fuchsia-500/15 border border-fuchsia-500/40"
      >
        <UIcon name="i-tabler-package" class="size-4 text-fuchsia-300" />
      </div>

      <div v-if="viewMode === 'list'" class="min-w-0 flex-1 flex items-center gap-2">
        <span class="text-sm font-medium truncate text-default">{{ displayName }}</span>
        <span v-if="pkg.root.description" class="text-[11px] text-dimmed truncate flex-1">{{ pkg.root.description }}</span>
        <UBadge size="xs" variant="subtle" color="neutral" class="shrink-0">
          {{ t('library.card.embedded_count', { n: embeddedCount }) }}
        </UBadge>
      </div>

      <template v-else>
        <div class="flex items-center gap-2 mb-2">
          <UIcon name="i-tabler-package" class="size-4 text-fuchsia-300" />
          <span class="text-sm font-medium truncate flex-1">{{ displayName }}</span>
          <UBadge size="xs" variant="subtle" color="neutral">{{ embeddedCount }}</UBadge>
        </div>
        <p v-if="pkg.root.description" class="text-[11px] text-dimmed mb-2 line-clamp-2">{{ pkg.root.description }}</p>
        <div class="flex gap-1 flex-wrap">
          <UBadge v-if="pkg.templates.length > 0" size="xs" variant="outline" color="neutral">
            <UIcon name="i-tabler-photo" class="size-3 mr-0.5" />{{ t('library.card.templates_count', { n: pkg.templates.length }) }}
          </UBadge>
          <UBadge v-if="pkg.clips.length > 0" size="xs" variant="outline" color="neutral">
            <UIcon name="i-tabler-vinyl" class="size-3 mr-0.5" />{{ t('library.card.clips_count', { n: pkg.clips.length }) }}
          </UBadge>
        </div>
      </template>
    </div>
  </UContextMenu>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SubgraphPackage } from '@/lib/backend'
import { useToast } from '@nuxt/ui/composables'
import { useConfirm } from '@/composables/useConfirm'
import { useLibraryStore } from '@/stores/library'

const { t } = useI18n()

const props = withDefaults(
  defineProps<{
    sgID: string
    pkg: SubgraphPackage
    selected?: boolean
    viewMode?: 'grid' | 'list'
  }>(),
  { selected: false, viewMode: 'grid' },
)

const emit = defineEmits<{
  select: [sgID: string]
  import: [sgID: string]
}>()

const libraryStore = useLibraryStore()
const toast = useToast()
const { confirm } = useConfirm()

const displayName = computed(() => props.pkg.root.label || props.sgID)
const embeddedCount = computed(() => 1 + Object.keys(props.pkg.embedded ?? {}).length)

const ctxMenuItems = computed(() => [
  [
    {
      label: t('library.card.view_detail'),
      icon: 'i-tabler-info-circle',
      onSelect: () => emit('select', props.sgID),
    },
    {
      label: t('library.card.import'),
      icon: 'i-tabler-arrow-bar-to-down',
      onSelect: () => emit('import', props.sgID),
    },
  ],
  [
    {
      label: t('library.card.copy_id'),
      icon: 'i-tabler-copy',
      onSelect: () => onCopyID(),
    },
  ],
  [
    {
      label: t('library.card.delete'),
      icon: 'i-tabler-trash',
      color: 'error' as const,
      onSelect: () => onDelete(),
    },
  ],
])

async function onCopyID() {
  try {
    await navigator.clipboard.writeText(props.sgID)
    toast.add({ title: t('toast.copy_id_success'), color: 'success', icon: 'i-tabler-check' })
  } catch (e: any) {
    toast.add({ title: t('toast.copy_failed'), description: String(e?.message ?? e), color: 'error' })
  }
}

async function onDelete() {
  const yes = await confirm({
    title: t('library.card.delete_confirm_title'),
    description: t('library.card.delete_confirm_desc', { name: displayName.value }),
    color: 'error',
    confirmText: t('common.delete'),
  })
  if (yes !== true) return
  const ok = await libraryStore.deletePackage(props.sgID)
  if (ok) {
    toast.add({ title: t('toast.deleted'), color: 'success' })
  } else {
    toast.add({ title: t('toast.delete_failed'), color: 'error' })
  }
}
</script>
