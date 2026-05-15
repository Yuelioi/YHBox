<template>
  <div class="flex flex-col h-full bg-default">
    <!-- Header（modal 模式显示；独立窗口用外层 EditorTitleBar，避免重复） -->
    <header
      v-if="!hideHeader"
      class="flex items-center justify-between border-b border-default shrink-0"
      style="padding: 14px 20px"
    >
      <div class="flex items-center gap-2 min-w-0">
        <UIcon name="i-tabler-movie" class="size-4 text-dimmed shrink-0" />
        <h3 class="text-sm font-medium text-highlighted truncate">
          录制片段
          <span v-if="action" class="text-dimmed">·</span>
          <span v-if="action" class="text-default">{{ action.name || '(未命名)' }}</span>
        </h3>
      </div>
      <UButton
        v-if="showCloseButton"
        icon="i-tabler-x"
        variant="ghost"
        color="neutral"
        size="sm"
        @click="emit('close')"
      />
    </header>

    <!-- Missing action（独立窗口里 id 不存在） -->
    <div v-if="!action" class="flex-1 flex flex-col items-center justify-center gap-3 p-6">
      <UIcon name="i-tabler-alert-triangle" class="size-6 text-warning" />
      <p class="text-sm text-muted">动作不存在或已删除</p>
      <UButton size="sm" variant="soft" color="neutral" @click="emit('close')">关闭</UButton>
    </div>

    <!-- Body -->
    <template v-else>
      <div class="flex-1 overflow-y-auto space-y-4" style="padding: 16px 20px">
        <!-- 基本字段 -->
        <section class="space-y-3">
          <div>
            <label class="block text-xs text-dimmed mb-1">名称</label>
            <UInput v-model="draft.name" class="w-full" placeholder="动作名称" />
          </div>

          <div>
            <label class="block text-xs text-dimmed mb-1">备注</label>
            <UTextarea
              v-model="draft.description"
              :rows="2"
              class="w-full"
              placeholder="可选 · 用途、注意事项、依赖的窗口/分辨率等"
            />
          </div>

          <div
            v-if="
              draft.recordingContext?.resolution && draft.recordingContext.resolution.length === 2
            "
            class="text-[11px] text-dimmed"
          >
            录制分辨率: {{ draft.recordingContext.resolution[0] }} ×
            {{ draft.recordingContext.resolution[1] }}
          </div>
        </section>

        <!-- Params 声明（spec §6.5） -->
        <section>
          <div class="flex items-center justify-between mb-2">
            <h4 class="text-xs uppercase tracking-wider text-dimmed">
              参数 ({{ draft.params?.length ?? 0 }})
            </h4>
            <UButton size="xs" variant="soft" icon="i-tabler-plus" @click="addParam"
              >加参数</UButton
            >
          </div>
          <p v-if="(draft.params?.length ?? 0) === 0" class="text-[11px] text-dimmed py-1">
            参数声明后，可在 step 字段值里用
            <code class="bg-elevated/60 px-1 rounded">$params.&lt;name&gt;</code>
            引用。
          </p>
          <div v-else class="space-y-1">
            <div
              v-for="(p, i) in draft.params ?? []"
              :key="i"
              class="flex items-center gap-2 text-xs"
            >
              <UInput
                :model-value="p.name"
                size="xs"
                placeholder="参数名"
                class="w-32"
                @update:model-value="updateParam(i, 'name', String($event))"
              />
              <USelect
                :model-value="p.type"
                size="xs"
                :items="paramTypes"
                class="w-28"
                @update:model-value="updateParam(i, 'type', String($event))"
              />
              <UInput
                :model-value="formatDefault(p.default)"
                size="xs"
                placeholder="默认值"
                class="w-32"
                @update:model-value="updateParam(i, 'default', $event)"
              />
              <UButton
                size="xs"
                variant="ghost"
                color="error"
                icon="i-tabler-x"
                @click="removeParam(i)"
              />
            </div>
          </div>
        </section>

        <!-- Steps -->
        <section>
          <div class="flex items-center justify-between mb-2">
            <h4 class="text-xs uppercase tracking-wider text-dimmed">
              步骤 ({{ draft.steps.length }})
            </h4>
            <UButton
              size="xs"
              :color="recordingOrCounting ? 'error' : 'primary'"
              :variant="recordingOrCounting ? 'solid' : 'soft'"
              :icon="recordingOrCounting ? 'i-tabler-square' : 'i-tabler-circle-dot'"
              :title="recordingOrCounting ? '停止录制' : '继续录制追加到末尾'"
              @click="onContinueRecord"
            >
              {{ recordingOrCounting ? '停止' : '再录一段' }}
            </UButton>
          </div>

          <p v-if="!draft.steps.length" class="text-xs text-dimmed py-4 text-center">
            暂无步骤。点上方"再录一段"开始录制。
          </p>

          <div v-else class="space-y-1">
            <StepRow
              v-for="(s, i) in draft.steps"
              :key="s.id"
              :step="s"
              :index="i"
              :selected="false"
              :executing="isExecutingStep(s.id)"
              @update="(patch: Partial<Step>) => updateStep(i, patch)"
              @remove="removeStep(i)"
              @reorder="reorderSteps"
              @toggle-select="() => {}"
            />
          </div>
        </section>
      </div>

      <!-- Footer -->
      <footer
        class="flex items-center justify-end gap-2 border-t border-default shrink-0"
        style="padding: 12px 20px"
      >
        <UButton variant="ghost" color="neutral" @click="emit('close')">{{
          showCloseButton ? '取消' : '关闭'
        }}</UButton>
        <UButton color="primary" icon="i-tabler-check" :disabled="!dirty" @click="onSave"
          >保存</UButton
        >
      </footer>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, toRaw, watch } from 'vue'
import { useToast } from '@nuxt/ui/composables'
import { useActionsStore, type Action, type Step } from '@/stores/actions'
import StepRow from './StepRow.vue'

const props = defineProps<{
  // action 来源由父决定（Modal 传 prop；独立窗口从 store.list 查）
  action: Action | null
  // 关按钮显示 — modal 模式右上 X 关；独立窗口不需要（系统 title bar 有 ×）
  showCloseButton?: boolean
  // 隐藏内部 header — 独立窗口外层 EditorTitleBar 已经画了标题
  hideHeader?: boolean
}>()
const emit = defineEmits<{ close: [] }>()

const store = useActionsStore()
const toast = useToast()

// draft 跟随 action 重建（action 切换 / 保存后 fresh）
const draft = ref<Action>(makeDraft(props.action))
watch(
  () => props.action,
  (a) => {
    draft.value = makeDraft(a)
  },
)

function makeDraft(a: Action | null): Action {
  if (!a) return { id: '', name: '', steps: [], createdAt: '', updatedAt: '' }
  return structuredClone(toRaw(a))
}

function isExecutingStep(stepID: string): boolean {
  return !!props.action && store.runningID === props.action.id && store.runningStepID === stepID
}

const dirty = computed(() => {
  if (!props.action) return false
  return (
    JSON.stringify(stripTimestamps(draft.value)) !== JSON.stringify(stripTimestamps(props.action))
  )
})

function stripTimestamps(a: Action) {
  const { createdAt: _ca, updatedAt: _ua, ...rest } = a
  return rest
}

function updateStep(i: number, patch: Partial<Step>) {
  const next = { ...draft.value.steps[i], ...patch }
  draft.value.steps.splice(i, 1, next)
}

function removeStep(i: number) {
  draft.value.steps.splice(i, 1)
}

function reorderSteps(from: number, to: number) {
  if (from === to) return
  const arr = draft.value.steps
  const [moved] = arr.splice(from, 1)
  arr.splice(to, 0, moved)
}

async function onSave() {
  if (!props.action) return
  const { id: _id, createdAt: _ca, updatedAt: _ua, ...patch } = draft.value
  const ok = await store.update(props.action.id, patch)
  if (ok) {
    toast.add({ title: '已保存', color: 'success', icon: 'i-tabler-check' })
  }
}

// ---- Params editor ----

const paramTypes = [
  { label: 'number', value: 'number' },
  { label: 'bool', value: 'bool' },
  { label: 'string', value: 'string' },
  { label: 'point', value: 'point' },
]

function formatDefault(v: any): string {
  if (v == null) return ''
  if (typeof v === 'string') return v
  return JSON.stringify(v)
}

function parseDefault(input: any, type: string): any {
  const s = String(input ?? '').trim()
  if (s === '') return null
  switch (type) {
    case 'number': {
      const n = Number(s)
      return Number.isNaN(n) ? 0 : n
    }
    case 'bool':
      return s === 'true'
    case 'string':
      return s
    case 'point':
      try {
        return JSON.parse(s)
      } catch {
        return null
      }
  }
  return s
}

function addParam() {
  if (!draft.value.params) draft.value.params = []
  draft.value.params.push({ name: '', type: 'number', default: 0 })
}

function removeParam(i: number) {
  if (!draft.value.params) return
  draft.value.params.splice(i, 1)
}

function updateParam(i: number, field: 'name' | 'type' | 'default', val: any) {
  if (!draft.value.params) return
  const p = draft.value.params[i]
  if (field === 'name') p.name = String(val)
  else if (field === 'type') {
    p.type = val as any
    p.default = val === 'bool' ? false : val === 'string' ? '' : val === 'point' ? null : 0
  } else if (field === 'default') {
    p.default = parseDefault(val, p.type)
  }
}

// ---- 再录一段 ----

const recordingOrCounting = computed(() => store.recording || store.countdown !== null)

async function onContinueRecord() {
  // 再录一段：跟主页录制走同一路径（带 3 秒倒计时）。录完 reload 后 watch action 会重建 draft。
  await store.toggleRecording()
}
</script>
