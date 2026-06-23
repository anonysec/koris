<script setup lang="ts">
import { ref, computed } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useClipboard } from '@koris/composables/useClipboard'
import { useI18n } from '@koris/composables/useI18n'
import { useFreshData } from '@koris/composables/useFreshData'
import KButton from '@koris/ui/KButton.vue'
import KSkeleton from '@koris/ui/KSkeleton.vue'
import KEmptyState from '@koris/ui/KEmptyState.vue'
import QRCode from '@/components/QRCode.vue'

interface SubscriptionResponse {
  ok: boolean
  subscription: string
  links: string[]
}

const { get, loading } = useApi()
const { copy, copied } = useClipboard()
const { t } = useI18n()

const subscription = ref('')
const links = ref<string[]>([])
const showSubQR = ref(false)
const showLinkQR = ref<Record<number, boolean>>({})

useFreshData(async () => {
  await fetchSubscription()
})

async function fetchSubscription() {
  try {
    const res = await get<SubscriptionResponse>('/api/portal/xray/subscription?format=json')
    subscription.value = res.subscription || ''
    links.value = res.links || []
  } catch {
    // keep empty state
  }
}

function handleCopySubscription() {
  if (subscription.value) {
    copy(subscription.value)
  }
}

function handleCopyLink(link: string) {
  copy(link)
}

function toggleSubQR() {
  showSubQR.value = !showSubQR.value
}

function toggleLinkQR(index: number) {
  showLinkQR.value[index] = !showLinkQR.value[index]
}

function getProtocolLabel(link: string): string {
  if (link.startsWith('vless://')) return 'VLESS'
  if (link.startsWith('vmess://')) return 'VMess'
  if (link.startsWith('trojan://')) return 'Trojan'
  if (link.startsWith('ss://')) return 'Shadowsocks'
  return 'Unknown'
}

function getProtocolIcon(link: string): string {
  if (link.startsWith('vless://')) return '⚡'
  if (link.startsWith('vmess://')) return '🔷'
  if (link.startsWith('trojan://')) return '🐴'
  if (link.startsWith('ss://')) return '🔮'
  return '🔗'
}

function getLinkRemark(link: string): string {
  try {
    // Most links have remark after # at the end
    const hashIdx = link.lastIndexOf('#')
    if (hashIdx !== -1) {
      return decodeURIComponent(link.substring(hashIdx + 1))
    }
  } catch { /* ignore */ }
  return ''
}
</script>
<template>
  <div class="xray-sub">
    <h1 class="xray-sub__title">{{ t('portal.xray.title') }}</h1>
    <p class="xray-sub__subtitle">{{ t('portal.xray.subtitle') }}</p>

    <KSkeleton v-if="loading && !subscription && !links.length" type="card" :count="2" />

    <template v-else>
      <!-- Subscription Link Section -->
      <section v-if="subscription" class="xray-sub__section xray-sub__section--primary">
        <h2 class="xray-sub__section-title">
          🔗 {{ t('portal.xray.subscriptionTitle') }}
        </h2>
        <p class="xray-sub__section-desc">{{ t('portal.xray.subscriptionDesc') }}</p>

        <div class="xray-sub__url-row">
          <input
            type="text"
            :value="subscription"
            class="xray-sub__url-input"
            readonly
            @click="($event.target as HTMLInputElement).select()"
          />
          <KButton variant="primary" size="sm" @click="handleCopySubscription">
            {{ copied ? '✓ ' + t('portal.xray.copied') : '📋 ' + t('portal.xray.copy') }}
          </KButton>
          <KButton variant="ghost" size="sm" @click="toggleSubQR">
            {{ showSubQR ? '🔼' : '📱 QR' }}
          </KButton>
        </div>

        <QRCode
          v-if="showSubQR"
          :value="subscription"
          :size="260"
          :visible="showSubQR"
        />

        <div class="xray-sub__actions">
          <KButton variant="ghost" size="sm" @click="fetchSubscription" :loading="loading">
            🔄 {{ t('portal.xray.refresh') || 'Refresh' }}
          </KButton>
        </div>
      </section>

      <!-- Individual Share Links -->
      <section v-if="links.length" class="xray-sub__section">
        <h2 class="xray-sub__section-title">
          ⚡ {{ t('portal.xray.configsTitle') }}
        </h2>
        <p class="xray-sub__section-desc">{{ t('portal.xray.configsDesc') }}</p>

        <div class="xray-sub__links-list">
          <div v-for="(link, idx) in links" :key="idx" class="xray-sub__link-card">
            <div class="xray-sub__link-header">
              <span class="xray-sub__link-icon">{{ getProtocolIcon(link) }}</span>
              <span class="xray-sub__link-proto">{{ getProtocolLabel(link) }}</span>
              <span v-if="getLinkRemark(link)" class="xray-sub__link-remark">{{ getLinkRemark(link) }}</span>
            </div>
            <div class="xray-sub__link-body">
              <input
                type="text"
                :value="link"
                class="xray-sub__url-input xray-sub__url-input--sm"
                readonly
                @click="($event.target as HTMLInputElement).select()"
              />
              <KButton variant="primary" size="sm" @click="handleCopyLink(link)">
                📋 {{ t('portal.xray.copy') }}
              </KButton>
              <KButton variant="ghost" size="sm" @click="toggleLinkQR(idx)">
                📱 QR
              </KButton>
            </div>
            <QRCode
              v-if="showLinkQR[idx]"
              :value="link"
              :size="200"
              :visible="showLinkQR[idx]"
            />
          </div>
        </div>
      </section>

      <!-- Empty State -->
      <KEmptyState
        v-if="!subscription && !links.length && !loading"
        :title="t('portal.xray.noConfigs')"
        :description="t('portal.xray.noConfigsDesc')"
        icon="⚡"
      />
    </template>
  </div>
</template>
<style scoped>
.xray-sub {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
  padding-bottom: calc(var(--space-8) + env(safe-area-inset-bottom, 20px));
}
.xray-sub__title {
  font-size: var(--text-xl);
  font-weight: 700;
}
.xray-sub__subtitle {
  font-size: var(--text-sm);
  color: var(--color-muted);
  margin-top: calc(-1 * var(--space-3));
}
.xray-sub__section {
  padding: var(--space-5);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}
.xray-sub__section--primary {
  border-color: var(--color-primary);
  border-width: 2px;
}
.xray-sub__section-title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--text-md);
  font-weight: 600;
  margin-bottom: var(--space-2);
}
.xray-sub__section-desc {
  font-size: var(--text-sm);
  color: var(--color-muted);
  margin-bottom: var(--space-4);
  line-height: 1.5;
}
.xray-sub__url-row {
  display: flex;
  gap: var(--space-2);
  align-items: center;
  margin-bottom: var(--space-3);
}
.xray-sub__url-input {
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
.xray-sub__url-input--sm {
  font-size: var(--text-xs);
}
.xray-sub__actions {
  margin-top: var(--space-3);
}
.xray-sub__links-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.xray-sub__link-card {
  padding: var(--space-4);
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}
.xray-sub__link-header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: var(--space-3);
}
.xray-sub__link-icon {
  font-size: 1.2rem;
}
.xray-sub__link-proto {
  font-size: var(--text-sm);
  font-weight: 600;
}
.xray-sub__link-remark {
  font-size: var(--text-xs);
  color: var(--color-muted);
  margin-inline-start: var(--space-2);
}
.xray-sub__link-body {
  display: flex;
  gap: var(--space-2);
  align-items: center;
}

/* Mobile */
@media (max-width: 640px) {
  .xray-sub__url-row,
  .xray-sub__link-body {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
