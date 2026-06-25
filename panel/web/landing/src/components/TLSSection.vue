<script setup lang="ts">
</script>

<template>
  <section class="tls-section" id="security">
    <div class="tls-container">
      <h2 class="tls-title">Built-In TLS &amp; Security</h2>
      <p class="tls-subtitle">
        Automatic certificate management with ACME, self-signed, or manual upload options.
      </p>

      <div class="tls-grid">
        <div class="tls-card">
          <div class="tls-card__badge">ACME</div>
          <h3 class="tls-card__title">Let's Encrypt</h3>
          <p class="tls-card__desc">
            Automatic certificate issuance and renewal via ACME. Zero manual intervention.
          </p>
        </div>
        <div class="tls-card">
          <div class="tls-card__badge">mTLS</div>
          <h3 class="tls-card__title">Node Authentication</h3>
          <p class="tls-card__desc">
            Mutual TLS between panel and nodes ensures secure gRPC communication.
            API keys add an extra authentication layer.
          </p>
        </div>
        <div class="tls-card">
          <div class="tls-card__badge">Manual</div>
          <h3 class="tls-card__title">Custom Certificates</h3>
          <p class="tls-card__desc">
            Upload your own certificates when using enterprise CAs or internal PKI.
          </p>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.tls-section {
  padding: 6rem 2rem;
  background: #0a0e1a;
}

.tls-container {
  max-width: 1000px;
  margin: 0 auto;
}

.tls-title {
  text-align: center;
  font-size: clamp(1.75rem, 3.5vw, 2.5rem);
  font-weight: 700;
  color: #f1f5f9;
  margin: 0 0 0.75rem;
}

.tls-subtitle {
  text-align: center;
  font-size: 1rem;
  color: #94a3b8;
  margin: 0 0 3rem;
  max-width: 520px;
  margin-left: auto;
  margin-right: auto;
  line-height: 1.6;
}

.tls-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1.5rem;
}

.tls-card {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 12px;
  padding: 2rem 1.5rem;
  transition: border-color 0.2s ease;
}

.tls-card:hover {
  border-color: rgba(34, 197, 94, 0.3);
}

.tls-card__badge {
  display: inline-block;
  padding: 3px 10px;
  background: rgba(34, 197, 94, 0.12);
  color: #22c55e;
  font-size: 0.75rem;
  font-weight: 700;
  border-radius: 12px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  margin-bottom: 1rem;
}

.tls-card__title {
  font-size: 1.05rem;
  font-weight: 600;
  color: #e2e8f0;
  margin: 0 0 0.5rem;
}

.tls-card__desc {
  font-size: 0.85rem;
  color: #94a3b8;
  line-height: 1.6;
  margin: 0;
}

@media (max-width: 768px) {
  .tls-grid {
    grid-template-columns: 1fr;
  }
  .tls-section {
    padding: 4rem 1.25rem;
  }
}
</style>
