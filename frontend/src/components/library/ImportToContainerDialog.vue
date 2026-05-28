<template>
  <UModal v-model:open="open">
    <UCard>
      <template #header>
        <div class="text-sm font-medium">{{ t('library.import.title_prefix') }} <strong>{{ libSgId }}</strong> {{ t('library.import.title_suffix') }}</div>
      </template>

      <div v-if="step === 'select'" class="space-y-4">
        <UFormField :label="t('library.import.target')">
          <USelect v-model="targetContainerID" :items="containerOptions" :placeholder="t('library.import.select_placeholder')" />
        </UFormField>
        <div class="flex gap-2 justify-end">
          <UButton variant="ghost" color="neutral" @click="cancel">{{ t('common.cancel') }}</UButton>
          <UButton :disabled="!targetContainerID" @click="doDryRun" :loading="busy">{{ t('library.import.next') }}</UButton>
        </div>
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
        <div class="flex gap-2 justify-end">
          <UButton variant="ghost" color="neutral" @click="cancel">{{ t('common.cancel') }}</UButton>
          <UButton color="primary" @click="doAddGlobalsAndImport" :loading="busy">{{ t('library.import.add_vars_confirm') }}</UButton>
        </div>
      </div>

      <div v-else-if="step === 'conflicts'" class="space-y-4">
        <div class="text-sm text-toned">{{ t('library.import.conflicts_detected', { n: conflicts.length }) }}</div>
        <ul class="text-xs space-y-1 max-h-56 overflow-auto border border-default rounded p-2">
          <li v-for="c in conflicts" :key="c.kind + ':' + c.key" class="text-dimmed">
            <span class="text-error">✗</span>
            <strong class="ml-1">{{ c.kind }}</strong>: {{ c.key }}
          </li>
        </ul>
        <div class="flex gap-2 justify-end">
          <UButton color="neutral" variant="ghost" @click="cancel">{{ t('common.cancel') }}</UButton>
          <UButton @click="resolve('skip_all')" :loading="busy">{{ t('library.import.skip_all') }}</UButton>
          <UButton color="primary" @click="resolve('overwrite_all')" :loading="busy">{{ t('library.import.overwrite_all') }}</UButton>
        </div>
      </div>

      <div v-else-if="step === 'done'" class="space-y-4">
        <div class="text-sm text-success flex items-center gap-2">
          <UIcon name="i-tabler-circle-check" class="size-5" />
          {{ addedGlobalsCount ? t('library.import.complete_full', { count: importedCount, varsCount: addedGlobalsCount }) : t('library.import.complete_simple', { count: importedCount }) }}
        </div>
        <div class="flex justify-end">
          <UButton @click="cancel">{{ t('library.import.done') }}</UButton>
        </div>
      </div>
    </UCard>
  </UModal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend, type ImportConflict, type ImportResult, type SubgraphRequiredGlobal, type VarDecl } from '@/lib/backend'
import { useContainersStore } from '@/stores/containers'
import { useToast } from '@nuxt/ui/composables'

const { t } = useI18n()

const props = defineProps<{ open: boolean; libSgId: string }>()
const emit = defineEmits<{ 'update:open': [v: boolean] }>()

const open = computed({
  get: () => props.open,
  set: v => emit('update:open', v),
})

const step = ref<'select' | 'globals' | 'conflicts' | 'done'>('select')
const targetContainerID = ref('')
const conflicts = ref<ImportConflict[]>([])
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
    conflicts.value = []
    missingGlobals.value = []
    importedCount.value = 0
    addedGlobalsCount.value = 0
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
    missingGlobals.value = r.missingGlobals ?? []
    conflicts.value = r.conflicts ?? []
    // B11 优先级: globals > conflicts > 直接 import (无任何 prompt 时)
    if (missingGlobals.value.length > 0) {
      step.value = 'globals'
    } else if (conflicts.value.length > 0) {
      step.value = 'conflicts'
    } else {
      await doRealImport('')
    }
  } catch (e: any) {
    toast.add({ title: t('toast.import_failed'), description: String(e?.message ?? e), color: 'error' })
  } finally {
    busy.value = false
  }
}

// B11: 用 container.update RPC 把 missing globals 追加到容器 Vars 然后继续 import.
async function doAddGlobalsAndImport() {
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
    if (toAdd.length === 0) {
      // race: 别人补过了, 直接进 conflicts/done
      addedGlobalsCount.value = 0
    } else {
      const nextVars = [...existing, ...toAdd]
      await backend.containers.update(targetContainerID.value, JSON.stringify({ vars: nextVars }))
      addedGlobalsCount.value = toAdd.length
    }
    // globals 已加, 看 conflicts 决定下一步
    if (conflicts.value.length > 0) {
      step.value = 'conflicts'
    } else {
      await doRealImport('')
    }
  } catch (e: any) {
    toast.add({ title: t('library.import.add_vars_failed'), description: String(e?.message ?? e), color: 'error' })
  } finally {
    busy.value = false
  }
}

async function resolve(strategy: 'overwrite_all' | 'skip_all') {
  await doRealImport(strategy)
}

async function doRealImport(strategy: string) {
  busy.value = true
  try {
    const result = await backend.library.importToContainer(props.libSgId, targetContainerID.value, strategy)
    importedCount.value = ((result as ImportResult)?.imported ?? []).length
    step.value = 'done'
  } catch (e: any) {
    toast.add({ title: t('toast.import_failed'), description: String(e?.message ?? e), color: 'error' })
  } finally {
    busy.value = false
  }
}

function cancel() {
  open.value = false
}
</script>
