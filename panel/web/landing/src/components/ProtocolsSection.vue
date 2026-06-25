<script setup lang="ts">
import { useI18n } from '../i18n'

const { t } = useI18n()

const protocols = [
  { name: 'OpenVPN', icon: '🔐', desc: 'Industry standard, cross-platform VPN protocol with strong encryption.' },
  { name: 'WireGuard', icon: '⚡', desc: 'Modern, high-performance protocol with minimal attack surface.' },
  { name: 'IKEv2/IPsec', icon: '🛡️', desc: 'Enterprise-grade protocol with seamless reconnection on network changes.' },
  { name: 'L2TP/IPsec', icon: '🔒', desc: 'Widely supported tunneling protocol with IPsec encryption layer.' },
  { name: 'SSH Tunnel', icon: '🖥️', desc: 'Encrypted tunneling via SSH for restrictive networks.' },
  { name: 'MTProto', icon: '📱', desc: 'Telegram-optimized proxy protocol for censorship circumvention.' },
]
</script>

<template>
  <section class="protocols" id="protocols">
    <div class="protocols-container">
      <h2 class="protocols-title">Multi-Protocol Support</h2>
      <p class="protocols-subtitle">
        Deploy multiple VPN protocols on each node. Users connect with their preferred method.
      </p>

      <div class="protocols-grid">
        <div v-for="proto in protocols" :key="proto.name" class="protocol-item">
          <span class="protocol-item__icon">{{ proto.icon }}</span>
          <div class="protocol-item__info">
            <h3 class="protocol-item__name">{{ proto.name }}</h3>
            <p class="protocol-item__desc">{{ proto.desc }}</p>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.protocols {
  padding: 6rem 2rem;
  background: #0a0e1a;
}

.protocols-container {
  max-width: 1000px;
  margin: 0 auto;
}

.protocols-title {
  text-align: center;
  font-size: clamp(1.75rem, 3.5vw, 2.5rem);
  font-weight: 700;
  color: #f1f5f9;
  margin: 0 0 0.75rem;
}

.protocols-subtitle {
  text-align: center;
  font-size: 1rem;
  color: #94a3b8;
  margin: 0 0 3rem;
  max-width: 520px;
  margin-left: auto;
  margin-right: auto;
  line-height: 1.6;
}

.protocols-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 1.25rem;
}

.protocol-item {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
  padding: 1.5rem;
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 10px;
  transition: border-color 0.2s ease;
}

.protocol-item:hover {
  border-color: rgba(99, 102, 241, 0.3);
}

.protocol-item__icon {
  font-size: 1.75rem;
  flex-shrink: 0;
}

.protocol-item__name {
  font-size: 1rem;
  font-weight: 600;
  color: #e2e8f0;
  margin: 0 0 0.25rem;
}

.protocol-item__desc {
  font-size: 0.85rem;
  color: #94a3b8;
  line-height: 1.5;
  margin: 0;
}

@media (max-width: 640px) {
  .protocols-grid {
    grid-template-columns: 1fr;
  }
  .protocols {
    padding: 4rem 1.25rem;
  }
}
</style>
