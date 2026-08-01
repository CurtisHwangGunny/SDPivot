<template>
  <section class="sdp-error-state" :class="`sdp-error-state--${type}`" role="alert" aria-live="assertive">
    <span class="sdp-error-state__icon" aria-hidden="true">
      <svg v-if="type === 'network'" viewBox="0 0 24 24" fill="none">
        <path d="M3 8.5a14 14 0 0 1 18 0M6.5 12a9 9 0 0 1 11 0M10 15.5a4 4 0 0 1 4 0M12 20h.01" />
        <path d="m4 4 16 16" />
      </svg>
      <svg v-else-if="type === 'server'" viewBox="0 0 24 24" fill="none">
        <rect x="3" y="4" width="18" height="6" rx="2" />
        <rect x="3" y="14" width="18" height="6" rx="2" />
        <path d="M7 7h.01M7 17h.01M16 7h2M16 17h2" />
      </svg>
      <svg v-else-if="type === 'forbidden'" viewBox="0 0 24 24" fill="none">
        <circle cx="12" cy="12" r="9" />
        <path d="m6 6 12 12" />
      </svg>
      <svg v-else-if="type === 'not-found'" viewBox="0 0 24 24" fill="none">
        <circle cx="10.5" cy="10.5" r="6.5" />
        <path d="m15.5 15.5 5 5M8.5 8.5l4 4M12.5 8.5l-4 4" />
      </svg>
      <svg v-else viewBox="0 0 24 24" fill="none">
        <path d="M10.3 4.2 2.8 17a2 2 0 0 0 1.7 3h15a2 2 0 0 0 1.7-3L13.7 4.2a2 2 0 0 0-3.4 0Z" />
        <path d="M12 9v4M12 17h.01" />
      </svg>
    </span>
    <div class="sdp-error-state__copy">
      <h2 class="sdp-error-state__title">{{ title }}</h2>
      <p class="sdp-error-state__description">{{ description }}</p>
    </div>
    <SdpButton
      v-if="retryable"
      variant="secondary"
      aria-label="Retry"
      @click="emit('retry')"
    >
      Retry
    </SdpButton>
  </section>
</template>

<script setup lang="ts">
import SdpButton from './SdpButton.vue'

withDefaults(defineProps<{
  type?: 'network' | 'server' | 'forbidden' | 'not-found' | 'warning'
  title: string
  description: string
  retryable?: boolean
}>(), {
  type: 'warning',
  retryable: false,
})

const emit = defineEmits<{
  retry: []
}>()
</script>

<style scoped>
.sdp-error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-4);
  width: 100%;
  padding: var(--space-8);
  border: 1px solid var(--ink-200);
  border-radius: var(--radius-lg);
  color: var(--ink-900);
  background: var(--ink-50);
  text-align: center;
  font-family: var(--font-body);
}

.sdp-error-state__icon {
  display: inline-flex;
  width: var(--space-14);
  height: var(--space-14);
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-pill);
  color: var(--ink-900);
  background: var(--ink-100);
}

.sdp-error-state--warning .sdp-error-state__icon,
.sdp-error-state--not-found .sdp-error-state__icon {
  color: var(--ink-800);
  background: var(--ink-200);
}

.sdp-error-state--network .sdp-error-state__icon {
  color: var(--brand-800);
  background: var(--brand-100);
}

.sdp-error-state__icon svg {
  width: var(--space-8);
  height: var(--space-8);
  stroke: currentColor;
  stroke-width: 1.8;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.sdp-error-state__copy {
  display: grid;
  gap: var(--space-2);
  max-width: 40rem;
}

.sdp-error-state__title {
  color: var(--ink-900);
  font-family: var(--font-display);
  font-size: var(--text-xl);
  font-weight: var(--font-weight-semibold);
  line-height: var(--leading-tight);
}

.sdp-error-state__description {
  color: var(--ink-700);
  font-size: var(--text-sm);
  line-height: var(--leading-relaxed);
}
</style>
