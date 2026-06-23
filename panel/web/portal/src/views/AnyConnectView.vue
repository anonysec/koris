<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from '@koris/composables/useI18n'
import KButton from '@koris/ui/KButton.vue'

const { t } = useI18n()
const downloading = ref(false)

function handleDownloadProfile() {
  downloading.value = true
  // Open profile download in new tab/window — backend returns the file directly
  window.open('/api/portal/anyconnect/profile', '_blank')
  setTimeout(() => { downloading.value = false }, 1500)
}
</script>
<template>
  <div class="anyconnect">
    <h1 class="anyconnect__title">Cisco AnyConnect</h1>
    <p class="anyconnect__subtitle">{{ t('portal.cisco.desc') }}</p>

    <!-- Download Section -->
    <section class="anyconnect__section anyconnect__section--primary">
      <h2 class="anyconnect__section-title">📥 Download Profile</h2>
      <p class="anyconnect__section-desc">
        Download your AnyConnect connection profile to import into the Cisco AnyConnect client.
      </p>

      <KButton
        variant="primary"
        :loading="downloading"
        @click="handleDownloadProfile"
      >
        📄 Download Profile
      </KButton>
    </section>

    <!-- Connection Instructions -->
    <section class="anyconnect__section">
      <h2 class="anyconnect__section-title">📋 {{ t('portal.cisco.setupTitle') }}</h2>

      <div class="anyconnect__instructions">
        <div class="anyconnect__step">
          <div class="anyconnect__step-num">1</div>
          <div class="anyconnect__step-text">
            Download and install the <strong>Cisco AnyConnect Secure Mobility Client</strong> from your platform's app store or the official Cisco website.
          </div>
        </div>

        <div class="anyconnect__step">
          <div class="anyconnect__step-num">2</div>
          <div class="anyconnect__step-text">
            Download your connection profile using the button above.
          </div>
        </div>

        <div class="anyconnect__step">
          <div class="anyconnect__step-num">3</div>
          <div class="anyconnect__step-text">
            Import the downloaded profile into the AnyConnect client, or enter the server address manually in the connection field.
          </div>
        </div>

        <div class="anyconnect__step">
          <div class="anyconnect__step-num">4</div>
          <div class="anyconnect__step-text">
            Use your VPN username and password to authenticate when prompted.
          </div>
        </div>
      </div>

      <div class="anyconnect__platforms">
        <h3 class="anyconnect__platforms-title">Supported Platforms</h3>
        <div class="anyconnect__platform-grid">
          <div class="anyconnect__platform-card">
            <span class="anyconnect__platform-icon">🖥️</span>
            <span class="anyconnect__platform-name">Windows</span>
          </div>
          <div class="anyconnect__platform-card">
            <span class="anyconnect__platform-icon">🍎</span>
            <span class="anyconnect__platform-name">macOS</span>
          </div>
          <div class="anyconnect__platform-card">
            <span class="anyconnect__platform-icon">🐧</span>
            <span class="anyconnect__platform-name">Linux</span>
          </div>
          <div class="anyconnect__platform-card">
            <span class="anyconnect__platform-icon">📱</span>
            <span class="anyconnect__platform-name">iOS / Android</span>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>
<style scoped>
.anyconnect {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
  padding-bottom: calc(var(--space-8) + env(safe-area-inset-bottom, 20px));
}
.anyconnect__title {
  font-size: var(--text-xl);
  font-weight: 700;
}
.anyconnect__subtitle {
  font-size: var(--text-sm);
  color: var(--color-muted);
  margin-top: calc(-1 * var(--space-3));
}
.anyconnect__section {
  padding: var(--space-5);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}
.anyconnect__section--primary {
  border-color: var(--color-primary);
  border-width: 2px;
}
.anyconnect__section-title {
  font-size: var(--text-md);
  font-weight: 600;
  margin-bottom: var(--space-2);
}
.anyconnect__section-desc {
  font-size: var(--text-sm);
  color: var(--color-muted);
  margin-bottom: var(--space-4);
  line-height: 1.5;
}
.anyconnect__instructions {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  margin-bottom: var(--space-6);
}
.anyconnect__step {
  display: flex;
  gap: var(--space-3);
  align-items: flex-start;
}
.anyconnect__step-num {
  width: 28px;
  height: 28px;
  min-width: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-primary);
  color: #fff;
  border-radius: 50%;
  font-size: var(--text-sm);
  font-weight: 600;
}
.anyconnect__step-text {
  font-size: var(--text-sm);
  color: var(--color-text);
  line-height: 1.6;
  padding-top: var(--space-1);
}
.anyconnect__platforms {
  margin-top: var(--space-4);
  padding-top: var(--space-4);
  border-top: 1px solid var(--color-border);
}
.anyconnect__platforms-title {
  font-size: var(--text-sm);
  font-weight: 600;
  margin-bottom: var(--space-3);
}
.anyconnect__platform-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: var(--space-3);
}
.anyconnect__platform-card {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3);
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}
.anyconnect__platform-icon {
  font-size: 1.2rem;
}
.anyconnect__platform-name {
  font-size: var(--text-sm);
  font-weight: 500;
}
</style>
