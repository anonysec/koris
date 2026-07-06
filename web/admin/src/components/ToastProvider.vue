<template>
  <div class="toast-provider" aria-live="polite" aria-relevant="additions removals">
    <TransitionGroup name="toast-list">
      <Toast
        v-for="toast in toasts"
        :key="toast.id"
        :message="toast.message"
        :type="toast.type"
        :duration="toast.duration"
        :visible="true"
        @close="removeToast(toast.id)"
      />
    </TransitionGroup>
  </div>
</template>

<script setup lang="ts">
import Toast from '@koris/ui/Toast.vue'
import { useToast } from '@koris/composables/useToast'

const { toasts, removeToast } = useToast()
</script>

<style scoped>
.toast-provider {
  position: fixed;
  top: var(--space-4);
  right: var(--space-4);
  z-index: var(--z-toast);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  pointer-events: none;
}

.toast-provider > :deep(*) {
  pointer-events: auto;
}

/* ─── List Transition ─── */

.toast-list-enter-active,
.toast-list-leave-active {
  transition:
    opacity var(--duration-slow) var(--ease-out),
    transform var(--duration-slow) var(--ease-out);
}

.toast-list-enter-from {
  opacity: 0;
  transform: translateX(20px);
}

.toast-list-leave-to {
  opacity: 0;
  transform: translateX(20px);
}

.toast-list-move {
  transition: transform var(--duration-slow) var(--ease-out);
}

@media (prefers-reduced-motion: reduce) {
  .toast-list-enter-active,
  .toast-list-leave-active,
  .toast-list-move {
    transition: opacity var(--duration-fast) var(--ease-default);
  }
  .toast-list-enter-from,
  .toast-list-leave-to {
    transform: none;
  }
}
</style>
