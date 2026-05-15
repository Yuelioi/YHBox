<template>
  <!--
    独立编辑器窗口的自定义 title bar。
    主窗口用 AppTitleBar 有 brand + 中间 view title + 三按钮；编辑器只需要 view title
    + 三按钮，没有 brand 占位。整行高度跟主窗口对齐（h-11 = 44px）保持视觉一致。
  -->
  <div class="h-12 shrink-0 flex items-stretch bg-default border-b border-default select-none">
    <!-- 中间 drag region + 标题 -->
    <div class="flex-1 flex items-center min-w-0" style="--wails-draggable: drag; padding: 0 16px">
      <UIcon name="i-tabler-pencil" class="size-4 text-primary shrink-0 mr-2" />
      <span class="text-sm font-medium text-highlighted truncate">
        {{ title }}
      </span>
    </div>

    <!-- 右侧 window controls -->
    <div class="shrink-0 flex items-stretch">
      <button
        type="button"
        class="w-11 flex items-center justify-center text-muted hover:bg-elevated/60 hover:text-highlighted transition-colors duration-150"
        title="最小化"
        @click="onMinimise"
      >
        <UIcon name="i-tabler-minus" class="size-4" />
      </button>
      <button
        type="button"
        class="w-11 flex items-center justify-center text-muted hover:bg-elevated/60 hover:text-highlighted transition-colors duration-150"
        :title="isMaximised ? '还原' : '最大化'"
        @click="onToggleMaximise"
      >
        <UIcon :name="isMaximised ? 'i-tabler-copy' : 'i-tabler-square'" class="size-3.5" />
      </button>
      <button
        type="button"
        class="w-11 flex items-center justify-center text-muted hover:bg-error hover:text-highlighted transition-colors duration-150"
        title="关闭"
        @click="onClose"
      >
        <UIcon name="i-tabler-x" class="size-4" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { Window } from '@wailsio/runtime'

defineProps<{ title: string }>()

const isMaximised = ref(false)
let pollTimer: ReturnType<typeof setInterval> | null = null

async function pollMaximised() {
  try {
    isMaximised.value = await Window.IsMaximised()
  } catch {
    /* runtime not ready 时静默 */
  }
}

onMounted(() => {
  pollMaximised()
  pollTimer = setInterval(pollMaximised, 500)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})

function onMinimise() {
  Window.Minimise()
}

function onToggleMaximise() {
  Window.ToggleMaximise()
  setTimeout(pollMaximised, 50)
}

function onClose() {
  Window.Close()
}
</script>
