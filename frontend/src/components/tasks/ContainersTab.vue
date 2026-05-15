<template>
  <div class="space-y-4">
    <header class="flex items-center gap-3">
      <UInput v-model="search" icon="i-tabler-search" placeholder="搜索容器..." class="flex-1" />
      <UButton color="primary" icon="i-tabler-plus" @click="onCreate">新建容器</UButton>
    </header>

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
        class="rounded-xl bg-default border border-default p-4 flex flex-col gap-3 transition-colors hover:border-accented"
      >
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
import type { Container } from '@/lib/backend'

const store = useContainersStore()
const execStore = useExecutionStore()
const toast = useToast()
const search = ref('')
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

const filtered = computed(() => {
  if (!search.value) return store.list
  const q = search.value.toLowerCase()
  return store.list.filter((c) => c.name?.toLowerCase().includes(q))
})

async function onCreate() {
  const name = `容器 ${store.list.length + 1}`
  const c = await store.create(name)
  if (c) {
    onEdit(c)
  }
}

function onEdit(c: Container) {
  // 同窗口跳子路由（v1 不开独立窗口；后续 plan 6 决定）
  window.location.hash = `#/container-editor?id=${encodeURIComponent(c.id)}`
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
