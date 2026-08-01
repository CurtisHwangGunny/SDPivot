<template>
  <section
    class="sdp-empty-state"
    :class="`sdp-empty-state--${variant}`"
    role="status"
    aria-live="polite"
    aria-atomic="true"
  >
    <img
      v-if="image"
      class="sdp-empty-state__image"
      :src="image"
      alt=""
      aria-hidden="true"
    >
    <div class="sdp-empty-state__copy">
      <h2 class="sdp-empty-state__title">{{ title }}</h2>
      <p class="sdp-empty-state__description">{{ description }}</p>
    </div>
    <div v-if="$slots.actions" class="sdp-empty-state__actions">
      <slot name="actions" />
    </div>
  </section>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  variant?: 'full' | 'compact' | 'minimal'
  title: string
  description: string
  image?: string
}>(), {
  variant: 'full',
  image: undefined,
})
</script>

<style scoped>
.sdp-empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-4);
  width: 100%;
  color: var(--ink-900);
  text-align: center;
  font-family: var(--font-body);
}

.sdp-empty-state--full {
  min-height: var(--space-24);
  padding: var(--space-12) var(--space-8);
  border: 1px solid var(--ink-200);
  border-radius: var(--radius-lg);
  background: var(--ink-50);
}

.sdp-empty-state--compact {
  padding: var(--space-8) var(--space-6);
  border: 1px solid var(--ink-200);
  border-radius: var(--radius-md);
  background: var(--ink-50);
}

.sdp-empty-state--minimal {
  padding: var(--space-4);
  gap: var(--space-2);
}

.sdp-empty-state__image {
  display: block;
  width: var(--space-20);
  height: var(--space-20);
  object-fit: contain;
}

.sdp-empty-state--compact .sdp-empty-state__image {
  width: var(--space-14);
  height: var(--space-14);
}

.sdp-empty-state--minimal .sdp-empty-state__image {
  width: var(--space-10);
  height: var(--space-10);
}

.sdp-empty-state__copy {
  display: grid;
  gap: var(--space-2);
  width: 100%;
}

.sdp-empty-state__title {
  color: var(--ink-900);
  font-family: var(--font-display);
  font-size: var(--text-xl);
  font-weight: var(--font-weight-semibold);
  line-height: var(--leading-tight);
}

.sdp-empty-state--compact .sdp-empty-state__title {
  font-size: var(--text-lg);
}

.sdp-empty-state--minimal .sdp-empty-state__title {
  font-family: var(--font-body);
  font-size: var(--text-base);
}

.sdp-empty-state__description {
  color: var(--ink-600);
  font-size: var(--text-sm);
  line-height: var(--leading-relaxed);
}

.sdp-empty-state--minimal .sdp-empty-state__description {
  font-size: var(--text-xs);
}

.sdp-empty-state__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: var(--space-3);
}
</style>
