<template>
  <div class="space-y-4">
    <!-- 顶栏：搜索最左，主操作最右 -->
    <header class="flex items-center gap-3 flex-wrap">
      <UInput
        v-model="search"
        icon="i-tabler-search"
        placeholder="搜索动作..."
        class="flex-1 min-w-48 max-w-md"
      />
      <UButton
        :color="recordingOrCounting ? 'error' : 'neutral'"
        :variant="recordingOrCounting ? 'solid' : 'soft'"
        :icon="recordingOrCounting ? 'i-tabler-square' : 'i-tabler-circle-dot'"
        :disabled="store.countdown !== null"
        @click="onToggleRecord"
      >
        {{ recordingOrCounting ? '停止录制' : '录制新动作' }}
      </UButton>
      <UButton color="primary" icon="i-tabler-plus" @click="onCreate">新建动作</UButton>
    </header>

    <!-- 提示行 -->
    <div class="flex items-center justify-between gap-3 flex-wrap text-xs text-dimmed">
      <p>
        Action 是「录制片段」：纯录制鼠键序列，没有循环/条件/模板检测。要编排请去
        <RouterLink to="/tasks" class="text-primary underline">容器</RouterLink>。
      </p>
      <span class="inline-flex items-center gap-1 shrink-0">
        <UIcon name="i-tabler-keyboard" class="size-3.5" />
        <code class="text-toned bg-elevated/60 px-1 rounded">Ctrl+Shift+R</code>
        切换录制
      </span>
    </div>

    <!-- 空状态 -->
    <div
      v-if="!store.list.length"
      class="rounded-xl bg-default/50 border border-default/60 border-dashed py-12 px-6 text-center"
    >
      <UIcon name="i-tabler-movie" class="size-8 text-dimmed mx-auto mb-3" />
      <p class="text-sm text-muted">还没有动作</p>
      <p class="text-xs text-dimmed mt-1">点上方"新建动作"或"录制新动作"开始。</p>
    </div>

    <!-- 卡片网格 -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
      <ActionCard
        v-for="a in filtered"
        :key="a.id"
        :action="a"
        :running="store.runningID === a.id"
        @run="store.run(a.id)"
        @stop="store.stop()"
        @edit="onEdit(a)"
        @delete="onAskDelete(a)"
      />
    </div>
    <p
      v-if="store.list.length > 0 && filtered.length === 0"
      class="text-xs text-dimmed text-center py-4"
    >
      没有匹配的动作。
    </p>

    <!-- 删除确认 -->
    <UModal
      :open="!!pendingDelete"
      :ui="{ content: 'sm:max-w-[440px]' }"
      @update:open="
        (v: boolean) => {
          if (!v) pendingDelete = null
        }
      "
    >
      <template #content>
        <div class="p-6 space-y-4 bg-default">
          <div class="flex items-center gap-2">
            <UIcon name="i-tabler-alert-triangle" class="size-4 text-warning" />
            <h3 class="text-sm font-medium text-highlighted">删除动作？</h3>
          </div>
          <p class="text-xs text-muted">
            将删除
            <span class="text-default">{{ pendingDelete?.name || '(未命名)' }}</span
            >，无法恢复。
          </p>
          <div class="flex justify-end gap-2 pt-2">
            <UButton variant="ghost" color="neutral" @click="pendingDelete = null">取消</UButton>
            <UButton color="error" icon="i-tabler-trash" @click="onConfirmDelete">删除</UButton>
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useToast } from '@nuxt/ui/composables'
import { useActionsStore, type Action } from '@/stores/actions'
import { backend } from '@/lib/backend'
import ActionCard from '@/components/actions/ActionCard.vue'

const store = useActionsStore()
const toast = useToast()

const search = ref('')
const pendingDelete = ref<Action | null>(null)
const recordingOrCounting = computed(() => store.recording || store.countdown !== null)

const filtered = computed(() => {
  if (!search.value) return store.list
  const q = search.value.toLowerCase()
  return store.list.filter(
    (a) => a.name?.toLowerCase().includes(q) || a.description?.toLowerCase().includes(q),
  )
})

onMounted(async () => {
  await store.reload()
})

async function onCreate() {
  const name = `动作 ${store.list.length + 1}`
  const a = await store.create(name)
  if (a) {
    await openEditor(a.id)
  }
}

async function onToggleRecord() {
  await store.toggleRecording()
}

async function onEdit(a: Action) {
  await openEditor(a.id)
}

async function openEditor(id: string) {
  const r = await backend.actions.openEditorWindow(id)
  if (r === undefined) {
    toast.add({ title: '打开编辑器失败', color: 'error', icon: 'i-tabler-x' })
  }
}

function onAskDelete(a: Action) {
  pendingDelete.value = a
}

async function onConfirmDelete() {
  const a = pendingDelete.value
  if (!a) return
  pendingDelete.value = null
  const ok = await store.remove(a.id)
  if (ok) {
    toast.add({ title: '已删除', color: 'neutral', icon: 'i-tabler-trash' })
  }
}
</script>
