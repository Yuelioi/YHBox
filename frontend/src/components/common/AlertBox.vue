<!-- 提示框: type 驱动配色 (扁平 tint + ring, 无浮层)。title 必传; 说明走默认 slot; icon 可覆盖默认。 -->
<template>
  <div class="flex gap-2.5 rounded-lg px-3 py-2.5 ring-1" :class="style.box">
    <UIcon :name="icon || style.defaultIcon" class="mt-0.5 size-4 shrink-0" :class="style.icon" />
    <div class="min-w-0 flex-1">
      <p v-if="title" class="text-xs font-medium text-highlighted">{{ title }}</p>
      <p v-if="$slots.default" class="text-xs text-muted" :class="title ? 'mt-0.5' : ''">
        <slot />
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { alertStyle, type AlertType } from './AlertBox.helpers'

const props = defineProps<{ type: AlertType; title?: string; icon?: string }>()
const style = computed(() => alertStyle(props.type))
</script>
