<template>
  <div v-if="tags.length > 0" class="overflow-x-auto whitespace-nowrap py-1 -mx-1 px-1 flex gap-1.5">
    <UButton
      v-for="t in tags"
      :key="t.tag"
      size="xs"
      :variant="selectedTags.includes(t.tag) ? 'solid' : 'soft'"
      :color="selectedTags.includes(t.tag) ? 'primary' : 'neutral'"
      class="shrink-0"
      @click="toggle(t.tag)"
    >
      {{ t.tag }}
      <span class="ml-1 text-[9px] opacity-60">{{ t.count }}</span>
    </UButton>
  </div>
</template>

<script setup lang="ts">
defineProps<{ tags: { tag: string; count: number }[] }>()
const selectedTags = defineModel<string[]>('selectedTags', { required: true })

function toggle(tag: string) {
  if (selectedTags.value.includes(tag)) {
    selectedTags.value = selectedTags.value.filter((t) => t !== tag)
  } else {
    selectedTags.value = [...selectedTags.value, tag]
  }
}
</script>
