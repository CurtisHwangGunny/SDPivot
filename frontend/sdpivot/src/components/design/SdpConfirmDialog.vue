<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="sdp-confirm-dialog__backdrop"
      @mousedown.self="handleCancel"
    >
      <section
        ref="dialogRef"
        class="sdp-confirm-dialog"
        :class="`sdp-confirm-dialog--${type}`"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="titleId"
        :aria-describedby="messageId"
        tabindex="-1"
        @keydown="handleKeydown"
      >
        <div class="sdp-confirm-dialog__heading">
          <span class="sdp-confirm-dialog__icon" aria-hidden="true">
            <svg viewBox="0 0 24 24" fill="none">
              <path v-if="type === 'normal'" d="M12 8h.01M11 12h1v4h1M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
              <template v-else>
                <path d="M10.3 4.2 2.8 17a2 2 0 0 0 1.7 3h15a2 2 0 0 0 1.7-3L13.7 4.2a2 2 0 0 0-3.4 0Z" />
                <path d="M12 9v4M12 17h.01" />
              </template>
            </svg>
          </span>
          <div class="sdp-confirm-dialog__copy">
            <h2 :id="titleId" class="sdp-confirm-dialog__title">{{ title }}</h2>
            <p :id="messageId" class="sdp-confirm-dialog__message">{{ message }}</p>
          </div>
        </div>
        <div class="sdp-confirm-dialog__actions">
          <SdpButton
            ref="cancelButtonRef"
            variant="secondary"
            :disabled="loading"
            :aria-label="cancelText"
            @click="handleCancel"
          >
            {{ cancelText }}
          </SdpButton>
          <SdpButton
            :variant="type === 'danger' ? 'danger' : 'primary'"
            :loading="loading"
            :aria-label="confirmText"
            @click="emit('confirm')"
          >
            {{ confirmText }}
          </SdpButton>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, useId, watch } from 'vue'
import SdpButton from './SdpButton.vue'

const props = withDefaults(defineProps<{
  type?: 'normal' | 'warning' | 'danger'
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  loading?: boolean
}>(), {
  type: 'normal',
  confirmText: 'Confirm',
  cancelText: 'Cancel',
  loading: false,
})

const visible = defineModel<boolean>('visible', { default: false })
const emit = defineEmits<{
  confirm: []
  cancel: []
}>()

const dialogRef = ref<HTMLElement | null>(null)
const cancelButtonRef = ref<InstanceType<typeof SdpButton> | null>(null)
const titleId = `sdp-confirm-dialog-title-${useId()}`
const messageId = `sdp-confirm-dialog-message-${useId()}`
let previouslyFocused: HTMLElement | null = null

const focusableSelector = [
  'button:not([disabled])',
  '[href]',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(',')

function closeDialog() {
  visible.value = false
}

function handleCancel() {
  if (props.loading) return
  emit('cancel')
  closeDialog()
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
    emit('cancel')
    closeDialog()
    return
  }

  if (event.key !== 'Tab' || !dialogRef.value) return

  const focusable = Array.from(dialogRef.value.querySelectorAll<HTMLElement>(focusableSelector))
  if (focusable.length === 0) {
    event.preventDefault()
    dialogRef.value.focus()
    return
  }

  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first.focus()
  }
}

watch(visible, async (isVisible) => {
  if (isVisible) {
    previouslyFocused = document.activeElement instanceof HTMLElement ? document.activeElement : null
    await nextTick()
    const cancelElement = cancelButtonRef.value?.$el as HTMLElement | undefined
    if (cancelElement && !cancelElement.matches(':disabled')) {
      cancelElement.focus()
    } else {
      dialogRef.value?.focus()
    }
    return
  }

  previouslyFocused?.focus()
  previouslyFocused = null
})

onBeforeUnmount(() => {
  previouslyFocused?.focus()
})
</script>

<style scoped>
.sdp-confirm-dialog__backdrop {
  position: fixed;
  z-index: var(--z-modal);
  inset: var(--space-0);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-4);
  background: var(--ink-950);
}

.sdp-confirm-dialog {
  display: grid;
  gap: var(--space-6);
  width: min(100%, 30rem);
  padding: var(--space-6);
  border: 1px solid var(--ink-200);
  border-radius: var(--radius-lg);
  color: var(--ink-900);
  background: var(--ink-50);
  box-shadow: var(--shadow-xl);
  font-family: var(--font-body);
}

.sdp-confirm-dialog:focus-visible {
  outline: 2px solid var(--brand-500);
  outline-offset: 2px;
}

.sdp-confirm-dialog__heading {
  display: grid;
  grid-template-columns: var(--space-10) 1fr;
  gap: var(--space-4);
  align-items: start;
}

.sdp-confirm-dialog__icon {
  display: inline-flex;
  width: var(--space-10);
  height: var(--space-10);
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-pill);
  color: var(--brand-700);
  background: var(--brand-100);
}

.sdp-confirm-dialog--warning .sdp-confirm-dialog__icon {
  color: var(--ink-900);
  background: var(--ink-200);
}

.sdp-confirm-dialog--danger .sdp-confirm-dialog__icon {
  color: var(--ink-950);
  background: var(--ink-200);
}

.sdp-confirm-dialog__icon svg {
  width: var(--space-6);
  height: var(--space-6);
  stroke: currentColor;
  stroke-width: 1.8;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.sdp-confirm-dialog__copy {
  display: grid;
  gap: var(--space-2);
}

.sdp-confirm-dialog__title {
  color: var(--ink-900);
  font-family: var(--font-display);
  font-size: var(--text-xl);
  font-weight: var(--font-weight-semibold);
  line-height: var(--leading-tight);
}

.sdp-confirm-dialog__message {
  color: var(--ink-700);
  font-size: var(--text-sm);
  line-height: var(--leading-relaxed);
}

.sdp-confirm-dialog__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: var(--space-3);
}

@media (max-width: 639px) {
  .sdp-confirm-dialog {
    padding: var(--space-5);
  }

  .sdp-confirm-dialog__heading {
    grid-template-columns: 1fr;
  }

  .sdp-confirm-dialog__actions {
    flex-direction: column-reverse;
  }

  .sdp-confirm-dialog__actions :deep(.sdp-button) {
    width: 100%;
  }
}
</style>
