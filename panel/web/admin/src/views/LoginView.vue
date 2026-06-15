<script setup lang="ts">
import { ref } from 'vue'
import { useAuthStore } from '@/stores/auth'

const store = useAuthStore()

const username = ref('')
const password = ref('')

async function handleLogin() {
  await store.login(username.value, password.value)
}
</script>

<template>
  <div class="page" style="display:flex;align-items:center;justify-content:center;min-height:100vh">
    <form class="login-form" style="width:100%;max-width:360px" @submit.prevent="handleLogin">
      <h3 style="margin-bottom:1.5rem">Admin Login</h3>
      <div style="margin-bottom:1rem">
        <label for="username" style="display:block;margin-bottom:0.25rem;color:var(--color-muted)">Username</label>
        <input id="username" v-model="username" type="text" autocomplete="username" required style="width:100%;padding:0.5rem;border-radius:0.375rem;border:1px solid var(--color-border, #1e293b);background:var(--color-surface, #0f1629);color:inherit" />
      </div>
      <div style="margin-bottom:1.5rem">
        <label for="password" style="display:block;margin-bottom:0.25rem;color:var(--color-muted)">Password</label>
        <input id="password" v-model="password" type="password" autocomplete="current-password" required style="width:100%;padding:0.5rem;border-radius:0.375rem;border:1px solid var(--color-border, #1e293b);background:var(--color-surface, #0f1629);color:inherit" />
      </div>
      <button type="submit" :disabled="store.loading" style="width:100%;padding:0.625rem;border-radius:0.375rem;background:var(--color-primary, #2563eb);color:#fff;border:none;cursor:pointer;font-weight:500">
        {{ store.loading ? 'Signing in...' : 'Sign In' }}
      </button>
      <p v-if="store.error" style="color:var(--color-danger, #ef4444);margin-top:0.75rem;font-size:0.875rem">{{ store.error }}</p>
    </form>
  </div>
</template>
