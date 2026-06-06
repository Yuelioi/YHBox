<template>
  <div class="space-y-4">
    <header class="flex items-center gap-3">
      <UInput v-model="search" icon="i-tabler-search" :placeholder="t('containers.search_placeholder')" class="flex-1" />
      <UButton
        size="sm"
        :variant="batch.enabled.value ? 'solid' : 'soft'"
        color="neutral"
        icon="i-tabler-checks"
        @click="batch.toggleMode()"
      >
        {{ batch.enabled.value ? t('containers.exit_select') : t('containers.select') }}
      </UButton>
      <UButton
        v-if="batch.enabled.value && batch.count.value > 0"
        size="sm"
        color="error"
        icon="i-tabler-trash"
        @click="onBatchDelete"
      >
        {{ t('containers.delete_count', { n: batch.count.value }) }}
      </UButton>
      <UButton color="primary" icon="i-tabler-plus" @click="onCreate">{{ t('containers.create') }}</UButton>
      <UButton size="xs" variant="soft" color="neutral" icon="i-tabler-layout-grid" @click="openLauncher">{{ t('containers.launcher') }}</UButton>
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
      <p class="text-sm text-muted">{{ t('containers.empty_title') }}</p>
      <p class="text-xs text-dimmed mt-1">
        {{ t('containers.empty_desc') }}
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
          :title="t('containers.editing_locked_tip')"
        />
        <div class="min-w-0">
          <h3 class="text-sm font-medium text-highlighted truncate">
            {{ c.name || t('common.untitled') }}
          </h3>
          <p v-if="c.description" class="text-xs text-dimmed truncate mt-0.5">
            {{ c.description }}
          </p>
          <div class="flex items-center gap-2 mt-1.5 flex-wrap">
            <span class="text-[11px] text-dimmed inline-flex items-center gap-1">
              <UIcon name="i-tabler-cpu" class="size-3" />
              {{ t('containers.node_count', { n: c.graph.nodes.length }) }}
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
            >{{ t('containers.run') }}</UButton
          >
          <UButton
            v-else
            size="xs"
            color="error"
            variant="soft"
            icon="i-tabler-square"
            @click="onStop()"
            >{{ t('containers.stop') }}</UButton
          >
          <UButton size="xs" variant="ghost" color="neutral" icon="i-tabler-edit" @click="onEdit(c)"
            >{{ t('containers.edit') }}</UButton
          >
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            icon="i-tabler-external-link"
            :title="t('containers.open_new_window_tip')"
            @click="onEditInWindow(c)"
          />
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
            <h3 class="text-sm font-medium">{{ t('containers.delete.title') }}</h3>
          </div>
          <p class="text-xs text-muted">
            {{ t('containers.delete.desc_prefix') }}<span class="text-default">{{ pendingDelete?.name }}</span>{{ t('containers.delete.desc_suffix') }}
          </p>
          <div class="flex justify-end gap-2 pt-2">
            <UButton variant="ghost" color="neutral" @click="pendingDelete = null">{{ t('common.cancel') }}</UButton>
            <UButton color="error" icon="i-tabler-trash" @click="onConfirmDelete">{{ t('containers.delete.confirm') }}</UButton>
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import { useContainersStore } from '@/stores/containers'
import { useExecutionStore } from '@/stores/execution'
import { useBatchSelect } from '@/composables/useBatchSelect'
import { useConfirm } from '@/composables/useConfirm'
import { backend, type Container } from '@/lib/backend'
import { errorMessage } from '@/lib/invoke'
import { useRouter } from 'vue-router'

const { t } = useI18n()

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
  if (ids.some((id) => store.isRecordingLocked(id))) {
    toast.add({ title: t('containers.toast.recording_locked'), color: 'warning' })
    return
  }
  const yes = await confirm({
    title: t('containers.batch_delete.title'),
    description: t('containers.batch_delete.desc', { n: ids.length }),
    color: 'error',
    confirmText: t('containers.batch_delete.confirm'),
  })
  if (yes !== true) return
  const ok = await store.deleteMany(ids)
  if (ok) {
    toast.add({ title: t('toast.deleted_count', { n: ids.length }), color: 'success' })
  } else {
    toast.add({ title: t('containers.toast.batch_partial_fail'), color: 'warning' })
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
    title: t('containers.toast.queue_added', { name: c.name }),
    color: 'success',
    icon: 'i-tabler-player-play',
  })
}

async function onStop() {
  await store.stopAll()
  toast.add({
    title: t('containers.toast.stop_signal'),
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
  const name = t('containers.create_default_name', { n: store.list.length + 1 })
  const c = await store.create(name)
  if (c) {
    onEdit(c)
  }
}

function onEdit(c: Container) {
  // 默认嵌入主壳; 用户在编辑器工具栏点 i-tabler-external-link 可拆独立窗口.
  router.push(`/containers/${c.id}/edit`)
}

async function onEditInWindow(c: Container) {
  // 直接开独立子窗口 (老行为). 同 id 重复点 → containerWindowAdapter focus 已有窗口.
  try {
    await backend.containers.openEditorWindow(c.id)
  } catch (e) {
    console.error('openEditorWindow failed:', e)
    toast.add({ title: t('toast.open_window_failed'), description: errorMessage(e), color: 'error' })
  }
}

function onAskDelete(c: Container) {
  if (store.isRecordingLocked(c.id)) {
    toast.add({ title: t('containers.toast.recording_locked'), color: 'warning' })
    return
  }
  pendingDelete.value = c
}

async function onConfirmDelete() {
  const c = pendingDelete.value
  if (!c) return
  pendingDelete.value = null
  await store.remove(c.id)
}

function openLauncher() {
  void backend.tools.openLauncher()
}
</script>
