<template>
  <BaseModal
    :open="open"
    :title="t('workflow.installation.update_title', { name })"
    icon="i-tabler-refresh"
    size="3xl"
    :dismissible="!applying"
    @update:open="emit('update:open', $event)"
  >
    <div v-if="loading" class="space-y-3" :aria-label="t('common.loading')">
      <USkeleton v-for="index in 4" :key="index" class="h-20 rounded-lg" />
    </div>

    <div v-else class="space-y-5" data-testid="workflow-installation-update-body">
      <p class="text-xs text-muted">{{ t('workflow.installation.update_description') }}</p>

      <div
        v-if="failure"
        class="rounded-lg border border-error/35 bg-error/10 px-4 py-3 text-sm text-error"
        role="alert"
      >
        {{ failure }}
      </div>

      <section class="space-y-3 rounded-lg border border-default bg-elevated/15 p-4">
        <div class="flex flex-wrap items-end gap-3">
          <UFormField :label="t('workflow.installation.update_candidate')" class="min-w-64 flex-1">
            <USelect
              v-model="candidateReleaseId"
              :items="candidateItems"
              value-key="value"
              label-key="label"
              :placeholder="t('workflow.installation.update_candidate_placeholder')"
              :disabled="previewing || applying"
            />
          </UFormField>
          <UButton
            data-testid="workflow-installation-preview-update"
            color="neutral"
            variant="soft"
            icon="i-tabler-file-diff"
            :label="t('workflow.installation.update_preview')"
            :loading="previewing && previewMode === 'update'"
            :disabled="!candidateReleaseId || previewing || applying"
            @click="previewUpdate"
          />
          <UButton
            v-if="hasRollback"
            data-testid="workflow-installation-preview-rollback"
            color="warning"
            variant="soft"
            icon="i-tabler-history"
            :label="t('workflow.installation.rollback_preview')"
            :loading="previewing && previewMode === 'rollback'"
            :disabled="previewing || applying"
            @click="previewRollback"
          />
        </div>
        <p v-if="candidates.length === 0" class="text-xs text-dimmed">
          {{ t('workflow.installation.update_empty') }}
        </p>
      </section>

      <template v-if="preview">
        <section class="rounded-lg border border-primary/30 bg-primary/5 p-4">
          <div class="flex flex-wrap items-center gap-2">
            <UBadge :color="preview.rollback ? 'warning' : 'primary'" variant="soft">
              {{
                t(
                  preview.rollback
                    ? 'workflow.installation.rollback_badge'
                    : 'workflow.installation.update_badge',
                )
              }}
            </UBadge>
            <span class="font-mono text-xs text-muted">{{ preview.diff.currentVersion }}</span>
            <UIcon name="i-tabler-arrow-right" class="size-4 text-dimmed" />
            <span class="font-mono text-xs font-semibold text-highlighted">
              {{ preview.diff.candidateVersion }}
            </span>
          </div>
          <p class="mt-2 break-all text-[10px] text-dimmed">
            {{ preview.diff.currentReleaseId }} → {{ preview.diff.candidateReleaseId }}
          </p>
        </section>

        <section class="grid gap-3 sm:grid-cols-3">
          <DiffCard
            :title="t('workflow.installation.update_content_changes')"
            icon="i-tabler-topology-star-3"
            :items="contentChanges"
          />
          <DiffCard
            :title="t('workflow.installation.update_local_changes')"
            icon="i-tabler-adjustments"
            :items="localChanges"
          />
          <DiffCard
            :title="t('workflow.installation.update_dependency_changes')"
            icon="i-tabler-packages"
            :items="dependencyChanges"
          />
        </section>

        <section aria-labelledby="installation-capability-diff-title">
          <div class="mb-2 flex items-center gap-2">
            <UIcon name="i-tabler-shield-lock" class="size-4 text-warning" />
            <h4
              id="installation-capability-diff-title"
              class="text-xs font-semibold text-highlighted"
            >
              {{ t('workflow.installation.update_permissions') }}
            </h4>
          </div>
          <div
            v-if="
              preview.diff.addedCapabilities.length === 0 &&
              preview.diff.removedCapabilities.length === 0
            "
            class="rounded-lg border border-default px-4 py-4 text-xs text-dimmed"
          >
            {{ t('workflow.installation.update_permissions_unchanged') }}
          </div>
          <div v-else class="grid gap-3 lg:grid-cols-2">
            <CapabilityList
              tone="added"
              :title="t('workflow.installation.update_permissions_added')"
              :items="preview.diff.addedCapabilities"
            />
            <CapabilityList
              tone="removed"
              :title="t('workflow.installation.update_permissions_removed')"
              :items="preview.diff.removedCapabilities"
            />
          </div>
        </section>

        <div
          v-if="preview.conflicts.length"
          class="rounded-lg border border-error/35 bg-error/10 px-4 py-3 text-xs text-error"
          role="alert"
        >
          <p class="font-semibold">{{ t('workflow.installation.update_conflicts') }}</p>
          <ul class="mt-1 list-disc space-y-1 pl-4">
            <li
              v-for="conflict in preview.conflicts"
              :key="`${conflict.kind}:${conflict.requirementId}`"
            >
              {{ conflict.kind }} · {{ conflict.requirementId }}
            </li>
          </ul>
        </div>

        <div
          class="rounded-lg border border-warning/35 bg-warning/10 px-4 py-3 text-xs text-warning"
        >
          {{ t('workflow.installation.update_consent_warning') }}
          <span v-if="preview.readiness.blockers.length">
            {{
              t('workflow.installation.update_blocker_count', {
                n: preview.readiness.blockers.length,
              })
            }}
          </span>
        </div>
      </template>
    </div>

    <template #footer>
      <UButton
        color="neutral"
        variant="ghost"
        :disabled="applying"
        :label="t('common.cancel')"
        @click="emit('update:open', false)"
      />
      <UButton
        data-testid="workflow-installation-apply-update"
        :color="preview?.rollback ? 'warning' : 'primary'"
        icon="i-tabler-check"
        :label="
          t(
            preview?.rollback
              ? 'workflow.installation.rollback_apply'
              : 'workflow.installation.update_apply',
          )
        "
        :loading="applying"
        :disabled="!preview || preview.conflicts.length > 0 || previewing"
        @click="applyPreview"
      />
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useToast } from '@nuxt/ui/composables'
import { useI18n } from 'vue-i18n'
import {
  workflowTransport,
  type InstallationUpdateCandidateView,
  type InstallationUpdatePreviewView,
} from '@/app/transport/workflow'
import { errorMessage } from '@/lib/invoke'
import BaseModal from '@/components/common/BaseModal.vue'
import DiffCard from './WorkflowInstallationUpdateDiffCard.vue'
import CapabilityList from './WorkflowInstallationCapabilityList.vue'

const props = defineProps<{ open: boolean; installationId: string; name: string }>()
const emit = defineEmits<{
  'update:open': [value: boolean]
  applied: []
}>()
const { t } = useI18n()
const toast = useToast()
const candidates = ref<InstallationUpdateCandidateView[]>([])
const candidateReleaseId = ref('')
const preview = ref<InstallationUpdatePreviewView | null>(null)
const loading = ref(false)
const previewing = ref(false)
const previewMode = ref<'update' | 'rollback'>('update')
const applying = ref(false)
const failure = ref('')

const hasRollback = computed(() =>
  candidates.value.some((candidate) => candidate.immediatePrevious),
)
const candidateItems = computed(() =>
  candidates.value
    .filter((candidate) => !candidate.immediatePrevious)
    .map((candidate) => ({
      value: candidate.releaseId,
      label: `${candidate.releaseVersion} · ${shortDigest(candidate.releaseId)}`,
    })),
)
const contentChanges = computed(() => {
  if (!preview.value) return []
  const diff = preview.value.diff
  return [
    diff.graphsChanged ? t('workflow.installation.update_graphs') : '',
    diff.resourcesChanged ? t('workflow.installation.update_resources') : '',
    diff.variablesChanged ? t('workflow.installation.update_variables') : '',
  ].filter(Boolean)
})
const localChanges = computed(() => {
  if (!preview.value) return []
  const diff = preview.value.diff
  return [
    ...diff.addedTargets.map((id) => `+ target · ${id}`),
    ...diff.changedTargets.map((id) => `~ target · ${id}`),
    ...diff.removedTargets.map((id) => `− target · ${id}`),
    ...diff.addedCredentials.map((id) => `+ credential · ${id}`),
    ...diff.changedCredentials.map((id) => `~ credential · ${id}`),
    ...diff.removedCredentials.map((id) => `− credential · ${id}`),
  ]
})
const dependencyChanges = computed(() => {
  if (!preview.value) return []
  return [
    ...preview.value.diff.addedDependencies.map(
      (item) => `+ ${item.packageId}@${item.packageVersion}`,
    ),
    ...preview.value.diff.removedDependencies.map(
      (item) => `− ${item.packageId}@${item.packageVersion}`,
    ),
  ]
})

watch(
  () => props.open,
  (open) => {
    if (open) void load()
    else reset()
  },
  { immediate: true },
)

async function load(): Promise<void> {
  loading.value = true
  failure.value = ''
  preview.value = null
  try {
    candidates.value = await workflowTransport.listInstallationUpdateCandidates(
      props.installationId,
    )
    candidateReleaseId.value =
      candidates.value.find((candidate) => !candidate.immediatePrevious)?.releaseId ?? ''
  } catch (error) {
    failure.value = errorMessage(error)
  } finally {
    loading.value = false
  }
}

async function previewUpdate(): Promise<void> {
  if (!candidateReleaseId.value || previewing.value) return
  previewMode.value = 'update'
  await stage(() =>
    workflowTransport.previewInstallationUpdate(props.installationId, candidateReleaseId.value),
  )
}

async function previewRollback(): Promise<void> {
  if (previewing.value) return
  previewMode.value = 'rollback'
  await stage(() => workflowTransport.previewInstallationRollback(props.installationId))
}

async function stage(action: () => Promise<InstallationUpdatePreviewView>): Promise<void> {
  previewing.value = true
  failure.value = ''
  preview.value = null
  try {
    preview.value = await action()
  } catch (error) {
    failure.value = errorMessage(error)
  } finally {
    previewing.value = false
  }
}

async function applyPreview(): Promise<void> {
  if (!preview.value || preview.value.conflicts.length || applying.value) return
  applying.value = true
  failure.value = ''
  try {
    const result = await workflowTransport.applyInstallationUpdate(preview.value.token)
    if (result.reconciliationRequired) {
      toast.add({
        title: t('workflow.installation.update_reconciliation_title'),
        description: t('workflow.installation.update_reconciliation_description'),
        color: 'warning',
      })
    }
    emit('applied')
    emit('update:open', false)
  } catch (error) {
    failure.value = errorMessage(error)
    preview.value = null
  } finally {
    applying.value = false
  }
}

function reset(): void {
  candidates.value = []
  candidateReleaseId.value = ''
  preview.value = null
  failure.value = ''
}

function shortDigest(value: string): string {
  return value.replace(/^sha256:/, '').slice(0, 10)
}
</script>
