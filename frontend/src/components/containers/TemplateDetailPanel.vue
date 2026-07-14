<!-- 模板库右栏详情 (就地编辑 + 变体管理): 名称/描述双击改, 分类/标签即改即存; 变体多分辨率档(重拍/新增/删档,
     当前窗口分辨率感知)吸纳自旧 TemplatePicker. 模板全局资产、无 rev. 变体逻辑由宿主显式传 containerId 定位窗口.
     pick 模式额外: 顶部「用于此节点」开关 (emit toggle-assign). -->
<template>
  <div class="w-full overflow-y-auto">
    <div v-if="!tpl" class="flex flex-col items-center justify-center text-center px-6 py-10">
      <UIcon name="i-tabler-pointer" class="size-10 text-dimmed mb-3" />
      <p class="text-sm text-toned">{{ t('template.detail.empty') }}</p>
    </div>

    <div v-else class="p-4 space-y-4">
      <!-- pick 模式: 用于此节点 开关 -->
      <UButton
        v-if="pickMode"
        block
        :variant="assigned ? 'solid' : 'soft'"
        :color="assigned ? 'primary' : 'neutral'"
        :icon="assigned ? 'i-tabler-circle-check' : 'i-tabler-circle-plus'"
        @click="emit('toggle-assign')"
      >
        {{ assigned ? t('template.picker.selected') : t('template.picker.use') }}
      </UButton>

      <!-- 缩略图 (当前变体档) -->
      <div
        class="relative flex h-40 items-center justify-center overflow-hidden rounded-md border border-default bg-elevated"
      >
        <CappedPreviewImage
          v-if="displayThumb"
          :src="displayThumb"
          :alt="tpl.name"
          :max-upscale="1"
        />
        <UIcon v-else name="i-tabler-photo" class="size-8 text-dimmed" />
        <UButton
          v-if="displayThumb"
          size="xs"
          variant="solid"
          color="neutral"
          icon="i-tabler-maximize"
          class="absolute bottom-2 right-2"
          @click="previewOpen = true"
        >
          {{ t('template.detail.view_large') }}
        </UButton>
      </div>

      <!-- 名称 -->
      <div>
        <UInput
          v-if="editingName"
          ref="nameInputRef"
          v-model="draftName"
          size="sm"
          :aria-label="t('common.name')"
          @keyup.enter="saveName"
          @keydown.esc.stop="editingName = false"
          @blur="saveName"
        />
        <h3 v-else>
          <button
            type="button"
            class="group flex w-full items-center gap-1 text-left text-sm font-medium leading-tight text-highlighted"
            :title="t('common.edit')"
            @click="enterEditName"
          >
            <span class="min-w-0 truncate">{{ tpl.name || tpl.guid }}</span>
            <UIcon name="i-tabler-pencil" class="size-3 shrink-0 text-dimmed" />
          </button>
        </h3>
      </div>

      <!-- 分辨率变体是模板的核心操作，前置到名称后，dock 与 workspace 共用同一入口。 -->
      <section class="space-y-1.5">
        <label class="block text-xs text-toned">{{ t('template.picker.variants_label') }}</label>
        <p class="text-xs leading-relaxed">
          <template v-if="resolutionLoading">
            <UIcon name="i-tabler-loader-2" class="mr-1 inline size-3 animate-spin text-dimmed" />
            <span class="text-dimmed">{{ t('common.loading') }}</span>
          </template>
          <template v-else-if="curRes">
            <span class="text-dimmed">{{ t('template.picker.current_window') }}: </span>
            <span class="text-toned">{{ curResLabel }}</span>
            <span v-if="curResHint" class="text-warning/80">
              · {{ t('template.picker.scaled_from', { res: curResHint }) }}</span
            >
          </template>
          <span v-else class="text-dimmed">{{ t('template.picker.window_not_open') }}</span>
        </p>
        <div v-if="recordLoading" class="flex gap-1.5" aria-hidden="true">
          <USkeleton class="h-7 w-24" />
          <USkeleton class="h-7 w-20" />
        </div>
        <div v-else-if="detailRecord?.variants?.length" class="flex flex-wrap gap-1.5">
          <div
            v-for="(v, i) in detailRecord.variants"
            :key="i"
            class="flex items-center rounded-md bg-elevated/50 p-0.5"
          >
            <UButton
              size="xs"
              :variant="activeVariantIdx === i ? 'solid' : 'ghost'"
              :color="activeVariantIdx === i ? 'primary' : 'neutral'"
              @click="activeVariantIdx = i"
            >
              {{ v.resolution[0] }}×{{ v.resolution[1] }}
              <span v-if="isCurResVariant(v)" class="ml-1 text-[10px] opacity-80">{{
                t('template.picker.current_badge')
              }}</span>
            </UButton>
            <UButton
              v-if="(detailRecord?.variants?.length ?? 0) > 1"
              size="xs"
              variant="ghost"
              color="error"
              icon="i-tabler-x"
              class="size-6 p-0"
              :aria-label="
                t('template.picker.del_variant_title', {
                  res: `${v.resolution[0]}×${v.resolution[1]}`,
                })
              "
              @click="removeVariant(v.resolution)"
            />
          </div>
        </div>
        <UButton
          size="xs"
          variant="soft"
          color="neutral"
          block
          :icon="curRes && !curResExact ? 'i-tabler-plus' : 'i-tabler-refresh'"
          @click="onRecapture"
        >
          {{ recaptureLabel }}
        </UButton>
      </section>

      <!-- 描述 -->
      <section class="space-y-1.5">
        <label class="block text-xs text-toned">{{ t('library.detail.description') }}</label>
        <UTextarea
          v-if="editingDesc"
          ref="descInputRef"
          v-model="draftDesc"
          :rows="3"
          size="sm"
          :aria-label="t('library.detail.description')"
          @keydown.esc.stop="editingDesc = false"
          @blur="saveDesc"
        />
        <button
          v-else-if="tpl.description"
          type="button"
          class="w-full whitespace-pre-line text-left text-xs text-default"
          :title="t('common.edit')"
          @click="enterEditDesc"
        >
          {{ tpl.description }}
        </button>
        <button
          v-else
          type="button"
          class="w-full text-left text-xs italic text-dimmed"
          @click="enterEditDesc"
        >
          {{ t('library.detail.desc_empty') }}
        </button>
      </section>

      <!-- 分类 -->
      <section class="space-y-1.5">
        <label class="block text-xs text-toned">{{ t('common.category') }}</label>
        <UInputMenu
          :model-value="tpl.category ?? ''"
          :create-item="'always'"
          :items="allCategories"
          size="sm"
          :placeholder="t('library.explorer.category_placeholder')"
          :aria-label="t('common.category')"
          @update:model-value="(v: string) => patch({ category: v ?? '' })"
          @create="(v: string) => patch({ category: v })"
        />
      </section>

      <!-- 标签 -->
      <section class="space-y-1.5">
        <label class="block text-xs text-toned">{{ t('library.detail.tags') }}</label>
        <UInputMenu
          :model-value="tpl.tags ?? []"
          multiple
          :create-item="'always'"
          :items="allTags"
          size="sm"
          :aria-label="t('library.detail.tags')"
          @update:model-value="(v: string[]) => patch({ tags: v })"
          @create="(v: string) => patch({ tags: [...(tpl?.tags ?? []), v] })"
        />
      </section>

      <!-- 元信息 -->
      <section v-if="tpl.createdAt" class="space-y-1 text-xs text-dimmed">
        <div class="flex justify-between">
          <span>{{ t('library.detail.created_at') }}</span>
          <span>{{ new Date(tpl.createdAt).toLocaleString() }}</span>
        </div>
      </section>

      <!-- ID -->
      <section class="space-y-1.5">
        <label class="block text-xs font-medium text-muted">ID</label>
        <button
          type="button"
          class="w-full text-left text-[11px] font-mono bg-elevated/40 rounded px-2 py-1 hover:bg-elevated/60 transition-colors truncate flex items-center gap-1.5"
          :class="copied ? 'text-success' : 'text-dimmed'"
          @click="onCopyID"
        >
          <UIcon v-if="copied" name="i-tabler-check" class="size-3 shrink-0" />
          <span class="truncate">{{ copied ? t('common.copied') : tpl.guid }}</span>
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

  <BaseModal
    v-model:open="previewOpen"
    :title="tpl?.name || t('template.capture.preview')"
    icon="i-tabler-photo"
    size="6xl"
  >
    <div class="flex min-h-80 items-center justify-center rounded-md bg-sunken p-4">
      <CappedPreviewImage
        v-if="displayThumb"
        :src="displayThumb"
        class="h-[70vh] w-full"
        :alt="tpl?.name || t('template.capture.preview')"
        :max-upscale="2"
      />
    </div>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend, type AssetSummary, type AssetRecord } from '@/lib/backend'
import { useTemplatesStore } from '@/stores/templates'
import { useConfirm } from '@/composables/useConfirm'
import { useToast } from '@nuxt/ui/composables'
import { errorMessage } from '@/lib/invoke'
import { awaitWailsEvent } from '@/composables/useWailsEvent'
import BaseModal from '@/components/common/BaseModal.vue'
import CappedPreviewImage from '@/components/common/CappedPreviewImage.vue'

const { t } = useI18n()
const props = defineProps<{
  guid: string | null
  containerId: string
  pickMode?: boolean
  assigned?: boolean
}>()
const emit = defineEmits<{ 'toggle-assign': [] }>()
const store = useTemplatesStore()
const { confirm } = useConfirm()
const toast = useToast()
const previewOpen = ref(false)

const tpl = computed<AssetSummary | undefined>(() =>
  props.guid ? store.map[props.guid] : undefined,
)

const allCategories = computed(() => {
  const set = new Set<string>()
  for (const s of Object.values(store.map)) if (s.category) set.add(s.category)
  return [...set].sort()
})
const allTags = computed(() => {
  const set = new Set<string>()
  for (const s of Object.values(store.map)) for (const tg of s.tags ?? []) set.add(tg)
  return [...set].sort()
})

// 字段级保存 — 模板无 rev, updateMeta 全覆盖; 缺的字段用当前值补全, 改完 store 自 reload.
async function patch(p: {
  name?: string
  description?: string
  category?: string
  tags?: string[]
}) {
  const s = tpl.value
  if (!s) return
  await store.updateMeta(
    s.guid,
    p.name ?? s.name,
    p.description ?? s.description ?? '',
    p.category ?? s.category ?? '',
    p.tags ?? s.tags ?? [],
  )
}

// ── 变体管理 (吸纳自 TemplatePicker) ──────────────────────
const detailRecord = ref<AssetRecord | null>(null)
const recordLoading = ref(false)
const resolutionLoading = ref(false)
const variantThumbs = ref<Record<string, string>>({}) // blobSha → dataURL
const activeVariantIdx = ref(0)
const curRes = ref<[number, number] | null>(null)
const curResExact = ref(false)

const activeVariant = computed(() => detailRecord.value?.variants?.[activeVariantIdx.value])
// 代表缩略图: 与网格 TemplateThumb 同源同法 (独立 watch on firstBlobSha),
// 不挂在 guid 那个含 backend.assets.get 的大 watch 上 → 不受变体加载链中断影响, 保证显示.
const baseThumb = ref<string | null>(null)
watch(
  () => tpl.value?.firstBlobSha,
  async (sha) => {
    baseThumb.value = sha ? await store.readBlobDataURL(sha) : null
  },
  { immediate: true },
)
const displayThumb = computed(() => {
  const sha = activeVariant.value?.blob
  return (sha && variantThumbs.value[sha]) || baseThumb.value
})

async function loadVariantThumb(sha: string) {
  if (!sha || variantThumbs.value[sha]) return
  const r = await store.readBlobDataURL(sha)
  if (typeof r === 'string') variantThumbs.value[sha] = r
}

async function refreshCurRes(guid = props.guid) {
  const containerId = props.containerId
  resolutionLoading.value = true
  try {
    const next = (await backend.assets.currentResolution(containerId)) ?? null
    if (props.guid === guid && props.containerId === containerId) curRes.value = next
  } finally {
    if (props.guid === guid && props.containerId === containerId) resolutionLoading.value = false
  }
}
// 用当前分辨率挑"运行时真会用的那档" → 自动切到该档 (挑档算法在后端).
async function applyCurResPick(guid: string) {
  if (!curRes.value) {
    curResExact.value = false
    return
  }
  const p = await backend.assets.pickVariant(guid, curRes.value[0], curRes.value[1])
  if (p) {
    activeVariantIdx.value = p.index
    curResExact.value = p.exact
  } else {
    curResExact.value = false
  }
}
const curResLabel = computed(() => (curRes.value ? `${curRes.value[0]}×${curRes.value[1]}` : ''))
const curResHint = computed(() => {
  if (!curRes.value || curResExact.value) return ''
  const v = activeVariant.value
  return v ? `${v.resolution[0]}×${v.resolution[1]}` : ''
})
const recaptureLabel = computed(() =>
  curRes.value && !curResExact.value
    ? t('template.picker.add_variant', { res: curResLabel.value })
    : t('template.picker.recapture'),
)
function isCurResVariant(v: { resolution: number[] }): boolean {
  return (
    !!curRes.value && v.resolution[0] === curRes.value[0] && v.resolution[1] === curRes.value[1]
  )
}

// 完整 record 独立加载，不能被最长 3 秒的窗口解析拖住；已有变体应立即可见。
watch(
  () => props.guid,
  async (guid) => {
    detailRecord.value = null
    recordLoading.value = !!guid
    variantThumbs.value = {}
    activeVariantIdx.value = 0
    previewOpen.value = false
    editingName.value = false
    editingDesc.value = false
    if (!guid) return
    try {
      const rec = await backend.assets.get(guid)
      if (props.guid !== guid) return
      if (rec) {
        detailRecord.value = rec
        for (const v of rec.variants ?? []) void loadVariantThumb(v.blob)
      }
    } finally {
      if (props.guid === guid) recordLoading.value = false
    }
  },
  { immediate: true },
)

// 当前窗口分辨率有独立状态：pending 不等于窗口未开；容器上下文变化也必须重新检测。
watch(
  [() => props.guid, () => props.containerId],
  async ([guid]) => {
    curRes.value = null
    curResExact.value = false
    if (!guid) {
      resolutionLoading.value = false
      return
    }
    await refreshCurRes(guid)
    if (props.guid === guid) await applyCurResPick(guid)
  },
  { immediate: true },
)

async function removeVariant(res: number[]) {
  const guid = props.guid
  if (!guid || (detailRecord.value?.variants?.length ?? 0) <= 1) return
  const yes = await confirm({
    title: t('template.picker.del_variant_title', { res: `${res[0]}×${res[1]}` }),
    description: t('template.picker.del_variant_desc'),
    confirmText: t('common.delete'),
    color: 'error',
  })
  if (yes !== true) return
  const r = await backend.assets.removeVariant(guid, res[0], res[1])
  if (r === undefined) return
  variantThumbs.value = {}
  await store.reload()
  const rec = await backend.assets.get(guid)
  if (rec) {
    detailRecord.value = rec
    for (const v of rec.variants ?? []) void loadVariantThumb(v.blob)
  }
  await applyCurResPick(guid)
}

async function onRecapture() {
  const guid = props.guid
  if (!guid) return
  const id = 'recap-' + Math.random().toString(36).slice(2, 10) + '-' + Date.now()
  const waiter = awaitWailsEvent<{ id: string; mode: string; payload: any }>(
    'tools:picker-result',
    (p) => p?.id === id,
  )
  await backend.tools.openScreenPicker('template_recapture', id, props.containerId, '', '', guid)
  const result = await waiter
  if (result.payload?.cancelled || !result.payload?.guid) return
  variantThumbs.value = {}
  await store.reload()
  const rec = await backend.assets.get(guid)
  if (rec) {
    detailRecord.value = rec
    for (const v of rec.variants ?? []) void loadVariantThumb(v.blob)
  }
  await refreshCurRes()
  await applyCurResPick(guid)
}

// 名称双击编辑
const editingName = ref(false)
const draftName = ref('')
const nameInputRef = ref<any>(null)
async function enterEditName() {
  if (!tpl.value) return
  draftName.value = tpl.value.name ?? ''
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
  if (!next || next === tpl.value?.name) return
  void patch({ name: next })
}

// 描述双击编辑
const editingDesc = ref(false)
const draftDesc = ref('')
const descInputRef = ref<any>(null)
async function enterEditDesc() {
  if (!tpl.value) return
  draftDesc.value = tpl.value.description ?? ''
  editingDesc.value = true
  await nextTick()
  const el: HTMLTextAreaElement | undefined = descInputRef.value?.textareaRef
  el?.focus()
}
function saveDesc() {
  if (!editingDesc.value) return
  editingDesc.value = false
  const next = draftDesc.value.trim()
  if (next === (tpl.value?.description ?? '')) return
  void patch({ description: next })
}

const copied = ref(false)
let copiedTimer = 0
async function onCopyID() {
  if (!props.guid) return
  try {
    await navigator.clipboard.writeText(props.guid)
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
  const s = tpl.value
  if (!s) return
  const refs = await backend.assets.referrers(s.guid)
  const n = refs?.length ?? 0
  const description =
    n > 0
      ? t('template.manager.delete_confirm_referenced', { key: s.name, n })
      : t('template.manager.delete_confirm', { key: s.name })
  const yes = await confirm({
    title: t('template.manager.delete_title'),
    description,
    color: 'error',
    confirmText: t('common.delete'),
  })
  if (yes !== true) return
  await store.remove(s.guid)
}
</script>
