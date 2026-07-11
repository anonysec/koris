import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      component: () => import('@/layouts/PortalShell.vue'),
      meta: { requiresAuth: true },
      children: [
        { path: '', name: 'portal-home', component: () => import('@/views/SinglePage.vue') },
        { path: 'billing', name: 'portal-billing', component: () => import('@/views/Billing.vue'), meta: { requiresBilling: true } },
        { path: 'profile', name: 'portal-profile', component: () => import('@/views/Profile.vue') },
        { path: 'support', redirect: '/' },
        { path: 'connections', name: 'portal-connections', component: () => import('@/views/Connections.vue') },
        { path: 'configs', name: 'portal-configs', component: () => import('@/views/ConfigDownloads.vue') },
        { path: 'invoices', name: 'portal-invoices', component: () => import('@/views/Invoices.vue'), meta: { requiresBilling: true } },
        { path: 'kb', name: 'portal-kb', component: () => import('@/views/KnowledgeBase.vue') },
        { path: 'payment', name: 'portal-payment', component: () => import('@/views/Payment.vue'), meta: { requiresBilling: true } },
        { path: 'wireguard', redirect: '/' },
      ]
    },
    { path: '/login', name: 'portal-login', component: () => import('@/views/Login.vue') },
    // Redirect old routes to home
    { path: '/usage', redirect: '/' },
    { path: '/vpn-profiles', redirect: '/' },
    { path: '/anyconnect', redirect: '/' },
    { path: '/:pathMatch(.*)*', redirect: '/' }
  ]
})

// Verify the customer session at most once per page load. Re-checking on every
// navigation (especially when a 401 fires onUnauthorized → push to login →
// re-guard) produced an unbounded checkAuth→401→redirect loop. One shot is enough.
let authResolved = false

router.beforeEach(async (to) => {
  const { usePortalAuthStore } = await import('@/stores/auth')
  const auth = usePortalAuthStore()

  // Going to the login page — never re-hit /api/portal/me here.
  if (to.name === 'portal-login') {
    return
  }

  // If already authenticated (e.g. just logged in), skip the network check
  if (auth.isAuthenticated) {
    if (to.name === 'portal-login') {
      return { name: 'portal-home' }
    }
    // Block billing route when billing is disabled
    if (to.meta.requiresBilling && !auth.billingEnabled) {
      return { name: 'portal-home' }
    }
    return // allow navigation
  }

  // Not authenticated yet — verify the session once, then trust the result.
  if (!authResolved && !auth.loading) {
    authResolved = true
    await auth.checkAuth()
  }

  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { name: 'portal-login' }
  }

  if (to.name === 'portal-login' && auth.isAuthenticated) {
    return { name: 'portal-home' }
  }

  // Block billing route when billing is disabled (after auth check)
  if (to.meta.requiresBilling && !auth.billingEnabled) {
    return { name: 'portal-home' }
  }
})

export default router
