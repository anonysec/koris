<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { usePortalAuthStore } from '@/stores/auth'
import { useTheme } from '@koris/composables/useTheme'
import NotificationCenter from '@/components/NotificationCenter.vue'

const router = useRouter()
const auth = usePortalAuthStore()
const { isDark, toggle: toggleTheme } = useTheme()

const userMenuOpen = ref(false)

function toggleUserMenu() {
  userMenuOpen.value = !userMenuOpen.value
}

function closeUserMenu() {
  userMenuOpen.value = false
}

function goToProfile() {
  closeUserMenu()
  router.push({ name: 'portal-profile' })
}

async function logout() {
  closeUserMenu()
  await auth.logout()
  router.push({ name: 'portal-login' })
}
</script>
<template>
  <div class="portal-shell">
    <header class="portal-nav">
      <div class="portal-nav__brand"><span class="portal-nav__logo">K</span><span class="portal-nav__title">KorisPanel</span></div>
      <nav class="portal-nav__links">
        <router-link :to="{ name: 'portal-dashboard' }">Dashboard</router-link>
        <router-link :to="{ name: 'portal-billing' }">Billing</router-link>
        <router-link :to="{ name: 'portal-support' }">Support</router-link>
        <router-link :to="{ name: 'portal-vpn' }">VPN Profiles</router-link>
      </nav>
      <div class="portal-nav__actions">
        <NotificationCenter />
        <button @click="toggleTheme" class="portal-nav__btn">{{ isDark ? '☀️' : '🌙' }}</button>
        <div class="portal-nav__user-menu">
          <button class="portal-nav__user-toggle" @click="toggleUserMenu">
            <span class="portal-nav__user">{{ auth.user?.username }}</span>
            <svg class="portal-nav__chevron" :class="{ 'portal-nav__chevron--open': userMenuOpen }" viewBox="0 0 20 20" fill="currentColor" width="16" height="16">
              <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clip-rule="evenodd" />
            </svg>
          </button>
          <div v-if="userMenuOpen" class="portal-nav__dropdown-backdrop" @click="closeUserMenu"></div>
          <div v-if="userMenuOpen" class="portal-nav__dropdown">
            <div class="portal-nav__dropdown-header">{{ auth.user?.username }}</div>
            <button class="portal-nav__dropdown-item" @click="goToProfile">Profile Settings</button>
            <button class="portal-nav__dropdown-item portal-nav__dropdown-item--danger" @click="logout">Logout</button>
          </div>
        </div>
      </div>
    </header>
    <main class="portal-main">
      <router-view v-slot="{ Component }">
        <transition name="fade" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </main>
  </div>
</template>
<style scoped>
.portal-shell { min-height:100vh;background:var(--color-bg); }
.portal-nav { display:flex;align-items:center;gap:var(--space-4);padding:var(--space-3) var(--space-6);border-bottom:1px solid var(--color-border);background:var(--color-surface); }
.portal-nav__brand { display:flex;align-items:center;gap:var(--space-2); }
.portal-nav__logo { width:32px;height:32px;border-radius:var(--radius-md);background:var(--gradient-brand);display:flex;align-items:center;justify-content:center;color:#fff;font-weight:800;font-size:14px; }
.portal-nav__title { font-weight:700;font-size:var(--text-md); }
.portal-nav__links { display:flex;gap:var(--space-1);margin-left:var(--space-6); }
.portal-nav__links a { padding:var(--space-2) var(--space-3);border-radius:var(--radius-md);font-size:var(--text-sm);color:var(--color-muted);text-decoration:none;transition:all var(--duration-fast); }
.portal-nav__links a:hover { color:var(--color-text);background:var(--color-surface-2); }
.portal-nav__links a.router-link-active { color:var(--color-primary);background:rgba(37,99,235,0.08); }
.portal-nav__actions { margin-left:auto;display:flex;align-items:center;gap:var(--space-3); }
.portal-nav__user { font-size:var(--text-sm);color:var(--color-muted); }
.portal-nav__btn { background:none;border:none;color:var(--color-muted);cursor:pointer;font-size:var(--text-sm);padding:var(--space-1) var(--space-2);border-radius:var(--radius-sm); }
.portal-nav__btn:hover { color:var(--color-text); }
.portal-nav__user-menu { position:relative; }
.portal-nav__user-toggle { display:flex;align-items:center;gap:var(--space-1);background:none;border:none;cursor:pointer;padding:var(--space-1) var(--space-2);border-radius:var(--radius-sm);transition:background var(--duration-fast); }
.portal-nav__user-toggle:hover { background:var(--color-surface-2); }
.portal-nav__chevron { transition:transform var(--duration-fast);color:var(--color-muted); }
.portal-nav__chevron--open { transform:rotate(180deg); }
.portal-nav__dropdown { position:absolute;top:calc(100% + var(--space-2));right:0;min-width:180px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-md);box-shadow:0 4px 12px rgba(0,0,0,0.1);z-index:100;overflow:hidden; }
.portal-nav__dropdown-backdrop { position:fixed;inset:0;z-index:99; }
.portal-nav__dropdown-header { padding:var(--space-3) var(--space-4);font-size:var(--text-xs);color:var(--color-muted);border-bottom:1px solid var(--color-border);font-weight:500; }
.portal-nav__dropdown-item { display:block;width:100%;padding:var(--space-3) var(--space-4);font-size:var(--text-sm);color:var(--color-text);background:none;border:none;text-align:left;cursor:pointer;transition:background var(--duration-fast); }
.portal-nav__dropdown-item:hover { background:var(--color-surface-2); }
.portal-nav__dropdown-item--danger { color:var(--color-danger); }
.portal-nav__dropdown-item--danger:hover { background:var(--color-danger-bg, #fef2f2); }
.portal-main { padding:var(--space-6);max-width:1200px;margin:0 auto; }
.fade-enter-active, .fade-leave-active { transition:opacity 0.2s ease; }
.fade-enter-from, .fade-leave-to { opacity:0; }
</style>
