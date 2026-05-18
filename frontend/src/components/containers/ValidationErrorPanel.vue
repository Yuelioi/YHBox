<template>
  <UModal
    :open="open"
    :ui="{ content: 'sm:max-w-[640px]' }"
    @update:open="(v: boolean) => !v && emit('close')"
  >
    <template #content>
      <div class="p-5 space-y-4 bg-default max-h-[70vh] flex flex-col">
        <header class="flex items-center gap-2 shrink-0">
          <UIcon
            :name="hasErrors ? 'i-tabler-alert-octagon' : 'i-tabler-check'"
            class="size-5"
            :class="hasErrors ? 'text-error' : 'text-success'"
          />
          <h3 class="text-sm font-medium">
            {{ hasErrors ? `校验失败 — ${errorCount} 错${warnCount ? ' / ' + warnCount + ' 警告' : ''}` : '校验通过' }}
          </h3>
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            icon="i-tabler-x"
            class="ml-auto"
            @click="emit('close')"
          />
        </header>

        <p v-if="!hasErrors && warnCount === 0" class="text-xs text-muted">
          没有发现校验错误。容器可以试运行。
        </p>

        <div v-else class="flex-1 overflow-y-auto space-y-2 pr-1">
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
            <!-- v4 (spec §12): prefer t('error.<CODE>', params) — falls back to backend.message
                 if i18n key not registered, then to the raw code as last resort. -->
            <div class="text-xs text-toned leading-relaxed pl-5">{{ errorText(e) }}</div>
            <div v-if="e.nodeId" class="text-[10px] text-dimmed pl-5">
              节点: <code class="font-mono">{{ e.nodeId }}</code>
            </div>
            <div v-if="e.code === 'MISSING_WINDOW_TARGET'" class="pl-5 pt-1">
              <UButton
                size="xs"
                color="primary"
                icon="i-tabler-wand"
                @click="emit('fix-missing-window-target')"
              >
                一键添加 WindowTarget 节点
              </UButton>
            </div>
          </div>
        </div>

        <div class="flex justify-end gap-2 pt-2 shrink-0">
          <UButton variant="ghost" color="neutral" @click="emit('close')">关闭</UButton>
          <UButton
            v-if="!hasErrors"
            color="primary"
            icon="i-tabler-player-play"
            @click="emit('run')"
          >
            通过 — 继续运行
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ValidationError } from '@/lib/backend'

const props = defineProps<{
  open: boolean
  errors: ValidationError[]
}>()

const emit = defineEmits<{
  close: []
  run: []
  'fix-missing-window-target': []
}>()

const errorCount = computed(() => props.errors.filter((e) => e.severity === 'error').length)
const warnCount = computed(() => props.errors.filter((e) => e.severity === 'warning').length)
const hasErrors = computed(() => errorCount.value > 0)

// v4 (spec §12): code+params → t('error.<CODE>', params). Fallback chain:
//   1. i18n key registered → translated string
//   2. backend legacy Message field non-empty → use it (back-compat for codes without keys yet)
//   3. raw code (last resort, hints at missing translation)
const { t, te } = useI18n()
function errorText(e: ValidationError & { params?: Record<string, any> }): string {
  const key = `error.${e.code}`
  if (te(key)) {
    return t(key, (e as any).params ?? {})
  }
  if (e.message) return e.message
  return e.code
}
</script>
