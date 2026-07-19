<template>
  <div class="workspace-page" data-testid="schedules-view">
    <header class="workspace-page__header">
      <div class="min-w-0">
        <div class="flex items-center gap-3">
          <span class="workspace-page__mark">
            <UIcon name="i-tabler-calendar-time" class="size-5" />
          </span>
          <div class="min-w-0">
            <p class="workspace-page__eyebrow">{{ t('schedule.workspace.eyebrow') }}</p>
            <h1 class="workspace-page__title truncate">{{ t('schedule.workspace.title') }}</h1>
          </div>
        </div>
      </div>
      <div class="flex shrink-0 flex-wrap items-center justify-end gap-2">
        <UButton
          color="primary"
          icon="i-tabler-plus"
          data-testid="schedule-create"
          @click="onCreate"
          >{{ t('schedule.create') }}</UButton
        >
      </div>
    </header>

    <main class="min-h-0 flex-1 overflow-y-auto px-6 pb-6 pt-2 sm:px-8">
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
        <AdaptiveSelect
          v-model="statusFilter"
          :items="statusItems"
          icon="i-tabler-adjustments-horizontal"
          class="shrink-0"
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
    </main>

    <BaseModal
      :open="!!editing"
      :title="editing?.name ?? t('schedule.create')"
      icon="i-tabler-calendar-time"
      size="4xl"
      tall
      @update:open="(open) => !open && (editing = null)"
    >
      <ScheduleEditorPanel
        v-if="editing"
        :schedule="editing"
        :workflows="workflows"
        @save="onSaveEdit"
        @cancel="editing = null"
      />
    </BaseModal>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, shallowRef, toRaw } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import { useSchedulesStore } from '@/stores/schedules'
import { useConfirm } from '@/composables/useConfirm'
import { errorMessage } from '@/lib/invoke'
import type { Schedule } from '@/lib/backend'
import { workflowTransport, type SourceView } from '@/app/transport/workflow'
import ScheduleListPanel from '@/components/schedules/ScheduleListPanel.vue'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'
import ScheduleEditorPanel from '@/components/schedules/ScheduleEditorPanel.vue'
import BaseModal from '@/components/common/BaseModal.vue'

const { t } = useI18n()
const store = useSchedulesStore()
const toast = useToast()
const { confirm } = useConfirm()
const editing = shallowRef<Schedule | null>(null)
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
  try {
    const [, sources] = await Promise.all([store.reload(), workflowTransport.listSources()])
    workflows.value = sources
  } catch (error) {
    showError(t('workflow.toast.list_failed'), error)
  }
})

async function onCreate() {
  try {
    editing.value = await store.createDraft(
      t('schedule.create_default_name', { n: store.list.length + 1 }),
    )
  } catch (error) {
    showError(t('toast.operation_failed'), error)
  }
}
function onEdit(schedule: Schedule) {
  editing.value = structuredClone(toRaw(schedule))
}
async function onSaveEdit(schedule: Schedule) {
  try {
    await store.save(schedule)
    editing.value = null
  } catch (error) {
    showError(t('toast.save_failed'), error)
  }
}
async function onToggle(schedule: Schedule, enabled: boolean) {
  try {
    await store.update(schedule.id, { enabled })
  } catch (error) {
    showError(t('toast.operation_failed'), error)
  }
}
async function onDelete(schedule: Schedule) {
  const yes = await confirm({
    title: t('schedule.delete_title'),
    description: t('schedule.delete_desc', { name: schedule.name }),
    color: 'error',
    confirmText: t('common.delete'),
  })
  if (yes !== true) return
  try {
    await store.remove(schedule.id)
  } catch (error) {
    showError(t('toast.operation_failed'), error)
  }
}

function showError(title: string, error: unknown): void {
  toast.add({ title, description: errorMessage(error), color: 'error' })
}
</script>
