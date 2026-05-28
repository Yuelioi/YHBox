<template>
  <div class="px-8 py-8">
    <!-- Header row -->
    <div class="flex items-center justify-between mb-8">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight text-highlighted">{{ title }}</h1>
        <p class="mt-1 text-sm text-dimmed">{{ subtitle }}</p>
      </div>
      <UBadge :color="badgeColor" variant="subtle" class="font-medium tracking-wide">
        {{ badgeLabel }}
      </UBadge>
    </div>

    <!-- Central illustration card -->
    <div class="max-w-xl">
      <div class="rounded-xl bg-default border border-default p-8">
        <div class="flex flex-col items-center text-center">
          <div class="size-20 rounded-full bg-elevated flex items-center justify-center mb-5">
            <UIcon :name="icon" class="size-9 text-muted" />
          </div>
          <h2 class="text-lg font-semibold text-highlighted mb-2">{{ cardTitle }}</h2>
          <p class="text-sm text-muted leading-relaxed max-w-sm">{{ cardBody }}</p>

          <div v-if="actionLabel" class="mt-6">
            <UButton
              :icon="actionIcon"
              size="md"
              color="neutral"
              variant="soft"
              :disabled="true"
              class="cursor-not-allowed opacity-60"
            >
              {{ actionLabel }}
            </UButton>
          </div>
        </div>
      </div>

      <slot name="extra" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps<{
  title: string
  subtitle: string
  icon: string
  cardTitle: string
  cardBody: string
  /** "敬请期待" | "Current" — Phase 4 about uses Current */
  badge?: 'pending' | 'current'
  /** Disabled action button label (only shown for bot views) */
  actionLabel?: string
  actionIcon?: string
}>()

const badgeColor = computed(() => (props.badge === 'current' ? 'success' : 'warning'))
const badgeLabel = computed(() => (props.badge === 'current' ? 'Current' : t('common.coming_soon')))
</script>
