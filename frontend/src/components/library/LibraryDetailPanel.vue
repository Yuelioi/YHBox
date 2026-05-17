<!--
LibraryDetailPanel 库视图右侧详情/编辑面板（UE Content Browser-style inspector）.

- selection 通过 prop 传 unified item: { kind: 'subgraph' | 'template' | 'clip', payload: ... }
- subgraph / template 因后端无 update RPC，目前只读元数据 + 删除按钮（用户原话"通用编辑能力"在 v1
  仅 clip 完整支持，subgraph/template 受限说明放在 footer）
- clip 全字段编辑 → clips.update(id, { label, description, tags })
- 风格对齐 ContainerPropsPanel / SubgraphPropsPanel：semantic colors, UInput/UTextarea/UInputMenu
-->
<template>
  <aside class="w-96 shrink-0 border-l border-default overflow-y-auto bg-default">
    <!-- 未选中 -->
    <div
      v-if="!selection"
      class="h-full flex flex-col items-center justify-center text-center px-6 py-10"
    >
      <UIcon name="i-tabler-pointer" class="size-10 text-dimmed mb-3" />
      <p class="text-sm text-toned">未选择</p>
      <p class="text-[11px] text-dimmed mt-1">单击卡片预览，双击进入</p>
    </div>

    <!-- subgraph -->
    <div v-else-if="selection.kind === 'subgraph'" class="p-4 space-y-4">
      <header class="flex items-start gap-3 pb-3 border-b border-default">
        <div
          class="size-10 rounded-lg flex items-center justify-center shrink-0 bg-fuchsia-500/15 border border-fuchsia-500/40"
        >
          <UIcon name="i-tabler-package" class="size-5 text-fuchsia-300" />
        </div>
        <div class="min-w-0 flex-1">
          <h3 class="text-sm font-medium text-highlighted truncate leading-tight">
            {{ subgraph.label || subgraph.id }}
          </h3>
          <p class="text-[11px] text-dimmed mt-0.5">
            {{ subgraph.graph?.nodes?.length ?? 0 }} 节点 ·
            {{ subgraph.outputPins?.length ?? 0 }} 出口
          </p>
        </div>
      </header>

      <section class="space-y-1.5">
        <label class="block text-xs text-toned">名称</label>
        <UInput
          :model-value="subgraph.label"
          size="sm"
          class="w-full"
          disabled
          :ui="{ base: 'opacity-70' }"
        />
        <p class="text-[10px] text-dimmed">库 subgraph 元数据当前只读；要改名先导入到容器编辑器</p>
      </section>

      <section v-if="subgraph.description" class="space-y-1.5">
        <label class="block text-xs text-toned">描述</label>
        <p class="text-xs text-default whitespace-pre-line">{{ subgraph.description }}</p>
      </section>

      <section v-if="(subgraph.tags ?? []).length > 0" class="space-y-1.5">
        <label class="block text-xs text-toned">标签</label>
        <div class="flex flex-wrap gap-1">
          <UBadge v-for="t in subgraph.tags ?? []" :key="t" size="xs" variant="subtle">
            {{ t }}
          </UBadge>
        </div>
      </section>

      <section v-if="(subgraph.requiredTemplates ?? []).length > 0" class="space-y-1.5">
        <label class="block text-xs text-toned">夹带模板</label>
        <div class="flex flex-wrap gap-1">
          <UBadge
            v-for="t in subgraph.requiredTemplates ?? []"
            :key="t.sha256"
            size="xs"
            variant="outline"
            color="neutral"
          >
            <UIcon name="i-tabler-photo" class="size-3 mr-1" />
            {{ t.name }}
          </UBadge>
        </div>
      </section>

      <section v-if="subgraph.recordingContext" class="space-y-1 rounded-md bg-elevated/30 border border-default/40 p-3">
        <div class="flex items-center gap-1.5 mb-1">
          <UIcon name="i-tabler-clipboard-data" class="size-3.5 text-toned" />
          <span class="text-xs text-toned font-medium">录制元数据</span>
        </div>
        <p class="text-[11px] text-dimmed">
          {{ subgraph.recordingContext.mouseCounts360 }} counts/360° ·
          {{ subgraph.recordingContext.resolution?.[0] ?? 0 }}×{{ subgraph.recordingContext.resolution?.[1] ?? 0 }}
        </p>
        <p class="text-[10px] text-dimmed">{{ subgraph.recordingContext.recordedAt || '—' }}</p>
      </section>

      <section class="space-y-1.5">
        <label class="block text-[10px] uppercase tracking-[0.08em] font-semibold text-dimmed">ID</label>
        <button
          type="button"
          class="w-full text-left text-[11px] text-dimmed font-mono bg-elevated/40 rounded px-2 py-1 hover:bg-elevated/60 transition-colors truncate"
          :title="'点击复制 — ' + subgraph.id"
          @click="onCopyID(subgraph.id)"
        >
          {{ subgraph.id }}
        </button>
      </section>

      <div class="pt-3 border-t border-default flex flex-col gap-2">
        <UButton
          size="sm"
          variant="soft"
          color="primary"
          icon="i-tabler-arrow-bar-to-down"
          :disabled="!editorStore.activeContainerID"
          @click="$emit('import-subgraph', subgraph.id)"
        >
          导入到当前容器
        </UButton>
        <UButton size="sm" variant="soft" color="error" icon="i-tabler-trash" @click="onDeleteSubgraph">
          删除
        </UButton>
      </div>
    </div>

    <!-- template -->
    <div v-else-if="selection.kind === 'template'" class="p-4 space-y-4">
      <header class="flex items-start gap-3 pb-3 border-b border-default">
        <div
          class="size-10 rounded-lg flex items-center justify-center shrink-0 bg-violet-500/15 border border-violet-500/40"
        >
          <UIcon name="i-tabler-photo" class="size-5 text-violet-300" />
        </div>
        <div class="min-w-0 flex-1">
          <h3 class="text-sm font-medium text-highlighted truncate leading-tight">
            {{ tpl.meta?.name || tpl.key }}
          </h3>
          <p class="text-[11px] text-dimmed mt-0.5">
            {{ tpl.meta?.width ?? 0 }}×{{ tpl.meta?.height ?? 0 }} ·
            {{ tpl.meta?.recordedResolution?.[0] ?? 0 }}×{{ tpl.meta?.recordedResolution?.[1] ?? 0 }}
          </p>
        </div>
      </header>

      <section v-if="tpl.meta?.description" class="space-y-1.5">
        <label class="block text-xs text-toned">描述</label>
        <p class="text-xs text-default whitespace-pre-line">{{ tpl.meta.description }}</p>
      </section>

      <section v-if="(tpl.meta?.tags ?? []).length > 0" class="space-y-1.5">
        <label class="block text-xs text-toned">标签</label>
        <div class="flex flex-wrap gap-1">
          <UBadge v-for="t in tpl.meta?.tags ?? []" :key="t" size="xs" variant="subtle">
            {{ t }}
          </UBadge>
        </div>
      </section>

      <section v-if="tpl.meta?.origin" class="space-y-1.5">
        <label class="block text-xs text-toned">来源</label>
        <UBadge size="xs" variant="outline" color="neutral">
          {{ tpl.meta.origin.kind }}{{ tpl.meta.origin.sourceID ? ' · ' + tpl.meta.origin.sourceID.slice(0, 8) : '' }}
        </UBadge>
      </section>

      <section class="space-y-1.5">
        <label class="block text-[10px] uppercase tracking-[0.08em] font-semibold text-dimmed">Key</label>
        <button
          type="button"
          class="w-full text-left text-[11px] text-dimmed font-mono bg-elevated/40 rounded px-2 py-1 hover:bg-elevated/60 transition-colors truncate"
          :title="'点击复制 — ' + tpl.key"
          @click="onCopyID(tpl.key)"
        >
          {{ tpl.key }}
        </button>
      </section>

      <p class="text-[10px] text-dimmed leading-relaxed">
        库模板编辑（改名 / 描述 / 标签）当前未在库视图开放；从容器编辑器使用此模板时可在 Inspector 改 meta。
      </p>

      <div class="pt-3 border-t border-default flex flex-col gap-2">
        <UButton size="sm" variant="soft" color="error" icon="i-tabler-trash" @click="onDeleteTemplate">
          删除
        </UButton>
      </div>
    </div>

    <!-- clip -->
    <div v-else-if="selection.kind === 'clip'" class="p-4 space-y-4">
      <header class="flex items-start gap-3 pb-3 border-b border-default">
        <div
          class="size-10 rounded-lg flex items-center justify-center shrink-0 bg-emerald-500/15 border border-emerald-500/40"
        >
          <UIcon name="i-tabler-vinyl" class="size-5 text-emerald-300" />
        </div>
        <div class="min-w-0 flex-1">
          <h3 class="text-sm font-medium text-highlighted truncate leading-tight">
            {{ clipDraft.label || clip.id }}
          </h3>
          <p class="text-[11px] text-dimmed mt-0.5">
            {{ clipDurationLabel }} · {{ clip.eventCount }} ev · {{ clipCreatedLabel }}
          </p>
        </div>
      </header>

      <section class="space-y-1.5">
        <label class="block text-xs text-toned">名称</label>
        <UInput
          v-model="clipDraft.label"
          size="sm"
          class="w-full"
          placeholder="给片段起个名字"
          @blur="commitClip"
          @keydown.enter="commitClip"
        />
      </section>

      <section class="space-y-1.5">
        <label class="block text-xs text-toned">描述</label>
        <UTextarea
          v-model="clipDraft.description"
          :rows="3"
          size="sm"
          class="w-full"
          placeholder="可选 · 这段录的是什么"
          @blur="commitClip"
        />
      </section>

      <section class="space-y-1.5">
        <label class="block text-xs text-toned">标签</label>
        <UInputMenu
          v-model="clipDraft.tags"
          multiple
          creatable
          :items="allClipTags"
          size="sm"
          class="w-full"
          placeholder="添加标签..."
          @update:model-value="commitClip"
        />
      </section>

      <section v-if="clip.meta" class="space-y-1 rounded-md bg-elevated/30 border border-default/40 p-3">
        <div class="flex items-center gap-1.5 mb-1">
          <UIcon name="i-tabler-clipboard-data" class="size-3.5 text-toned" />
          <span class="text-xs text-toned font-medium">录制元数据</span>
        </div>
        <p class="text-[11px] text-dimmed">
          模式：{{ clip.meta.mouseMode }} ·
          {{ clip.meta.baseResolution?.[0] ?? 0 }}×{{ clip.meta.baseResolution?.[1] ?? 0 }} ·
          {{ clip.meta.windowMode }}
        </p>
        <p class="text-[10px] text-dimmed">
          counts/360° = {{ clip.meta.mouseCounts360 }} · filter = {{ clip.meta.filterMode }}
        </p>
      </section>

      <section class="space-y-1.5">
        <label class="block text-[10px] uppercase tracking-[0.08em] font-semibold text-dimmed">ID</label>
        <button
          type="button"
          class="w-full text-left text-[11px] text-dimmed font-mono bg-elevated/40 rounded px-2 py-1 hover:bg-elevated/60 transition-colors truncate"
          :title="'点击复制 — ' + clip.id"
          @click="onCopyID(clip.id)"
        >
          {{ clip.id }}
        </button>
      </section>

      <div class="pt-3 border-t border-default flex flex-col gap-2">
        <UButton size="sm" variant="soft" color="error" icon="i-tabler-trash" @click="onDeleteClip">
          删除
        </UButton>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { LibrarySubgraph, LibraryTemplate } from '@/lib/backend'
import { useClipsStore, type ClipSummary } from '@/stores/clips'
import { useLibraryStore } from '@/stores/library'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { backend } from '@/lib/backend'
import { useConfirm } from '@/composables/useConfirm'
import { useToast } from '@nuxt/ui/composables'
import { fmtMillis } from '@/lib/format'

export type LibrarySelection =
  | { kind: 'subgraph'; payload: LibrarySubgraph }
  | { kind: 'template'; payload: LibraryTemplate }
  | { kind: 'clip'; payload: ClipSummary }

const props = defineProps<{ selection: LibrarySelection | null }>()

const emit = defineEmits<{
  'import-subgraph': [id: string]
  'cleared': []
}>()

const clipsStore = useClipsStore()
const libraryStore = useLibraryStore()
const editorStore = useContainerEditorStore()
const { confirm } = useConfirm()
const toast = useToast()

// 联合 narrow helpers — 让 template 段不需要重复 `as any`
const subgraph = computed<LibrarySubgraph>(() =>
  props.selection?.kind === 'subgraph' ? props.selection.payload : ({} as LibrarySubgraph),
)
const tpl = computed<LibraryTemplate>(() =>
  props.selection?.kind === 'template' ? props.selection.payload : ({} as LibraryTemplate),
)
const clip = computed<ClipSummary>(() =>
  props.selection?.kind === 'clip' ? props.selection.payload : ({} as ClipSummary),
)

// ─── clip 草稿 — 跟随 selection 变化重置, blur/enter/tag-change 时提交 ───
const clipDraft = ref({
  label: '',
  description: '',
  tags: [] as string[],
})

watch(
  () => (props.selection?.kind === 'clip' ? props.selection.payload : null),
  (c) => {
    if (!c) return
    clipDraft.value = {
      label: c.label ?? '',
      description: c.description ?? '',
      tags: [...(c.tags ?? [])],
    }
  },
  { immediate: true },
)

const allClipTags = computed(() => {
  const set = new Set<string>()
  for (const c of clipsStore.clips) for (const t of c.tags ?? []) set.add(t)
  return [...set]
})

const clipDurationLabel = computed(() =>
  fmtMillis(Math.floor((clip.value.durationUs ?? 0) / 1000)),
)
const clipCreatedLabel = computed(() => {
  const iso = clip.value.createdAt
  if (!iso) return '—'
  const d = new Date(iso)
  if (isNaN(d.getTime())) return '—'
  const M = String(d.getMonth() + 1).padStart(2, '0')
  const D = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const mm = String(d.getMinutes()).padStart(2, '0')
  return `${M}-${D} ${hh}:${mm}`
})

async function commitClip() {
  if (props.selection?.kind !== 'clip') return
  const c = props.selection.payload
  const nextLabel = clipDraft.value.label.trim()
  const nextDesc = clipDraft.value.description
  const nextTags = clipDraft.value.tags
  // 跟原值一样就不重写盘 — 减少不必要的 emit + 列表 reload
  if (
    nextLabel === (c.label ?? '') &&
    nextDesc === (c.description ?? '') &&
    JSON.stringify(nextTags) === JSON.stringify(c.tags ?? [])
  ) {
    return
  }
  try {
    await clipsStore.update(c.id, {
      label: nextLabel || c.label,
      description: nextDesc,
      tags: nextTags,
    })
  } catch (e: any) {
    toast.add({ title: '保存失败', description: String(e?.message ?? e), color: 'error' })
  }
}

async function onCopyID(id: string) {
  try {
    await navigator.clipboard.writeText(id)
    toast.add({ title: '已复制 ID', color: 'success', icon: 'i-tabler-check' })
  } catch (e: any) {
    toast.add({ title: '复制失败', description: String(e?.message ?? e), color: 'error' })
  }
}

async function onDeleteSubgraph() {
  if (props.selection?.kind !== 'subgraph') return
  const sg = props.selection.payload
  const yes = await confirm({
    title: '删除库子图',
    description: `确认删除子图 "${sg.label || sg.id}"？此操作不可恢复。`,
    color: 'error',
    confirmText: '删除',
  })
  if (yes !== true) return
  try {
    await backend.library.deleteSubgraph(sg.id)
    toast.add({ title: '已删除', color: 'success' })
    await libraryStore.reload()
    emit('cleared')
  } catch (e: any) {
    toast.add({ title: '删除失败', description: String(e?.message ?? e), color: 'error' })
  }
}

async function onDeleteTemplate() {
  if (props.selection?.kind !== 'template') return
  const t = props.selection.payload
  const yes = await confirm({
    title: '删除库模板',
    description: `确认删除模板 "${t.meta?.name || t.key}"？此操作不可恢复。`,
    color: 'error',
    confirmText: '删除',
  })
  if (yes !== true) return
  try {
    await backend.library.deleteTemplateMany([t.key])
    toast.add({ title: '已删除', color: 'success' })
    await libraryStore.reload()
    emit('cleared')
  } catch (e: any) {
    toast.add({ title: '删除失败', description: String(e?.message ?? e), color: 'error' })
  }
}

async function onDeleteClip() {
  if (props.selection?.kind !== 'clip') return
  const c = props.selection.payload
  const yes = await confirm({
    title: '删除录制片段',
    description: `确认删除片段 "${c.label || c.id}"？此操作不可恢复。`,
    color: 'error',
    confirmText: '删除',
  })
  if (yes !== true) return
  try {
    await clipsStore.remove(c.id)
    toast.add({ title: '已删除', color: 'success' })
    emit('cleared')
  } catch (e: any) {
    toast.add({ title: '删除失败', description: String(e?.message ?? e), color: 'error' })
  }
}
</script>
