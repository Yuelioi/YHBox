<!-- SaveSnippetDrawer — 右侧抽屉式 panel, 保存 / 编辑 snippet. -->
<template>
  <transition name="drawer">
    <div v-if="open" class="drawer-mask" @click="close">
      <div class="drawer-panel bg-default" @click.stop>
        <div class="drawer-header">
          <UIcon name="i-tabler-bookmark-plus" class="size-5 text-primary" />
          <div class="flex-1">
            <div class="text-[13px] font-semibold text-default">
              {{ editingID ? '编辑 Snippet' : '保存为 Snippet' }}
            </div>
            <div class="text-[10px] text-dimmed">
              {{ sourceKind && !editingID ? `源节点: ${sourceKind}` : '从已有 snippet 编辑' }}
            </div>
          </div>
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            icon="i-tabler-x"
            @click="close"
            title="关闭 (Esc)"
          />
        </div>

        <div class="drawer-body">
          <!-- Name -->
          <div class="field">
            <label>名称 <span class="text-error">*</span></label>
            <UInput v-model="form.name" size="sm" placeholder="例: 异环钓鱼窗口" class="w-full" />
          </div>

          <!-- Description -->
          <div class="field">
            <label>描述</label>
            <UTextarea v-model="form.description" size="sm" :rows="2" placeholder="可选 — 给自己看的说明" class="w-full" />
          </div>

          <!-- Category — NuxtUI UInputMenu (单值 + creatable 自由输入) -->
          <div class="field">
            <label>分类 (sidebar 树)</label>
            <UInputMenu
              v-model="form.category"
              :items="existingCategories"
              create-item
              size="sm"
              placeholder="例: 异环 / 原神 / 通用 — 空 = 通用"
              class="w-full"
            />
          </div>

          <!-- Tags — UInputMenu multi -->
          <div class="field">
            <label>标签 (filter)</label>
            <UInputMenu
              v-model="form.tags"
              :items="allTags"
              multiple
              create-item
              size="sm"
              placeholder="添加 tag..."
              class="w-full"
            />
          </div>

          <!-- Color picker -->
          <div class="field">
            <label>颜色 (视觉记忆)</label>
            <div class="flex flex-wrap gap-1.5">
              <button
                v-for="c in colorPalette"
                :key="c.value"
                type="button"
                class="color-chip"
                :style="{ background: c.value }"
                :class="form.color === c.value ? 'is-selected' : ''"
                :title="c.label"
                @click="form.color = form.color === c.value ? undefined : c.value"
              />
            </div>
          </div>

          <!-- Icon picker -->
          <div class="field">
            <label>图标</label>
            <div class="flex flex-wrap gap-1">
              <button
                v-for="ic in iconPalette"
                :key="ic"
                type="button"
                class="icon-chip"
                :class="form.icon === ic ? 'is-selected' : ''"
                :title="ic"
                @click="form.icon = form.icon === ic ? undefined : ic"
              >
                <UIcon :name="ic" class="size-4" />
              </button>
            </div>
          </div>

          <!-- Shortcut -->
          <div class="field">
            <label>全局快捷键 (可选)</label>
            <div class="flex gap-2">
              <UInput
                v-model="form.shortcut"
                size="sm"
                placeholder="例: Ctrl+Shift+F / Alt+1"
                class="flex-1"
                @keydown.escape.stop="form.shortcut = ''"
              />
              <UButton
                :variant="capturing ? 'solid' : 'soft'"
                :color="capturing ? 'primary' : 'neutral'"
                size="sm"
                :icon="capturing ? 'i-tabler-keyboard' : 'i-tabler-target'"
                @click="toggleCapture"
                :title="capturing ? '按任意组合键 (Esc 取消)' : '点击后按下组合键自动填'"
              />
            </div>
            <div v-if="shortcutError" class="text-[10px] text-error mt-1">⚠ {{ shortcutError }}</div>
            <div v-else-if="form.shortcut" class="text-[10px] text-dimmed mt-1">
              normalize: <code class="text-primary">{{ normalizeShortcut(form.shortcut) }}</code>
            </div>
          </div>
        </div>

        <div class="drawer-footer">
          <UButton
            v-if="editingID"
            size="sm"
            color="error"
            variant="ghost"
            icon="i-tabler-trash"
            @click="onDelete"
          >删除</UButton>
          <div class="flex-1" />
          <UButton size="sm" variant="ghost" color="neutral" @click="close">取消</UButton>
          <UButton size="sm" color="primary" :disabled="!canSave" @click="onSave">保存</UButton>
        </div>
      </div>
    </div>
  </transition>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch, onMounted } from 'vue'
import {
  useSnippetsStore,
  normalizeShortcut,
  isReservedShortcut,
  eventToShortcutKey,
  type Snippet,
} from '@/stores/snippets'
import { useConfirm } from '@/composables/useConfirm'

const props = defineProps<{
  open: boolean
  sourceKind?: string
  sourceConfig?: Record<string, unknown>
  editingID?: string
}>()
const emit = defineEmits<{
  'update:open': [v: boolean]
  saved: [s: Snippet]
}>()

const store = useSnippetsStore()
store.load()

const { confirm: confirmDialog } = useConfirm()

interface FormShape {
  name: string
  description: string
  category: string
  tags: string[]
  color?: string
  icon?: string
  shortcut: string
}

const form = reactive<FormShape>({
  name: '',
  description: '',
  category: '',
  tags: [],
  color: undefined,
  icon: undefined,
  shortcut: '',
})

function resetForm(patch: Partial<FormShape>) {
  Object.assign(form, {
    name: '',
    description: '',
    category: '',
    tags: [],
    color: undefined,
    icon: undefined,
    shortcut: '',
    ...patch,
  })
}

// open 变 true 时 prefill — flush 'post' 确保 props.editingID 在 DOM update 后是 final.
watch(
  () => props.open,
  (v) => {
    if (!v) {
      capturing.value = false
      window.removeEventListener('keydown', onCaptureKey, true)
      return
    }
    fillFromProps()
  },
  { flush: 'post' },
)

// setup time 也 fill 一次 (drawer 首次 mount 时 open 可能已经 true)
onMounted(() => {
  if (props.open) fillFromProps()
})

function fillFromProps() {
  if (props.editingID) {
    store.load() // defensive
    const s = store.getById(props.editingID)
    if (s) {
      resetForm({
        name: s.name,
        description: s.description ?? '',
        category: s.category ?? '',
        tags: [...s.tags],
        color: s.color,
        icon: s.icon,
        shortcut: s.shortcut ?? '',
      })
    } else {
      resetForm({})
    }
  } else {
    resetForm({ name: props.sourceKind ?? '' })
  }
}

const existingCategories = computed(() => store.allCategories)
const allTags = computed(() => store.allTags)

const colorPalette = [
  { label: '红 (危险)', value: '#ef4444' },
  { label: '橙', value: '#f97316' },
  { label: '黄', value: '#eab308' },
  { label: '绿 (OCR)', value: '#22c55e' },
  { label: '青 (通用)', value: '#06b6d4' },
  { label: '蓝 (窗口)', value: '#3b82f6' },
  { label: '紫', value: '#a855f7' },
  { label: '粉', value: '#ec4899' },
  { label: '灰', value: '#71717a' },
]

const iconPalette = [
  'i-tabler-bookmark', 'i-tabler-target', 'i-tabler-eye', 'i-tabler-mouse',
  'i-tabler-keyboard', 'i-tabler-clock', 'i-tabler-flag', 'i-tabler-bolt',
  'i-tabler-fish', 'i-tabler-sword', 'i-tabler-heart', 'i-tabler-coin',
  'i-tabler-package', 'i-tabler-settings', 'i-tabler-puzzle', 'i-tabler-wand',
]

const shortcutError = computed(() => {
  const s = form.shortcut.trim()
  if (!s) return ''
  const norm = normalizeShortcut(s)
  if (isReservedShortcut(s)) return `${norm} 是系统保留键, 请换`
  const existing = store.byShortcut.get(norm)
  if (existing && existing.id !== props.editingID) {
    return `${norm} 已被 "${existing.name}" 占用`
  }
  return ''
})

const canSave = computed(() => form.name.trim().length > 0 && !shortcutError.value)

const capturing = ref(false)
function toggleCapture() {
  capturing.value = !capturing.value
  if (capturing.value) window.addEventListener('keydown', onCaptureKey, true)
  else window.removeEventListener('keydown', onCaptureKey, true)
}
function onCaptureKey(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    capturing.value = false
    window.removeEventListener('keydown', onCaptureKey, true)
    return
  }
  if (['Control', 'Shift', 'Alt', 'Meta'].includes(e.key)) return
  e.preventDefault()
  e.stopPropagation()
  form.shortcut = eventToShortcutKey(e)
  capturing.value = false
  window.removeEventListener('keydown', onCaptureKey, true)
}

function onSave() {
  if (!canSave.value) return
  const payload = props.editingID
    ? store.getById(props.editingID)?.payload
    : ({ type: 'node' as const, kind: props.sourceKind!, config: props.sourceConfig ?? {} })
  if (!payload) return

  const data = {
    name: form.name.trim(),
    description: form.description.trim() || undefined,
    category: form.category.trim() || undefined,
    tags: form.tags.map((t) => t.trim()).filter(Boolean),
    color: form.color,
    icon: form.icon,
    shortcut: form.shortcut.trim() ? normalizeShortcut(form.shortcut) : undefined,
    payload,
  }

  let saved: Snippet | null = null
  if (props.editingID) {
    saved = store.update(props.editingID, data)
  } else {
    saved = store.create({ ...data, lastUsedAt: undefined })
  }
  if (saved) emit('saved', saved)
  close()
}

async function onDelete() {
  if (!props.editingID) return
  const s = store.getById(props.editingID)
  const ok = await confirmDialog({
    title: '删除 Snippet',
    description: `确定删除 "${s?.name ?? '此 snippet'}"?\n此操作不可撤销.`,
    confirmText: '删除',
    cancelText: '取消',
    color: 'error',
  })
  if (!ok) return
  store.remove(props.editingID)
  close()
}

function close() {
  emit('update:open', false)
}
</script>

<style scoped>
.drawer-mask {
  position: fixed;
  inset: 0;
  z-index: 60;
  background: rgba(8, 8, 12, 0.65);
  backdrop-filter: blur(8px) saturate(120%);
  display: flex;
  justify-content: flex-end;
}
.drawer-panel {
  width: 400px;
  height: 100%;
  border-left: 1px solid var(--ui-border);
  display: flex;
  flex-direction: column;
  box-shadow: -20px 0 50px -10px rgba(0, 0, 0, 0.7);
  font-family:
    system-ui, -apple-system, 'Segoe UI Variable Text', 'PingFang SC', sans-serif;
}
.drawer-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--ui-border);
  background-image: linear-gradient(
    135deg,
    rgba(255, 255, 255, 0.04) 0%,
    transparent 60%
  );
}
.drawer-body {
  flex: 1;
  overflow-y: auto;
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.field label {
  font-size: 11px;
  color: var(--ui-text-toned);
  font-weight: 500;
}
.drawer-footer {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid var(--ui-border);
  background: rgba(0, 0, 0, 0.2);
}

.color-chip {
  width: 26px;
  height: 26px;
  border-radius: 6px;
  border: 2px solid transparent;
  cursor: pointer;
  transition: transform 120ms ease;
}
.color-chip:hover {
  transform: scale(1.1);
}
.color-chip.is-selected {
  border-color: rgba(255, 255, 255, 0.8);
  box-shadow: 0 0 0 2px rgba(255, 255, 255, 0.2);
}

.icon-chip {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.06);
  color: var(--ui-text-toned);
  cursor: pointer;
  transition: all 120ms ease;
}
.icon-chip:hover {
  background: rgba(255, 255, 255, 0.1);
  color: var(--ui-text-default);
}
.icon-chip.is-selected {
  background: var(--ui-primary, #6366f1);
  border-color: var(--ui-primary, #6366f1);
  color: white;
}

.drawer-enter-active,
.drawer-leave-active {
  transition: opacity 180ms ease;
}
.drawer-enter-from,
.drawer-leave-to {
  opacity: 0;
}
.drawer-enter-active .drawer-panel,
.drawer-leave-active .drawer-panel {
  transition: transform 220ms cubic-bezier(0.4, 0, 0.2, 1);
}
.drawer-enter-from .drawer-panel,
.drawer-leave-to .drawer-panel {
  transform: translateX(100%);
}
</style>
