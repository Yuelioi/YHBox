<template>
  <div>
    <USelect
      :model-value="modelValue"
      :items="items"
      size="sm"
      placeholder="选择动作..."
      :ui="{ content: 'min-w-[280px]' }"
      @update:model-value="$emit('update:modelValue', String($event))"
    />
    <p v-if="actionsStore.list.length === 0" class="text-[10px] text-amber-300/80 mt-1">
      <UIcon name="i-tabler-alert-triangle" class="size-3 inline" />
      还没有动作。请去 任务 → 动作 tab 录制或新建一个
    </p>
    <p v-else-if="modelValue && !found" class="text-[10px] text-rose-300/80 mt-1">
      <UIcon name="i-tabler-x" class="size-3 inline" />
      引用的动作 {{ modelValue.slice(0, 8) }}... 已删除
    </p>
    <p v-else-if="found" class="text-[10px] text-dimmed mt-1">
      {{ found.steps.length }} 步
      <span v-if="found.recordingContext?.resolution">
        · {{ found.recordingContext.resolution[0] }}×{{ found.recordingContext.resolution[1] }}
        分辨率
      </span>
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useActionsStore } from '@/stores/actions'

const props = defineProps<{ modelValue: string }>()
defineEmits<{ 'update:modelValue': [v: string] }>()

const actionsStore = useActionsStore()
onMounted(() => {
  if (actionsStore.list.length === 0) void actionsStore.reload()
})

const items = computed(() =>
  actionsStore.list.map((a) => ({
    label: a.name || `(未命名 ${a.id.slice(0, 6)})`,
    value: a.id,
  })),
)

const found = computed(() => actionsStore.list.find((a) => a.id === props.modelValue))
</script>
