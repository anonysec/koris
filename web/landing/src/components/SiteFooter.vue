<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from '../i18n'

const { t } = useI18n()

const currentYear = new Date().getFullYear()

const productLinks = computed(() => [
  { label: t('footer.features'), href: '#features' },
  { label: t('footer.pricing'), href: '#pricing' },
  { label: t('footer.documentation'), href: '/docs' },
])

const resourceLinks = computed(() => [
  { label: t('footer.dashboard'), href: '/dashboard/' },
  { label: t('footer.portal'), href: '/portal/' },
  { label: t('footer.api_docs'), href: '/docs/api' },
])

const legalLinks = computed(() => [
  { label: t('footer.privacy'), href: '/privacy' },
  { label: t('footer.terms'), href: '/terms' },
])
</script>

<template>
  <footer class="site-footer">
    <div class="footer-inner">
      <!-- Top columns -->
      <div class="footer-columns">
        <!-- Brand column -->
        <div class="footer-brand">
          <a href="/" class="footer-logo">Koris</a>
          <p class="footer-tagline">{{ t('footer.tagline') }}</p>
        </div>

        <!-- Product links -->
        <div class="footer-col">
          <h4 class="footer-col-title">{{ t('footer.product') }}</h4>
          <ul class="footer-links">
            <li v-for="link in productLinks" :key="link.label">
              <a :href="link.href" class="footer-link">{{ link.label }}</a>
            </li>
          </ul>
        </div>

        <!-- Resources -->
        <div class="footer-col">
          <h4 class="footer-col-title">{{ t('footer.resources') }}</h4>
          <ul class="footer-links">
            <li v-for="link in resourceLinks" :key="link.label">
              <a :href="link.href" class="footer-link">{{ link.label }}</a>
            </li>
          </ul>
        </div>

        <!-- Legal -->
        <div class="footer-col">
          <h4 class="footer-col-title">{{ t('footer.legal') }}</h4>
          <ul class="footer-links">
            <li v-for="link in legalLinks" :key="link.label">
              <a :href="link.href" class="footer-link">{{ link.label }}</a>
            </li>
          </ul>
        </div>
      </div>

      <!-- Bottom bar -->
      <div class="footer-bottom">
        <p class="footer-copyright">{{ t('footer.copyright', { year: currentYear }) }}</p>

        <!-- Social icons -->
        <div class="footer-socials">
          <a href="https://github.com" target="_blank" rel="noopener noreferrer" class="social-link" aria-label="GitHub">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 0C5.374 0 0 5.373 0 12c0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23A11.509 11.509 0 0112 5.803c1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576C20.566 21.797 24 17.3 24 12c0-6.627-5.373-12-12-12z"/>
            </svg>
          </a>
          <a href="https://t.me" target="_blank" rel="noopener noreferrer" class="social-link" aria-label="Telegram">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
              <path d="M11.944 0A12 12 0 000 12a12 12 0 0012 12 12 12 0 0012-12A12 12 0 0012 0h-.056zm4.962 7.224c.1-.002.321.023.465.14a.506.506 0 01.171.325c.016.093.036.306.02.472-.18 1.898-.962 6.502-1.36 8.627-.168.9-.499 1.201-.82 1.23-.696.065-1.225-.46-1.9-.902-1.056-.693-1.653-1.124-2.678-1.8-1.185-.78-.417-1.21.258-1.91.177-.184 3.247-2.977 3.307-3.23.007-.032.014-.15-.056-.212s-.174-.041-.249-.024c-.106.024-1.793 1.14-5.061 3.345-.479.33-.913.49-1.302.48-.428-.008-1.252-.241-1.865-.44-.752-.245-1.349-.374-1.297-.789.027-.216.325-.437.893-.663 3.498-1.524 5.83-2.529 6.998-3.014 3.332-1.386 4.025-1.627 4.476-1.635z"/>
            </svg>
          </a>
          <a href="https://twitter.com" target="_blank" rel="noopener noreferrer" class="social-link" aria-label="Twitter">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
              <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z"/>
            </svg>
          </a>
        </div>
      </div>
    </div>
  </footer>
</template>

<style scoped>
.site-footer {
  border-top: 1px solid rgba(148, 163, 184, 0.08);
  padding: 4rem 2rem 2rem;
  margin-top: 2rem;
}

.footer-inner {
  max-width: 1200px;
  margin: 0 auto;
}

/* Columns layout */
.footer-columns {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr;
  gap: 3rem;
  padding-bottom: 3rem;
  border-bottom: 1px solid rgba(148, 163, 184, 0.08);
}

/* Brand */
.footer-brand {
  max-width: 280px;
}

.footer-logo {
  font-size: 1.5rem;
  font-weight: 800;
  color: #f1f5f9;
  text-decoration: none;
  letter-spacing: -0.02em;
}

.footer-logo:hover {
  color: #6366f1;
}

.footer-tagline {
  margin-top: 0.75rem;
  color: #64748b;
  font-size: 0.9rem;
  line-height: 1.5;
}

/* Link columns */
.footer-col-title {
  color: #f1f5f9;
  font-size: 0.85rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin: 0 0 1rem;
}

.footer-links {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
}

.footer-link {
  color: #94a3b8;
  text-decoration: none;
  font-size: 0.9rem;
  transition: color 0.2s ease;
}

.footer-link:hover {
  color: #f1f5f9;
}

/* Bottom bar */
.footer-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 2rem;
}

.footer-copyright {
  color: #475569;
  font-size: 0.85rem;
  margin: 0;
}

/* Social icons */
.footer-socials {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.social-link {
  color: #64748b;
  transition: color 0.2s ease, transform 0.15s ease;
  display: flex;
  align-items: center;
  justify-content: center;
}

.social-link:hover {
  color: #f1f5f9;
  transform: translateY(-1px);
}

/* Responsive */
@media (max-width: 768px) {
  .footer-columns {
    grid-template-columns: 1fr 1fr;
    gap: 2rem;
  }

  .footer-brand {
    grid-column: 1 / -1;
    max-width: 100%;
  }

  .footer-bottom {
    flex-direction: column;
    gap: 1.25rem;
    align-items: center;
    text-align: center;
  }
}

@media (max-width: 480px) {
  .footer-columns {
    grid-template-columns: 1fr;
    gap: 1.5rem;
  }
}
</style>
