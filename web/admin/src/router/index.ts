import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

/** Routes that resellers are NOT allowed to access */
const adminOnlyRoutes = new Set([
  'overview',
  'services',
  'node-detail',
  'node-compare',
  'metrics',
  'landing-editor',
  'settings',
  'tickets',
  'ticket-detail',
  'payments',
  'billing',
  'templates',
  'notifications',
  'plans',
  'canned-responses',
  'sla-config',
  'knowledge-base',
  'user-tags',
  'filter-presets',
  'protocols',
  'telegram-bot',
])

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      component: () => import('@/layouts/AppShell.vue'),
      meta: { requiresAuth: true },
      children: [
        { path: '', name: 'overview', component: () => import('@/views/Dashboard.vue') },
        { path: 'users', name: 'users', component: () => import('@/views/Customers.vue') },
        { path: 'users/:id', name: 'user-detail', component: () => import('@/views/CustomerDetail.vue'), props: true },
        { path: 'plans', name: 'plans', component: () => import('@/views/Plans.vue') },
        { path: 'payments', name: 'payments', component: () => import('@/views/Payments.vue') },
        { path: 'billing', name: 'billing', component: () => import('@/views/Billing.vue') },
        { path: 'tickets', name: 'tickets', component: () => import('@/views/Tickets.vue') },
        { path: 'tickets/:id', name: 'ticket-detail', component: () => import('@/views/TicketDetail.vue'), props: true },
        { path: 'services', name: 'services', component: () => import('@/views/Cores.vue') },
        { path: 'nodes', redirect: '/dashboard/services' },
        { path: 'nodes/compare', name: 'node-compare', component: () => import('@/views/NodeCompare.vue') },
        { path: 'nodes/:id/:tab?', name: 'node-detail', component: () => import('@/views/NodeDetail.vue'), props: true },
        { path: 'cores', redirect: '/dashboard/services' },
        { path: 'metrics', name: 'metrics', component: () => import('@/views/MetricsDashboard.vue') },
        { path: 'landing-editor', name: 'landing-editor', component: () => import('@/views/LandingPageEditor.vue') },
        { path: 'templates', name: 'templates', component: () => import('@/views/Templates.vue') },
        { path: 'settings/:tab?', name: 'settings', component: () => import('@/views/Settings.vue'), props: true },
                { path: 'wireguard', redirect: '/dashboard/services' },
        { path: 'notifications', name: 'notifications', component: () => import('@/views/Notifications.vue') },
                { path: 'mtproto', redirect: '/dashboard/services' },
        { path: 'canned-responses', name: 'canned-responses', component: () => import('@/views/CannedResponses.vue') },
        { path: 'sla-config', name: 'sla-config', component: () => import('@/views/SLAConfig.vue') },
        { path: 'knowledge-base', name: 'knowledge-base', component: () => import('@/views/KnowledgeBase.vue') },
        { path: 'user-tags', name: 'user-tags', component: () => import('@/views/UserTags.vue') },
        { path: 'filter-presets', name: 'filter-presets', component: () => import('@/views/FilterPresets.vue') },
        { path: 'protocols', name: 'protocols', component: () => import('@/views/Protocols.vue') },
        { path: 'telegram-bot', name: 'telegram-bot', component: () => import('@/views/TelegramBot.vue') },
        // Redirects from old paths
        { path: 'customers', redirect: '/dashboard/users' },
        { path: 'customers/:id', redirect: (to: any) => `/dashboard/users/${to.params.id}` },
        { path: 'resellers', name: 'resellers', component: () => import('@/views/Customers.vue') },
        // Reseller-specific routes
        { path: 'reseller-dashboard', name: 'reseller-dashboard', component: () => import('@/views/ResellerDashboard.vue') },
        { path: 'reseller-plans', name: 'reseller-plans', component: () => import('@/views/ResellerPlans.vue') },
        { path: 'reseller-transactions', name: 'reseller-transactions', component: () => import('@/views/ResellerTransactions.vue') },
        { path: 'reseller-tickets', name: 'reseller-tickets', component: () => import('@/views/ResellerTickets.vue') },
        { path: 'reseller-tickets/:id', name: 'reseller-ticket-detail', component: () => import('@/views/ResellerTicketDetail.vue'), props: true },
        { path: 'reseller-settings', name: 'reseller-settings', component: () => import('@/views/ResellerSettings.vue') },
      ]
    },
    { path: '/login', name: 'login', component: () => import('@/views/Login.vue') },
    { path: '/setup', name: 'setup', component: () => import('@/views/Setup.vue') },
    { path: '/:pathMatch(.*)*', redirect: '/' }
  ]
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()

  if (!auth.initialized) {
    await auth.checkAuth()
  }

  if (auth.setupRequired && to.name !== 'setup') {
    return { name: 'setup' }
  }

  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }

  if ((to.name === 'login' || to.name === 'setup') && auth.isAuthenticated) {
    return { name: 'overview' }
  }

  // Role-based access: resellers can only access allowed routes
  if (auth.user?.role === 'reseller' && to.name && adminOnlyRoutes.has(to.name as string)) {
    return { name: 'reseller-dashboard' }
  }

  // Reseller landing page: redirect root to reseller-dashboard
  if (auth.user?.role === 'reseller' && (to.name === 'overview' || to.path === '/' || to.path === '')) {
    return { name: 'reseller-dashboard' }
  }

  // Legacy meta-based role check
  if (to.meta.roles && auth.user) {
    const roles = to.meta.roles as string[]
    if (!roles.includes(auth.user.role)) {
      return { name: 'overview' }
    }
  }
})

export default router
