<template>
  <section
    class="overflow-hidden rounded-lg border"
    :class="tone === 'added' ? 'border-warning/35' : 'border-default'"
  >
    <h5
      class="border-b px-3 py-2 text-xs font-semibold"
      :class="
        tone === 'added'
          ? 'border-warning/25 bg-warning/10 text-warning'
          : 'border-default bg-elevated/20 text-muted'
      "
    >
      {{ title }} · {{ items.length }}
    </h5>
    <div class="max-h-52 divide-y divide-default overflow-y-auto">
      <article v-for="item in items" :key="scopeKey(item)" class="space-y-1 px-3 py-2">
        <div class="flex flex-wrap items-center gap-1.5">
          <span class="break-all font-mono text-[10px] text-highlighted">
            {{ item.capabilityId }}
          </span>
          <UBadge color="neutral" variant="soft" size="xs">{{ item.risk }}</UBadge>
          <UBadge color="neutral" variant="subtle" size="xs">{{ item.consent }}</UBadge>
        </div>
        <p class="break-all text-[10px] text-muted">
          {{ item.operations.join(', ') }} · target={{ item.targetSlot }}
          <template v-if="item.credentialSlot"> · credential={{ item.credentialSlot }}</template>
        </p>
        <p class="break-all font-mono text-[10px] text-dimmed">
          scope={{ item.scope }} · node={{ item.nodeTypeId }}
        </p>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { InstallationUpdatePreviewView } from '@/app/transport/workflow'

type Scope = InstallationUpdatePreviewView['diff']['addedCapabilities'][number]
defineProps<{ title: string; tone: 'added' | 'removed'; items: Scope[] }>()

function scopeKey(item: Scope): string {
  return [
    item.nodeTypeId,
    item.capabilityId,
    item.operations.join(','),
    item.targetSlot,
    item.credentialSlot,
    item.scope,
  ].join(':')
}
</script>
