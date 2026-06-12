<template>
  <aside class="w-80 shrink-0 border-l border-default overflow-y-auto bg-default">
    <div
      v-if="!sg"
      class="h-full flex flex-col items-center justify-center text-center px-6 py-10"
    >
      <UIcon name="i-tabler-pointer" class="size-10 text-dimmed mb-3" />
      <p class="text-sm text-toned">{{ t('library.detail.empty') }}</p>
      <p class="text-[11px] text-dimmed mt-1">{{ t('library.detail.empty_hint') }}</p>
    </div>

    <div v-else class="p-4 space-y-4">
      <header class="flex items-start gap-3 pb-3 border-b border-default">
        <div class="size-10 rounded-lg flex items-center justify-center shrink-0 bg-fuchsia-500/15 border border-fuchsia-500/40">
          <UIcon name="i-tabler-subtask" class="size-5 text-fuchsia-300" />
        </div>
        <div class="min-w-0 flex-1">
          <h3 class="text-sm font-medium text-highlighted truncate leading-tight">
            {{ sg.label || sg.id }}
          </h3>
          <p class="text-[11px] text-dimmed mt-0.5">
            {{ t('library.detail.nodes_and_outputs', { n: sg.graph?.nodes?.length ?? 0, m: sg.outputPins?.length ?? 0 }) }}
          </p>
        </div>
      </header>

      <section v-if="sg.description" class="space-y-1.5">
        <label class="block text-xs text-toned">{{ t('library.detail.description') }}</label>
        <p class="text-xs text-default whitespace-pre-line">{{ sg.description }}</p>
      </section>

      <!-- 引用计数: 让用户知道在动共享物 (即扫即得, 选中时拉一次)。 -->
      <section class="space-y-1 text-[11px] text-dimmed">
        <div class="flex justify-between">
          <span>{{ t('library.detail.used_by') }}</span>
          <span>{{ useCount === null ? '…' : t('library.detail.used_by_n', { n: useCount }) }}</span>
        </div>
        <div v-if="sg.createdAt" class="flex justify-between">
          <span>{{ t('library.detail.created_at') }}</span>
          <span>{{ new Date(sg.createdAt).toLocaleString() }}</span>
        </div>
      </section>

      <section v-if="(sg.tags ?? []).length > 0" class="space-y-1.5">
        <label class="block text-xs text-toned">{{ t('library.detail.tags') }}</label>
        <div class="flex flex-wrap gap-1">
          <UBadge v-for="tag in sg.tags ?? []" :key="tag" size="xs" variant="subtle">{{ tag }}</UBadge>
        </div>
      </section>

      <section class="space-y-1.5">
        <label class="block text-[10px] uppercase tracking-[0.08em] font-semibold text-dimmed">ID</label>
        <button
          type="button"
          class="w-full text-left text-[11px] font-mono bg-elevated/40 rounded px-2 py-1 hover:bg-elevated/60 transition-colors truncate flex items-center gap-1.5"
          :class="copied ? 'text-success' : 'text-dimmed'"
          :title="t('library.detail.click_to_copy') + sg.id"
          @click="onCopyID"
        >
          <UIcon v-if="copied" name="i-tabler-check" class="size-3 shrink-0" />
          <span class="truncate">{{ copied ? t('common.copied') : sg.id }}</span>
        </button>
      </section>

      <div class="pt-3 border-t border-default flex flex-col gap-2">
        <UButton
          size="sm"
          variant="soft"
          color="primary"
          icon="i-tabler-copy-plus"
          @click="onDuplicate"
        >
          {{ t('library.card.duplicate') }}
        </UButton>
        <UButton size="sm" variant="soft" color="error" icon="i-tabler-trash" @click="onDelete">
          {{ t('library.detail.delete') }}
        </UButton>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Subgraph } from '@/lib/backend'
import { useLibraryStore } from '@/stores/library'
import { useConfirm } from '@/composables/useConfirm'
import { useToast } from '@nuxt/ui/composables'
import { errorMessage } from '@/lib/invoke'

const { t } = useI18n()

const props = defineProps<{ sgID: string | null }>()

const emit = defineEmits<{
  cleared: []
}>()

const libraryStore = useLibraryStore()
const { confirm } = useConfirm()
const toast = useToast()

const sg = computed<Subgraph | undefined>(() => (props.sgID ? libraryStore.byId(props.sgID) : undefined))

// 「被 N 个容器使用」— 选中时拉一次 referrers (null = 加载中)。
const useCount = ref<number | null>(null)
watch(() => props.sgID, async (id) => {
  useCount.value = null
  if (!id) return
  const refs = await libraryStore.referrersOf(id)
  useCount.value = libraryStore.containerUseCount(refs)
}, { immediate: true })

const copied = ref(false)
let copiedTimer = 0
async function onCopyID() {
  if (!props.sgID) return
  try {
    await navigator.clipboard.writeText(props.sgID)
    copied.value = true
    window.clearTimeout(copiedTimer)
    copiedTimer = window.setTimeout(() => { copied.value = false }, 1500)
  } catch (e: any) {
    toast.add({ title: t('toast.copy_failed'), description: errorMessage(e), color: 'error' })
  }
}

async function onDuplicate() {
  if (!props.sgID) return
  const dup = await libraryStore.duplicateSubgraph(props.sgID)
  if (dup) {
    toast.add({ title: t('library.card.duplicated', { name: dup.label }), color: 'success', icon: 'i-tabler-check' })
  }
}

async function onDelete() {
  if (!props.sgID || !sg.value) return
  const refs = await libraryStore.referrersOf(props.sgID)
  const n = libraryStore.containerUseCount(refs)
  const desc = n > 0
    ? t('library.card.delete_confirm_referenced', { name: sg.value.label || props.sgID, n })
    : t('library.card.delete_confirm_desc', { name: sg.value.label || props.sgID })
  const yes = await confirm({
    title: t('library.card.delete_confirm_title'),
    description: desc,
    color: 'error',
    confirmText: t('common.delete'),
  })
  if (yes !== true) return
  const ok = await libraryStore.deleteSubgraph(props.sgID)
  if (ok) {
    emit('cleared')
  } else {
    toast.add({ title: t('toast.delete_failed'), color: 'error' })
  }
}
</script>
