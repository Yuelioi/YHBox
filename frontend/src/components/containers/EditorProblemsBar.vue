<!-- 底部「问题」条: 人类可读说明优先，错误码和节点 ID 退居技术元数据。挤画布不盖。 -->
<template>
  <div class="shrink-0 border-t border-default bg-default">
    <!-- 展开面板 (在 strip 之上): 有问题 → 扁平列表 -->
    <div
      v-if="expanded && summary.status !== 'pass'"
      id="editor-problems-panel"
      class="max-h-64 overflow-y-auto bg-elevated/10"
    >
      <div class="flex items-center gap-3 border-b border-default px-4 py-2.5">
        <span class="text-[13px] font-medium text-toned">{{ t('validation.bar_title') }}</span>
        <span v-if="summary.errorCount > 0" class="text-xs tabular-nums text-error">
          {{ t('validation.error_count', { n: summary.errorCount }) }}
        </span>
        <span v-if="summary.warnCount > 0" class="text-xs tabular-nums text-warning">
          {{ t('validation.warning_count', { n: summary.warnCount }) }}
        </span>
      </div>
      <article
        v-for="(e, idx) in errors"
        :key="idx"
        class="flex gap-3 border-b border-default px-4 py-3 last:border-b-0"
      >
        <UIcon
          :name="e.severity === 'error' ? 'i-tabler-circle-x' : 'i-tabler-alert-circle'"
          class="mt-0.5 size-4 shrink-0"
          :class="e.severity === 'error' ? 'text-error' : 'text-warning'"
        />
        <div class="min-w-0 flex-1">
          <p class="text-[13px] leading-5 text-default">{{ errorText(e) }}</p>
          <div class="mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-dimmed">
            <code class="font-mono">{{ e.code }}</code>
            <span v-if="e.graphPath.length > 0" class="truncate">{{
              e.graphPath.join(' › ')
            }}</span>
            <code v-if="e.nodeId" class="font-mono">{{ e.nodeId }}</code>
          </div>
        </div>
        <div
          v-if="e.nodeId || canFix(e) || e.code === 'MISSING_WIN32_WINDOW_TARGET'"
          class="flex shrink-0 flex-wrap items-center justify-end gap-2"
        >
          <UButton
            v-if="e.nodeId"
            size="xs"
            variant="soft"
            color="neutral"
            icon="i-tabler-focus-2"
            @click="emit('jump', e)"
            >{{ t('validation.jump') }}</UButton
          >
          <UButton
            v-if="canFix(e)"
            size="xs"
            color="primary"
            icon="i-tabler-wand"
            @click="emit('fix', e)"
            >{{ t('validation.fix') }}</UButton
          >
          <UButton
            v-if="e.code === 'MISSING_WIN32_WINDOW_TARGET'"
            size="xs"
            color="primary"
            icon="i-tabler-wand"
            @click="emit('fix-missing-win32-window-target')"
            >{{ t('validation.fix_missing_win32_window_target') }}</UButton
          >
        </div>
      </article>
    </div>
    <!-- 展开但已通过: 通过提示 + 运行 -->
    <div
      v-else-if="expanded"
      id="editor-problems-panel"
      class="flex items-center gap-2 bg-elevated/10 px-4 py-3"
    >
      <UIcon name="i-tabler-check" class="size-4 text-success" />
      <span class="text-[13px] text-muted">{{ t('validation.desc_no_issues') }}</span>
      <UButton
        class="ml-auto"
        size="xs"
        color="primary"
        icon="i-tabler-player-play"
        @click="emit('run')"
        >{{ t('validation.run_button') }}</UButton
      >
    </div>

    <!-- 收起 strip (常驻, 点切展开) -->
    <button
      type="button"
      class="flex h-8 w-full items-center gap-2 px-3 text-xs transition-colors hover:bg-elevated/30"
      :aria-expanded="expanded"
      aria-controls="editor-problems-panel"
      @click="emit('update:expanded', !expanded)"
    >
      <UIcon
        :name="
          validated && summary.status === 'pass'
            ? 'i-tabler-circle-check'
            : summary.status === 'fail'
              ? 'i-tabler-circle-x'
              : 'i-tabler-alert-circle'
        "
        class="size-3.5"
        :class="
          !validated
            ? 'text-dimmed'
            : summary.status === 'fail'
              ? 'text-error'
              : summary.status === 'warn'
                ? 'text-warning'
                : 'text-success'
        "
      />
      <span class="font-medium text-muted">{{ t('validation.bar_title') }}</span>
      <template v-if="!validated">
        <span class="text-dimmed">· {{ t('validation.unchecked') }}</span>
      </template>
      <template v-else>
        <span v-if="summary.errorCount > 0" class="tabular-nums text-error">
          · {{ t('validation.error_count', { n: summary.errorCount }) }}
        </span>
        <span v-if="summary.warnCount > 0" class="tabular-nums text-warning">
          · {{ t('validation.warning_count', { n: summary.warnCount }) }}
        </span>
        <span v-if="summary.status === 'pass'" class="text-success flex items-center gap-1">
          · {{ t('validation.passed_short') }}
        </span>
      </template>
      <div class="flex-1" />
      <UIcon
        :name="expanded ? 'i-tabler-chevron-down' : 'i-tabler-chevron-up'"
        class="size-3.5 text-dimmed"
      />
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ValidationError } from '@/lib/backend'
import { safeCoerceForFix } from '@/components/containers/inline/coerceLiteral'
import { summarizeProblems } from '@/composables/editor/problemsBar'

const props = defineProps<{
  errors: ValidationError[]
  expanded: boolean
  /** 是否已跑过校验 (区分「通过=0问题」与「还没查」)。 */
  validated: boolean
}>()
const emit = defineEmits<{
  'update:expanded': [v: boolean]
  jump: [e: ValidationError]
  fix: [e: ValidationError]
  'fix-missing-win32-window-target': []
  run: []
}>()

const { t, te } = useI18n()
const summary = computed(() => summarizeProblems(props.errors))

// canFix: 仅 LITERAL_TYPE_MISMATCH 且能安全 coerce 时显「修复」(含糊值不显)。
function canFix(e: ValidationError): boolean {
  return (
    e.code === 'LITERAL_TYPE_MISMATCH' &&
    safeCoerceForFix(e.params?.value, String(e.params?.expected ?? '')) !== undefined
  )
}
// 链: t('error.<CODE>', params) → raw code (兜底)。
function errorText(e: ValidationError): string {
  const key = `error.${e.code}`
  return te(key) ? t(key, (e.params ?? {}) as Record<string, unknown>) : e.code
}
</script>
