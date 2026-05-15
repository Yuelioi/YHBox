<template>
  <div
    class="flex items-center gap-2 py-1.5 px-2 border rounded-md transition-colors"
    :class="[
      dragging ? 'opacity-40' : '',
      executing
        ? 'bg-primary/20 border-primary'
        : selected
          ? 'bg-primary/10 border-primary/30'
          : 'border-transparent hover:bg-elevated/30',
    ]"
    draggable="true"
    @dragstart="onDragStart"
    @dragend="onDragEnd"
    @dragover.prevent="onDragOver"
    @drop.prevent="onDrop"
  >
    <input
      type="checkbox"
      class="shrink-0 cursor-pointer accent-primary"
      :checked="selected"
      @change="emit('toggle-select', ($event.target as HTMLInputElement).checked)"
    />
    <UIcon name="i-tabler-grip-vertical" class="cursor-grab text-dimmed size-4 shrink-0" />
    <span class="text-xs text-dimmed w-6 tabular-nums shrink-0">{{ index + 1 }}.</span>
    <span class="text-xs text-toned w-16 shrink-0">{{ kindLabel }}</span>
    <component :is="renderer" :step="step" class="flex-1 min-w-0" @update="onStepUpdate" />
    <button
      type="button"
      class="shrink-0 size-6 rounded inline-flex items-center justify-center text-dimmed hover:text-error hover:bg-error/10 transition-colors cursor-pointer"
      title="删除"
      @click="emit('remove')"
    >
      <UIcon name="i-tabler-x" class="size-3.5" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Step, StepKind } from '@/stores/actions'
import StepClickEditor from './steps/StepClickEditor.vue'
import StepKeyEditor from './steps/StepKeyEditor.vue'
import StepSleepEditor from './steps/StepSleepEditor.vue'
import StepMouseMoveEditor from './steps/StepMouseMoveEditor.vue'
import StepMouseDragEditor from './steps/StepMouseDragEditor.vue'
import StepScrollEditor from './steps/StepScrollEditor.vue'
import StepMouseMoveRelEditor from './steps/StepMouseMoveRelEditor.vue'

const props = defineProps<{
  step: Step
  index: number
  selected?: boolean
  executing?: boolean // 当前 runner 正在执行这一步 → 高亮
}>()
const emit = defineEmits<{
  update: [Partial<Step>]
  remove: []
  reorder: [from: number, to: number]
  'toggle-select': [v: boolean]
}>()

const KIND_LABEL: Record<StepKind, string> = {
  click_left: '左键',
  click_middle: '中键',
  click_right: '右键',
  key: '按键',
  key_down: '按下',
  key_up: '松开',
  sleep: '休眠',
  mouse_move: '移光标',
  mouse_drag: '拖拽',
  scroll: '滚轮',
  mouse_move_rel: '转视角',
}

const kindLabel = computed(() => KIND_LABEL[props.step.kind] ?? props.step.kind)

const renderer = computed(() => {
  const k = props.step.kind
  if (k === 'sleep') return StepSleepEditor
  if (k === 'mouse_move') return StepMouseMoveEditor
  if (k === 'mouse_drag') return StepMouseDragEditor
  if (k === 'scroll') return StepScrollEditor
  if (k === 'mouse_move_rel') return StepMouseMoveRelEditor
  if (k === 'click_left' || k === 'click_middle' || k === 'click_right') return StepClickEditor
  return StepKeyEditor
})

const dragging = ref(false)

function onStepUpdate(patch: Partial<Step>) {
  emit('update', patch)
}

function onDragStart(e: DragEvent) {
  dragging.value = true
  if (e.dataTransfer) {
    e.dataTransfer.effectAllowed = 'move'
    e.dataTransfer.setData('text/yhbox-step-index', String(props.index))
  }
}
function onDragEnd() {
  dragging.value = false
}
function onDragOver(e: DragEvent) {
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
}
function onDrop(e: DragEvent) {
  const raw = e.dataTransfer?.getData('text/yhbox-step-index')
  if (raw == null || raw === '') return
  const from = Number(raw)
  if (Number.isNaN(from) || from === props.index) return
  emit('reorder', from, props.index)
}
</script>
