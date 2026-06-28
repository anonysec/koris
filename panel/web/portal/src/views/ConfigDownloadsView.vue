<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useClipboard } from '@koris/composables/useClipboard'
import KButton from '@koris/ui/KButton.vue'

interface VpnProfile {
  type: string
  name: string
  available: boolean
  download: string
  description?: string
  node: string
}

const { get } = useApi()
const { copy, copied } = useClipboard()
const profiles = ref<VpnProfile[]>([])
const loading = ref(false)
const downloading = ref<string | null>(null)
const copiedUrl = ref<string | null>(null)

const protocolIcons: Record<string, string> = {
  'openvpn-udp': '⚡',
  'openvpn-tcp': '🔐',
  'openvpn': '🔐',
  'l2tp': '🔒',
  'ikev2': '🛡️',
  'cisco-ipsec': '🔑',
}

async function loadProtocols() {
  loading.value = true
  try {
    const res = await get<{ ok: boolean; profiles: VpnProfile[] }>('/api/portal/profiles')
    if (res?.ok && res.profiles) {
      profiles.value = res.profiles.filter(p => p.available)
    }
  } catch {
    profiles.value = []
  } finally {
    loading.value = false
  }
}

async function downloadConfig(profile: VpnProfile) {
  downloading.value = profile.type
  try {
    const res = await fetch(profile.download, { credentials: 'include' })
    if (!res.ok) throw new Error('Download failed')
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = profile.name.replace(/[^a-zA-Z0-9_.-]/g, '_') + (profile.type.startsWith('openvpn') ? '.ovpn' : '.conf')
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  } catch {
    // Error handled silently
  } finally {
    downloading.value = null
  }
}

function copyUrl(downloadPath: string) {
  const fullUrl = `${window.location.origin}${downloadPath}`
  copy(fullUrl)
  copiedUrl.value = downloadPath
  setTimeout(() => {
    if (copiedUrl.value === downloadPath) copiedUrl.value = null
  }, 2000)
}

onMounted(loadProtocols)
</script>

<template>
  <div class="config-downloads-view">
    <h2 class="page-title">Configuration Downloads</h2>
    <p class="page-desc">Download VPN configuration files for your device.</p>

    <div v-if="loading" class="loading-text">Loading available protocols...</div>

    <div v-else-if="profiles.length === 0" class="empty-state">
      <p>No configurations available at this time.</p>
    </div>

    <div v-else class="protocols-grid">
      <div
        v-for="profile in profiles"
        :key="profile.type"
        class="download-card"
      >
        <span class="download-card__icon">{{ protocolIcons[profile.type] || '📄' }}</span>
        <span class="download-card__label">{{ profile.name }}</span>
        <span v-if="profile.description" class="download-card__desc">{{ profile.description }}</span>
        <div class="download-card__actions">
          <KButton
            variant="primary"
            size="sm"
            :loading="downloading === profile.type"
            @click="downloadConfig(profile)"
          >
            Download
          </KButton>
          <KButton
            v-if="profile.type.startsWith('openvpn')"
            variant="ghost"
            size="sm"
            @click="copyUrl(profile.download)"
          >
            {{ copiedUrl === profile.download ? '✓ Copied' : '📋 Copy URL' }}
          </KButton>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.config-downloads-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  padding: var(--space-5);
}

.page-title {
  font-size: var(--text-2xl);
  font-weight: var(--font-semibold);
  color: var(--color-text);
  margin: 0;
}

.page-desc {
  font-size: var(--text-sm);
  color: var(--color-muted);
  margin: 0;
}

.loading-text {
  font-size: var(--text-sm);
  color: var(--color-muted);
}

.empty-state {
  text-align: center;
  padding: var(--space-8);
  color: var(--color-muted);
}

.protocols-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: var(--space-3);
}

.download-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-5) var(--space-4);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}

.download-card__icon {
  font-size: 2rem;
}

.download-card__label {
  font-size: var(--text-md);
  font-weight: var(--font-semibold);
  color: var(--color-text);
  text-align: center;
}

.download-card__desc {
  font-size: var(--text-xs);
  color: var(--color-muted);
  text-align: center;
}

.download-card__actions {
  display: flex;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
