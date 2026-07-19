<template>
  <div
    class="flex min-h-12 flex-wrap items-center gap-2 border-s-2 border-primary bg-primary/5 px-3 py-2"
    role="toolbar"
    :aria-label="label"
    data-testid="library-selection-toolbar"
  >
    <div class="mr-auto flex min-w-0 items-center gap-2">
      <span
        class="flex size-7 shrink-0 items-center justify-center rounded-md bg-primary/15 text-primary"
        aria-hidden="true"
      >
        <UIcon name="i-tabler-checks" class="size-4" />
      </span>
      <span class="truncate text-sm font-medium text-highlighted">{{ label }}</span>
      <span v-if="hint" class="hidden text-xs text-dimmed xl:inline">{{ hint }}</span>
      <UButton
        size="xs"
        color="neutral"
        variant="ghost"
        icon="i-tabler-x"
        :label="clearLabel"
        @click="$emit('clear')"
      />
    </div>
    <div class="flex flex-wrap items-center justify-end gap-1.5">
      <slot />
      <template v-if="$slots.destructive">
        <span class="mx-1 h-5 w-px bg-accented" aria-hidden="true" />
        <slot name="destructive" />
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  label: string
  hint?: string
  clearLabel: string
}>()

defineEmits<{ clear: [] }>()
</script>
