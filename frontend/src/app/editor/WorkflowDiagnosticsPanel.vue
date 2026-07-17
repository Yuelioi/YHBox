<template>
  <section
    data-testid="workflow-diagnostics"
    class="max-h-60 shrink-0 overflow-y-auto border-b border-default bg-default"
    :aria-label="t('workflow.diagnostics.title')"
  >
    <header
      class="sticky top-0 z-10 flex items-center border-b border-default bg-default px-4 py-2"
    >
      <div class="min-w-0 flex-1">
        <h2 class="text-xs font-semibold text-highlighted">
          {{ t('workflow.diagnostics.title') }}
        </h2>
        <p class="mt-0.5 text-[10px] text-dimmed">
          {{ t('workflow.diagnostics.summary', { n: diagnostics.length }) }}
        </p>
      </div>
      <UButton
        icon="i-tabler-x"
        color="neutral"
        variant="ghost"
        size="xs"
        :aria-label="t('workflow.diagnostics.close')"
        @click="emit('close')"
      />
    </header>

    <div class="grid gap-3 px-4 py-3 lg:grid-cols-3">
      <section v-for="group in groups" :key="group.severity" class="min-w-0">
        <h3
          class="mb-1.5 flex items-center gap-2 text-[10px] font-semibold uppercase tracking-wide"
          :class="severityText[group.severity]"
        >
          {{ t(`workflow.diagnostics.${group.severity}`) }}
          <UBadge :color="severityColor[group.severity]" variant="soft" size="xs">{{
            group.diagnostics.length
          }}</UBadge>
        </h3>
        <div class="space-y-1.5">
          <UButton
            v-for="(diagnostic, index) in group.diagnostics"
            :key="`${diagnostic.code}:${index}`"
            color="neutral"
            variant="ghost"
            class="h-auto w-full justify-start border border-default px-2.5 py-2 text-left"
            @click="emit('focus', diagnostic)"
          >
            <span class="min-w-0 flex-1">
              <span class="block truncate text-[11px] font-medium text-toned">{{
                diagnosticMessage(diagnostic)
              }}</span>
              <span
                class="mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-1 font-mono text-[9px] text-dimmed"
              >
                <span>{{ diagnostic.code }}</span>
                <span v-if="diagnostic.nodeId">{{ diagnostic.nodeId }}</span>
                <span v-if="diagnosticFieldLocation(diagnostic)">{{
                  diagnosticFieldLocation(diagnostic)
                }}</span>
              </span>
              <span
                v-if="diagnostic.fix"
                class="mt-1 flex items-center gap-1 text-[9px] text-primary"
              >
                <UIcon name="i-tabler-wand" class="size-3" />
                {{ t(`workflow.diagnostics.fix_${diagnostic.fix.kind}`) }}
              </span>
            </span>
          </UButton>
        </div>
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  diagnosticFieldLocation,
  groupDiagnostics,
  type DiagnosticSeverity,
  type WorkflowDiagnostic,
} from './workflowDiagnostics'

const props = defineProps<{ diagnostics: WorkflowDiagnostic[] }>()
const emit = defineEmits<{ close: []; focus: [diagnostic: WorkflowDiagnostic] }>()
const { t, te } = useI18n()

const groups = computed(() => groupDiagnostics(props.diagnostics))
const severityColor: Record<DiagnosticSeverity, 'error' | 'warning' | 'info'> = {
  error: 'error',
  warning: 'warning',
  info: 'info',
}
const severityText: Record<DiagnosticSeverity, string> = {
  error: 'text-error',
  warning: 'text-warning',
  info: 'text-info',
}

function diagnosticMessage(diagnostic: WorkflowDiagnostic): string {
  const key = `error.${diagnostic.code}`
  if (te(key)) return t(key, diagnostic.params ?? {})
  return diagnostic.message || diagnostic.code
}
</script>
