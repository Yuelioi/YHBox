<template>
  <div class="space-y-4">
    <header class="flex flex-col gap-3 xl:flex-row xl:items-center">
      <div class="flex min-w-0 flex-1 flex-wrap items-center gap-2">
        <UInput
          v-model="search"
          icon="i-tabler-search"
          :placeholder="t('containers.search_placeholder')"
          class="min-w-60 flex-1"
        />
        <USelect
          v-model="sortKey"
          :items="sortItems"
          size="sm"
          class="w-40"
          :aria-label="t('containers.sort.label')"
        />
        <UButton
          size="sm"
          variant="soft"
          color="neutral"
          :icon="sortDesc ? 'i-tabler-sort-descending' : 'i-tabler-sort-ascending'"
          :aria-label="sortDirectionLabel"
          :title="sortDirectionLabel"
          @click="sortDesc = !sortDesc"
        />
        <div class="inline-flex rounded-md border border-default bg-muted/20 p-0.5">
          <UButton
            size="sm"
            color="neutral"
            :variant="viewMode === 'cards' ? 'soft' : 'ghost'"
            icon="i-tabler-layout-grid"
            :aria-label="t('containers.view.cards')"
            @click="viewMode = 'cards'"
          >
            {{ t('containers.view.cards') }}
          </UButton>
          <UButton
            size="sm"
            color="neutral"
            :variant="viewMode === 'list' ? 'soft' : 'ghost'"
            icon="i-tabler-list"
            :aria-label="t('containers.view.list')"
            @click="viewMode = 'list'"
          >
            {{ t('containers.view.list') }}
          </UButton>
        </div>
      </div>
      <div class="flex shrink-0 items-center gap-2">
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
      </div>
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

    <EmptyState
      v-if="store.list.length === 0"
      icon="i-tabler-schema"
      :title="t('containers.empty_title')"
      :description="t('containers.empty_desc')"
    >
      <template #action>
        <UButton color="primary" icon="i-tabler-plus" @click="onCreate">
          {{ t('containers.empty_cta') }}
        </UButton>
      </template>
    </EmptyState>

    <EmptyState
      v-else-if="pageResult.total === 0"
      icon="i-tabler-filter-off"
      :title="t('containers.no_match_title')"
      :description="t('containers.no_match_desc')"
    >
      <template #action>
        <UButton color="neutral" variant="soft" icon="i-tabler-refresh" @click="resetFilters">
          {{ t('containers.reset_filters') }}
        </UButton>
      </template>
    </EmptyState>

    <div
      v-else-if="viewMode === 'cards'"
      class="grid gap-3"
      style="grid-template-columns: repeat(auto-fill, minmax(min(260px, 100%), 1fr));"
    >
      <AppCard
        v-for="c in visibleContainers"
        :key="c.id"
        padding="panel"
        hover
        class="flex flex-col gap-3 relative"
        :class="batch.isSelected(c.id) ? '!border-primary ring-2 ring-primary/40' : ''"
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
        <div class="min-w-0">
          <div class="flex items-center justify-between gap-2">
            <h3 class="text-sm font-medium text-highlighted truncate">
              {{ c.name || t('common.untitled') }}
            </h3>
            <StatusPill
              :status="isRunning(c.id) ? 'online' : 'ready'"
              :label="isRunning(c.id) ? t('containers.status.running') : t('containers.status.idle')"
              :dot="isRunning(c.id)"
              class="shrink-0"
            />
          </div>
          <p v-if="c.description" class="text-xs text-dimmed truncate mt-0.5">
            {{ c.description }}
          </p>
          <div class="flex items-center gap-2 mt-1.5 flex-wrap">
            <span class="text-[11px] text-dimmed inline-flex items-center gap-1 font-mono tabular-nums">
              <UIcon name="i-tabler-cpu" class="size-3" />
              {{ t('containers.node_count', { n: c.graph.nodes.length }) }}
            </span>
            <span v-if="c.hotkey" class="text-[11px] text-dimmed inline-flex items-center gap-1">
              <UIcon name="i-tabler-keyboard" class="size-3" />
              <code class="text-toned bg-elevated/60 px-1 rounded font-mono">{{ c.hotkey }}</code>
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
            @click.stop="onRun(c)"
            >{{ t('containers.run') }}</UButton
          >
          <UButton
            v-else
            size="xs"
            color="error"
            variant="soft"
            icon="i-tabler-square"
            @click.stop="onStop()"
            >{{ t('containers.stop') }}</UButton
          >
          <UButton size="xs" variant="ghost" color="neutral" icon="i-tabler-edit" @click.stop="onEdit(c)"
            >{{ t('containers.edit') }}</UButton
          >
          <div class="flex-1" />
          <UButton
            size="xs"
            variant="ghost"
            color="error"
            icon="i-tabler-trash"
            :aria-label="t('containers.delete.title')"
            @click.stop="onAskDelete(c)"
          />
        </div>
      </AppCard>
    </div>

    <div v-else class="overflow-x-auto rounded-lg border border-default/60">
      <div class="min-w-[980px]">
        <div
          class="grid items-center gap-3 border-b border-default/60 bg-muted/30 px-3 py-2 text-[11px] font-medium uppercase text-dimmed"
          style="grid-template-columns: 40px minmax(220px, 1fr) 110px 90px 150px 150px 120px 150px;"
        >
          <span />
          <span>{{ t('containers.list.name') }}</span>
          <span>{{ t('containers.list.status') }}</span>
          <span>{{ t('containers.list.nodes') }}</span>
          <span>{{ t('containers.list.created_at') }}</span>
          <span>{{ t('containers.list.updated_at') }}</span>
          <span>{{ t('containers.list.hotkey') }}</span>
          <span class="text-right">{{ t('containers.list.actions') }}</span>
        </div>
        <div
          v-for="c in visibleContainers"
          :key="c.id"
          class="grid items-center gap-3 border-b border-default/40 px-3 py-2 last:border-b-0"
          :class="[
            batch.enabled.value ? 'cursor-pointer hover:bg-elevated/40' : 'hover:bg-elevated/20',
            batch.isSelected(c.id) ? 'bg-primary/10 ring-1 ring-inset ring-primary/40' : '',
          ]"
          style="grid-template-columns: 40px minmax(220px, 1fr) 110px 90px 150px 150px 120px 150px;"
          @click="batch.enabled.value ? batch.toggle(c.id) : undefined"
        >
          <div class="flex items-center justify-center">
            <UCheckbox
              v-if="batch.enabled.value"
              :model-value="batch.isSelected(c.id)"
              size="sm"
              @click.stop
              @update:model-value="batch.toggle(c.id)"
            />
            <UIcon v-else name="i-tabler-schema" class="size-4 text-dimmed" />
          </div>
          <div class="min-w-0">
            <div class="truncate text-sm font-medium text-highlighted">{{ c.name || t('common.untitled') }}</div>
            <div v-if="c.description" class="mt-0.5 truncate text-xs text-dimmed">{{ c.description }}</div>
            <div v-if="c.tags && c.tags.length > 0" class="mt-1 flex min-w-0 flex-wrap gap-1">
              <span
                v-for="tag in c.tags.slice(0, 3)"
                :key="tag"
                class="rounded bg-elevated/60 px-1.5 py-0.5 text-[10px] text-dimmed"
              >
                {{ tag }}
              </span>
            </div>
          </div>
          <StatusPill
            :status="isRunning(c.id) ? 'online' : 'ready'"
            :label="isRunning(c.id) ? t('containers.status.running') : t('containers.status.idle')"
            :dot="isRunning(c.id)"
            class="w-fit"
          />
          <span class="font-mono text-xs tabular-nums text-toned">{{ containerNodeCount(c) }}</span>
          <span class="font-mono text-xs text-dimmed">{{ formatContainerDate(c.createdAt) }}</span>
          <span class="font-mono text-xs text-dimmed">{{ formatContainerDate(c.updatedAt) }}</span>
          <span class="truncate text-xs text-dimmed">
            <code v-if="c.hotkey" class="rounded bg-elevated/60 px-1 font-mono text-toned">{{ c.hotkey }}</code>
            <span v-else>-</span>
          </span>
          <div class="flex items-center justify-end gap-1">
            <UButton
              v-if="!isRunning(c.id)"
              size="xs"
              color="primary"
              variant="soft"
              icon="i-tabler-player-play"
              :aria-label="t('containers.run')"
              @click.stop="onRun(c)"
            />
            <UButton
              v-else
              size="xs"
              color="error"
              variant="soft"
              icon="i-tabler-square"
              :aria-label="t('containers.stop')"
              @click.stop="onStop()"
            />
            <UButton
              size="xs"
              variant="ghost"
              color="neutral"
              icon="i-tabler-edit"
              :aria-label="t('containers.edit')"
              @click.stop="onEdit(c)"
            />
            <UButton
              size="xs"
              variant="ghost"
              color="error"
              icon="i-tabler-trash"
              :aria-label="t('containers.delete.title')"
              @click.stop="onAskDelete(c)"
            />
          </div>
        </div>
      </div>
    </div>

    <footer
      v-if="store.list.length > 0"
      class="flex flex-col gap-3 border-t border-default/60 pt-3 sm:flex-row sm:items-center sm:justify-between"
    >
      <span class="text-xs text-dimmed">{{ pageTotalLabel }}</span>
      <div class="flex items-center gap-2">
        <UPagination
          v-if="pageResult.totalPages > 1"
          v-model:page="page"
          :total="pageResult.total"
          :items-per-page="pageSize"
          :sibling-count="1"
          size="xs"
        />
        <USelect v-model="pageSize" :items="pageSizeItems" size="xs" class="w-28" />
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import { useContainersStore } from '@/stores/containers'
import { useExecutionStore } from '@/stores/execution'
import { useBatchSelect } from '@/composables/useBatchSelect'
import { useConfirm } from '@/composables/useConfirm'
import { type Container } from '@/lib/backend'
import { useRouter } from 'vue-router'
import AppCard from '@/components/common/AppCard.vue'
import StatusPill from '@/components/common/StatusPill.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import {
  buildContainerPage,
  containerNodeCount,
  formatContainerDate,
  type ContainerSortKey,
} from '@/lib/containerList'

const { t } = useI18n()

const router = useRouter()
const store = useContainersStore()
const execStore = useExecutionStore()
const toast = useToast()
const search = ref('')
const sortKey = ref<ContainerSortKey>('updatedAt')
const sortDesc = ref(true)
const viewMode = ref<'cards' | 'list'>('cards')
const page = ref(1)
const pageSize = ref(24)

// 批量删除（E.5）
const batch = useBatchSelect()
const { confirm } = useConfirm()

const sortItems = computed(() => [
  { label: t('containers.sort.name'), value: 'name' },
  { label: t('containers.sort.created_at'), value: 'createdAt' },
  { label: t('containers.sort.updated_at'), value: 'updatedAt' },
  { label: t('containers.sort.nodes'), value: 'nodes' },
])
const sortDirectionLabel = computed(() => sortDesc.value ? t('containers.sort.desc') : t('containers.sort.asc'))
const pageSizeItems = computed(() => [12, 24, 48, 96].map((n) => ({ label: t('containers.pagination.per_page', { n }), value: n })))

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
  if (!ok) {
    toast.add({ title: t('containers.toast.batch_partial_fail'), color: 'warning' })
  }
  batch.clear()
  batch.disable()
}
function isRunning(id: string): boolean {
  return execStore.running && execStore.currentTargetID === id
}

async function onRun(c: Container) {
  await store.run(c.id)
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

const pageResult = computed(() => buildContainerPage(store.list, {
  query: search.value,
  tags: selectedTags.value,
  sortKey: sortKey.value,
  sortDesc: sortDesc.value,
  page: page.value,
  pageSize: pageSize.value,
}))
const visibleContainers = computed(() => pageResult.value.pageItems)
const pageTotalLabel = computed(() => {
  if (pageResult.value.total === 0) return t('containers.pagination.empty')
  return t('containers.pagination.range', {
    start: pageResult.value.start,
    end: pageResult.value.end,
    total: pageResult.value.total,
  })
})

watch([search, selectedTags, sortKey, sortDesc, viewMode, pageSize], () => { page.value = 1 })
watch(() => pageResult.value.page, (p) => { if (page.value !== p) page.value = p })

function resetFilters() {
  search.value = ''
  selectedTags.value = []
}

async function onCreate() {
  const name = t('containers.create_default_name', { n: store.list.length + 1 })
  const c = await store.create(name)
  if (c) {
    onEdit(c)
  }
}

function onEdit(c: Container) {
  router.push(`/containers/${c.id}/edit`)
}

// (旧「导出到库」已删 — 子图全局化后无库包概念; 跨机分享按 spec 留口子, 真需要时做 zip 导出。)

async function onAskDelete(c: Container) {
  if (store.isRecordingLocked(c.id)) {
    toast.add({ title: t('containers.toast.recording_locked'), color: 'warning' })
    return
  }
  const yes = await confirm({
    title: t('containers.delete.title'),
    description: `${t('containers.delete.desc_prefix')}${c.name}${t('containers.delete.desc_suffix')}`,
    color: 'error',
    confirmText: t('containers.delete.confirm'),
  })
  if (yes !== true) return
  await store.remove(c.id)
}
</script>
