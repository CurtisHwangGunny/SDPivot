<template>
  <button
    class="sdp-button"
    :class="[`sdp-button--${variant}`, `sdp-button--${size}`]"
    type="button"
    :disabled="disabled || loading"
    :aria-label="ariaLabel"
    :aria-busy="loading"
    @click="handleClick"
  >
    <span v-if="loading" class="sdp-button__spinner" aria-hidden="true" />
    <span class="sdp-button__content">
      <slot />
    </span>
  </button>
</template>

<script setup lang="ts">
const props = withDefaults(defineProps<{
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger'
  size?: 'sm' | 'md' | 'lg'
  loading?: boolean
  disabled?: boolean
  ariaLabel?: string
}>(), {
  variant: 'primary',
  size: 'md',
  loading: false,
  disabled: false,
  ariaLabel: undefined,
})

const emit = defineEmits<{
  click: [event: MouseEvent]
}>()

function handleClick(event: MouseEvent) {
  if (!props.disabled && !props.loading) {
    emit('click', event)
  }
}
</script>

<style scoped>
.sdp-button {
  display: inline-flex;
  min-width: var(--space-8);
  min-height: var(--space-8);
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  border: 1px solid var(--ink-300);
  border-radius: var(--radius-sm);
  font-family: var(--font-body);
  font-weight: var(--font-weight-semibold);
  line-height: var(--leading-tight);
  cursor: pointer;
  transition:
    background-color var(--duration-fast) var(--ease-out-quart),
    border-color var(--duration-fast) var(--ease-out-quart),
    color var(--duration-fast) var(--ease-out-quart),
    box-shadow var(--duration-fast) var(--ease-out-quart);
}

.sdp-button:hover:not(:disabled) {
  box-shadow: var(--shadow-sm);
}

.sdp-button:focus-visible {
  outline: 2px solid var(--brand-500);
  outline-offset: 2px;
}

.sdp-button:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.sdp-button--sm {
  padding: var(--space-2) var(--space-3);
  font-size: var(--text-xs);
}

.sdp-button--md {
  padding: var(--space-3) var(--space-4);
  font-size: var(--text-sm);
}

.sdp-button--lg {
  padding: var(--space-4) var(--space-6);
  font-size: var(--text-base);
}

.sdp-button--primary {
  border-color: var(--brand-500);
  color: var(--ink-950);
  background: var(--brand-500);
}

.sdp-button--primary:hover:not(:disabled) {
  border-color: var(--brand-600);
  background: var(--brand-600);
}

.sdp-button--secondary {
  border-color: var(--ink-300);
  color: var(--ink-900);
  background: var(--ink-50);
}

.sdp-button--secondary:hover:not(:disabled) {
  border-color: var(--ink-500);
  background: var(--ink-100);
}

.sdp-button--ghost {
  border-color: var(--ink-50);
  color: var(--ink-800);
  background: var(--ink-50);
}

.sdp-button--ghost:hover:not(:disabled) {
  border-color: var(--ink-200);
  background: var(--ink-100);
}

.sdp-button--danger {
  border-color: var(--danger-500);
  color: var(--ink-50);
  background: var(--danger-500);
}

.sdp-button--danger:hover:not(:disabled) {
  border-color: var(--danger-500);
  background: var(--danger-500);
}

:global([data-theme="dark"]) .sdp-button--primary {
  color: var(--ink-50);
}

.sdp-button__spinner {
  width: var(--space-4);
  height: var(--space-4);
  flex: 0 0 var(--space-4);
  border: 2px solid currentColor;
  border-right-color: var(--ink-300);
  border-radius: var(--radius-pill);
  animation: sdp-button-spin var(--duration-slow) linear infinite;
}

.sdp-button__content {
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

@media (max-width: 639px) {
  .sdp-button {
    min-width: 44px;
    min-height: 44px;
  }
}

@keyframes sdp-button-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
