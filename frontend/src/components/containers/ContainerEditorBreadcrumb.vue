<template>
  <div class="shrink-0 h-9 px-4 border-b border-default flex items-center gap-2 text-xs">
    <UButton
      v-if="editorPath.length > 0"
      size="xs"
      variant="ghost"
      color="neutral"
      icon="i-tabler-arrow-left"
      @click="$emit('pop')"
    >上一层</UButton>
    <span class="text-toned">{{ rootLabel ?? '...' }}</span>
    <template v-for="(sgID, idx) in editorPath" :key="sgID">
      <UIcon name="i-tabler-chevron-right" class="size-3 text-dimmed" />
      <UButton
        size="xs"
        variant="ghost"
        color="neutral"
        class="!px-1.5 !py-0 !h-auto"
        @click="$emit('goto', idx)"
      >{{ sgLabelFn(sgID) }}</UButton>
    </template>
    <!-- 当前层级节点数 (Subgraph 算 1, 主图节点总数 / 子图内部节点总数) -->
    <span
      v-if="activeNodeCount !== null"
      class="ml-2 text-[10px] text-dimmed inline-flex items-center gap-1"
      :title="editorPath.length === 0 ? '主图节点数 (子图算 1)' : '当前子图内部节点数'"
    >
      <UIcon name="i-tabler-cpu" class="size-3" />
      {{ activeNodeCount }} 节点
    </span>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  rootLabel: string | undefined
  editorPath: readonly string[]
  sgLabelFn: (id: string) => string
  activeNodeCount: number | null
}>()

defineEmits<{
  'pop': []
  'goto': [idx: number]
}>()
</script>
