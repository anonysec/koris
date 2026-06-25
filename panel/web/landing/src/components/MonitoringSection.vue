<script setup lang="ts">
</script>

<template>
  <section class="monitoring" id="monitoring">
    <div class="monitoring-container">
      <h2 class="monitoring-title">Real-Time Monitoring</h2>
      <p class="monitoring-subtitle">
        Live metrics via WebSocket. CPU, RAM, disk, bandwidth, and sessions at a glance.
      </p>

      <div class="monitoring-features">
        <div class="monitoring-card">
          <div class="monitoring-card__icon">📊</div>
          <h3 class="monitoring-card__title">Live Metrics Dashboard</h3>
          <p class="monitoring-card__desc">
            Per-node CPU, RAM, and disk usage with configurable alert thresholds.
            Get notified when resources exceed limits.
          </p>
        </div>
        <div class="monitoring-card">
          <div class="monitoring-card__icon">🔔</div>
          <h3 class="monitoring-card__title">Smart Alerts</h3>
          <p class="monitoring-card__desc">
            Customizable thresholds for CPU, RAM, and disk. Alerts appear in the dashboard
            and can trigger Telegram notifications.
          </p>
        </div>
        <div class="monitoring-card">
          <div class="monitoring-card__icon">📈</div>
          <h3 class="monitoring-card__title">Historical Charts</h3>
          <p class="monitoring-card__desc">
            View metrics over 1h, 6h, or 24h ranges with bandwidth charts
            and session counts per node.
          </p>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.monitoring {
  padding: 6rem 2rem;
  background: #070a12;
}

.monitoring-container {
  max-width: 1000px;
  margin: 0 auto;
}

.monitoring-title {
  text-align: center;
  font-size: clamp(1.75rem, 3.5vw, 2.5rem);
  font-weight: 700;
  color: #f1f5f9;
  margin: 0 0 0.75rem;
}

.monitoring-subtitle {
  text-align: center;
  font-size: 1rem;
  color: #94a3b8;
  margin: 0 0 3rem;
  max-width: 520px;
  margin-left: auto;
  margin-right: auto;
  line-height: 1.6;
}

.monitoring-features {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1.5rem;
}

.monitoring-card {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 12px;
  padding: 2rem 1.5rem;
  text-align: center;
  transition: border-color 0.2s ease, transform 0.2s ease;
}

.monitoring-card:hover {
  border-color: rgba(99, 102, 241, 0.25);
  transform: translateY(-2px);
}

.monitoring-card__icon {
  font-size: 2.5rem;
  margin-bottom: 1rem;
}

.monitoring-card__title {
  font-size: 1.05rem;
  font-weight: 600;
  color: #e2e8f0;
  margin: 0 0 0.5rem;
}

.monitoring-card__desc {
  font-size: 0.85rem;
  color: #94a3b8;
  line-height: 1.6;
  margin: 0;
}

@media (max-width: 768px) {
  .monitoring-features {
    grid-template-columns: 1fr;
  }
  .monitoring {
    padding: 4rem 1.25rem;
  }
}
</style>
