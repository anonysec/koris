<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useApi } from '@koris/composables/useApi'

interface ProtocolConfig {
  protocol: string
  label: string
  available: boolean
}

const { get } = useApi()
const protocols = ref<ProtocolConfig[]>([])
const loading = ref(false)
const downloading = ref<string | null>(null)

const protocolLabels: Record<string, string> = {
  openvpn: 'OpenVPN',
  wireguard: 'WireGuard',
  l2tp: 'L2TP/IPsec',
  ikev2: 'IKEv2',
  ssh: 'SSH Tunnel',
}

const protocolIcons: Record<string, string> = {
  openvpn: '🔐',
  wireguard: '⚡',
  l2tp: '🔒',
  ikev2: '🛡️',
  ssh: '🖥️',
}

async function loadProtocols() {
  loading.value = true
  try {
    const res = await get<{ ok: boolean; protocols: string[] }>('/api/portal/available-protocols')
    if (res?.ok && res.protocols) {
      protocols.value = res.protocols.map(p => ({
        protocol: p,
        label: protocolLabels[p] || p,
        available: true,
      }))
    }
  } catch {
    // Fallback: show common protocols
    protocols.value = ['openvpn', 'wireguard', 'ikev2'].map(p => ({
      protocol: p,
      label: protocolLabels[p] || p,
      available: true,
    }))
  } finally {
    loading.value = false
  }
}

async function downloadConfig(protocol: string) {
  downloading.value = protocol
  try {
    const res = await fetch(`/api/portal/configs/${protocol}`, { credentials: 'include' })
    if (!res.ok) throw new Error('Download failed')
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${protocol}-config.${protocol === 'wireguard' ? 'conf' : 'ovpn'}`
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

onMounted(loadProtocols)
</script>

<template>
  <div class="config-downloads-view">
    <h2 class="page-title">Configuration Downloads</h2>
    <p class="page-desc">Download VPN configuration files for your device.</p>

    <div v-if="loading" class="loading-text">Loading available protocols...</div>

    <div v-else-if="protocols.length === 0" class="empty-state">
      <p>No configurations available at this time.</p>
    </div>

    <div v-else class="protocols-grid">
      <button
        v-for="proto in protocols"
        :key="proto.protocol"
        class="download-card"
        :disabled="downloading === proto.protocol"
        @click="downloadConfig(proto.protocol)"
      >
        <span class="download-card__icon">{{ protocolIcons[proto.protocol] || '📄' }}</span>
        <span class="download-card__label">{{ proto.label }}</span>
        <span class="download-card__action">
          {{ downloading === proto.protocol ? 'Downloading...' : 'Download' }}
        </span>
      </button>
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
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
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
  cursor: pointer;
  transition: border-color 0.15s ease, transform 0.15s ease;
}

.download-card:hover:not(:disabled) {
  border-color: var(--color-primary);
  transform: translateY(-2px);
}

.download-card:disabled {
  opacity: 0.6;
  cursor: wait;
}

.download-card__icon {
  font-size: 2rem;
}

.download-card__label {
  font-size: var(--text-md);
  font-weight: var(--font-semibold);
  color: var(--color-text);
}

.download-card__action {
  font-size: var(--text-xs);
  color: var(--color-primary);
  font-weight: var(--font-medium);
}
</style>
