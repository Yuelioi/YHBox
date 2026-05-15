<template>
  <div class="flex items-center gap-3 p-3 rounded-md bg-elevated/40 border border-default/60">
    <UInput v-model="search" icon="i-tabler-search" placeholder="搜索动作..." class="flex-1" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ search: string }>()
const emit = defineEmits<{ 'update:search': [v: string] }>()

const search = computed({ get: () => props.search, set: (v) => emit('update:search', v) })
</script>
