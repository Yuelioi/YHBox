<!-- 底部「问题」条 (VS Code Problems 式): 收起=状态徽章; 展开=错误列表 (跳转/修复)。挤画布不盖。 -->
<template>
  <div class="shrink-0 border-t border-default bg-default">
    <!-- 展开面板 (在 strip 之上): 有错误 → 列表 -->
    <div v-if="expanded && summary.status !== 'pass'" class="max-h-64 overflow-y-auto px-3 py-2 space-y-2">
      <div
        v-for="(e, idx) in errors"
        :key="idx"
        class="rounded-md border px-3 py-2 space-y-1"
        :class="e.severity === 'error' ? 'border-error/40 bg-error/5' : 'border-warning/40 bg-warning/5'"
      >
        <div class="flex items-center gap-2 text-xs">
          <UIcon :name="e.severity === 'error' ? 'i-tabler-alert-triangle' : 'i-tabler-info-circle'"
                 class="size-3.5 shrink-0" :class="e.severity === 'error' ? 'text-error' : 'text-warning'" />
          <code class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-elevated/60"
                :class="e.severity === 'error' ? 'text-error' : 'text-warning'">{{ e.code }}</code>
          <span v-if="e.graphPath.length > 0" class="text-[10px] text-dimmed truncate">{{ e.graphPath.join(' › ') }}</span>
        </div>
        <div class="text-xs text-toned leading-relaxed pl-5">{{ errorText(e) }}</div>
        <div v-if="e.nodeId || canFix(e) || e.code === 'MISSING_WIN32_WINDOW_TARGET'"
             class="flex flex-wrap items-center gap-2 pl-5 pt-1">
          <code v-if="e.nodeId" class="font-mono text-[10px] text-dimmed">{{ e.nodeId }}</code>
          <UButton v-if="e.nodeId" size="xs" variant="soft" color="neutral" icon="i-tabler-focus-2"
                   @click="emit('jump', e)">{{ t('validation.jump') }}</UButton>
          <UButton v-if="canFix(e)" size="xs" color="primary" icon="i-tabler-wand"
                   @click="emit('fix', e)">{{ t('validation.fix') }}</UButton>
          <UButton v-if="e.code === 'MISSING_WIN32_WINDOW_TARGET'" size="xs" color="primary" icon="i-tabler-wand"
                   @click="emit('fix-missing-win32-window-target')">{{ t('validation.fix_missing_win32_window_target') }}</UButton>
        </div>
      </div>
    </div>
    <!-- 展开但已通过: 通过提示 + 运行 -->
    <div v-else-if="expanded" class="px-3 py-2 flex items-center gap-2">
      <UIcon name="i-tabler-check" class="size-4 text-success" />
      <span class="text-xs text-muted">{{ t('validation.desc_no_issues') }}</span>
      <UButton class="ml-auto" size="xs" color="primary" icon="i-tabler-player-play"
               @click="emit('run')">{{ t('validation.run_button') }}</UButton>
    </div>

    <!-- 收起 strip (常驻, 点切展开) -->
    <button type="button"
            class="w-full h-7 px-3 flex items-center gap-2 text-[11px] hover:bg-elevated/30 transition-colors"
            @click="emit('update:expanded', !expanded)">
      <UIcon name="i-tabler-alert-circle" class="size-3.5 text-dimmed" />
      <span class="text-dimmed">{{ t('validation.bar_title') }}</span>
      <template v-if="!validated">
        <span class="text-dimmed">· {{ t('validation.unchecked') }}</span>
      </template>
      <template v-else>
        <UBadge v-if="summary.errorCount > 0" size="xs" color="error" variant="soft">{{ summary.errorCount }}</UBadge>
        <UBadge v-if="summary.warnCount > 0" size="xs" color="warning" variant="soft">{{ summary.warnCount }}</UBadge>
        <span v-if="summary.status === 'pass'" class="text-success flex items-center gap-1">
          <UIcon name="i-tabler-check" class="size-3.5" />{{ t('validation.passed_short') }}
        </span>
      </template>
      <div class="flex-1" />
      <UIcon :name="expanded ? 'i-tabler-chevron-down' : 'i-tabler-chevron-up'" class="size-3.5 text-dimmed" />
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
  return e.code === 'LITERAL_TYPE_MISMATCH'
    && safeCoerceForFix(e.params?.value, String(e.params?.expected ?? '')) !== undefined
}
// 链: t('error.<CODE>', params) → raw code (兜底)。
function errorText(e: ValidationError): string {
  const key = `error.${e.code}`
  return te(key) ? t(key, (e.params ?? {}) as Record<string, unknown>) : e.code
}
</script>
