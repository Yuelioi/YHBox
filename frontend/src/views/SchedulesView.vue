<template>
  <div class="px-8 py-6 space-y-5">
    <header class="flex items-center gap-3">
      <UButton v-if="!editing" color="primary" icon="i-tabler-plus" @click="onCreate"
        >{{ t('schedule.create') }}</UButton
      >
      <UButton
        v-else
        variant="ghost"
        color="neutral"
        icon="i-tabler-arrow-left"
        @click="editing = null"
        >{{ t('schedule.back_to_list') }}</UButton
      >
    </header>

    <ScheduleListPanel v-if="!editing" :list="store.list" @edit="onEdit" @delete="onDelete" />

    <ScheduleEditorPanel
      v-else
      :schedule="editing"
      :containers="containersStore.list"
      @save="onSaveEdit"
      @cancel="editing = null"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import { useSchedulesStore } from '@/stores/schedules'
import { useContainersStore } from '@/stores/containers'
import { useConfirm } from '@/composables/useConfirm'
import type { Schedule } from '@/lib/backend'
import ScheduleListPanel from '@/components/schedules/ScheduleListPanel.vue'
import ScheduleEditorPanel from '@/components/schedules/ScheduleEditorPanel.vue'

const { t } = useI18n()

const store = useSchedulesStore()
const containersStore = useContainersStore()
const toast = useToast()
const { confirm } = useConfirm()

const editing = ref<Schedule | null>(null)

onMounted(() => {
  void store.reload()
  void containersStore.reload()
})

async function onCreate() {
  const draft = await store.createDraft(t('schedule.create_default_name', { n: store.list.length + 1 }))
  if (draft) editing.value = draft
}

function onEdit(sc: Schedule) {
  editing.value = JSON.parse(JSON.stringify(sc))
}

async function onSaveEdit(sc: Schedule) {
  const ok = await store.save(sc)
  if (ok) {
    editing.value = null
  } else {
    toast.add({ title: t('toast.save_failed'), color: 'error', icon: 'i-tabler-x' })
  }
}

async function onDelete(sc: Schedule) {
  const yes = await confirm({
    title: t('schedule.delete_title'),
    description: t('schedule.delete_desc', { name: sc.name }),
    color: 'error',
    confirmText: t('common.delete'),
  })
  if (yes !== true) return
  await store.remove(sc.id)
}
</script>
