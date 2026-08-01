<template>
  <section
    v-if="visible"
    class="sdp-notice"
    :class="`sdp-notice--${type}`"
    role="alert"
    aria-live="polite"
    aria-atomic="true"
  >
    <span class="sdp-notice__icon" aria-hidden="true">
      <svg viewBox="0 0 24 24" fill="none">
        <path v-if="type === 'success'" d="m5 12 4 4L19 6" />
        <template v-else-if="type === 'warning' || type === 'danger'">
          <path d="M10.3 4.2 2.8 17a2 2 0 0 0 1.7 3h15a2 2 0 0 0 1.7-3L13.7 4.2a2 2 0 0 0-3.4 0Z" />
          <path d="M12 9v4M12 17h.01" />
        </template>
        <template v-else>
          <circle cx="12" cy="12" r="9" />
          <path d="M12 11v5M12 8h.01" />
        </template>
      </svg>
    </span>
    <div class="sdp-notice__copy">
      <h2 class="sdp-notice__title">{{ title }}</h2>
      <div v-if="$slots.default" class="sdp-notice__content">
        <slot />
      </div>
    </div>
    <button
      v-if="closable"
      class="sdp-notice__close"
      type="button"
      :aria-label="`Dismiss notice: ${title}`"
      @click="visible = false"
    >
      <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <path d="m6 6 12 12M18 6 6 18" />
      </svg>
    </button>
  </section>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  type?: 'success' | 'info' | 'warning' | 'danger'
  title: string
  closable?: boolean
}>(), {
  type: 'info',
  closable: false,
})

const visible = defineModel<boolean>('visible', { default: true })
</script>

<style scoped>
.sdp-notice {
  display: grid;
  grid-template-columns: var(--space-6) 1fr auto;
  gap: var(--space-3);
  align-items: start;
  width: 100%;
  padding: var(--space-4);
  border: 1px solid var(--brand-300);
  border-radius: var(--radius-md);
  color: var(--ink-900);
  background: var(--brand-50);
  font-family: var(--font-body);
}

.sdp-notice--warning {
  border-color: var(--ink-400);
  background: var(--ink-100);
}

.sdp-notice--danger {
  border-color: var(--ink-700);
  background: var(--ink-100);
}

.sdp-notice__icon {
  display: inline-flex;
  width: var(--space-6);
  height: var(--space-6);
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-pill);
  color: var(--brand-800);
  background: var(--brand-100);
}

.sdp-notice--warning .sdp-notice__icon,
.sdp-notice--danger .sdp-notice__icon {
  color: var(--ink-900);
  background: var(--ink-200);
}

.sdp-notice__icon svg,
.sdp-notice__close svg {
  width: var(--space-4);
  height: var(--space-4);
  stroke: currentColor;
  stroke-width: 2;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.sdp-notice__copy {
  display: grid;
  gap: var(--space-1);
  padding-block: var(--space-1);
}

.sdp-notice__title {
  color: var(--ink-900);
  font-family: var(--font-display);
  font-size: var(--text-base);
  font-weight: var(--font-weight-semibold);
  line-height: var(--leading-tight);
}

.sdp-notice__content {
  color: var(--ink-700);
  font-size: var(--text-sm);
  line-height: var(--leading-relaxed);
}

.sdp-notice__close {
  display: inline-flex;
  width: var(--space-8);
  height: var(--space-8);
  align-items: center;
  justify-content: center;
  border: 1px solid var(--ink-300);
  border-radius: var(--radius-sm);
  color: var(--ink-700);
  background: var(--ink-50);
  cursor: pointer;
}

.sdp-notice__close:hover {
  color: var(--ink-900);
  background: var(--ink-100);
}

.sdp-notice__close:focus-visible {
  outline: 2px solid var(--brand-500);
  outline-offset: 2px;
}
</style>
