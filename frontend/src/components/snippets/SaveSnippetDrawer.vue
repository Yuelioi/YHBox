<!-- SaveSnippetDrawer — 右侧抽屉式 panel, 保存 / 编辑 snippet.
     字段 (GPT critique): name / description / category / tags / color / icon / shortcut.
     冲突检测 + 系统键禁 + tabler icon picker (简化版 chip 列表). -->
<template>
  <transition name="drawer">
    <div v-if="open" class="drawer-mask" @click="close">
      <div class="drawer-panel" @click.stop>
        <div class="drawer-header">
          <UIcon name="i-tabler-bookmark-plus" class="size-5 text-primary" />
          <div class="flex-1">
            <div class="text-[13px] font-semibold">
              {{ editingID ? '编辑 Snippet' : '保存为 Snippet' }}
            </div>
            <div class="text-[10px] text-dimmed">
              {{ sourceKind ? `源节点: ${sourceKind}` : '从已有 snippet 编辑' }}
            </div>
          </div>
          <button class="drawer-close" @click="close" title="关闭 (Esc)">
            <UIcon name="i-tabler-x" class="size-4" />
          </button>
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

          <!-- Category -->
          <div class="field">
            <label>分类 (sidebar 树)</label>
            <UInput
              v-model="form.category"
              size="sm"
              placeholder="例: 异环 / 原神 / 通用 — 空 = 通用"
              class="w-full"
              :list="categoryListID"
            />
            <datalist :id="categoryListID">
              <option v-for="c in existingCategories" :key="c" :value="c" />
            </datalist>
          </div>

          <!-- Tags -->
          <div class="field">
            <label>标签 (filter, 逗号分隔)</label>
            <UInput
              v-model="tagsRaw"
              size="sm"
              placeholder="例: fishing, window, qte"
              class="w-full"
            />
            <div v-if="parsedTags.length > 0" class="flex flex-wrap gap-1 mt-1">
              <span
                v-for="t in parsedTags"
                :key="t"
                class="text-[10px] px-1.5 py-0.5 rounded bg-primary/20 text-primary"
              >#{{ t }}</span>
            </div>
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
              <button
                type="button"
                class="capture-btn"
                :class="capturing ? 'is-active' : ''"
                @click="toggleCapture"
                :title="capturing ? '按任意组合键 (Esc 取消)' : '点击后按下组合键自动填'"
              >
                <UIcon :name="capturing ? 'i-tabler-keyboard' : 'i-tabler-target'" class="size-4" />
              </button>
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
            @click="onDelete"
          >删除</UButton>
          <div class="flex-1" />
          <UButton size="sm" variant="ghost" @click="close">取消</UButton>
          <UButton size="sm" color="primary" :disabled="!canSave" @click="onSave">保存</UButton>
        </div>
      </div>
    </div>
  </transition>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  useSnippetsStore,
  normalizeShortcut,
  isReservedShortcut,
  eventToShortcutKey,
  type Snippet,
} from '@/stores/snippets'

const props = defineProps<{
  open: boolean
  /** 'create' 模式: 从节点新建 */
  sourceKind?: string
  sourceConfig?: Record<string, unknown>
  /** 'edit' 模式: 编辑已有 snippet */
  editingID?: string
}>()
const emit = defineEmits<{
  'update:open': [v: boolean]
  saved: [s: Snippet]
}>()

const store = useSnippetsStore()

const form = ref<{
  name: string
  description: string
  category: string
  color?: string
  icon?: string
  shortcut: string
}>({
  name: '',
  description: '',
  category: '',
  shortcut: '',
})

const tagsRaw = ref('')

const parsedTags = computed(() =>
  tagsRaw.value
    .split(',')
    .map((t) => t.trim())
    .filter(Boolean),
)

const existingCategories = computed(() => store.allCategories)

const categoryListID = `category-list-${Math.random().toString(36).slice(2, 8)}`

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
  const s = form.value.shortcut.trim()
  if (!s) return ''
  const norm = normalizeShortcut(s)
  if (isReservedShortcut(s)) return `${norm} 是系统保留键, 请换`
  // 冲突检测 — 同 normalize key 已存在 snippet 且不是自己
  const existing = store.byShortcut.get(norm)
  if (existing && existing.id !== props.editingID) {
    return `${norm} 已被 "${existing.name}" 占用`
  }
  return ''
})

const canSave = computed(() => form.value.name.trim().length > 0 && !shortcutError.value)

// 快捷键 capture: 监听 keydown 自动填表单
const capturing = ref(false)
function toggleCapture() {
  capturing.value = !capturing.value
  if (capturing.value) {
    window.addEventListener('keydown', onCaptureKey, true)
  } else {
    window.removeEventListener('keydown', onCaptureKey, true)
  }
}
function onCaptureKey(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    capturing.value = false
    window.removeEventListener('keydown', onCaptureKey, true)
    return
  }
  // 修饰键单独不算
  if (['Control', 'Shift', 'Alt', 'Meta'].includes(e.key)) return
  e.preventDefault()
  e.stopPropagation()
  form.value.shortcut = eventToShortcutKey(e)
  capturing.value = false
  window.removeEventListener('keydown', onCaptureKey, true)
}

// open 时 reset form (按 props 决定 create or edit)
watch(
  () => props.open,
  (v) => {
    if (!v) {
      capturing.value = false
      window.removeEventListener('keydown', onCaptureKey, true)
      return
    }
    if (props.editingID) {
      const s = store.getById(props.editingID)
      if (s) {
        form.value = {
          name: s.name,
          description: s.description ?? '',
          category: s.category ?? '',
          color: s.color,
          icon: s.icon,
          shortcut: s.shortcut ?? '',
        }
        tagsRaw.value = s.tags.join(', ')
      }
    } else {
      form.value = {
        name: props.sourceKind ?? '',
        description: '',
        category: '',
        shortcut: '',
      }
      tagsRaw.value = ''
    }
  },
)

function onSave() {
  if (!canSave.value) return
  const payload = props.editingID
    ? store.getById(props.editingID)?.payload
    : ({ type: 'node' as const, kind: props.sourceKind!, config: props.sourceConfig ?? {} })
  if (!payload) return

  const data = {
    name: form.value.name.trim(),
    description: form.value.description.trim() || undefined,
    category: form.value.category.trim() || undefined,
    tags: parsedTags.value,
    color: form.value.color,
    icon: form.value.icon,
    shortcut: form.value.shortcut.trim() ? normalizeShortcut(form.value.shortcut) : undefined,
    payload,
  }

  let saved: Snippet | null = null
  if (props.editingID) {
    saved = store.update(props.editingID, data)
  } else {
    saved = store.create({
      ...data,
      lastUsedAt: undefined,
    })
  }
  if (saved) emit('saved', saved)
  close()
}

function onDelete() {
  if (!props.editingID) return
  if (!confirm('确定删除此 snippet?')) return
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
  background: rgba(0, 0, 0, 0.4);
  backdrop-filter: blur(2px);
  display: flex;
  justify-content: flex-end;
}
.drawer-panel {
  width: 380px;
  height: 100%;
  background: var(--ui-bg-default);
  border-left: 1px solid var(--ui-border);
  display: flex;
  flex-direction: column;
  box-shadow: -20px 0 40px -10px rgba(0, 0, 0, 0.6);
  font-family:
    system-ui, -apple-system, 'Segoe UI Variable Text', 'PingFang SC', sans-serif;
}
.drawer-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--ui-border);
}
.drawer-close {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  color: var(--ui-text-dimmed);
  background: transparent;
  border: none;
  cursor: pointer;
  transition: background 120ms ease;
}
.drawer-close:hover {
  background: rgba(255, 255, 255, 0.06);
  color: var(--ui-text-default);
}
.drawer-body {
  flex: 1;
  overflow-y: auto;
  padding: 12px 16px;
  display: flex;
  flex-direction: column;
  gap: 14px;
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
  padding: 10px 16px;
  border-top: 1px solid var(--ui-border);
  background: rgba(0, 0, 0, 0.15);
}

.color-chip {
  width: 24px;
  height: 24px;
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

.capture-btn {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: var(--ui-text-toned);
  cursor: pointer;
  transition: all 120ms ease;
}
.capture-btn.is-active {
  background: var(--ui-primary, #6366f1);
  color: white;
  animation: pulse 1s ease-in-out infinite;
}
@keyframes pulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(99, 102, 241, 0.6); }
  50% { box-shadow: 0 0 0 6px rgba(99, 102, 241, 0); }
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
