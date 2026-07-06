<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useApi, getCsrfToken } from '@koris/composables/useApi'
import { useToast } from '@koris/composables/useToast'
import { useI18n } from '@koris/composables/useI18n'
import { useConfirm } from '@koris/composables/useConfirm'
import Button from '@koris/ui/Button.vue'
import Input from '@koris/ui/Input.vue'
import Select from '@koris/ui/Select.vue'
import StatusPill from '@koris/ui/StatusPill.vue'
import EmptyState from '@koris/ui/EmptyState.vue'
import Skeleton from '@koris/ui/Skeleton.vue'

const { get, post } = useApi()
const toast = useToast()
const { t } = useI18n()
const { confirm } = useConfirm()

interface Proxy {
  id: number
  node_id: number
  port: number
  secret: string
  tag: string
  status: string
  share_link: string
  tg_link: string
  connections_count: number
  last_health_check: string | null
  created_at: string
}
interface Node { id: number; name: string }

const proxies = ref<Proxy[]>([])
const nodes = ref<Node[]>([])
const loading = ref(false)
const loadingNodes = ref(false)
const showCreate = ref(false)
const creating = ref(false)
const rotatingAll = ref(false)
const form = ref<{ node_id: number | null; port: number; tag: string }>({ node_id: null, port: 443, tag: '' })

const nodeOptions = computed(() => nodes.value.map((n) => ({ label: n.name, value: n.id })))
const hasProxies = computed(() => proxies.value.length > 0)

async function load() {
  loading.value = true
  try {
    const res = await get<{ ok: boolean; proxies: Proxy[] }>('/api/admin/telegram-proxies')
    proxies.value = res.proxies || []
  } catch { /* surfaced by useApi */ } finally { loading.value = false }
}

async function loadNodes() {
  loadingNodes.value = true
  try {
    const res = await get<{ nodes: Node[] }>('/api/admin/knode/nodes')
    nodes.value = res.nodes || []
  } catch { /* surfaced by useApi */ } finally { loadingNodes.value = false }
}

async function createProxy() {
  if (!form.value.node_id || !form.value.port) {
    toast.error(t('teleproxy.select_node'))
    return
  }
  creating.value = true
  try {
    const res = await post<{ ok: boolean; proxy: Proxy }>('/api/admin/telegram-proxies', {
      node_id: form.value.node_id,
      port: form.value.port,
      tag: form.value.tag,
    })
    if (res.ok) {
      toast.success(t('teleproxy.created_success'))
      showCreate.value = false
      form.value = { node_id: null, port: 443, tag: '' }
      await load()
    }
  } catch { /* surfaced by useApi */ } finally { creating.value = false }
}

async function startProxy(p: Proxy) {
  try {
    const res = await post<{ ok: boolean }>(`/api/admin/telegram-proxies/${p.id}/start`, {})
    if (res.ok) { toast.success(t('teleproxy.start_success')); await load() }
  } catch { /* surfaced */ }
}
async function stopProxy(p: Proxy) {
  try {
    const res = await post<{ ok: boolean }>(`/api/admin/telegram-proxies/${p.id}/stop`, {})
    if (res.ok) { toast.success(t('teleproxy.stop_success')); await load() }
  } catch { /* surfaced */ }
}
async function rotateAll() {
  const ok = await confirm({ title: t('teleproxy.confirm_rotate_title'), message: t('teleproxy.confirm_rotate_msg'), variant: 'danger' })
  if (!ok) return
  rotatingAll.value = true
  try {
    for (const p of proxies.value) {
      await post<{ ok: boolean }>('/api/admin/telegram-proxies/rotate', { id: p.id })
    }
    toast.success(t('teleproxy.rotated_success'))
    await load()
  } catch { /* surfaced */ } finally { rotatingAll.value = false }
}
async function deleteProxy(p: Proxy) {
  const ok = await confirm({ title: t('teleproxy.confirm_delete_title'), message: t('teleproxy.confirm_delete_msg'), variant: 'danger' })
  if (!ok) return
  const token = getCsrfToken()
  try {
    const res = await fetch('/api/admin/telegram-proxies', {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json', ...(token ? { 'X-CSRF-Token': token } : {}) },
      credentials: 'same-origin',
      body: JSON.stringify({ id: p.id }),
    })
    const data = await res.json()
    if (data.ok) { toast.success(t('teleproxy.deleted_success')); await load() }
    else toast.error(data.error || 'Delete failed')
  } catch { toast.error('Delete failed') }
}
async function copyLink(p: Proxy) {
  const link = p.share_link || p.tg_link
  if (!link) return
  try { await navigator.clipboard.writeText(link); toast.success(t('teleproxy.link_copied')) } catch { toast.error('Copy failed') }
}

function nodeName(id: number): string { return nodes.value.find((n) => n.id === id)?.name || `#${id}` }
function fmtDate(s: string | null): string { if (!s) return '—'; return new Date(s).toLocaleString() }
function isActive(p: Proxy): boolean { return p.status === 'active' }

onMounted(() => { load(); loadNodes() })
</script>

<template>
  <div class="page teleproxy-view">
    <header class="page-header">
      <div>
        <h1>{{ t('teleproxy.title') }}</h1>
        <p class="subtitle">{{ t('teleproxy.empty_desc') }}</p>
      </div>
      <div class="header-actions">
        <Button v-if="hasProxies" variant="ghost" size="sm" :loading="rotatingAll" @click="rotateAll">{{ t('teleproxy.rotate_all') }}</Button>
        <Button variant="primary" size="sm" @click="showCreate = !showCreate">{{ t('teleproxy.add_proxy') }}</Button>
      </div>
    </header>

    <section v-if="showCreate" class="card create-card">
      <h3 class="card-title">{{ t('teleproxy.create_title') }}</h3>
      <div class="create-grid">
        <Select v-model="form.node_id" :options="nodeOptions" :placeholder="t('teleproxy.select_node')" />
        <Input v-model.number="form.port" type="number" :placeholder="t('teleproxy.field_port')" />
        <Input v-model="form.tag" :placeholder="t('teleproxy.tag_placeholder')" />
      </div>
      <div class="form-actions-row">
        <Button variant="primary" size="sm" :loading="creating" @click="createProxy">{{ t('teleproxy.add_proxy') }}</Button>
        <Button variant="ghost" size="sm" @click="showCreate = false">Cancel</Button>
      </div>
    </section>

    <div v-if="loading" class="skeleton-wrap">
      <Skeleton v-for="i in 3" :key="i" variant="rect" width="100%" :height="56" />
    </div>

    <EmptyState
      v-else-if="!hasProxies"
      icon="🛡️"
      :title="t('teleproxy.empty_title')"
      :description="t('teleproxy.empty_desc')"
    />

    <div v-else class="table-wrap">
      <table class="k-table teleproxy-table">
        <thead>
          <tr>
            <th>{{ t('teleproxy.col_node') }}</th>
            <th>{{ t('teleproxy.col_port') }}</th>
            <th>{{ t('teleproxy.col_tag') }}</th>
            <th>{{ t('teleproxy.col_status') }}</th>
            <th>{{ t('teleproxy.col_connections') }}</th>
            <th>{{ t('teleproxy.col_health_check') }}</th>
            <th>{{ t('teleproxy.col_actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in proxies" :key="p.id">
            <td>{{ nodeName(p.node_id) }}</td>
            <td>{{ p.port }}</td>
            <td>{{ p.tag || '—' }}</td>
            <td><StatusPill :status="p.status" size="sm" /></td>
            <td>{{ p.connections_count }}</td>
            <td class="muted">{{ fmtDate(p.last_health_check) }}</td>
            <td>
              <div class="row-actions">
                <Button v-if="!isActive(p)" variant="ghost" size="sm" @click="startProxy(p)">{{ t('teleproxy.start') }}</Button>
                <Button v-else variant="ghost" size="sm" @click="stopProxy(p)">{{ t('teleproxy.stop') }}</Button>
                <Button variant="ghost" size="sm" @click="copyLink(p)">{{ t('teleproxy.copy_link') }}</Button>
                <Button variant="danger" size="sm" @click="deleteProxy(p)">🗑</Button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.teleproxy-view { padding: var(--space-6, 24px); max-width: 1200px; margin: 0 auto; }
.page-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 20px; flex-wrap: wrap; }
.page-header h1 { font-size: var(--text-2xl, 24px); font-weight: 700; margin: 0; }
.subtitle { color: var(--color-muted, #8b98a5); margin: 6px 0 0; font-size: var(--text-sm, 13px); }
.header-actions { display: flex; gap: 10px; }
.card { background: var(--color-surface); border: 1px solid var(--color-border, #28333f); border-radius: var(--radius-lg, 12px); padding: 20px; box-shadow: var(--shadow-sm, 0 1px 3px rgba(0,0,0,.3)); }
.card-title { margin: 0 0 14px; font-size: var(--text-lg, 16px); font-weight: 700; }
.create-card { margin-bottom: 18px; }
.create-grid { display: grid; grid-template-columns: 1.2fr 0.8fr 1fr; gap: 12px; margin-bottom: 14px; }
.form-actions-row { display: flex; gap: 10px; }
.skeleton-wrap { display: flex; flex-direction: column; gap: 10px; }
.table-wrap { overflow-x: auto; background: var(--color-surface); border: 1px solid var(--color-border, #28333f); border-radius: var(--radius-lg, 12px); }
.teleproxy-table { width: 100%; border-collapse: collapse; }
.teleproxy-table th, .teleproxy-table td { padding: 12px 14px; text-align: left; border-bottom: 1px solid var(--color-border, #28333f); font-size: var(--text-sm, 13px); }
.teleproxy-table th { color: var(--color-muted, #8b98a5); font-weight: 600; background: var(--color-surface-2, #1e2630); }
.teleproxy-table tbody tr:hover { background: var(--color-surface-2, #1e2630); }
.muted { color: var(--color-muted, #8b98a5); }
.row-actions { display: flex; gap: 6px; flex-wrap: wrap; }
@media (max-width: 760px) {
  .create-grid { grid-template-columns: 1fr; }
  .page-header { flex-direction: column; align-items: flex-start; }
}
</style>
