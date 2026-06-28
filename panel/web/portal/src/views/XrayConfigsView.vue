<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useClipboard } from '@koris/composables/useClipboard'
import { useI18n } from '@koris/composables/useI18n'
import KButton from '@koris/ui/KButton.vue'
import KSkeleton from '@koris/ui/KSkeleton.vue'
import KEmptyState from '@koris/ui/KEmptyState.vue'
import ConfigLinkCard from '@/components/ConfigLinkCard.vue'
import QRCode from '@/components/QRCode.vue'

interface XrayConfig {
  protocol: string
  link: string
  qr_data: string
  remark?: string
  node_name?: string
}

interface XrayConfigsResponse {
  ok: boolean
  configs: XrayConfig[]
  subscription_url: string
}

const { get, loading } = useApi()
const { copy, copied } = useClipboard()
const { t } = useI18n()

const configs = ref<XrayConfig[]>([])
const subscriptionUrl = ref('')
const showSubQR = ref(false)

// Group configs by protocol
const groupedConfigs = computed(() => {
  const groups: Record<string, XrayConfig[]> = {}
  for (const config of configs.value) {
    const proto = config.protocol.toLowerCase()
    if (!groups[proto]) groups[proto] = []
    groups[proto].push(config)
  }
  return groups
})

const protocolOrder = ['vless', 'vmess', 'trojan', 'shadowsocks', 'ss']

const sortedGroups = computed(() => {
  const entries = Object.entries(groupedConfigs.value)
  entries.sort((a, b) => {
    const aIdx = protocolOrder.indexOf(a[0])
    const bIdx = protocolOrder.indexOf(b[0])
    return (aIdx === -1 ? 99 : aIdx) - (bIdx === -1 ? 99 : bIdx)
  })
  return entries
})

// App download links
const appLinks = [
  { name: 'v2rayNG', platform: 'Android', icon: '📱', url: 'https://play.google.com/store/apps/details?id=com.v2ray.ang' },
  { name: 'Shadowrocket', platform: 'iOS', icon: '🍎', url: 'https://apps.apple.com/app/shadowrocket/id932747118' },
  { name: 'Clash for Android', platform: 'Android', icon: '🤖', url: 'https://play.google.com/store/apps/details?id=com.github.kr328.clash' },
  { name: 'v2rayN', platform: 'Windows', icon: '🖥️', url: 'https://github.com/2dust/v2rayN/releases' },
  { name: 'Clash Verge', platform: 'Desktop', icon: '💻', url: 'https://github.com/clash-verge-rev/clash-verge-rev/releases' },
]

onMounted(async () => {
  try {
    const res = await get<XrayConfigsResponse>('/api/portal/xray/links')
    configs.value = res.configs || []
    subscriptionUrl.value = res.subscription_url || ''
  } catch {
    // keep empty state
  }
})

function handleCopySubUrl() {
  if (subscriptionUrl.value) {
    copy(subscriptionUrl.value)
  }
}

function toggleSubQR() {
  showSubQR.value = !showSubQR.value
}
</script>
<template>
  <div class="xray-configs">
    <h1 class="xray-configs__title">{{ t('portal.xray.title') }}</h1>
    <p class="xray-configs__subtitle">{{ t('portal.xray.subtitle') }}</p>

    <KSkeleton v-if="loading && !configs.length" type="card" :count="3" />

    <template v-else>
      <!-- Subscription Link Section -->
      <section v-if="subscriptionUrl" class="xray-configs__section xray-configs__section--sub">
        <div class="xray-configs__section-header">
          <h2 class="xray-configs__section-title">
            <svg viewBox="0 0 20 20" fill="currentColor" width="20" height="20"><path fill-rule="evenodd" d="M12.586 4.586a2 2 0 112.828 2.828l-3 3a2 2 0 01-2.828 0 1 1 0 00-1.414 1.414 4 4 0 005.656 0l3-3a4 4 0 00-5.656-5.656l-1.5 1.5a1 1 0 101.414 1.414l1.5-1.5zm-5 5a2 2 0 012.828 0 1 1 0 101.414-1.414 4 4 0 00-5.656 0l-3 3a4 4 0 105.656 5.656l1.5-1.5a1 1 0 10-1.414-1.414l-1.5 1.5a2 2 0 11-2.828-2.828l3-3z" clip-rule="evenodd"/></svg>
            {{ t('portal.xray.subscriptionTitle') }}
          </h2>
        </div>
        <p class="xray-configs__section-desc">{{ t('portal.xray.subscriptionDesc') }}</p>

        <div class="xray-configs__sub-url-row">
          <input
            type="text"
            :value="subscriptionUrl"
            class="xray-configs__url-input"
            readonly
            @click="($event.target as HTMLInputElement).select()"
          />
          <button class="xray-configs__btn xray-configs__btn--primary" @click="handleCopySubUrl" type="button">
            {{ copied ? '✓ ' + t('portal.xray.copied') : '📋 ' + t('portal.xray.copy') }}
          </button>
          <button class="xray-configs__btn" @click="toggleSubQR" type="button">
            {{ showSubQR ? '🔼' : '📱 QR' }}
          </button>
        </div>

        <QRCode
          v-if="showSubQR"
          :value="subscriptionUrl"
          :size="260"
          :visible="showSubQR"
        />
      </section>

      <!-- Config Links Section -->
      <section v-if="configs.length" class="xray-configs__section">
        <h2 class="xray-configs__section-title">
          <svg viewBox="0 0 20 20" fill="currentColor" width="20" height="20"><path fill-rule="evenodd" d="M11.3 1.046A1 1 0 0112 2v5h4a1 1 0 01.82 1.573l-7 10A1 1 0 018 18v-5H4a1 1 0 01-.82-1.573l7-10a1 1 0 011.12-.38z" clip-rule="evenodd"/></svg>
          {{ t('portal.xray.configsTitle') }}
        </h2>
        <p class="xray-configs__section-desc">{{ t('portal.xray.configsDesc') }}</p>

        <div v-for="[protocol, items] in sortedGroups" :key="protocol" class="xray-configs__group">
          <div class="xray-configs__configs-list">
            <ConfigLinkCard
              v-for="(config, idx) in items"
              :key="`${protocol}-${idx}`"
              :config="config"
            />
          </div>
        </div>
      </section>

      <!-- Empty State -->
      <KEmptyState
        v-if="!configs.length && !loading"
        :title="t('portal.xray.noConfigs')"
        :description="t('portal.xray.noConfigsDesc')"
        icon="⚡"
      />

      <!-- App Download Links -->
      <section class="xray-configs__section">
        <h2 class="xray-configs__section-title">
          <svg viewBox="0 0 20 20" fill="currentColor" width="20" height="20"><path fill-rule="evenodd" d="M3 17a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm3.293-7.707a1 1 0 011.414 0L9 10.586V3a1 1 0 112 0v7.586l1.293-1.293a1 1 0 111.414 1.414l-3 3a1 1 0 01-1.414 0l-3-3a1 1 0 010-1.414z" clip-rule="evenodd"/></svg>
          {{ t('portal.xray.appsTitle') }}
        </h2>
        <p class="xray-configs__section-desc">{{ t('portal.xray.appsDesc') }}</p>

        <div class="xray-configs__apps-grid">
          <a
            v-for="app in appLinks"
            :key="app.name"
            :href="app.url"
            target="_blank"
            rel="noopener noreferrer"
            class="xray-configs__app-card"
          >
            <span class="xray-configs__app-icon">{{ app.icon }}</span>
            <span class="xray-configs__app-info">
              <span class="xray-configs__app-name">{{ app.name }}</span>
              <span class="xray-configs__app-platform">{{ app.platform }}</span>
            </span>
          </a>
        </div>
      </section>
    </template>
  </div>
</template>
<style scoped>
.xray-configs {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
  padding-bottom: calc(var(--space-8) + env(safe-area-inset-bottom, 20px));
}
.xray-configs__title {
  font-size: var(--text-xl);
  font-weight: 700;
}
.xray-configs__subtitle {
  font-size: var(--text-sm);
  color: var(--color-muted);
  margin-top: calc(-1 * var(--space-3));
}

/* Sections */
.xray-configs__section {
  padding: var(--space-5);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}
.xray-configs__section--sub {
  border-color: var(--color-primary);
  border-width: 2px;
}
.xray-configs__section-header {
  margin-bottom: var(--space-2);
}
.xray-configs__section-title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--text-md);
  font-weight: 600;
  margin-bottom: var(--space-2);
  color: var(--color-text);
}
.xray-configs__section-title svg {
  color: var(--color-primary);
  flex-shrink: 0;
}
.xray-configs__section-desc {
  font-size: var(--text-sm);
  color: var(--color-muted);
  margin-bottom: var(--space-4);
  line-height: 1.5;
}

/* Subscription URL */
.xray-configs__sub-url-row {
  display: flex;
  gap: var(--space-2);
  align-items: center;
  margin-bottom: var(--space-3);
}
.xray-configs__url-input {
  flex: 1;
  padding: var(--space-2) var(--space-3);
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  color: var(--color-text);
  font-size: var(--text-sm);
  font-family: monospace;
  min-width: 0;
}
.xray-configs__btn {
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-2) var(--space-3);
  font-size: var(--text-sm);
  cursor: pointer;
  transition: background 0.15s;
  color: var(--color-text);
  white-space: nowrap;
}
.xray-configs__btn:hover {
  background: var(--color-surface-2);
}
.xray-configs__btn--primary {
  background: var(--color-primary);
  color: #fff;
  border-color: var(--color-primary);
}
.xray-configs__btn--primary:hover {
  opacity: 0.9;
  background: var(--color-primary);
}

/* Config groups */
.xray-configs__group {
  margin-bottom: var(--space-3);
}
.xray-configs__group:last-child {
  margin-bottom: 0;
}
.xray-configs__configs-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

/* App links */
.xray-configs__apps-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: var(--space-3);
}
.xray-configs__app-card {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  text-decoration: none;
  color: var(--color-text);
  transition: border-color 0.15s, box-shadow 0.15s;
}
.xray-configs__app-card:hover {
  border-color: var(--color-primary);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}
.xray-configs__app-icon {
  font-size: 1.5rem;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-surface-2);
  border-radius: var(--radius-md);
  flex-shrink: 0;
}
.xray-configs__app-info {
  display: flex;
  flex-direction: column;
}
.xray-configs__app-name {
  font-size: var(--text-sm);
  font-weight: 600;
}
.xray-configs__app-platform {
  font-size: var(--text-xs);
  color: var(--color-muted);
}

/* Mobile */
@media (max-width: 640px) {
  .xray-configs__sub-url-row {
    flex-direction: column;
    align-items: stretch;
  }
  .xray-configs__sub-url-row .xray-configs__btn {
    text-align: center;
  }
  .xray-configs__apps-grid {
    grid-template-columns: 1fr;
  }
}
</style>
