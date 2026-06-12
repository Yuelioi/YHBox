<!-- 子图库右栏详情 (就地编辑): 名称/描述双击改, 标签/分类行内即改即存;
     插入引用 = 唯一主 CTA; 复制为新/删除弱化到底部一行。 -->
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
      <header class="flex items-start gap-3">
        <div class="size-10 rounded-lg flex items-center justify-center shrink-0 bg-fuchsia-500/15 border border-fuchsia-500/40">
          <UIcon name="i-tabler-subtask" class="size-5 text-fuchsia-300" />
        </div>
        <div class="min-w-0 flex-1">
          <UInput
            v-if="editingName"
            ref="nameInputRef"
            v-model="draftName"
            size="sm"
            @keyup.enter="saveName"
            @keydown.esc.stop="editingName = false"
            @blur="saveName"
          />
          <h3
            v-else
            class="group flex items-center gap-1 text-sm font-medium text-highlighted leading-tight cursor-text"
            :title="t('library.detail.dblclick_edit')"
            @dblclick="enterEditName"
          >
            <span class="truncate min-w-0">{{ sg.label || sg.id }}</span>
            <UIcon name="i-tabler-pencil" class="size-3 shrink-0 text-dimmed opacity-0 group-hover:opacity-100" />
          </h3>
          <p class="text-[11px] text-dimmed mt-0.5">
            {{ t('library.detail.nodes_and_outputs', { n: sg.graph?.nodes?.length ?? 0, m: sg.outputPins?.length ?? 0 }) }}
          </p>
        </div>
      </header>

      <UButton color="primary" icon="i-tabler-package-import" block @click="emit('insert')">
        {{ t('library.explorer.insert') }}
      </UButton>

      <section class="space-y-1.5">
        <label class="block text-xs text-toned">{{ t('library.detail.description') }}</label>
        <UTextarea
          v-if="editingDesc"
          ref="descInputRef"
          v-model="draftDesc"
          :rows="3"
          size="sm"
          @keydown.esc.stop="editingDesc = false"
          @blur="saveDesc"
        />
        <p
          v-else-if="sg.description"
          class="text-xs text-default whitespace-pre-line cursor-text"
          :title="t('library.detail.dblclick_edit')"
          @dblclick="enterEditDesc"
        >
          {{ sg.description }}
        </p>
        <p v-else class="text-xs text-dimmed italic cursor-text" @dblclick="enterEditDesc">
          {{ t('library.detail.desc_empty') }}
        </p>
      </section>

      <section class="space-y-1.5">
        <label class="block text-xs text-toned">{{ t('common.category') }}</label>
        <UInputMenu
          :model-value="sg.category ?? ''"
          creatable
          :items="allCategories"
          size="sm"
          :placeholder="t('library.explorer.category_placeholder')"
          @update:model-value="(v: string) => patchField({ category: v ?? '' })"
        />
      </section>

      <section class="space-y-1.5">
        <label class="block text-xs text-toned">{{ t('library.detail.tags') }}</label>
        <UInputMenu
          :model-value="sg.tags ?? []"
          multiple
          creatable
          :items="allTags"
          size="sm"
          @update:model-value="(v: string[]) => patchField({ tags: v })"
        />
      </section>

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

      <div class="pt-3 border-t border-default flex items-center gap-2">
        <UButton size="xs" variant="soft" color="neutral" icon="i-tabler-copy-plus" @click="onDuplicate">
          {{ t('library.card.duplicate') }}
        </UButton>
        <UButton size="xs" variant="soft" color="error" icon="i-tabler-trash" class="ml-auto" @click="onDelete">
          {{ t('library.detail.delete') }}
        </UButton>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend, type Subgraph } from '@/lib/backend'
import { useLibraryStore } from '@/stores/library'
import { useConfirm } from '@/composables/useConfirm'
import { useToast } from '@nuxt/ui/composables'
import { errorMessage, toastError } from '@/lib/invoke'

const { t } = useI18n()

const props = defineProps<{ sgID: string | null }>()
const emit = defineEmits<{ insert: [] }>()

const libraryStore = useLibraryStore()
const { confirm } = useConfirm()
const toast = useToast()

const sg = computed<Subgraph | undefined>(() => (props.sgID ? libraryStore.byId(props.sgID) : undefined))

const allCategories = computed(() => {
  const set = new Set<string>()
  for (const s of libraryStore.subgraphs) if (s.category) set.add(s.category)
  return [...set].sort()
})
const allTags = computed(() => {
  const set = new Set<string>()
  for (const s of libraryStore.subgraphs) for (const tg of s.tags ?? []) set.add(tg)
  return [...set].sort()
})

// ── 字段级保存 (merge patch + rev 乐观锁); 失败 toast, 成败都 reload 对齐磁盘 ──
async function patchField(patch: Record<string, unknown>) {
  if (!sg.value) return
  try {
    await backend.subgraphs.updateSilent(sg.value.id, JSON.stringify(patch), sg.value.rev)
  } catch (e) {
    toastError(errorMessage(e))
  }
  await libraryStore.reload()
}

// ── 名称双击编辑 (CommentBoxNode 习语: editing + draft + nextTick focus) ──
const editingName = ref(false)
const draftName = ref('')
const nameInputRef = ref<any>(null)

async function enterEditName() {
  if (!sg.value) return
  draftName.value = sg.value.label ?? ''
  editingName.value = true
  await nextTick()
  const el: HTMLInputElement | undefined = nameInputRef.value?.inputRef
  el?.focus()
  el?.select()
}

function saveName() {
  if (!editingName.value) return
  editingName.value = false
  const next = draftName.value.trim()
  if (!next || next === sg.value?.label) return
  void patchField({ label: next })
}

// ── 描述双击编辑 ──
const editingDesc = ref(false)
const draftDesc = ref('')
const descInputRef = ref<any>(null)

async function enterEditDesc() {
  if (!sg.value) return
  draftDesc.value = sg.value.description ?? ''
  editingDesc.value = true
  await nextTick()
  const el: HTMLTextAreaElement | undefined = descInputRef.value?.textareaRef
  el?.focus()
}

function saveDesc() {
  if (!editingDesc.value) return
  editingDesc.value = false
  const next = draftDesc.value.trim()
  if (next === (sg.value?.description ?? '')) return
  void patchField({ description: next })
}

// 切换选中项时退出编辑态, 防 draft 串台
watch(() => props.sgID, () => {
  editingName.value = false
  editingDesc.value = false
})

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
  if (!ok) {
    toast.add({ title: t('toast.delete_failed'), color: 'error' })
  }
  // 列表收缩 → 选中/锚点由 useListSelection 剪枝, 面板自动回空态
}
</script>
