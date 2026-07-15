<!-- Blob preview adapter 落地前只显示占位；禁止把大对象 base64 后穿过 Wails RPC。 -->
<template>
  <div
    class="flex h-full w-full flex-col items-center justify-center gap-1.5 text-dimmed"
    :aria-label="alt"
  >
    <UIcon name="i-tabler-photo" class="size-6" />
  </div>
</template>

<script setup lang="ts">
const { alt } = defineProps<{
  alt?: string
  maxUpscale?: number
}>()
</script>
