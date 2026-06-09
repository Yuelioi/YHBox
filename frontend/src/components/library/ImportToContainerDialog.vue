<template>
  <BaseModal v-model:open="open" :title="dialogTitle" icon="i-tabler-arrow-bar-to-down" size="md">
    <div v-if="step === 'select'" class="space-y-4">
      <UFormField :label="t('library.import.target')">
        <USelect v-model="targetContainerID" :items="containerOptions" :placeholder="t('library.import.select_placeholder')" />
      </UFormField>
    </div>

    <div v-else-if="step === 'globals'" class="space-y-4">
      <div class="text-sm text-toned">
        {{ t('library.import.vars_needed_desc', { n: missingGlobals.length }) }}
      </div>
      <ul class="text-xs space-y-1 max-h-56 overflow-auto border border-default rounded p-2">
        <li v-for="g in missingGlobals" :key="g.name" class="flex items-center gap-2">
          <UIcon name="i-tabler-variable" class="size-3.5 text-info" />
          <strong>{{ g.name }}</strong>
          <span class="text-dimmed">({{ g.type || 'any' }})</span>
          <span v-if="g.default !== undefined && g.default !== null" class="text-dimmed">
            = {{ JSON.stringify(g.default) }}
          </span>
        </li>
      </ul>
    </div>

    <div v-else-if="step === 'done'" class="text-sm text-success flex items-center gap-2">
      <UIcon name="i-tabler-circle-check" class="size-5" />
      {{ addedGlobalsCount ? t('library.import.complete_full', { count: importedCount, varsCount: addedGlobalsCount }) : t('library.import.complete_simple', { count: importedCount }) }}
    </div>

    <template #footer>
      <template v-if="step === 'select'">
        <UButton variant="ghost" color="neutral" @click="cancel">{{ t('common.cancel') }}</UButton>
        <UButton :disabled="!targetContainerID" @click="doImport" :loading="busy">{{ t('library.import.next') }}</UButton>
      </template>
      <template v-else-if="step === 'globals'">
        <UButton variant="ghost" color="neutral" @click="cancel">{{ t('common.cancel') }}</UButton>
        <UButton color="primary" @click="doAddGlobals" :loading="busy">{{ t('library.import.add_vars_confirm') }}</UButton>
      </template>
      <UButton v-else-if="step === 'done'" @click="cancel">{{ t('library.import.done') }}</UButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend, type ImportResult, type SubgraphRequiredGlobal, type VarDecl } from '@/lib/backend'
import { useContainersStore } from '@/stores/containers'
import { useToast } from '@nuxt/ui/composables'
import { errorMessage } from '@/lib/invoke'
import BaseModal from '@/components/common/BaseModal.vue'

const { t } = useI18n()

const props = defineProps<{ open: boolean; libSgId: string }>()
const emit = defineEmits<{ 'update:open': [v: boolean] }>()

const open = computed({
  get: () => props.open,
  set: v => emit('update:open', v),
})

const dialogTitle = computed(
  () => `${t('library.import.title_prefix')} ${props.libSgId} ${t('library.import.title_suffix')}`,
)

const step = ref<'select' | 'globals' | 'done'>('select')
const targetContainerID = ref('')
const missingGlobals = ref<SubgraphRequiredGlobal[]>([])
const importedCount = ref(0)
const addedGlobalsCount = ref(0)
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
    missingGlobals.value = []
    importedCount.value = 0
    addedGlobalsCount.value = 0
    busy.value = false
    void containersStore.reload()
  }
})

// 单次幂等导入: 资产按 GUID/sha 合并、子图直接写入, 无 conflict/strategy.
// 导入即发生; 返回的 missingGlobals 是 import 的 sg 需要但目标容器未声明的 var, 据此提示补充.
async function doImport() {
  if (!targetContainerID.value) return
  busy.value = true
  try {
    const result = await backend.library.importToContainer(props.libSgId, targetContainerID.value)
    if (!result) return
    const r = result as ImportResult
    importedCount.value = (r.imported ?? []).length
    missingGlobals.value = r.missingGlobals ?? []
    step.value = missingGlobals.value.length > 0 ? 'globals' : 'done'
  } catch (e: any) {
    toast.add({ title: t('toast.import_failed'), description: errorMessage(e), color: 'error' })
  } finally {
    busy.value = false
  }
}

// B11: 导入已完成, 这步只把 missing globals 追加到目标容器 Vars (var 声明跟资产正交, 顺序无关).
async function doAddGlobals() {
  busy.value = true
  try {
    const c = await backend.containers.get(targetContainerID.value)
    if (!c) {
      toast.add({ title: t('toast.container_not_found'), color: 'error' })
      return
    }
    const existing = (c.vars ?? []) as VarDecl[]
    const existingNames = new Set(existing.map(v => v.name))
    const toAdd: VarDecl[] = missingGlobals.value
      .filter(g => !existingNames.has(g.name))
      .map(g => ({
        name: g.name,
        type: (g.type || 'any') as VarDecl['type'],
        default: (g.default ?? null) as VarDecl['default'],
      }))
    if (toAdd.length > 0) {
      const nextVars = [...existing, ...toAdd]
      await backend.containers.update(targetContainerID.value, JSON.stringify({ vars: nextVars }))
      addedGlobalsCount.value = toAdd.length
    } else {
      addedGlobalsCount.value = 0 // race: 别人补过了
    }
    step.value = 'done'
  } catch (e: any) {
    toast.add({ title: t('library.import.add_vars_failed'), description: errorMessage(e), color: 'error' })
  } finally {
    busy.value = false
  }
}

function cancel() {
  open.value = false
}
</script>
