<template>
  <div class="workspace-page">
    <header class="workspace-page__header">
      <div class="min-w-0">
        <div class="flex items-center gap-3">
          <span class="workspace-page__mark"
            ><UIcon name="i-tabler-calendar-time" class="size-5"
          /></span>
          <div>
            <p class="workspace-page__eyebrow">{{ t('schedule.workspace.eyebrow') }}</p>
            <h1 class="workspace-page__title">
              {{ editing ? editing.name : t('schedule.workspace.title') }}
            </h1>
          </div>
        </div>
        <p class="workspace-page__description">
          {{
            editing
              ? t('schedule.workspace.editor_description')
              : t('schedule.workspace.description')
          }}
        </p>
      </div>
      <div class="flex shrink-0 items-center gap-2">
        <UButton
          v-if="editing"
          variant="ghost"
          color="neutral"
          icon="i-tabler-arrow-left"
          @click="editing = null"
        >
          {{ t('schedule.back_to_list') }}
        </UButton>
        <UButton v-else color="primary" icon="i-tabler-plus" @click="onCreate">{{
          t('schedule.create')
        }}</UButton>
      </div>
    </header>

    <main class="min-h-0 flex-1 overflow-y-auto px-6 pb-6 sm:px-8">
      <template v-if="!editing">
        <section class="workspace-metrics mb-4" :aria-label="t('schedule.workspace.summary')">
          <div class="workspace-metric workspace-metric--primary">
            <span>{{ t('schedule.workspace.total') }}</span
            ><strong>{{ store.list.length }}</strong>
          </div>
          <div class="workspace-metric">
            <span>{{ t('schedule.workspace.enabled') }}</span
            ><strong>{{ enabledCount }}</strong>
          </div>
          <div class="workspace-metric">
            <span>{{ t('schedule.workspace.automatic') }}</span
            ><strong>{{ automaticCount }}</strong>
          </div>
          <div class="workspace-metric">
            <span>{{ t('schedule.workspace.targets') }}</span
            ><strong>{{ targetCount }}</strong>
          </div>
        </section>

        <div class="schedule-toolbar">
          <UInput
            v-model="search"
            icon="i-tabler-search"
            :placeholder="t('schedule.search_placeholder')"
            class="min-w-0 flex-1 sm:max-w-sm"
          />
          <USelect
            v-model="statusFilter"
            :items="statusItems"
            icon="i-tabler-adjustments-horizontal"
            class="w-40"
            :aria-label="t('schedule.status_filter')"
          />
          <span class="text-xs text-dimmed">{{
            t('schedule.workspace.showing', { n: filteredSchedules.length })
          }}</span>
        </div>

        <ScheduleListPanel
          :list="filteredSchedules"
          :workflows="workflows"
          @edit="onEdit"
          @delete="onDelete"
          @toggle="onToggle"
        />
      </template>

      <ScheduleEditorPanel
        v-else
        :schedule="editing"
        :workflows="workflows"
        @save="onSaveEdit"
        @cancel="editing = null"
      />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import { useSchedulesStore } from '@/stores/schedules'
import { useConfirm } from '@/composables/useConfirm'
import type { Schedule } from '@/lib/backend'
import { workflowTransport, type SourceView } from '@/app/transport/workflow'
import ScheduleListPanel from '@/components/schedules/ScheduleListPanel.vue'
import ScheduleEditorPanel from '@/components/schedules/ScheduleEditorPanel.vue'

const { t } = useI18n()
const store = useSchedulesStore()
const toast = useToast()
const { confirm } = useConfirm()
const editing = ref<Schedule | null>(null)
const search = ref('')
const statusFilter = ref<'all' | 'enabled' | 'disabled'>('all')
const workflows = ref<SourceView[]>([])

const enabledCount = computed(() => store.list.filter((schedule) => schedule.enabled).length)
const automaticCount = computed(
  () => store.list.filter((schedule) => schedule.trigger.kind !== 'manual').length,
)
const targetCount = computed(() =>
  store.list.reduce((total, schedule) => total + schedule.targets.length, 0),
)
const statusItems = computed(() => [
  { label: t('schedule.filter.all'), value: 'all' },
  { label: t('schedule.filter.enabled'), value: 'enabled' },
  { label: t('schedule.filter.disabled'), value: 'disabled' },
])
const filteredSchedules = computed(() => {
  const query = search.value.trim().toLocaleLowerCase()
  return store.list.filter((schedule) => {
    if (statusFilter.value === 'enabled' && !schedule.enabled) return false
    if (statusFilter.value === 'disabled' && schedule.enabled) return false
    return !query || schedule.name.toLocaleLowerCase().includes(query)
  })
})

onMounted(async () => {
  void store.reload()
  try {
    workflows.value = await workflowTransport.listSources()
  } catch (error) {
    toast.add({
      title: t('workflow.toast.list_failed'),
      description: error instanceof Error ? error.message : String(error),
      color: 'error',
    })
  }
})

async function onCreate() {
  const draft = await store.createDraft(
    t('schedule.create_default_name', { n: store.list.length + 1 }),
  )
  if (draft) editing.value = draft
}
function onEdit(schedule: Schedule) {
  editing.value = structuredClone(schedule)
}
async function onSaveEdit(schedule: Schedule) {
  if (await store.save(schedule)) editing.value = null
  else toast.add({ title: t('toast.save_failed'), color: 'error', icon: 'i-tabler-x' })
}
async function onToggle(schedule: Schedule, enabled: boolean) {
  await store.update(schedule.id, { enabled })
}
async function onDelete(schedule: Schedule) {
  const yes = await confirm({
    title: t('schedule.delete_title'),
    description: t('schedule.delete_desc', { name: schedule.name }),
    color: 'error',
    confirmText: t('common.delete'),
  })
  if (yes === true) await store.remove(schedule.id)
}
</script>
