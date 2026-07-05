<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useToast } from '@koris/composables/useToast'
import { useI18n } from '@koris/composables/useI18n'
import KButton from '@koris/ui/KButton.vue'
import KInput from '@koris/ui/KInput.vue'
import KFormField from '@koris/ui/KFormField.vue'

const { get, patch, post } = useApi()
const toast = useToast()
const { t } = useI18n()

const telegramToken = ref('')
const telegramChatId = ref('')
const saving = ref(false)
const testing = ref(false)
const botStatus = ref<'unknown' | 'online' | 'offline'>('unknown')

const botConfigured = computed(() => !!(telegramToken.value || '').trim())
const statusLabel = computed(() => {
  if (testing.value) return 'Checking…'
  if (!botConfigured.value) return 'Not set up'
  return botStatus.value === 'offline' ? 'Saved · not confirmed' : 'Connected'
})

async function loadSettings() {
  try {
    const res = await get<{ ok: boolean; settings: Record<string, string> }>('/api/panel-settings')
    if (res.settings) {
      telegramToken.value = res.settings.telegram_token || ''
      telegramChatId.value = res.settings.telegram_chat_id || ''
    }
  } catch { /* defaults */ }
}
async function saveSettings() {
  saving.value = true
  try {
    await patch<{ ok: boolean }>('/api/panel-settings', { telegram_token: telegramToken.value, telegram_chat_id: telegramChatId.value })
    toast.success(t('settings.telegram_save_success'))
  } catch { toast.error(t('settings.telegram_save_error')) } finally { saving.value = false }
}
async function testBot() {
  testing.value = true
  botStatus.value = 'unknown'
  try {
    await patch<{ ok: boolean }>('/api/panel-settings', { telegram_token: telegramToken.value, telegram_chat_id: telegramChatId.value })
    const res = await post<{ ok: boolean }>('/api/admin/bot/restart', {})
    if (res.ok) { botStatus.value = 'online'; toast.success(t('settings.bot_restart_success')) }
    else { botStatus.value = 'offline'; toast.error(t('settings.bot_restart_error')) }
  } catch { botStatus.value = 'offline'; toast.error(t('settings.bot_restart_error')) } finally { testing.value = false }
}
onMounted(loadSettings)
</script>

<template>
  <div class="page telegram-view">
    <header class="page-header">
      <div>
        <h1>Telegram Bot</h1>
        <p class="subtitle">Connect a Telegram bot to receive admin alerts and run commands from chat.</p>
      </div>
      <span class="bot-status" :class="botConfigured ? 'is-on' : 'is-off'">
        <span class="bot-status__dot" />
        {{ statusLabel }}
      </span>
    </header>

    <div class="tg-grid">
      <section class="card tg-config">
        <h3 class="card-title">Configuration</h3>
        <form class="settings-form" autocomplete="off" @submit.prevent="saveSettings">
          <KFormField name="tg-token" label="Bot Token" hint="Get the token from @BotFather">
            <template #default="{ fieldId }">
              <KInput :id="fieldId" v-model="telegramToken" placeholder="123456:ABC-DEF..." type="password" autocomplete="new-password" />
            </template>
          </KFormField>
          <KFormField name="tg-chat" label="Admin Chat / Group ID" hint="Chat or group ID that receives alerts">
            <template #default="{ fieldId }">
              <KInput :id="fieldId" v-model="telegramChatId" placeholder="-1001234567890" />
            </template>
          </KFormField>
          <div class="form-actions-row">
            <KButton type="submit" variant="primary" size="sm" :loading="saving">Save</KButton>
            <KButton type="button" variant="ghost" size="sm" :loading="testing" @click="testBot">Test &amp; Restart</KButton>
          </div>
        </form>
      </section>

      <section class="card tg-help">
        <h3 class="card-title">How it works</h3>
        <ol class="help-list">
          <li>Create a bot with <code>@BotFather</code> and copy its token.</li>
          <li>Save the token and your admin chat ID above.</li>
          <li>Press <strong>Test &amp; Restart</strong> to bring the bot online — it reads config from the database and starts.</li>
          <li>The bot alerts the configured admin chats about node, ticket, and maintenance events.</li>
        </ol>
        <div class="capabilities">
          <span class="k-badge">Node alerts</span>
          <span class="k-badge">Ticket notifications</span>
          <span class="k-badge">Maintenance events</span>
          <span class="k-badge">Chat commands</span>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.telegram-view { padding: var(--space-6, 24px); max-width: 1100px; margin: 0 auto; }
.page-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 24px; flex-wrap: wrap; }
.page-header h1 { font-size: var(--text-2xl, 24px); font-weight: var(--font-bold, 700); margin: 0; }
.subtitle { color: var(--color-muted, #8b98a5); margin: 6px 0 0; font-size: var(--text-sm, 13px); }
.bot-status { display: inline-flex; align-items: center; gap: 7px; padding: 5px 11px; border-radius: 999px; font-size: var(--text-xs, 11px); font-weight: var(--font-semibold, 600); border: 1px solid var(--color-border, #28333f); background: var(--color-surface-2, #1e2630); color: var(--color-muted, #8b98a5); }
.bot-status__dot { width: 8px; height: 8px; border-radius: 50%; background: var(--color-muted, #8b98a5); }
.bot-status.is-on { color: var(--color-success, #22c55e); border-color: color-mix(in srgb, var(--color-success, #22c55e) 40%, var(--color-border, #28333f)); }
.bot-status.is-on .bot-status__dot { background: var(--color-success, #22c55e); box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-success, #22c55e) 22%, transparent); }
.tg-grid { display: grid; grid-template-columns: 1.1fr 1fr; gap: 16px; align-items: start; }
.card { background: var(--color-surface); border: 1px solid var(--color-border, #28333f); border-radius: var(--radius-lg, 12px); padding: var(--space-5, 20px); box-shadow: var(--shadow-sm, 0 1px 3px rgba(0,0,0,.3)); }
.card-title { margin: 0 0 14px; font-size: var(--text-lg, 16px); font-weight: var(--font-bold, 700); }
.settings-form { display: flex; flex-direction: column; gap: 14px; }
.form-actions-row { display: flex; gap: 10px; }
.help-list { margin: 0; padding-left: 18px; display: flex; flex-direction: column; gap: 8px; color: var(--color-muted, #8b98a5); font-size: var(--text-sm, 13px); line-height: 1.55; }
.help-list code { font-family: var(--font-mono, monospace); color: var(--color-text); }
.capabilities { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 16px; }
@media (max-width: 760px) {
  .tg-grid { grid-template-columns: 1fr; }
}
</style>

