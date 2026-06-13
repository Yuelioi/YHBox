<template>
  <BaseModal
    :open="open"
    :title="titleText"
    :icon="hasErrors ? 'i-tabler-alert-octagon' : 'i-tabler-check'"
    :icon-color="hasErrors ? 'error' : 'success'"
    size="2xl"
    @update:open="(v: boolean) => !v && emit('close')"
  >
    <div class="space-y-4">
      <p v-if="!hasErrors && warnCount === 0" class="text-xs text-muted">
        {{ t('validation.desc_no_issues') }}
      </p>

      <div v-else class="space-y-2">
        <div
          v-for="(e, idx) in errors"
          :key="idx"
          class="rounded-md border px-3 py-2 space-y-1"
          :class="
            e.severity === 'error'
              ? 'border-error/40 bg-error/5'
              : 'border-warning/40 bg-warning/5'
          "
        >
          <div class="flex items-center gap-2 text-xs">
            <UIcon
              :name="e.severity === 'error' ? 'i-tabler-alert-triangle' : 'i-tabler-info-circle'"
              class="size-3.5 shrink-0"
              :class="e.severity === 'error' ? 'text-error' : 'text-warning'"
            />
            <code
              class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-elevated/60"
              :class="e.severity === 'error' ? 'text-error' : 'text-warning'"
            >{{ e.code }}</code>
            <span v-if="e.graphPath.length > 0" class="text-[10px] text-dimmed truncate">
              {{ e.graphPath.join(' › ') }}
            </span>
          </div>
          <!-- t('error.<CODE>', params) → backend.message → raw code (fallback 链). -->
          <div class="text-xs text-toned leading-relaxed pl-5">{{ errorText(e) }}</div>
          <!-- 动作行: nodeId + 跳转(定位到出错节点, 含跳进子图) + 修复(能机械改的才显) -->
          <div
            v-if="e.nodeId || canFix(e) || e.code === 'MISSING_WINDOW_TARGET'"
            class="flex flex-wrap items-center gap-2 pl-5 pt-1"
          >
            <code v-if="e.nodeId" class="font-mono text-[10px] text-dimmed">{{ e.nodeId }}</code>
            <UButton
              v-if="e.nodeId"
              size="xs"
              variant="soft"
              color="neutral"
              icon="i-tabler-focus-2"
              @click="emit('jump', e)"
            >
              {{ t('validation.jump') }}
            </UButton>
            <UButton
              v-if="canFix(e)"
              size="xs"
              color="primary"
              icon="i-tabler-wand"
              @click="emit('fix', e)"
            >
              {{ t('validation.fix') }}
            </UButton>
            <UButton
              v-if="e.code === 'MISSING_WINDOW_TARGET'"
              size="xs"
              color="primary"
              icon="i-tabler-wand"
              @click="emit('fix-missing-window-target')"
            >
              {{ t('validation.fix_missing_window_target') }}
            </UButton>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <UButton variant="ghost" color="neutral" @click="emit('close')">{{ t('validation.close') }}</UButton>
      <UButton
        v-if="!hasErrors"
        color="primary"
        icon="i-tabler-player-play"
        @click="emit('run')"
      >
        {{ t('validation.run_button') }}
      </UButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ValidationError } from '@/lib/backend'
import BaseModal from '@/components/common/BaseModal.vue'
import { safeCoerceForFix } from '@/components/containers/inline/coerceLiteral'

const props = defineProps<{
  open: boolean
  errors: ValidationError[]
}>()

const emit = defineEmits<{
  close: []
  run: []
  'fix-missing-window-target': []
  jump: [e: ValidationError]
  fix: [e: ValidationError]
}>()

// canFix: 这条错误能不能「一键修复」。目前只覆盖 LITERAL_TYPE_MISMATCH 中能安全 coerce 的
// (干净数字串→number / 真假串→bool / 标量→string)。含糊值 (如 "500ms") → 不显修复按钮。
function canFix(e: ValidationError): boolean {
  return (
    e.code === 'LITERAL_TYPE_MISMATCH' &&
    safeCoerceForFix(e.params?.value, String(e.params?.expected ?? '')) !== undefined
  )
}

const { t, te } = useI18n()
const errorCount = computed(() => props.errors.filter((e) => e.severity === 'error').length)
const warnCount = computed(() => props.errors.filter((e) => e.severity === 'warning').length)
const hasErrors = computed(() => errorCount.value > 0)

const titleText = computed(() => {
  if (!hasErrors.value) return t('validation.title_passed')
  return warnCount.value > 0
    ? t('validation.title_failed_with_warns', { errorCount: errorCount.value, warnCount: warnCount.value })
    : t('validation.title_failed', { errorCount: errorCount.value })
})

// B5 ship 后 Code+Params 纯模型 — 不再有 Message fallback.
// 链: t('error.<CODE>', params) → raw code (兜底, hint missing translation).
function errorText(e: ValidationError): string {
  const key = `error.${e.code}`
  if (te(key)) {
    return t(key, (e.params ?? {}) as Record<string, unknown>)
  }
  return e.code
}
</script>
