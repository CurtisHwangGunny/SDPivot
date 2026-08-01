<template>
  <Teleport v-if="stackReady" :to="`#${stackId}`">
    <div
      v-if="visible"
      class="sdp-toast"
      :class="`sdp-toast--${type}`"
      role="alert"
      aria-live="assertive"
      aria-atomic="true"
    >
      <span class="sdp-toast__icon" aria-hidden="true">
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
      <p class="sdp-toast__message">{{ message }}</p>
      <button
        v-if="closable"
        class="sdp-toast__close"
        type="button"
        :aria-label="`Dismiss notification: ${message}`"
        @click="dismiss"
      >
        <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
          <path d="m6 6 12 12M18 6 6 18" />
        </svg>
      </button>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'

const stackId = 'sdp-toast-stack'
let stackUsers = 0

const props = withDefaults(defineProps<{
  type?: 'success' | 'info' | 'warning' | 'danger'
  message: string
  duration?: number
  closable?: boolean
}>(), {
  type: 'info',
  duration: 3000,
  closable: false,
})

const emit = defineEmits<{
  dismiss: []
}>()

const visible = ref(true)
const stackReady = ref(false)
let dismissTimer: ReturnType<typeof setTimeout> | undefined

function clearDismissTimer() {
  if (dismissTimer !== undefined) {
    clearTimeout(dismissTimer)
    dismissTimer = undefined
  }
}

function dismiss() {
  if (!visible.value) return
  clearDismissTimer()
  visible.value = false
  emit('dismiss')
}

function startDismissTimer() {
  clearDismissTimer()
  if (props.duration > 0) {
    dismissTimer = setTimeout(dismiss, props.duration)
  }
}

watch(() => [props.message, props.duration], () => {
  visible.value = true
  startDismissTimer()
})

onMounted(() => {
  let stack = document.getElementById(stackId)
  if (!stack) {
    stack = document.createElement('div')
    stack.id = stackId
    stack.className = 'sdp-toast-stack'
    stack.setAttribute('aria-label', 'Notifications')
    document.body.appendChild(stack)
  }

  stackUsers += 1
  stackReady.value = true
  startDismissTimer()
})

onBeforeUnmount(() => {
  clearDismissTimer()
  stackUsers = Math.max(0, stackUsers - 1)
  if (stackUsers === 0) {
    document.getElementById(stackId)?.remove()
  }
})
</script>

<style scoped>
:global(.sdp-toast-stack) {
  position: fixed;
  z-index: var(--z-toast);
  top: var(--space-4);
  right: var(--space-4);
  display: flex;
  width: min(calc(100% - (var(--space-4) * 2)), 24rem);
  flex-direction: column;
  gap: var(--space-3);
  pointer-events: none;
}

.sdp-toast {
  display: grid;
  grid-template-columns: var(--space-6) 1fr auto;
  gap: var(--space-3);
  align-items: start;
  padding: var(--space-4);
  border: 1px solid var(--ink-300);
  border-left: var(--space-1) solid var(--brand-600);
  border-radius: var(--radius-md);
  color: var(--ink-900);
  background: var(--ink-50);
  box-shadow: var(--shadow-lg);
  font-family: var(--font-body);
  pointer-events: auto;
}

.sdp-toast--warning {
  border-left-color: var(--ink-600);
}

.sdp-toast--danger {
  border-left-color: var(--ink-900);
}

.sdp-toast__icon {
  display: inline-flex;
  width: var(--space-6);
  height: var(--space-6);
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-pill);
  color: var(--brand-800);
  background: var(--brand-100);
}

.sdp-toast--warning .sdp-toast__icon,
.sdp-toast--danger .sdp-toast__icon {
  color: var(--ink-900);
  background: var(--ink-200);
}

.sdp-toast__icon svg,
.sdp-toast__close svg {
  width: var(--space-4);
  height: var(--space-4);
  stroke: currentColor;
  stroke-width: 2;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.sdp-toast__message {
  padding-block: var(--space-1);
  color: var(--ink-900);
  font-size: var(--text-sm);
  line-height: var(--leading-normal);
}

.sdp-toast__close {
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

.sdp-toast__close:hover {
  color: var(--ink-900);
  background: var(--ink-100);
}

.sdp-toast__close:focus-visible {
  outline: 2px solid var(--brand-500);
  outline-offset: 2px;
}

@media (max-width: 639px) {
  :global(.sdp-toast-stack) {
    right: var(--space-3);
    left: var(--space-3);
    width: auto;
  }
}
</style>
