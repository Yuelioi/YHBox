<template>
  <!-- Frameless 子窗口：顶部自定义 title bar + 下方编辑器内容 -->
  <div class="flex flex-col bg-default" style="height: 100vh; width: 100vw">
    <EditorTitleBar :title="titleText" />
    <!-- inline padding 兜底：Tailwind v4 在子窗口 webview 偶尔漏扫新文件 class -->
    <div class="flex-1 min-h-0" style="padding: 12px 12px; box-sizing: border-box">
      <div class="h-full w-full rounded-lg border border-default overflow-hidden bg-default">
        <ActionEditorContent
          :action="action"
          :show-close-button="false"
          :hide-header="true"
          @close="onCloseRequest"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Window } from '@wailsio/runtime'
import { useActionsStore } from '@/stores/actions'
import ActionEditorContent from '@/components/actions/ActionEditorContent.vue'
import EditorTitleBar from '@/components/actions/EditorTitleBar.vue'

const route = useRoute()
const store = useActionsStore()

// 从 URL query 读 id；找不到时 ActionEditorContent 内部会显示"动作不存在"
const actionID = computed(() => String(route.query.id ?? ''))
const action = computed(() => store.list.find((a) => a.id === actionID.value) ?? null)

// title bar 显示文本 + 同步设到 document.title 给系统 task bar
const titleText = computed(() => {
  const name = action.value?.name?.trim() || '(未命名)'
  return `编辑动作 · ${name}`
})

onMounted(async () => {
  // 子窗口的 Pinia 是独立实例，自己 hydrate 一次拿到 action 列表
  if (store.list.length === 0) {
    await store.reload()
  }
})

watch(
  titleText,
  (v) => {
    document.title = v
  },
  { immediate: true },
)

function onCloseRequest() {
  Window.Close()
}
</script>
