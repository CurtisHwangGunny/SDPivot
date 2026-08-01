<template>
  <div
    class="sdp-skeleton"
    :class="`sdp-skeleton--${variant}`"
    role="status"
    aria-live="polite"
    :aria-label="loadingLabel"
  >
    <div
      v-for="item in normalizedCount"
      :key="item"
      class="sdp-skeleton__item"
      aria-hidden="true"
    >
      <template v-if="variant === 'card'">
        <span class="sdp-skeleton__media" />
        <span class="sdp-skeleton__line sdp-skeleton__line--title" />
        <span class="sdp-skeleton__line" />
        <span class="sdp-skeleton__line sdp-skeleton__line--short" />
      </template>

      <template v-else-if="variant === 'grid'">
        <span class="sdp-skeleton__tile" />
        <span class="sdp-skeleton__line sdp-skeleton__line--title" />
        <span class="sdp-skeleton__line sdp-skeleton__line--short" />
      </template>

      <template v-else>
        <span class="sdp-skeleton__avatar" />
        <span class="sdp-skeleton__list-copy">
          <span class="sdp-skeleton__line sdp-skeleton__line--title" />
          <span class="sdp-skeleton__line" />
        </span>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  variant?: 'card' | 'grid' | 'list'
  count?: number
}>(), {
  variant: 'card',
  count: 1,
})

const normalizedCount = computed(() => Math.max(0, Math.floor(props.count)))
const loadingLabel = computed(() => `Loading ${normalizedCount.value} ${props.variant} item${normalizedCount.value === 1 ? '' : 's'}`)
</script>

<style scoped>
.sdp-skeleton {
  display: grid;
  gap: var(--space-4);
  width: 100%;
}

.sdp-skeleton--grid {
  grid-template-columns: repeat(auto-fit, minmax(var(--space-24), 1fr));
}

.sdp-skeleton__item {
  display: grid;
  gap: var(--space-3);
  padding: var(--space-4);
  border: 1px solid var(--ink-200);
  border-radius: var(--radius-md);
  background: var(--ink-50);
}

.sdp-skeleton--list .sdp-skeleton__item {
  grid-template-columns: var(--space-12) 1fr;
  align-items: center;
}

.sdp-skeleton__media,
.sdp-skeleton__tile,
.sdp-skeleton__line,
.sdp-skeleton__avatar {
  display: block;
  overflow: hidden;
  position: relative;
  background: var(--ink-200);
}

.sdp-skeleton__media::after,
.sdp-skeleton__tile::after,
.sdp-skeleton__line::after,
.sdp-skeleton__avatar::after {
  position: absolute;
  inset: var(--space-0);
  background: linear-gradient(90deg, var(--ink-200), var(--ink-100), var(--ink-200));
  content: '';
  transform: translateX(-100%);
  animation: sdp-skeleton-shimmer var(--duration-slow) var(--ease-in-out) infinite;
}

.sdp-skeleton__media {
  height: var(--space-24);
  border-radius: var(--radius-sm);
}

.sdp-skeleton__tile {
  height: var(--space-20);
  border-radius: var(--radius-sm);
}

.sdp-skeleton__line {
  width: 100%;
  height: var(--space-3);
  border-radius: var(--radius-pill);
}

.sdp-skeleton__line--title {
  width: 70%;
  height: var(--space-4);
}

.sdp-skeleton__line--short {
  width: 45%;
}

.sdp-skeleton__avatar {
  width: var(--space-12);
  height: var(--space-12);
  border-radius: var(--radius-pill);
}

.sdp-skeleton__list-copy {
  display: grid;
  gap: var(--space-2);
}

@keyframes sdp-skeleton-shimmer {
  to {
    transform: translateX(100%);
  }
}
</style>
