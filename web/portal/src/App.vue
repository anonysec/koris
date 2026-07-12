<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useTheme, availableThemes } from '@koris/composables/useTheme'
import { useDirection } from '@koris/composables/useDirection'
import type { ThemeMode, UITheme } from '@koris/composables/useTheme'
import router from './router'

const { get } = useApi()
const { setMode, setTheme } = useTheme()
useDirection()

// Show a splash until the initial route (including the auth check in the
// router guard) has resolved — otherwise a hard refresh of an authenticated
// /account/ page flashes a blank white screen while /api/portal/me is in flight.
const ready = ref(false)
router.isReady().then(() => { ready.value = true })

onMounted(async () => {
  try {
    const res = await get<{ ok: boolean; settings: Record<string, string> }>('/api/public-settings')
    if (res.settings) {
      if (res.settings.ui_theme && availableThemes.some((t) => t.id === res.settings.ui_theme)) {
        setTheme(res.settings.ui_theme as UITheme)
      }
      if (res.settings.ui_mode && ['system', 'dark', 'light'].includes(res.settings.ui_mode)) {
        setMode(res.settings.ui_mode as ThemeMode)
      }
    }
  } catch {
    // Use localStorage defaults on error
  }
})
</script>

<template>
  <div v-if="!ready" class="portal-boot" role="status" aria-label="Loading">
    <span class="portal-boot__spinner" />
  </div>
  <router-view v-else />
</template>

<style>
.portal-boot {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg, #070a12);
  z-index: 2000;
}
.portal-boot__spinner {
  width: 36px;
  height: 36px;
  border: 3px solid var(--color-border, #28333f);
  border-top-color: var(--color-primary, #6366f1);
  border-radius: 50%;
  animation: portal-boot-spin 0.8s linear infinite;
}
@keyframes portal-boot-spin {
  to { transform: rotate(360deg); }
}
@media (prefers-reduced-motion: reduce) {
  .portal-boot__spinner { animation-duration: 2s; }
}
</style>
