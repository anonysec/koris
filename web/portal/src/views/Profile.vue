<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { usePortalAuthStore } from '@/stores/auth'
import { useI18n } from '@koris/composables/useI18n'
import Button from '@koris/ui/Button.vue'
import FormField from '@koris/ui/FormField.vue'
import Input from '@koris/ui/Input.vue'

const router = useRouter()
const auth = usePortalAuthStore()
const { t } = useI18n()

const profileForm = ref({
  display_name: auth.user?.display_name || '',
})

const passwordForm = ref({
  current_password: '',
  new_password: '',
  confirm_password: '',
})

const notice = ref('')
const passwordError = ref('')

async function handleUpdateProfile() {
  notice.value = ''
  const success = await auth.updateProfile({
    display_name: profileForm.value.display_name,
  })
  if (success) {
    notice.value = t('portal.profile.updated')
  }
}

async function handleChangePassword() {
  notice.value = ''
  passwordError.value = ''

  if (!passwordForm.value.current_password || !passwordForm.value.new_password) {
    passwordError.value = t('portal.profile.fillAll')
    return
  }

  if (passwordForm.value.new_password !== passwordForm.value.confirm_password) {
    passwordError.value = t('portal.profile.mismatch')
    return
  }

  if (passwordForm.value.new_password.length < 6) {
    passwordError.value = t('portal.profile.tooShort')
    return
  }

  const success = await auth.updateProfile({
    current_password: passwordForm.value.current_password,
    password: passwordForm.value.new_password,
  })

  if (success) {
    notice.value = t('portal.profile.passwordChanged')
    passwordForm.value = { current_password: '', new_password: '', confirm_password: '' }
  } else {
    passwordError.value = auth.error || t('portal.profile.changeFailed')
  }
}
</script>
<template>
  <div class="profile">
    <div class="profile__back">
      <Button variant="ghost" size="sm" @click="router.push({ name: 'portal-home' })">
        ← {{ t('portal.profile.back') }}
      </Button>
    </div>

    <h1 class="profile__title">{{ t('portal.profile.title') }}</h1>

    <div v-if="notice" class="profile__notice" role="status">{{ notice }}</div>

    <!-- Account Info -->
    <section class="profile__section">
      <h2 class="profile__section-title">{{ t('portal.profile.accountInfo') }}</h2>
      <div class="profile__info">
        <div class="profile__info-item">
          <span class="profile__info-label">{{ t('portal.profile.username') }}</span>
          <span class="profile__info-value">{{ auth.username }}</span>
        </div>
        <div class="profile__info-item">
          <span class="profile__info-label">{{ t('portal.profile.status') }}</span>
          <span class="profile__info-value">{{ auth.status }}</span>
        </div>
        <div class="profile__info-item">
          <span class="profile__info-label">{{ t('portal.profile.plan') }}</span>
          <span class="profile__info-value">{{ auth.planName }}</span>
        </div>
      </div>
    </section>

    <!-- Update Display Name -->
    <section class="profile__section">
      <h2 class="profile__section-title">{{ t('portal.profile.displayName') }}</h2>
      <form class="profile__form" @submit.prevent="handleUpdateProfile">
        <FormField :label="t('portal.profile.displayName')">
          <Input v-model="profileForm.display_name" :placeholder="t('portal.profile.displayName')" />
        </FormField>
        <Button type="submit" variant="primary" :loading="auth.loading">
          {{ t('portal.profile.updateName') }}
        </Button>
      </form>
    </section>

    <!-- Change Password -->
    <section class="profile__section">
      <h2 class="profile__section-title">{{ t('portal.profile.changePassword') }}</h2>
      <form class="profile__form" @submit.prevent="handleChangePassword">
        <FormField :label="t('portal.profile.currentPassword')" :required="true">
          <Input v-model="passwordForm.current_password" type="password" :placeholder="t('portal.profile.currentPassword')" autocomplete="current-password" />
        </FormField>

        <FormField :label="t('portal.profile.newPassword')" :required="true">
          <Input v-model="passwordForm.new_password" type="password" :placeholder="t('portal.profile.newPassword')" autocomplete="new-password" />
        </FormField>

        <FormField :label="t('portal.profile.confirmPassword')" :required="true">
          <Input v-model="passwordForm.confirm_password" type="password" :placeholder="t('portal.profile.confirmPassword')" autocomplete="new-password" />
        </FormField>

        <div v-if="passwordError" class="profile__error" role="alert">
          {{ passwordError }}
        </div>

        <Button type="submit" variant="primary" :loading="auth.loading">
          {{ t('portal.profile.changePassword') }}
        </Button>
      </form>
    </section>
  </div>
</template>
<style scoped>
.profile__back {
  margin-bottom: var(--space-4);
}
.profile__title {
  font-size: var(--text-2xl);
  font-weight: 700;
  margin-bottom: var(--space-6);
}
.profile__notice {
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-md);
  background: rgba(34, 197, 94, 0.1);
  color: var(--color-success);
  font-size: var(--text-sm);
  margin-bottom: var(--space-4);
  border: 1px solid rgba(34, 197, 94, 0.2);
}
.profile__section {
  padding: var(--space-5);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  margin-bottom: var(--space-4);
}
.profile__section-title {
  font-size: var(--text-md);
  font-weight: 600;
  margin-bottom: var(--space-4);
}
.profile__info {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.profile__info-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--space-2) 0;
  border-bottom: 1px solid var(--color-border);
}
.profile__info-item:last-child {
  border-bottom: none;
}
.profile__info-label {
  font-size: var(--text-sm);
  color: var(--color-muted);
}
.profile__info-value {
  font-size: var(--text-sm);
  font-weight: 500;
}
.profile__form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  max-width: 400px;
}
.profile__error {
  padding: var(--space-3);
  border-radius: var(--radius-md);
  background: rgba(239, 68, 68, 0.1);
  color: var(--color-danger);
  font-size: var(--text-sm);
  border: 1px solid rgba(239, 68, 68, 0.2);
}
</style>
