<script setup lang="ts">
/**
 * SlideOver — A slide-over panel from the right side.
 *
 * This is a convenience wrapper around Drawer with a simplified API.
 * - Props: open (boolean), title (string), width (string, default '480px')
 * - Emits: close
 * - Accessible: role="dialog", aria-modal, focus trap, Escape to close
 * - Mobile (max-width 640px): goes full-width
 */
import Drawer from './Drawer.vue'

withDefaults(defineProps<{
  open: boolean
  title?: string
  width?: string
}>(), {
  title: '',
  width: '480px',
})

defineEmits<{
  (e: 'close'): void
}>()
</script>

<template>
  <Drawer
    :open="open"
    :title="title"
    :width="width"
    side="right"
    :closable="true"
    :overlay="true"
    class="k-slide-over"
    @close="$emit('close')"
  >
    <slot />
    <template v-if="$slots.footer" #footer>
      <slot name="footer" />
    </template>
  </Drawer>
</template>

<style>
/* Mobile full-width override for SlideOver */
@media (max-width: 640px) {
  .k-slide-over.k-drawer,
  .k-slide-over .k-drawer {
    width: 100vw !important;
    max-width: 100vw !important;
    border-radius: 0 !important;
  }
}
</style>
