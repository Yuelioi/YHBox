<template>
  <aside class="w-80 shrink-0 border-l border-default overflow-y-auto bg-default">
    <div
      v-if="!sgID"
      class="h-full flex flex-col items-center justify-center text-center px-6 py-10"
    >
      <UIcon name="i-tabler-pointer" class="size-10 text-dimmed mb-3" />
      <p class="text-sm text-toned">未选择</p>
      <p class="text-[11px] text-dimmed mt-1">单击卡片查看详情</p>
    </div>

    <div v-else-if="pkg" class="p-4 space-y-4">
      <header class="flex items-start gap-3 pb-3 border-b border-default">
        <div class="size-10 rounded-lg flex items-center justify-center shrink-0 bg-fuchsia-500/15 border border-fuchsia-500/40">
          <UIcon name="i-tabler-package" class="size-5 text-fuchsia-300" />
        </div>
        <div class="min-w-0 flex-1">
          <h3 class="text-sm font-medium text-highlighted truncate leading-tight">
            {{ pkg.root.label || sgID }}
          </h3>
          <p class="text-[11px] text-dimmed mt-0.5">
            {{ pkg.root.graph?.nodes?.length ?? 0 }} 节点 · {{ pkg.root.outputPins?.length ?? 0 }} 出口
          </p>
        </div>
      </header>

      <section v-if="pkg.root.description" class="space-y-1.5">
        <label class="block text-xs text-toned">描述</label>
        <p class="text-xs text-default whitespace-pre-line">{{ pkg.root.description }}</p>
      </section>

      <section class="space-y-1 text-[11px] text-dimmed">
        <div class="flex justify-between">
          <span>嵌入子图</span>
          <span>{{ Object.keys(pkg.embedded ?? {}).length }} 个</span>
        </div>
        <div class="flex justify-between">
          <span>模板</span>
          <span>{{ pkg.templates.length }} 个</span>
        </div>
        <div class="flex justify-between">
          <span>录制片段</span>
          <span>{{ pkg.clips.length }} 个</span>
        </div>
      </section>

      <section v-if="(pkg.root.tags ?? []).length > 0" class="space-y-1.5">
        <label class="block text-xs text-toned">标签</label>
        <div class="flex flex-wrap gap-1">
          <UBadge v-for="t in pkg.root.tags ?? []" :key="t" size="xs" variant="subtle">{{ t }}</UBadge>
        </div>
      </section>

      <section class="space-y-1.5">
        <label class="block text-[10px] uppercase tracking-[0.08em] font-semibold text-dimmed">ID</label>
        <button
          type="button"
          class="w-full text-left text-[11px] text-dimmed font-mono bg-elevated/40 rounded px-2 py-1 hover:bg-elevated/60 transition-colors truncate"
          :title="'点击复制 — ' + sgID"
          @click="onCopyID"
        >
          {{ sgID }}
        </button>
      </section>

      <div class="pt-3 border-t border-default flex flex-col gap-2">
        <UButton
          size="sm"
          variant="soft"
          color="primary"
          icon="i-tabler-arrow-bar-to-down"
          @click="$emit('import', sgID)"
        >
          导入到容器
        </UButton>
        <UButton size="sm" variant="soft" color="error" icon="i-tabler-trash" @click="onDelete">
          删除
        </UButton>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { SubgraphPackage } from '@/lib/backend'
import { useLibraryStore } from '@/stores/library'
import { useConfirm } from '@/composables/useConfirm'
import { useToast } from '@nuxt/ui/composables'

const props = defineProps<{ sgID: string | null }>()

const emit = defineEmits<{
  import: [sgID: string]
  cleared: []
}>()

const libraryStore = useLibraryStore()
const { confirm } = useConfirm()
const toast = useToast()

const pkg = computed<SubgraphPackage | undefined>(() =>
  props.sgID ? libraryStore.packages[props.sgID] : undefined,
)

async function onCopyID() {
  if (!props.sgID) return
  try {
    await navigator.clipboard.writeText(props.sgID)
    toast.add({ title: '已复制 ID', color: 'success', icon: 'i-tabler-check' })
  } catch (e: any) {
    toast.add({ title: '复制失败', description: String(e?.message ?? e), color: 'error' })
  }
}

async function onDelete() {
  if (!props.sgID || !pkg.value) return
  const yes = await confirm({
    title: '删除库子图',
    description: `确认删除 "${pkg.value.root.label || props.sgID}"？此操作不可恢复。`,
    color: 'error',
    confirmText: '删除',
  })
  if (yes !== true) return
  const ok = await libraryStore.deletePackage(props.sgID)
  if (ok) {
    toast.add({ title: '已删除', color: 'success' })
    emit('cleared')
  } else {
    toast.add({ title: '删除失败', color: 'error' })
  }
}
</script>
