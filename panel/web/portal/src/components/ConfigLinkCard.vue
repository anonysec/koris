<script setup lang="ts">
import { ref } from 'vue'
import { useClipboard } from '@koris/composables/useClipboard'
import { useI18n } from '@koris/composables/useI18n'
import QRCode from './QRCode.vue'

export interface ConfigLink {
  protocol: string
  link: string
  qr_data: string
  remark?: string
  node_name?: string
}

const props = defineProps<{
  config: ConfigLink
}>()

const { copy, copied } = useClipboard()
const { t } = useI18n()
const showQR = ref(false)

function handleCopy() {
  copy(props.config.link)
}

function toggleQR() {
  showQR.value = !showQR.value
}

function getProtocolIcon(protocol: string): string {
  switch (protocol.toLowerCase()) {
    case 'vless': return '⚡'
    case 'vmess': return '🔷'
    case 'trojan': return '🐴'
    case 'shadowsocks':
    case 'ss': return '🔮'
    default: return '🔗'
  }
}

function getProtocolLabel(protocol: string): string {
  switch (protocol.toLowerCase()) {
    case 'vless': return 'VLESS'
    case 'vmess': return 'VMess'
    case 'trojan': return 'Trojan'
    case 'shadowsocks':
    case 'ss': return 'Shadowsocks'
    default: return protocol.toUpperCase()
  }
}

function truncateLink(link: string, maxLen = 50): string {
  if (link.length <= maxLen) return link
  return link.substring(0, maxLen) + '...'
}
</script>
<template>
  <div class="config-card">
    <div class="config-card__header">
      <div class="config-card__icon">{{ getProtocolIcon(config.protocol) }}</div>
      <div class="config-card__info">
        <div class="config-card__protocol">{{ getProtocolLabel(config.protocol) }}</div>
        <div v-if="config.remark || config.node_name" class="config-card__remark">
          {{ config.remark || config.node_name }}
        </div>
      </div>
    </div>

    <div class="config-card__link">
      <code class="config-card__link-text">{{ truncateLink(config.link) }}</code>
    </div>

    <div class="config-card__actions">
      <button class="config-card__btn config-card__btn--copy" @click="handleCopy" type="button">
        {{ copied ? '✓ ' + t('portal.xray.copied') : '📋 ' + t('portal.xray.copy') }}
      </button>
      <button class="config-card__btn config-card__btn--qr" @click="toggleQR" type="button">
        {{ showQR ? '🔼 ' + t('portal.xray.hideQR') : '📱 ' + t('portal.xray.showQR') }}
      </button>
    </div>

    <QRCode
      v-if="showQR"
      :value="config.qr_data || config.link"
      :size="240"
      :visible="showQR"
    />
  </div>
</template>
<style scoped>
.config-card {
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.config-card__header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}
.config-card__icon {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.25rem;
  background: var(--color-surface-2);
  border-radius: var(--radius-md);
  flex-shrink: 0;
}
.config-card__info {
  flex: 1;
  min-width: 0;
}
.config-card__protocol {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
}
.config-card__remark {
  font-size: var(--text-xs);
  color: var(--color-muted);
  margin-top: 2px;
}
.config-card__link {
  background: var(--color-surface-2);
  border-radius: var(--radius-sm);
  padding: var(--space-2) var(--space-3);
  overflow: hidden;
}
.config-card__link-text {
  font-size: var(--text-xs);
  color: var(--color-muted);
  font-family: monospace;
  word-break: break-all;
  line-height: 1.4;
}
.config-card__actions {
  display: flex;
  gap: var(--space-2);
  flex-wrap: wrap;
}
.config-card__btn {
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-2) var(--space-3);
  font-size: var(--text-xs);
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;
  color: var(--color-text);
  white-space: nowrap;
}
.config-card__btn:hover {
  background: var(--color-surface-2);
}
.config-card__btn--copy {
  border-color: var(--color-primary);
  color: var(--color-primary);
}
.config-card__btn--copy:hover {
  background: var(--color-primary);
  color: #fff;
}

@media (max-width: 480px) {
  .config-card__actions {
    flex-direction: column;
  }
  .config-card__btn {
    text-align: center;
  }
}
</style>
