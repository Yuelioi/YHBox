<template>
  <div class="flex min-h-full flex-col bg-default">
    <header class="flex items-end gap-6 border-b border-default px-8 py-6">
      <div class="min-w-0 flex-1">
        <h1 class="text-xl font-semibold tracking-tight text-highlighted">
          {{ t('workflow31.list.title') }}
        </h1>
        <p class="mt-1 max-w-2xl text-sm text-muted">
          {{ t('workflow31.list.description') }}
        </p>
      </div>
      <form class="flex items-end gap-2" @submit.prevent="createWorkflow">
        <UFormField :label="t('workflow31.list.new_workflow')" class="w-64">
          <UInput
            v-model="newName"
            :placeholder="t('workflow31.list.name_placeholder')"
            class="w-full"
          />
        </UFormField>
        <UButton
          type="submit"
          :label="t('workflow31.list.create')"
          icon="i-tabler-plus"
          :loading="creating"
          :disabled="!newName.trim()"
        />
      </form>
    </header>

    <main class="flex-1 px-8 py-6">
      <div v-if="loading" class="space-y-2" :aria-label="t('workflow31.list.loading')">
        <USkeleton v-for="index in 4" :key="index" class="h-16 w-full rounded-lg" />
      </div>

      <div
        v-else-if="failure"
        class="rounded-lg border border-error/35 bg-error/10 px-4 py-3 text-sm text-error"
        role="alert"
      >
        {{ failure }}
      </div>

      <div
        v-else-if="sources.length === 0"
        class="flex min-h-72 items-center justify-center rounded-lg border border-dashed border-default bg-elevated/20 px-8 text-center"
      >
        <div class="max-w-sm">
          <UIcon name="i-tabler-route" class="mx-auto mb-4 size-8 text-primary" />
          <h2 class="text-sm font-semibold text-highlighted">
            {{ t('workflow31.list.empty_title') }}
          </h2>
          <p class="mt-2 text-xs leading-5 text-muted">
            {{ t('workflow31.list.empty_description') }}
          </p>
        </div>
      </div>

      <div v-else class="overflow-hidden rounded-lg border border-default">
        <div
          class="grid grid-cols-[minmax(220px,1fr)_88px_minmax(180px,0.8fr)_170px] gap-4 bg-elevated/60 px-4 py-2 text-[11px] font-medium text-muted"
        >
          <span>{{ t('workflow31.list.name') }}</span>
          <span>{{ t('workflow31.list.revision') }}</span>
          <span>{{ t('workflow31.list.source_identity') }}</span>
          <span class="text-right">{{ t('workflow31.list.actions') }}</span>
        </div>
        <div class="divide-y divide-default">
          <article
            v-for="source in sources"
            :key="source.workflowId"
            class="grid grid-cols-[minmax(220px,1fr)_88px_minmax(180px,0.8fr)_170px] items-center gap-4 px-4 py-3 transition-colors hover:bg-elevated/35"
          >
            <div class="min-w-0">
              <p class="truncate text-sm font-medium text-highlighted">{{ source.name }}</p>
              <p class="mt-0.5 truncate font-mono text-[10px] text-dimmed">
                {{ source.workflowId }}
              </p>
            </div>
            <span class="font-mono text-xs tabular-nums text-toned">{{ source.revision }}</span>
            <span class="truncate font-mono text-[10px] text-dimmed">{{ source.sourceHash }}</span>
            <div class="flex justify-end gap-2">
              <UButton
                :label="t('workflow31.action.run')"
                icon="i-tabler-player-play"
                color="neutral"
                variant="ghost"
                size="xs"
                @click="runWorkflow(source.workflowId)"
              />
              <UButton
                :label="t('workflow31.action.edit')"
                icon="i-tabler-schema"
                size="xs"
                @click="router.push(`/workflows/${source.workflowId}/edit`)"
              />
            </div>
          </article>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from '@nuxt/ui/composables'
import { useI18n } from 'vue-i18n'
import { workflowTransport, type SourceView } from '@/app/transport/workflow31'

defineOptions({ name: 'WorkflowsView' })

const router = useRouter()
const toast = useToast()
const { t } = useI18n()
const sources = ref<SourceView[]>([])
const loading = ref(true)
const creating = ref(false)
const newName = ref('')
const failure = ref('')

onMounted(load)

async function load(): Promise<void> {
  loading.value = true
  failure.value = ''
  try {
    sources.value = await workflowTransport.listSources()
  } catch (error) {
    failure.value = errorText(error)
  } finally {
    loading.value = false
  }
}

async function createWorkflow(): Promise<void> {
  const name = newName.value.trim()
  if (!name || creating.value) return
  creating.value = true
  try {
    const created = await workflowTransport.createSource(name)
    newName.value = ''
    await router.push(`/workflows/${created.workflowId}/edit`)
  } catch (error) {
    toast.add({
      title: t('workflow31.toast.create_failed'),
      description: errorText(error),
      color: 'error',
    })
  } finally {
    creating.value = false
  }
}

async function runWorkflow(workflowId: string): Promise<void> {
  try {
    const started = await workflowTransport.startRun(workflowId)
    if (!started.run) {
      toast.add({
        title: t('workflow31.toast.not_started'),
        description: diagnosticText(started),
        color: 'warning',
      })
      return
    }
    toast.add({
      title: t('workflow31.toast.queued'),
      description: started.run.runId,
      color: 'success',
    })
  } catch (error) {
    toast.add({
      title: t('workflow31.toast.run_failed'),
      description: errorText(error),
      color: 'error',
    })
  }
}

function diagnosticText(value: { diagnostics: Array<{ code: string }> }): string {
  return (
    value.diagnostics.map((diagnostic) => diagnostic.code).join(', ') ||
    t('workflow31.toast.no_run_created')
  )
}

function errorText(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}
</script>
