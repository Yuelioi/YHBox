<template>
  <div class="space-y-1.5">
    <!-- 触发按钮：显示已选模板数 + 首个缩略图，点击打开资产浏览 modal -->
    <UButton
      variant="outline"
      color="neutral"
      size="sm"
      class="w-full justify-start font-normal"
      :title="selectedNames.join(', ') || t('template.picker.not_selected')"
      @click="open = true"
    >
      <div class="flex items-center gap-2 min-w-0 flex-1">
        <img
          v-if="firstThumb"
          :src="firstThumb"
          class="size-6 rounded object-contain bg-elevated shrink-0"
          alt=""
        />
        <UIcon
          v-else
          :name="selected.length ? 'i-tabler-photo' : 'i-tabler-photo-plus'"
          class="size-4 shrink-0"
          :class="selected.length ? 'text-dimmed' : 'text-warning'"
        />
        <div class="min-w-0 flex-1 text-left">
          <div class="text-xs text-highlighted truncate">{{ summaryLabel }}</div>
        </div>
        <UIcon name="i-tabler-chevron-down" class="size-3.5 text-dimmed shrink-0" />
      </div>
    </UButton>

    <!-- 资产浏览 modal：网格 ↔ 详情 钻入式 -->
    <UModal
      :open="open"
      :ui="{ content: 'sm:max-w-[900px]' }"
      @update:open="onModalToggle"
    >
      <template #content>
        <!-- ========== 网格页 ========== -->
        <div v-if="view === 'grid'" class="bg-default flex flex-col" style="height: 76vh">
          <header class="flex items-center gap-2 px-4 py-3 border-b border-default shrink-0">
            <UIcon name="i-tabler-photo" class="size-4 text-primary shrink-0" />
            <h3 class="text-sm font-medium shrink-0">{{ t('template.picker.browser_title') }}</h3>
            <UInput
              v-model="search"
              autofocus
              size="sm"
              class="flex-1 min-w-32"
              icon="i-tabler-search"
              :placeholder="t('template.manager.search')"
            />
            <UButton size="xs" variant="soft" color="primary" icon="i-tabler-camera" @click="onCaptureNew">
              {{ t('template.picker.capture_new') }}
            </UButton>
            <UButton size="xs" variant="ghost" color="neutral" icon="i-tabler-x" @click="open = false" />
          </header>

          <!-- 标签筛选条 -->
          <div
            v-if="allTags.length"
            class="px-4 py-2 border-b border-default/60 flex gap-1.5 flex-wrap shrink-0 max-h-20 overflow-y-auto"
          >
            <UButton
              v-for="tag in allTags"
              :key="tag"
              size="xs"
              :variant="selectedTags.includes(tag) ? 'solid' : 'soft'"
              :color="selectedTags.includes(tag) ? 'primary' : 'neutral'"
              @click="toggleTagFilter(tag)"
            >
              {{ tag }}
            </UButton>
          </div>

          <!-- 空态 -->
          <div v-if="entries.length === 0" class="flex-1 flex items-center justify-center p-6">
            <div class="text-center text-[12px] text-warning/80">
              <UIcon name="i-tabler-alert-triangle" class="size-6 mx-auto mb-2" />
              {{ t('template.picker.library_empty') }}
            </div>
          </div>
          <div v-else-if="filtered.length === 0" class="flex-1 flex items-center justify-center p-6 text-sm text-dimmed">
            {{ t('template.manager.no_match', { search }) }}
          </div>

          <!-- 虚拟滚动网格 -->
          <div v-else v-bind="containerProps" class="flex-1 overflow-y-auto px-4 py-3">
            <div v-bind="wrapperProps">
              <div
                v-for="row in vList"
                :key="row.index"
                class="grid gap-3 pb-3 content-start"
                :style="{ height: ROW_H + 'px', gridTemplateColumns: `repeat(${COLS}, minmax(0, 1fr))` }"
              >
                <div
                  v-for="s in row.data"
                  :key="s.guid"
                  class="group relative rounded-md border bg-elevated/30 overflow-hidden cursor-pointer transition-colors flex flex-col"
                  :class="
                    isSelected(s.guid)
                      ? 'border-primary ring-2 ring-primary/40'
                      : 'border-default hover:border-primary/60'
                  "
                  @click="openDetail(s.guid)"
                >
                  <!-- 选择 checkbox (节点多选) -->
                  <button
                    type="button"
                    class="absolute top-1 left-1 z-10 rounded-full bg-default/70 backdrop-blur p-0.5"
                    :title="isSelected(s.guid) ? t('template.picker.unselect') : t('template.picker.select')"
                    @click.stop="toggle(s.guid)"
                  >
                    <UIcon
                      :name="isSelected(s.guid) ? 'i-tabler-circle-check-filled' : 'i-tabler-circle'"
                      class="size-4"
                      :class="isSelected(s.guid) ? 'text-primary' : 'text-dimmed'"
                    />
                  </button>
                  <!-- 缩略图 (含 tags 浮层 + hover「查看」) -->
                  <div class="relative h-[124px] bg-elevated flex items-center justify-center overflow-hidden">
                    <img
                      v-if="thumbCache[s.guid]"
                      :src="thumbCache[s.guid]"
                      class="max-w-full max-h-full object-contain"
                      :alt="s.name"
                    />
                    <UIcon v-else name="i-tabler-photo" class="size-7 text-dimmed" />
                    <!-- tags 浮层 (非 hover 显示) -->
                    <div
                      v-if="s.tags?.length"
                      class="absolute bottom-1 left-1 right-1 flex flex-wrap gap-0.5 group-hover:opacity-0 transition-opacity"
                    >
                      <span
                        v-for="tag in s.tags.slice(0, 3)"
                        :key="tag"
                        class="text-[8px] leading-none px-1 py-0.5 rounded bg-default/80 text-toned backdrop-blur-sm truncate max-w-[72px]"
                      >
                        {{ tag }}
                      </span>
                    </div>
                    <!-- hover「查看」 -->
                    <div
                      class="absolute inset-0 flex items-center justify-center bg-default/40 opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none"
                    >
                      <span class="inline-flex items-center gap-1 px-2 py-1 rounded bg-elevated/90 text-[10px] text-toned">
                        <UIcon name="i-tabler-eye" class="size-3.5" /> {{ t('template.picker.view') }}
                      </span>
                    </div>
                  </div>
                  <!-- 名称 + 变体数 -->
                  <div class="px-2 py-1.5 min-w-0 flex items-center gap-1">
                    <span class="text-xs text-highlighted truncate flex-1" :title="s.name">
                      {{ s.name || s.guid }}
                    </span>
                    <span class="text-[9px] text-dimmed inline-flex items-center gap-0.5 shrink-0">
                      <UIcon name="i-tabler-layers" class="size-3" />{{ s.variantCount }}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <footer class="flex items-center gap-2 px-4 py-3 border-t border-default shrink-0">
            <span class="text-[11px] text-dimmed">
              {{ t('template.picker.selected_count', { n: selected.length }) }}
            </span>
            <UButton
              v-if="selected.length"
              size="xs"
              variant="soft"
              color="error"
              icon="i-tabler-trash"
              @click="batchDelete"
            >
              {{ t('template.picker.delete_selected') }}
            </UButton>
            <span class="flex-1" />
            <UButton size="sm" color="primary" @click="open = false">{{ t('template.picker.done') }}</UButton>
          </footer>
        </div>

        <!-- ========== 详情页 ========== -->
        <div v-else class="bg-default flex flex-col" style="height: 76vh">
          <header class="flex items-center gap-2 px-4 py-3 border-b border-default shrink-0">
            <UButton size="xs" variant="ghost" color="neutral" icon="i-tabler-arrow-left" @click="view = 'grid'">
              {{ t('template.picker.back') }}
            </UButton>
            <h3 class="text-sm font-medium truncate flex-1 min-w-0">{{ editName || detailGuid }}</h3>
            <UButton
              size="xs"
              :variant="detailGuid && isSelected(detailGuid) ? 'solid' : 'soft'"
              :color="detailGuid && isSelected(detailGuid) ? 'primary' : 'neutral'"
              :icon="detailGuid && isSelected(detailGuid) ? 'i-tabler-circle-check' : 'i-tabler-circle-plus'"
              @click="detailGuid && toggle(detailGuid)"
            >
              {{ detailGuid && isSelected(detailGuid) ? t('template.picker.selected') : t('template.picker.use') }}
            </UButton>
            <!-- ✕ 返回网格 (不直接关 modal — 避免误触全关) -->
            <UButton size="xs" variant="ghost" color="neutral" icon="i-tabler-x" @click="view = 'grid'" />
          </header>

          <div class="flex-1 flex min-h-0 gap-3 p-3">
            <!-- 左：缩放预览 (圆角描边框住, 不顶边) -->
            <div
              ref="zoomBox"
              class="flex-1 relative overflow-hidden rounded-lg border border-default bg-muted select-none"
              :class="dragging ? 'cursor-grabbing' : 'cursor-grab'"
              @wheel.prevent="onWheel"
              @pointerdown="onPointerDown"
              @pointermove="onPointerMove"
              @pointerup="onPointerUp"
              @pointerleave="onPointerUp"
              @dblclick="resetZoom"
            >
              <img
                v-if="activeVariantThumb"
                :src="activeVariantThumb"
                class="absolute top-1/2 left-1/2 max-w-none pointer-events-none"
                :style="imgStyle"
                draggable="false"
                @load="resetZoom"
              />
              <div v-else class="absolute inset-0 flex items-center justify-center text-xs text-dimmed">
                {{ t('common.loading') }}
              </div>
              <span class="absolute bottom-2 left-2 text-[10px] text-dimmed bg-default/60 rounded px-1.5 py-0.5">
                {{ t('template.picker.zoom_hint') }} · {{ Math.round(scale * 100) }}%
              </span>
              <!-- 上一个 / 下一个 (按 filtered 顺序切换) -->
              <UButton
                v-if="detailIndex > 0"
                class="absolute left-2 top-1/2 -translate-y-1/2 z-10"
                size="sm"
                variant="solid"
                color="neutral"
                icon="i-tabler-chevron-left"
                :title="t('template.picker.prev')"
                @click.stop="gotoOffset(-1)"
                @pointerdown.stop
                @dblclick.stop
              />
              <UButton
                v-if="detailIndex >= 0 && detailIndex < filtered.length - 1"
                class="absolute right-2 top-1/2 -translate-y-1/2 z-10"
                size="sm"
                variant="solid"
                color="neutral"
                icon="i-tabler-chevron-right"
                :title="t('template.picker.next')"
                @click.stop="gotoOffset(1)"
                @pointerdown.stop
                @dblclick.stop
              />
              <span class="absolute bottom-2 right-2 text-[10px] text-dimmed bg-default/60 rounded px-1.5 py-0.5">
                {{ detailIndex + 1 }} / {{ filtered.length }}
              </span>
            </div>

            <!-- 右：信息列 (圆角描边框住) -->
            <div class="w-72 shrink-0 rounded-lg border border-default bg-elevated/30 overflow-y-auto p-4 space-y-4">
              <!-- 名称 -->
              <div class="space-y-1">
                <label class="text-[10px] uppercase tracking-wider text-dimmed">{{ t('template.picker.name_label') }}</label>
                <UInput
                  v-model="editName"
                  size="sm"
                  class="w-full"
                  @blur="saveMeta"
                  @keyup.enter="saveMeta"
                />
              </div>

              <!-- 标签 -->
              <div class="space-y-1.5">
                <label class="text-[10px] uppercase tracking-wider text-dimmed">{{ t('template.picker.tags_label') }}</label>
                <div class="flex flex-wrap gap-1">
                  <span
                    v-for="(tag, i) in editTags"
                    :key="tag + i"
                    class="inline-flex items-center gap-1 rounded bg-elevated/60 border border-default/40 pl-1.5 pr-1 py-0.5 text-[10px] text-toned"
                  >
                    {{ tag }}
                    <UIcon
                      name="i-tabler-x"
                      class="size-3 cursor-pointer hover:text-error"
                      @click="removeTag(i)"
                    />
                  </span>
                </div>
                <UInput
                  v-model="tagInput"
                  size="xs"
                  class="w-full"
                  :placeholder="t('template.picker.add_tag')"
                  @keyup.enter="addTag"
                />
              </div>

              <!-- 分辨率变体 -->
              <div v-if="detailRecord?.variants?.length" class="space-y-1.5">
                <label class="text-[10px] uppercase tracking-wider text-dimmed">{{ t('template.picker.variants_label') }}</label>
                <!-- 当前窗口分辨率 (自动切档 / 新增·重拍 的依据) -->
                <p class="text-[10px] leading-tight">
                  <template v-if="curRes">
                    <span class="text-dimmed">{{ t('template.picker.current_window') }}: </span>
                    <span class="text-toned">{{ curResLabel }}</span>
                    <span v-if="curResHint" class="text-warning/80"> · {{ t('template.picker.scaled_from', { res: curResHint }) }}</span>
                  </template>
                  <span v-else class="text-dimmed">{{ t('template.picker.window_not_open') }}</span>
                </p>
                <div class="flex flex-wrap gap-1">
                  <UButton
                    v-for="(v, i) in detailRecord.variants"
                    :key="i"
                    size="xs"
                    :variant="activeVariantIdx === i ? 'solid' : 'soft'"
                    :color="activeVariantIdx === i ? 'primary' : 'neutral'"
                    @click="selectVariant(i)"
                  >
                    {{ v.resolution[0] }}×{{ v.resolution[1] }}
                    <span v-if="isCurResVariant(v)" class="ml-1 text-[9px] opacity-70">{{ t('template.picker.current_badge') }}</span>
                    <UIcon
                      v-if="(detailRecord?.variants?.length ?? 0) > 1"
                      name="i-tabler-x"
                      class="ml-1 size-3 opacity-60 hover:opacity-100 hover:text-error"
                      :title="t('template.picker.del_variant_title', { res: `${v.resolution[0]}×${v.resolution[1]}` })"
                      @click.stop="removeVariant(v.resolution)"
                    />
                  </UButton>
                </div>
              </div>

              <!-- 元信息 -->
              <div class="space-y-1 text-[11px] text-dimmed">
                <div class="flex items-center gap-1">
                  <UIcon name="i-tabler-layers" class="size-3" />
                  {{ detailRecord?.variants?.length ?? 0 }} {{ t('template.manager.variant_count') }}
                  <span v-if="activeVariantKB" class="text-toned">· ~{{ activeVariantKB }} KB</span>
                </div>
                <div v-if="detailRecord?.createdAt">
                  {{ t('template.manager.created_at', { time: formatTime(detailRecord.createdAt) }) }}
                </div>
                <button
                  type="button"
                  class="flex items-center gap-1 font-mono truncate max-w-full hover:text-toned"
                  :class="guidCopied ? 'text-success' : ''"
                  :title="detailGuid"
                  @click="copyGuid"
                >
                  <UIcon :name="guidCopied ? 'i-tabler-check' : 'i-tabler-copy'" class="size-3 shrink-0" />
                  <span class="truncate">{{ guidCopied ? t('common.copied') : detailGuid }}</span>
                </button>
              </div>

              <!-- 操作 -->
              <div class="flex gap-2 pt-1">
                <UButton
                  size="xs"
                  variant="soft"
                  color="neutral"
                  :icon="curRes && !curResExact ? 'i-tabler-plus' : 'i-tabler-refresh'"
                  class="flex-1 justify-center"
                  @click="onRecapture"
                >
                  {{ recaptureLabel }}
                </UButton>
                <UButton size="xs" variant="soft" color="error" icon="i-tabler-trash" class="flex-1 justify-center" @click="onDelete">
                  {{ t('common.delete') }}
                </UButton>
              </div>
            </div>
          </div>
        </div>
      </template>
    </UModal>

    <!-- 已选模板 chips（可逐个移除）+ 缺失告警 -->
    <div v-if="selected.length" class="flex flex-wrap gap-1">
      <span
        v-for="guid in selected"
        :key="guid"
        class="inline-flex items-center gap-1 rounded bg-elevated/60 border border-default/40 pl-1.5 pr-1 py-0.5 text-[10px]"
        :class="tplStore.map[guid] ? 'text-toned' : 'text-error/90 border-error/30'"
      >
        <UIcon v-if="!tplStore.map[guid]" name="i-tabler-alert-triangle" class="size-3" />
        <span class="truncate max-w-[140px]">{{ tplStore.map[guid]?.name || guid }}</span>
        <UIcon name="i-tabler-x" class="size-3 cursor-pointer hover:text-error" @click="toggle(guid)" />
      </span>
    </div>
    <p v-else class="text-[10px] text-dimmed">{{ t('template.picker.no_template_selected') }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useVirtualList } from '@vueuse/core'
import { useTemplatesStore } from '@/stores/templates'
import { awaitWailsEvent } from '@/composables/useWailsEvent'
import { useToast } from '@nuxt/ui/composables'
import { useConfirm } from '@/composables/useConfirm'
import { backend, type AssetSummary, type AssetRecord } from '@/lib/backend'

const { t } = useI18n()

const props = defineProps<{ modelValue: string[] }>()
const emit = defineEmits<{ 'update:modelValue': [v: string[]] }>()

const tplStore = useTemplatesStore()
const toast = useToast()
const { confirm } = useConfirm()

const open = ref(false)
const view = ref<'grid' | 'detail'>('grid')
const search = ref('')
const thumbCache = ref<Record<string, string>>({})

// modelValue 容错: 可能是 undefined / 单 string / string[].
const selected = computed<string[]>(() => {
  const v = props.modelValue as unknown
  if (Array.isArray(v)) return v.filter((x): x is string => typeof x === 'string')
  if (typeof v === 'string' && v) return [v]
  return []
})

onMounted(() => {
  if (Object.keys(tplStore.map).length === 0) void tplStore.reload()
})

function onModalToggle(v: boolean) {
  open.value = v
}
// 关 modal 复位到网格 — 覆盖所有关闭路径 (✕ / 完成 / 背景点击 / Esc).
// 开 modal 取一次当前窗口分辨率 (缓存供详情页挑档/显示用).
watch(open, (v) => {
  if (v) void refreshCurRes()
  else view.value = 'grid'
})

// ── 网格 ────────────────────────────────────────────────
const entries = computed(() => Object.values(tplStore.map))

// 标签筛选: 收集全部 tags + 多选 AND 过滤.
const selectedTags = ref<string[]>([])
const allTags = computed(() => {
  const set = new Set<string>()
  for (const s of entries.value) for (const tag of s.tags ?? []) set.add(tag)
  return [...set].sort()
})
function toggleTagFilter(tag: string) {
  selectedTags.value = selectedTags.value.includes(tag)
    ? selectedTags.value.filter((x) => x !== tag)
    : [...selectedTags.value, tag]
}

const filtered = computed(() => {
  let arr = entries.value
  const q = search.value.trim().toLowerCase()
  if (q) {
    arr = arr.filter(
      (s) =>
        s.guid.toLowerCase().includes(q) ||
        s.name?.toLowerCase().includes(q) ||
        s.tags?.some((tag) => tag.toLowerCase().includes(q)),
    )
  }
  if (selectedTags.value.length) {
    arr = arr.filter((s) => selectedTags.value.every((tag) => (s.tags ?? []).includes(tag)))
  }
  return arr
})

// 虚拟滚动: 把列表切成行 (每行 COLS 个), 只渲染可见行.
const COLS = 4
const ROW_H = 184 // 卡片 ~172 + 行间距 12
const rows = computed<AssetSummary[][]>(() => {
  const out: AssetSummary[][] = []
  for (let i = 0; i < filtered.value.length; i += COLS) {
    out.push(filtered.value.slice(i, i + COLS))
  }
  return out
})
const { list: vList, containerProps, wrapperProps } = useVirtualList(rows, {
  itemHeight: ROW_H,
  overscan: 3,
})

function isSelected(guid: string) {
  return selected.value.includes(guid)
}
function toggle(guid: string) {
  const cur = selected.value
  const next = cur.includes(guid) ? cur.filter((x) => x !== guid) : [...cur, guid]
  emit('update:modelValue', next)
}

const selectedNames = computed(() => selected.value.map((g) => tplStore.map[g]?.name || g))
const firstThumb = computed(() =>
  selected.value[0] ? thumbCache.value[selected.value[0]] : undefined,
)
const summaryLabel = computed(() => {
  if (selected.value.length === 0) return t('template.picker.select_placeholder')
  if (selected.value.length === 1) return tplStore.map[selected.value[0]]?.name || selected.value[0]
  return t('template.picker.selected_count', { n: selected.value.length })
})

async function loadThumb(summary: { guid: string; firstBlobSha?: string }) {
  if (thumbCache.value[summary.guid] || !summary.firstBlobSha) return
  const r = await tplStore.readBlobDataURL(summary.firstBlobSha)
  if (typeof r === 'string') thumbCache.value[summary.guid] = r
}

// 打开/筛选时拉可见缩略图
watch(
  [open, filtered, view],
  () => {
    if (!open.value || view.value !== 'grid') return
    for (const s of filtered.value) if (!thumbCache.value[s.guid]) void loadThumb(s)
  },
  { immediate: true },
)
watch(
  selected,
  (guids) => {
    for (const g of guids) {
      if (g && !thumbCache.value[g]) {
        const s = tplStore.map[g]
        if (s) void loadThumb(s)
      }
    }
  },
  { immediate: true },
)

// ── 详情 ────────────────────────────────────────────────
const detailGuid = ref('')
const detailRecord = ref<AssetRecord | null>(null)
const variantThumbs = ref<Record<string, string>>({}) // blobSha → dataURL
const activeVariantIdx = ref(0)
const editName = ref('')
const editTags = ref<string[]>([])
const tagInput = ref('')

// 预览翻页: 当前资产在 filtered 里的位置 + 切上一个/下一个.
const detailIndex = computed(() => filtered.value.findIndex((s) => s.guid === detailGuid.value))
function gotoOffset(delta: number) {
  const next = filtered.value[detailIndex.value + delta]
  if (next) void openDetail(next.guid)
}

async function openDetail(guid: string) {
  detailGuid.value = guid
  view.value = 'detail'
  activeVariantIdx.value = 0
  resetZoom()
  const rec = await backend.assets.get(guid)
  if (!rec) return
  detailRecord.value = rec
  editName.value = rec.name ?? ''
  editTags.value = [...(rec.tags ?? [])]
  for (const v of rec.variants ?? []) void loadVariantThumb(v.blob)
  await applyCurResPick(guid) // 自动切到当前分辨率运行时真会用的那档
}

async function loadVariantThumb(sha: string) {
  if (!sha || variantThumbs.value[sha]) return
  const r = await tplStore.readBlobDataURL(sha)
  if (typeof r === 'string') variantThumbs.value[sha] = r
}

const activeVariant = computed(() => detailRecord.value?.variants?.[activeVariantIdx.value])
const activeVariantThumb = computed(() => {
  const sha = activeVariant.value?.blob
  return sha ? variantThumbs.value[sha] : undefined
})
// base64 估算字节: dataURL = "data:image/png;base64,XXXX"; bytes ≈ b64len * 3/4.
const activeVariantKB = computed(() => {
  const d = activeVariantThumb.value
  if (!d) return 0
  const b64 = d.slice(d.indexOf(',') + 1)
  return Math.max(1, Math.round((b64.length * 0.75) / 1024))
})

function selectVariant(i: number) {
  activeVariantIdx.value = i
  resetZoom()
}

// ── 当前窗口分辨率感知 ─────────────────────────────────────
// modal 开时取一次当前游戏窗口客户区分辨率 (缓存, 避免每张卡都解析窗口; 窗口没开只等这一次).
// 用途: 进详情自动切"运行时真会用的那档" + 顶部显示当前分辨率 + 重拍/新增按钮语义化.
const curRes = ref<[number, number] | null>(null)
const curResExact = ref(false) // 当前分辨率是否有精确变体档 (有=重拍 / 无=新增)

async function refreshCurRes() {
  curRes.value = (await backend.assets.currentResolution(tplStore.containerId)) ?? null
}
// 用缓存分辨率挑"运行时真会用的那档" → 自动切到该档. 挑档算法在后端, 此处只接结果.
async function applyCurResPick(guid: string) {
  if (!curRes.value) {
    curResExact.value = false
    return // 窗口没开 → 不自动切 (openDetail 已置 activeVariantIdx=0)
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
// 无精确档时, 运行时实际会缩放使用的那档 (= 当前 activeVariant) 分辨率, 提示给用户.
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
  return !!curRes.value && v.resolution[0] === curRes.value[0] && v.resolution[1] === curRes.value[1]
}

// 删单个分辨率档 (仅 >1 档时给入口; 最后一档走整删). 资产仍在, 不查引用.
async function removeVariant(res: number[]) {
  const guid = detailGuid.value
  if (!guid || (detailRecord.value?.variants?.length ?? 0) <= 1) return
  const yes = await confirm({
    title: t('template.picker.del_variant_title', { res: `${res[0]}×${res[1]}` }),
    description: t('template.picker.del_variant_desc'),
    confirmText: t('common.delete'),
    color: 'error',
  })
  if (yes !== true) return
  const r = await backend.assets.removeVariant(guid, res[0], res[1])
  if (r === undefined) return // 失败 invoke 已 toast
  variantThumbs.value = {}
  delete thumbCache.value[guid]
  await tplStore.reload()
  const rec = await backend.assets.get(guid)
  if (rec) {
    detailRecord.value = rec
    for (const v of rec.variants ?? []) void loadVariantThumb(v.blob)
  }
  await applyCurResPick(guid)
}

async function saveMeta() {
  if (!detailGuid.value) return
  await backend.assets.updateMeta(detailGuid.value, editName.value.trim(), editTags.value)
  await tplStore.reload()
}
function addTag() {
  const v = tagInput.value.trim()
  if (!v || editTags.value.includes(v)) {
    tagInput.value = ''
    return
  }
  editTags.value = [...editTags.value, v]
  tagInput.value = ''
  void saveMeta()
}
function removeTag(i: number) {
  editTags.value = editTags.value.filter((_, idx) => idx !== i)
  void saveMeta()
}

const guidCopied = ref(false)
let guidCopiedTimer = 0
function copyGuid() {
  navigator.clipboard?.writeText(detailGuid.value).then(
    () => {
      guidCopied.value = true
      window.clearTimeout(guidCopiedTimer)
      guidCopiedTimer = window.setTimeout(() => {
        guidCopied.value = false
      }, 1500)
    },
    () => toast.add({ title: t('toast.copy_failed'), color: 'error' }),
  )
}

function formatTime(s?: string): string {
  if (!s) return '—'
  try {
    return new Date(s).toLocaleString()
  } catch {
    return s
  }
}

// ── 缩放预览 (滚轮缩放 + 拖拽平移 + 双击/换图复位) ──────
const scale = ref(1)
const tx = ref(0)
const ty = ref(0)
const dragging = ref(false)
let startX = 0
let startY = 0
const imgStyle = computed(() => ({
  transform: `translate(-50%, -50%) translate(${tx.value}px, ${ty.value}px) scale(${scale.value})`,
}))
function resetZoom() {
  scale.value = 1
  tx.value = 0
  ty.value = 0
}
function onWheel(e: WheelEvent) {
  const factor = e.deltaY < 0 ? 1.12 : 0.89
  scale.value = Math.min(8, Math.max(0.3, scale.value * factor))
}
function onPointerDown(e: PointerEvent) {
  dragging.value = true
  startX = e.clientX - tx.value
  startY = e.clientY - ty.value
}
function onPointerMove(e: PointerEvent) {
  if (!dragging.value) return
  tx.value = e.clientX - startX
  ty.value = e.clientY - startY
}
function onPointerUp() {
  dragging.value = false
}

// ── 重拍 / 删除 (详情页, 节点上下文有 containerId) ──────
async function onRecapture() {
  const guid = detailGuid.value
  if (!guid) return
  const id = 'recap-' + Math.random().toString(36).slice(2, 10) + '-' + Date.now()
  const waiter = awaitWailsEvent<{ id: string; mode: string; payload: any }>(
    'tools:picker-result',
    (p) => p?.id === id,
  )
  await backend.tools.openScreenPicker('template_recapture', id, tplStore.containerId, '', '', guid)
  const result = await waiter
  if (result.payload?.cancelled || !result.payload?.guid) return
  // 重拍换了 blob: 清详情变体缓存 + 网格缩略图, 重取记录.
  variantThumbs.value = {}
  delete thumbCache.value[guid]
  await tplStore.reload()
  const rec = await backend.assets.get(guid)
  if (rec) {
    detailRecord.value = rec
    for (const v of rec.variants ?? []) void loadVariantThumb(v.blob)
  }
  // 重拍/新增成功后重取当前分辨率 + 重新挑档 → 自动跳到刚拍那档 (此时变精确命中).
  await refreshCurRes()
  await applyCurResPick(guid)
  resetZoom()
}

async function onDelete() {
  const guid = detailGuid.value
  const name = detailRecord.value?.name || guid
  if (!guid) return
  const refs = await backend.assets.referrers(guid)
  const n = refs?.length ?? 0
  const yes = await confirm({
    title: t('template.manager.delete_title'),
    description:
      n > 0
        ? t('template.manager.delete_confirm_referenced', { key: name, n })
        : t('template.manager.delete_confirm', { key: name }),
    color: 'error',
    confirmText: t('common.delete'),
  })
  if (yes !== true) return
  const ok = await tplStore.remove(guid)
  if (!ok) return
  delete thumbCache.value[guid]
  if (selected.value.includes(guid)) {
    emit('update:modelValue', selected.value.filter((x) => x !== guid))
  }
  view.value = 'grid'
}

// 批量删除选中的模板 (网格页, 不用进详情). 删完从选中里清掉.
async function batchDelete() {
  const guids = [...selected.value]
  if (guids.length === 0) return
  // 汇总被引用处数 (跟单删一致, 让用户看清影响面).
  let refTotal = 0
  for (const g of guids) {
    const refs = await backend.assets.referrers(g)
    refTotal += refs?.length ?? 0
  }
  const yes = await confirm({
    title: t('template.manager.batch_delete_title'),
    description:
      refTotal > 0
        ? t('template.manager.batch_delete_confirm_referenced', { n: guids.length, refs: refTotal })
        : t('template.manager.batch_delete_confirm', { n: guids.length }),
    color: 'error',
    confirmText: t('common.delete'),
  })
  if (yes !== true) return
  for (const g of guids) {
    const ok = await tplStore.remove(g)
    if (ok) delete thumbCache.value[g]
  }
  // 删掉的从选中清掉 (map 已 reload, 不存在的过滤掉).
  emit('update:modelValue', selected.value.filter((g) => tplStore.map[g]))
}

// "现截一张" → ScreenPicker(template_save)，存成功追加到选中 (modal 不关).
async function onCaptureNew() {
  const id = 'tpl-' + Math.random().toString(36).slice(2, 10) + '-' + Date.now()
  const waiter = awaitWailsEvent<{ id: string; mode: string; payload: any }>(
    'tools:picker-result',
    (p) => p?.id === id,
  )
  await backend.tools.openScreenPicker('template_save', id, tplStore.containerId)
  const result = await waiter
  if (!result.payload?.cancelled && result.payload?.guid) {
    await tplStore.reload()
    if (!selected.value.includes(result.payload.guid)) {
      emit('update:modelValue', [...selected.value, result.payload.guid])
    }
  }
}
</script>
