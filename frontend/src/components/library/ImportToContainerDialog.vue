<template>
  <UModal v-model:open="open">
    <UCard>
      <template #header>
        <div class="text-sm font-medium">导入 <strong>{{ libSgId }}</strong> 到容器</div>
      </template>

      <div v-if="step === 'select'" class="space-y-4">
        <UFormField label="目标容器">
          <USelect v-model="targetContainerID" :items="containerOptions" placeholder="选择容器..." />
        </UFormField>
        <div class="flex gap-2 justify-end">
          <UButton variant="ghost" color="neutral" @click="cancel">取消</UButton>
          <UButton :disabled="!targetContainerID" @click="doDryRun" :loading="busy">下一步</UButton>
        </div>
      </div>

      <div v-else-if="step === 'conflicts'" class="space-y-4">
        <div class="text-sm text-toned">检测到 {{ conflicts.length }} 个冲突：</div>
        <ul class="text-xs space-y-1 max-h-56 overflow-auto border border-default rounded p-2">
          <li v-for="c in conflicts" :key="c.kind + ':' + c.key" class="text-dimmed">
            <span class="text-error">✗</span>
            <strong class="ml-1">{{ c.kind }}</strong>: {{ c.key }}
          </li>
        </ul>
        <div class="flex gap-2 justify-end">
          <UButton color="neutral" variant="ghost" @click="cancel">取消</UButton>
          <UButton @click="resolve('skip_all')" :loading="busy">全部跳过</UButton>
          <UButton color="primary" @click="resolve('overwrite_all')" :loading="busy">全部覆盖</UButton>
        </div>
      </div>

      <div v-else-if="step === 'done'" class="space-y-4">
        <div class="text-sm text-success flex items-center gap-2">
          <UIcon name="i-tabler-circle-check" class="size-5" />
          导入完成（{{ importedCount }} 项）
        </div>
        <div class="flex justify-end">
          <UButton @click="cancel">关闭</UButton>
        </div>
      </div>
    </UCard>
  </UModal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { backend, type ImportConflict, type ImportResult } from '@/lib/backend'
import { useContainersStore } from '@/stores/containers'
import { useToast } from '@nuxt/ui/composables'

const props = defineProps<{ open: boolean; libSgId: string }>()
const emit = defineEmits<{ 'update:open': [v: boolean] }>()

const open = computed({
  get: () => props.open,
  set: v => emit('update:open', v),
})

const step = ref<'select' | 'conflicts' | 'done'>('select')
const targetContainerID = ref('')
const conflicts = ref<ImportConflict[]>([])
const importedCount = ref(0)
const busy = ref(false)

const containersStore = useContainersStore()
const toast = useToast()

const containerOptions = computed(() =>
  containersStore.list.map(c => ({ label: c.name || c.id, value: c.id })),
)

watch(() => props.open, (v) => {
  if (v) {
    step.value = 'select'
    targetContainerID.value = ''
    conflicts.value = []
    importedCount.value = 0
    busy.value = false
    void containersStore.reload()
  }
})

async function doDryRun() {
  if (!targetContainerID.value) return
  busy.value = true
  try {
    const result = await backend.library.importToContainer(props.libSgId, targetContainerID.value, 'cancel')
    if (!result) return
    const r = result as ImportResult
    if (r.conflicts && r.conflicts.length > 0) {
      conflicts.value = r.conflicts
      step.value = 'conflicts'
    } else {
      const r2 = await backend.library.importToContainer(props.libSgId, targetContainerID.value, '')
      importedCount.value = ((r2 as ImportResult)?.imported ?? []).length
      step.value = 'done'
    }
  } catch (e: any) {
    toast.add({ title: '导入失败', description: String(e?.message ?? e), color: 'error' })
  } finally {
    busy.value = false
  }
}

async function resolve(strategy: 'overwrite_all' | 'skip_all') {
  busy.value = true
  try {
    const result = await backend.library.importToContainer(props.libSgId, targetContainerID.value, strategy)
    importedCount.value = ((result as ImportResult)?.imported ?? []).length
    step.value = 'done'
  } catch (e: any) {
    toast.add({ title: '导入失败', description: String(e?.message ?? e), color: 'error' })
  } finally {
    busy.value = false
  }
}

function cancel() {
  open.value = false
}
</script>
