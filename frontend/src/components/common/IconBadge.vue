<!-- 图标徽框: raised 小方框 + 居中图标。size/color/shape 可选; shape=round 时圆形。 -->
<template>
  <div class="flex shrink-0 items-center justify-center raised-surface" :class="[boxClass, shape === 'round' ? '!rounded-full' : '']">
    <UIcon :name="icon" :class="[iconSizeClass, iconColorClass]" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { badgeBoxClass, badgeIconSize, badgeIconColor, type BadgeSize, type BadgeColor } from './IconBadge.helpers'

const props = withDefaults(
  defineProps<{ icon: string; size?: BadgeSize; color?: BadgeColor; shape?: 'square' | 'round' }>(),
  { size: 'md', color: 'default', shape: 'square' },
)
const boxClass = computed(() => badgeBoxClass(props.size))
const iconSizeClass = computed(() => badgeIconSize(props.size))
const iconColorClass = computed(() => badgeIconColor(props.color))
</script>
