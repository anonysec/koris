<script setup lang="ts">
import { ref } from 'vue'
import { useAuthStore } from '@/stores/auth'

const store = useAuthStore()

const username = ref('')
const password = ref('')
const setupKey = ref('')

async function handleSetup() {
  const params: { username: string; password: string; setup_key?: string } = {
    username: username.value,
    password: password.value,
  }
  if (store.setupKeyRequired && setupKey.value) {
    params.setup_key = setupKey.value
  }
  await store.setup(params)
}
</script>

<template>
  <div class="page" style="display:flex;align-items:center;justify-content:center;min-height:100vh">
    <form class="setup-form" style="width:100%;max-width:360px" @submit.prevent="handleSetup">
      <h3 style="margin-bottom:0.5rem">Initial Setup</h3>
      <p style="color:var(--color-muted);margin-bottom:1.5rem;font-size:0.875rem">Create the owner account to get started.</p>
      <div style="margin-bottom:1rem">
        <label for="setup-username" style="display:block;margin-bottom:0.25rem;color:var(--color-muted)">Username</label>
        <input id="setup-username" v-model="username" type="text" autocomplete="username" required style="width:100%;padding:0.5rem;border-radius:0.375rem;border:1px solid var(--color-border, #1e293b);background:var(--color-surface, #0f1629);color:inherit" />
      </div>
      <div style="margin-bottom:1rem">
        <label for="setup-password" style="display:block;margin-bottom:0.25rem;color:var(--color-muted)">Password</label>
        <input id="setup-password" v-model="password" type="password" autocomplete="new-password" required style="width:100%;padding:0.5rem;border-radius:0.375rem;border:1px solid var(--color-border, #1e293b);background:var(--color-surface, #0f1629);color:inherit" />
      </div>
      <div v-if="store.setupKeyRequired" style="margin-bottom:1.5rem">
        <label for="setup-key" style="display:block;margin-bottom:0.25rem;color:var(--color-muted)">Setup Key</label>
        <input id="setup-key" v-model="setupKey" type="text" style="width:100%;padding:0.5rem;border-radius:0.375rem;border:1px solid var(--color-border, #1e293b);background:var(--color-surface, #0f1629);color:inherit" />
      </div>
      <button type="submit" :disabled="store.loading" style="width:100%;padding:0.625rem;border-radius:0.375rem;background:var(--color-primary, #2563eb);color:#fff;border:none;cursor:pointer;font-weight:500">
        {{ store.loading ? 'Creating...' : 'Create Account' }}
      </button>
      <p v-if="store.error" style="color:var(--color-danger, #ef4444);margin-top:0.75rem;font-size:0.875rem">{{ store.error }}</p>
    </form>
  </div>
</template>
