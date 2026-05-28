<template>
  <div class="space-y-4">
    <header class="flex items-center gap-3">
      <UInput v-model="search" icon="i-tabler-search" placeholder="搜索容器..." class="flex-1" />
      <UButton
        size="sm"
        :variant="batch.enabled.value ? 'solid' : 'soft'"
        color="neutral"
        icon="i-tabler-checks"
        @click="batch.toggleMode()"
      >
        {{ batch.enabled.value ? '退出选择' : '选择' }}
      </UButton>
      <UButton
        v-if="batch.enabled.value && batch.count.value > 0"
        size="sm"
        color="error"
        icon="i-tabler-trash"
        @click="onBatchDelete"
      >
        删除 ({{ batch.count.value }})
      </UButton>
      <UButton color="primary" icon="i-tabler-plus" @click="onCreate">新建容器</UButton>
    </header>

    <!-- Tag chip 筛选（按使用计数倒序，横向滚动） -->
    <div v-if="tagsByCount.length > 0" class="overflow-x-auto whitespace-nowrap py-1 flex gap-1.5">
      <UButton
        v-for="t in tagsByCount"
        :key="t.tag"
        size="xs"
        :variant="selectedTags.includes(t.tag) ? 'solid' : 'soft'"
        :color="selectedTags.includes(t.tag) ? 'primary' : 'neutral'"
        class="shrink-0"
        @click="toggleTag(t.tag)"
      >
        {{ t.tag }}
        <span class="ml-1 text-[9px] opacity-60">{{ t.count }}</span>
      </UButton>
    </div>

    <div
      v-if="filtered.length === 0"
      class="rounded-xl bg-default/50 border border-default/60 border-dashed py-12 px-6 text-center"
    >
      <UIcon name="i-tabler-schema" class="size-8 text-dimmed mx-auto mb-3" />
      <p class="text-sm text-muted">还没有容器</p>
      <p class="text-xs text-dimmed mt-1">
        容器是节点图蓝图，包含变量、控制流、模板检测和 Action 调用。
      </p>
    </div>

    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
      <div
        v-for="c in filtered"
        :key="c.id"
        class="rounded-xl bg-default border p-4 flex flex-col gap-3 transition-colors relative"
        :class="[
          store.isEditing(c.id) ? 'opacity-50 pointer-events-none' : 'hover:border-accented',
          batch.isSelected(c.id) ? 'border-primary ring-2 ring-primary/40' : 'border-default',
        ]"
        @click="batch.enabled.value ? batch.toggle(c.id) : undefined"
      >
        <UCheckbox
          v-if="batch.enabled.value"
          :model-value="batch.isSelected(c.id)"
          size="sm"
          class="absolute top-2 left-2"
          @click.stop
          @update:model-value="batch.toggle(c.id)"
        />
        <UIcon
          v-if="store.isEditing(c.id)"
          name="i-tabler-lock"
          class="absolute top-2 right-2 size-3.5 text-amber-300"
          :title="'另一窗口正在编辑中'"
        />
        <div class="min-w-0">
          <h3 class="text-sm font-medium text-highlighted truncate">
            {{ c.name || '(未命名)' }}
          </h3>
          <p v-if="c.description" class="text-xs text-dimmed truncate mt-0.5">
            {{ c.description }}
          </p>
          <div class="flex items-center gap-2 mt-1.5 flex-wrap">
            <span class="text-[11px] text-dimmed inline-flex items-center gap-1">
              <UIcon name="i-tabler-cpu" class="size-3" />
              {{ c.graph.nodes.length }} 节点
            </span>
            <span v-if="c.hotkey" class="text-[11px] text-dimmed inline-flex items-center gap-1">
              <UIcon name="i-tabler-keyboard" class="size-3" />
              <code class="text-toned bg-elevated/60 px-1 rounded">{{ c.hotkey }}</code>
            </span>
          </div>
        </div>
        <div class="flex items-center gap-1.5 pt-1 border-t border-default/60">
          <UButton
            v-if="!isRunning(c.id)"
            size="xs"
            color="primary"
            variant="soft"
            icon="i-tabler-player-play"
            @click="onRun(c)"
            >运行</UButton
          >
          <UButton
            v-else
            size="xs"
            color="error"
            variant="soft"
            icon="i-tabler-square"
            @click="onStop()"
            >停止</UButton
          >
          <UButton size="xs" variant="ghost" color="neutral" icon="i-tabler-edit" @click="onEdit(c)"
            >编辑</UButton
          >
          <div class="flex-1" />
          <UButton
            size="xs"
            variant="ghost"
            color="error"
            icon="i-tabler-trash"
            @click="onAskDelete(c)"
          />
        </div>
      </div>
    </div>

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
            <h3 class="text-sm font-medium">删除容器？</h3>
          </div>
          <p class="text-xs text-muted">
            将删除 <span class="text-default">{{ pendingDelete?.name }}</span
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
import { useContainersStore } from '@/stores/containers'
import { useExecutionStore } from '@/stores/execution'
import { useBatchSelect } from '@/composables/useBatchSelect'
import { useConfirm } from '@/composables/useConfirm'
import { backend, type Container } from '@/lib/backend'
import { useRouter } from 'vue-router'

const router = useRouter()
const store = useContainersStore()
const execStore = useExecutionStore()
const toast = useToast()
const search = ref('')

// 批量删除（E.5）
const batch = useBatchSelect()
const { confirm } = useConfirm()

async function onBatchDelete() {
  const ids = [...batch.selected.value]
  if (ids.length === 0) return
  const yes = await confirm({
    title: '批量删除容器',
    description: `确认删除 ${ids.length} 个容器？此操作不可恢复。`,
    color: 'error',
    confirmText: '删除',
  })
  if (yes !== true) return
  const ok = await store.deleteMany(ids)
  if (ok) {
    toast.add({ title: `已删除 ${ids.length} 个`, color: 'success' })
  } else {
    toast.add({ title: '批量删除部分失败（详情见日志）', color: 'warning' })
  }
  batch.clear()
  batch.disable()
}
const pendingDelete = ref<Container | null>(null)

function isRunning(id: string): boolean {
  return execStore.running && execStore.currentTargetID === id
}

async function onRun(c: Container) {
  await store.run(c.id)
  toast.add({
    title: `已加入运行队列: ${c.name}`,
    color: 'success',
    icon: 'i-tabler-player-play',
  })
}

async function onStop() {
  await store.stopAll()
  toast.add({
    title: '已发出停止信号',
    color: 'neutral',
    icon: 'i-tabler-square',
  })
}

onMounted(() => {
  void store.reload()
})

const selectedTags = ref<string[]>([])

const tagsByCount = computed(() => {
  const counts: Record<string, number> = {}
  for (const c of store.list ?? []) {
    for (const t of (c as any).tags ?? []) counts[t] = (counts[t] ?? 0) + 1
  }
  return Object.entries(counts).sort((a, b) => b[1] - a[1]).map(([tag, count]) => ({ tag, count }))
})

function toggleTag(tag: string) {
  if (selectedTags.value.includes(tag)) {
    selectedTags.value = selectedTags.value.filter((t) => t !== tag)
  } else {
    selectedTags.value = [...selectedTags.value, tag]
  }
}

const filtered = computed(() => {
  let list = store.list
  if (search.value) {
    const q = search.value.toLowerCase()
    list = list.filter((c) => c.name?.toLowerCase().includes(q))
  }
  if (selectedTags.value.length > 0) {
    list = list.filter((c) => selectedTags.value.every((t) => ((c as any).tags ?? []).includes(t)))
  }
  return list
})

async function onCreate() {
  const name = `容器 ${store.list.length + 1}`
  const c = await store.create(name)
  if (c) {
    onEdit(c)
  }
}

function onEdit(c: Container) {
  // 默认嵌入主壳; 用户在编辑器工具栏点 i-tabler-external-link 可拆独立窗口.
  router.push(`/containers/${c.id}/edit`)
}

function onAskDelete(c: Container) {
  pendingDelete.value = c
}

async function onConfirmDelete() {
  const c = pendingDelete.value
  if (!c) return
  pendingDelete.value = null
  await store.remove(c.id)
}
</script>
