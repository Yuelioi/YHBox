<!-- clip 库右栏详情 (就地编辑): 名称/描述双击改, 分类/标签即改即存. clip 是全局资产、无 rev.
     clip update 全覆盖 → 字段级 patch 缺省值用当前值补全. 无可视缩略图 (输入事件录制).
     插入引用 = 主 CTA (emit insert → 父插裸 PlayClip 节点); 删除弱化到底部. -->
<template>
  <div class="w-full overflow-y-auto">
    <div v-if="!clip" class="flex flex-col items-center justify-center text-center px-6 py-10">
      <UIcon name="i-tabler-pointer" class="size-10 text-dimmed mb-3" />
      <p class="text-sm text-toned">{{ t('clip.detail.empty') }}</p>
    </div>

    <div v-else class="p-4 space-y-4">
      <header class="flex items-start gap-3">
        <div
          class="size-10 rounded-lg flex items-center justify-center shrink-0 bg-primary/15 border border-primary/40"
        >
          <UIcon name="i-tabler-movie" class="size-5 text-primary" />
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
            <span class="truncate min-w-0">{{ clip.label || t('clip.manager.untitled') }}</span>
            <UIcon
              name="i-tabler-pencil"
              class="size-3 shrink-0 text-dimmed opacity-0 group-hover:opacity-100"
            />
          </h3>
          <p class="text-[11px] text-dimmed mt-0.5">
            {{ formatDuration(clip.durationUs) }} · {{ clip.eventCount }}
            {{ t('clip.manager.events') }}
          </p>
        </div>
      </header>

      <UButton color="primary" icon="i-tabler-package-import" block @click="emit('insert')">
        {{ t('library.explorer.insert') }}
      </UButton>

      <!-- 描述 -->
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
          v-else-if="clip.description"
          class="text-xs text-default whitespace-pre-line cursor-text"
          :title="t('library.detail.dblclick_edit')"
          @dblclick="enterEditDesc"
        >
          {{ clip.description }}
        </p>
        <p v-else class="text-xs text-dimmed italic cursor-text" @dblclick="enterEditDesc">
          {{ t('library.detail.desc_empty') }}
        </p>
      </section>

      <!-- 分类 -->
      <section class="space-y-1.5">
        <label class="block text-xs text-toned">{{ t('common.category') }}</label>
        <UInputMenu
          :model-value="clip.category ?? ''"
          :create-item="'always'"
          :items="allCategories"
          size="sm"
          :placeholder="t('library.explorer.category_placeholder')"
          @update:model-value="(v: string) => patch({ category: v ?? '' })"
          @create="(v: string) => patch({ category: v })"
        />
      </section>

      <!-- 标签 -->
      <section class="space-y-1.5">
        <label class="block text-xs text-toned">{{ t('library.detail.tags') }}</label>
        <UInputMenu
          :model-value="clip.tags ?? []"
          multiple
          :create-item="'always'"
          :items="allTags"
          size="sm"
          @update:model-value="(v: string[]) => patch({ tags: v })"
          @create="(v: string) => patch({ tags: [...(clip?.tags ?? []), v] })"
        />
      </section>

      <!-- 元信息 -->
      <section v-if="clip.createdAt" class="space-y-1 text-[11px] text-dimmed">
        <div class="flex justify-between">
          <span>{{ t('library.detail.created_at') }}</span>
          <span>{{ new Date(clip.createdAt).toLocaleString() }}</span>
        </div>
      </section>

      <!-- ID -->
      <section class="space-y-1.5">
        <label class="block text-[10px] uppercase tracking-[0.08em] font-semibold text-dimmed"
          >ID</label
        >
        <button
          type="button"
          class="w-full text-left text-[11px] font-mono bg-elevated/40 rounded px-2 py-1 hover:bg-elevated/60 transition-colors truncate flex items-center gap-1.5"
          :class="copied ? 'text-success' : 'text-dimmed'"
          @click="onCopyID"
        >
          <UIcon v-if="copied" name="i-tabler-check" class="size-3 shrink-0" />
          <span class="truncate">{{ copied ? t('common.copied') : clip.id }}</span>
        </button>
      </section>

      <div class="pt-3 border-t border-default flex">
        <UButton
          size="xs"
          variant="soft"
          color="error"
          icon="i-tabler-trash"
          class="ml-auto"
          @click="onDelete"
        >
          {{ t('common.delete') }}
        </UButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend } from '@/lib/backend'
import { useClipsStore, type ClipSummary } from '@/stores/clips'
import { useConfirm } from '@/composables/useConfirm'
import { useToast } from '@nuxt/ui/composables'
import { errorMessage } from '@/lib/invoke'

const { t } = useI18n()
const props = defineProps<{ clipId: string | null }>()
const emit = defineEmits<{ insert: [] }>()
const store = useClipsStore()
const { confirm } = useConfirm()
const toast = useToast()

const clip = computed<ClipSummary | undefined>(() =>
  props.clipId ? store.clips.find((c) => c.id === props.clipId) : undefined,
)

const allCategories = computed(() => {
  const set = new Set<string>()
  for (const c of store.clips) if (c.category) set.add(c.category)
  return [...set].sort()
})
const allTags = computed(() => {
  const set = new Set<string>()
  for (const c of store.clips) for (const tg of c.tags ?? []) set.add(tg)
  return [...set].sort()
})

function formatDuration(us: number): string {
  const ms = us / 1000
  if (ms < 1000) return `${Math.round(ms)} ms`
  return `${(ms / 1000).toFixed(1)} s`
}

// 字段级保存 — clip update 全覆盖; 缺的字段用当前值补全, 改完 store 自 refresh.
async function patch(p: {
  label?: string
  description?: string
  category?: string
  tags?: string[]
}) {
  const c = clip.value
  if (!c) return
  await store.update(c.id, {
    label: p.label ?? c.label,
    description: p.description ?? c.description ?? '',
    category: p.category ?? c.category ?? '',
    tags: p.tags ?? c.tags ?? [],
  })
}

// 名称双击编辑
const editingName = ref(false)
const draftName = ref('')
const nameInputRef = ref<any>(null)
async function enterEditName() {
  if (!clip.value) return
  draftName.value = clip.value.label ?? ''
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
  if (!next || next === clip.value?.label) return
  void patch({ label: next })
}

// 描述双击编辑
const editingDesc = ref(false)
const draftDesc = ref('')
const descInputRef = ref<any>(null)
async function enterEditDesc() {
  if (!clip.value) return
  draftDesc.value = clip.value.description ?? ''
  editingDesc.value = true
  await nextTick()
  const el: HTMLTextAreaElement | undefined = descInputRef.value?.textareaRef
  el?.focus()
}
function saveDesc() {
  if (!editingDesc.value) return
  editingDesc.value = false
  const next = draftDesc.value.trim()
  if (next === (clip.value?.description ?? '')) return
  void patch({ description: next })
}

watch(
  () => props.clipId,
  () => {
    editingName.value = false
    editingDesc.value = false
  },
)

const copied = ref(false)
let copiedTimer = 0
async function onCopyID() {
  if (!props.clipId) return
  try {
    await navigator.clipboard.writeText(props.clipId)
    copied.value = true
    window.clearTimeout(copiedTimer)
    copiedTimer = window.setTimeout(() => {
      copied.value = false
    }, 1500)
  } catch (e: any) {
    toast.add({ title: t('toast.copy_failed'), description: errorMessage(e), color: 'error' })
  }
}

async function onDelete() {
  const c = clip.value
  if (!c) return
  const refs = await backend.assets.referrers(c.id)
  const n = refs?.length ?? 0
  const description =
    n > 0
      ? t('clip.manager.delete_confirm_referenced', { name: c.label, n })
      : t('clip.manager.delete_confirm', { name: c.label })
  const yes = await confirm({
    title: t('clip.manager.delete_title'),
    description,
    color: 'error',
    confirmText: t('common.delete'),
  })
  if (yes !== true) return
  await store.remove(c.id)
}
</script>
