<template>
  <div
    class="sdp-loading-indicator"
    :class="[`sdp-loading-indicator--${type}`, `sdp-loading-indicator--${size}`]"
    role="progressbar"
    :aria-label="label"
    :aria-valuemin="type === 'progress' && isDeterminate ? 0 : undefined"
    :aria-valuemax="type === 'progress' && isDeterminate ? 100 : undefined"
    :aria-valuenow="type === 'progress' && isDeterminate ? normalizedValue : undefined"
  >
    <span v-if="type === 'spinner'" class="sdp-loading-indicator__spinner" aria-hidden="true" />
    <span v-else-if="type === 'progress'" class="sdp-loading-indicator__track" aria-hidden="true">
      <span
        class="sdp-loading-indicator__bar"
        :class="{ 'sdp-loading-indicator__bar--indeterminate': !isDeterminate }"
        :style="isDeterminate ? { width: `${normalizedValue}%` } : undefined"
      />
    </span>
    <span v-else class="sdp-loading-indicator__dot" aria-hidden="true" />
    <span v-if="type !== 'progress'" class="sdp-loading-indicator__label">{{ label }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  type?: 'spinner' | 'progress' | 'inline'
  size?: 'sm' | 'md' | 'lg'
  label: string
  value?: number
}>(), {
  type: 'spinner',
  size: 'md',
  value: undefined,
})

const isDeterminate = computed(() => Number.isFinite(props.value))
const normalizedValue = computed(() => Math.min(100, Math.max(0, props.value ?? 0)))
</script>

<style scoped>
.sdp-loading-indicator {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--ink-700);
  font-family: var(--font-body);
  line-height: var(--leading-normal);
}

.sdp-loading-indicator--sm {
  font-size: var(--text-xs);
}

.sdp-loading-indicator--md {
  font-size: var(--text-sm);
}

.sdp-loading-indicator--lg {
  font-size: var(--text-base);
}

.sdp-loading-indicator--progress {
  display: flex;
  width: 100%;
}

.sdp-loading-indicator__spinner {
  width: var(--space-5);
  height: var(--space-5);
  flex: 0 0 var(--space-5);
  border: 2px solid var(--brand-200);
  border-top-color: var(--brand-600);
  border-radius: var(--radius-pill);
  animation: sdp-loading-spin var(--duration-slow) linear infinite;
}

.sdp-loading-indicator--sm .sdp-loading-indicator__spinner {
  width: var(--space-4);
  height: var(--space-4);
  flex-basis: var(--space-4);
}

.sdp-loading-indicator--lg .sdp-loading-indicator__spinner {
  width: var(--space-8);
  height: var(--space-8);
  flex-basis: var(--space-8);
}

.sdp-loading-indicator__track {
  display: block;
  overflow: hidden;
  width: 100%;
  height: var(--space-2);
  border-radius: var(--radius-pill);
  background: var(--ink-200);
}

.sdp-loading-indicator--sm .sdp-loading-indicator__track {
  height: var(--space-1);
}

.sdp-loading-indicator--lg .sdp-loading-indicator__track {
  height: var(--space-3);
}

.sdp-loading-indicator__bar {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--brand-600);
  transition: width var(--duration-normal) var(--ease-out-quart);
}

.sdp-loading-indicator__bar--indeterminate {
  width: 40%;
  animation: sdp-loading-progress var(--duration-slow) var(--ease-in-out) infinite alternate;
}

.sdp-loading-indicator__dot {
  width: var(--space-2);
  height: var(--space-2);
  flex: 0 0 var(--space-2);
  border-radius: var(--radius-pill);
  background: var(--brand-600);
  animation: sdp-loading-pulse var(--duration-slow) var(--ease-in-out) infinite alternate;
}

.sdp-loading-indicator--sm .sdp-loading-indicator__dot {
  width: var(--space-1);
  height: var(--space-1);
  flex-basis: var(--space-1);
}

.sdp-loading-indicator--lg .sdp-loading-indicator__dot {
  width: var(--space-3);
  height: var(--space-3);
  flex-basis: var(--space-3);
}

@keyframes sdp-loading-spin {
  to {
    transform: rotate(360deg);
  }
}

@keyframes sdp-loading-progress {
  from {
    transform: translateX(-100%);
  }

  to {
    transform: translateX(250%);
  }
}

@keyframes sdp-loading-pulse {
  from {
    opacity: 0.45;
  }

  to {
    opacity: 1;
  }
}
</style>
