# Implementation Plan: KorisPanel UI Enhancement

## Overview

This implementation plan covers the complete refactoring of KorisPanel's monolithic frontend applications (Admin Panel and Customer Portal) into a modular architecture with Vue Router, Pinia state management, a shared component library, composable-based logic, interactive data visualization, WCAG 2.1 AA accessibility, and performance optimizations. The work is organized into 6 phases following the migration strategy defined in the design document. All code is TypeScript + Vue 3.

## Tasks

- [x] 1. Phase 1: Foundation — Shared Directory, Types, Composables, and Configuration
  - [x] 1.1 Create shared directory structure and base configuration files
    - Create `panel/web/shared/` directory with subdirectories: `components/`, `composables/`, `types/`, `styles/`
    - Create `panel/web/shared/types/api.ts` with `ApiResponse<T>`, `PaginatedResponse<T>` interfaces
    - Create `panel/web/shared/types/entities.ts` with `Customer`, `CustomerDetail`, `Plan`, `NodeItem`, `NodeMetrics`, `Ticket`, `Payment` interfaces
    - Create `panel/web/shared/types/components.ts` with component prop/emit interfaces (`KButtonProps`, `KDataTableColumn`, `KDataTableProps`, `KDrawerProps`, `ConfirmOptions`, `KChartProps`, `ChartDataPoint`, `ChartOptions`, `ValidationRule`, `KFormFieldProps`, `NavItem`, `Breadcrumb`)
    - _Requirements: 1.1, 1.2, 26.1_

  - [x] 1.2 Create shared design system styles
    - Create `panel/web/shared/styles/tokens.css` with CSS custom properties for colors (`#070a12` background, `#2563eb` primary, `#22d3ee` accent), spacing scale, typography, border-radius, and transition durations
    - Create `panel/web/shared/styles/reset.css` with a minimal CSS reset
    - Create `panel/web/shared/styles/utilities.css` with utility classes (flex, grid, spacing, text alignment)
    - _Requirements: 26.1, 26.2_

  - [x] 1.3 Install new dependencies and update Vite config for admin app
    - Add `vue-router@4`, `pinia@2`, `@vueuse/core` to `panel/web/admin/package.json`
    - Update `panel/web/admin/vite.config.ts` with path aliases (`@`, `@koris/ui`, `@koris/composables`, `@koris/types`, `@koris/styles`), set `base: '/dashboard/'`, and configure `manualChunks` for vendor and charts bundles
    - Update `panel/web/admin/tsconfig.json` with path mappings matching Vite aliases
    - _Requirements: 1.1, 20.1, 20.2_

  - [x] 1.4 Install new dependencies and update Vite config for portal app
    - Add `vue-router@4`, `pinia@2`, `@vueuse/core` to `panel/web/portal/package.json`
    - Update `panel/web/portal/vite.config.ts` with path aliases, set `base: '/portal/'`, and configure `manualChunks`
    - Update `panel/web/portal/tsconfig.json` with path mappings matching Vite aliases
    - _Requirements: 1.2, 20.1_

  - [x] 1.5 Implement useApi composable
    - Create `panel/web/shared/composables/useApi.ts` implementing the `UseApiReturn` interface
    - Implement `get`, `post`, `put`, `patch`, `del` methods wrapping `fetch` with `credentials: 'same-origin'`
    - Set `loading` ref to `true` before requests and `false` after (regardless of success/failure)
    - Automatically set `Content-Type: application/json` for POST/PUT/PATCH
    - Call `onUnauthorized` callback on HTTP 401 responses
    - Set `error.value` with human-readable message on non-2xx responses
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5_

  - [ ]* 1.6 Write property tests for useApi composable
    - **Property 3: API Loading State Bracketing** — For any API call, `loading.value` transitions from false→true→false
    - **Property 4: API Error Handling** — For non-2xx responses, `error.value` is set; for 401, `onUnauthorized` is called exactly once
    - **Property 5: API Content-Type Header** — POST/PUT/PATCH requests include `Content-Type: application/json`
    - **Validates: Requirements 11.2, 11.3, 11.4, 11.5**

  - [x] 1.7 Implement useWebSocket composable
    - Create `panel/web/shared/composables/useWebSocket.ts` implementing `UseWebSocketReturn`
    - Implement auto-connect when `autoConnect: true`
    - Implement exponential backoff reconnection (`baseDelay * 2^attempt + jitter`, capped at `maxDelay`)
    - Stop reconnection when `maxReconnectAttempts` exceeded
    - Clean up connection and timers on component unmount via `onUnmounted`
    - _Requirements: 12.1, 12.2, 12.3, 12.4, 12.5_

  - [ ]* 1.8 Write property tests for useWebSocket reconnection
    - **Property 20: WebSocket Reconnection Algorithm** — For attempt N < max, delay equals `min(baseDelay * 2^N + jitter, maxDelay)`; for N >= max, no reconnection
    - **Validates: Requirements 12.2, 12.5**

  - [x] 1.9 Implement useTheme composable
    - Create `panel/web/shared/composables/useTheme.ts`
    - Load persisted theme from `localStorage` on initialization (before first paint)
    - Apply theme by updating CSS custom properties on `document.documentElement`
    - `toggle()` switches theme and persists to `localStorage`
    - _Requirements: 14.1, 14.2, 14.3, 26.2_

  - [x] 1.10 Implement useI18n composable
    - Create `panel/web/shared/composables/useI18n.ts`
    - Support `en`, `fa`, `zh` locales
    - `t(key)` returns translation for active locale; falls back to English if key missing
    - Never return raw key string
    - _Requirements: 15.1, 15.2, 15.3_

  - [ ]* 1.11 Write property tests for useI18n
    - **Property 24: i18n Translation Lookup** — For any key and locale, `t(key)` returns a translation (active locale or English fallback), never a raw key
    - **Validates: Requirements 15.2, 15.3**

  - [x] 1.12 Implement useClipboard composable
    - Create `panel/web/shared/composables/useClipboard.ts`
    - `copy(text)` writes to system clipboard via `navigator.clipboard.writeText`
    - Set `copied.value = true` briefly on success for UI feedback
    - _Requirements: 17.1, 17.2_

- [x] 2. Checkpoint — Foundation complete
  - Ensure all tests pass, ask the user if questions arise.

- [x] 3. Phase 2: State Extraction — Pinia Stores
  - [x] 3.1 Implement admin authStore
    - Create `panel/web/admin/src/stores/auth.ts` using `defineStore` with Composition API style
    - Implement state: `user`, `isAuthenticated`, `setupRequired`, `setupKeyRequired`, `initialized`
    - Implement actions: `checkAuth()`, `login()`, `logout()`, `setup()`
    - Use `useApi` composable for all API calls with loading state management
    - On error, preserve existing state and surface error via toast
    - _Requirements: 2.1, 2.2, 2.3, 3.1, 3.3, 3.4_

  - [x] 3.2 Implement customersStore
    - Create `panel/web/admin/src/stores/customers.ts`
    - Implement state: `list`, `deleted`, `detail`, `usage`, `loading`, `detailLoading`, `filters`, `pagination`
    - Implement actions: `loadCustomers()`, `loadDetail(id)`, `createCustomer()`, `updateCustomer()`, `deleteCustomer()`, `archiveCustomer()`
    - Implement computed `filteredList` with search, status filter, sorting
    - Set `loading = true` before requests, `false` after (success or failure)
    - _Requirements: 3.1, 3.3, 3.4, 22.2, 22.3_

  - [x] 3.3 Implement plansStore
    - Create `panel/web/admin/src/stores/plans.ts`
    - State: `list`, `loading`
    - Actions: `loadPlans()`, `createPlan()`, `updatePlan()`, `deletePlan()`
    - _Requirements: 3.1, 3.3, 22.4_

  - [x] 3.4 Implement paymentsStore
    - Create `panel/web/admin/src/stores/payments.ts`
    - State: `list`, `loading`, `filters`
    - Actions: `loadPayments()`, `approvePayment()`, `rejectPayment()`
    - _Requirements: 3.1, 3.3, 22.5_

  - [x] 3.5 Implement ticketsStore
    - Create `panel/web/admin/src/stores/tickets.ts`
    - State: `list`, `detail`, `loading`
    - Actions: `loadTickets()`, `loadTicketDetail()`, `replyToTicket()`, `closeTicket()`
    - _Requirements: 3.1, 3.3, 22.6_

  - [x] 3.6 Implement nodesStore
    - Create `panel/web/admin/src/stores/nodes.ts`
    - State: `list`, `tasks`, `vpnSettings`, `vpnConfigs`, `loading`
    - Actions: `loadNodes()`, `loadNodeTasks()`, `loadVpnSettings()`, `updateNode()`
    - _Requirements: 3.1, 3.3, 22.8_

  - [x] 3.7 Implement resellersStore
    - Create `panel/web/admin/src/stores/resellers.ts`
    - State: `list`, `loading`
    - Actions: `loadResellers()`, `createReseller()`, `updateReseller()`
    - _Requirements: 3.1, 3.3, 22.7_

  - [x] 3.8 Implement realtimeStore
    - Create `panel/web/admin/src/stores/realtime.ts`
    - State: `connected`, `stats`, `liveSessions`, `rxHistory`, `txHistory`, `notifications`
    - Use `useWebSocket` composable with `autoConnect: true` and `reconnect: true`
    - Use `shallowRef` for large arrays (sessions, history) to avoid deep reactivity overhead
    - Throttle chart data updates to `requestAnimationFrame` cadence
    - _Requirements: 3.1, 12.1, 12.2, 21.2, 21.3_

  - [ ]* 3.9 Write property tests for store data preservation on error
    - **Property 6: Store Data Preservation on Error** — For any store with existing data, when API action errors, existing data remains unchanged
    - **Validates: Requirement 3.4**

- [x] 4. Phase 3: View Decomposition — Router, Views, and Navigation Guards
  - [x] 4.1 Implement admin Vue Router configuration
    - Create `panel/web/admin/src/router/index.ts`
    - Configure `createWebHistory('/dashboard/')` with all routes (DashboardView, CustomersView, CustomerDetailView, PlansView, PaymentsView, TicketsView, TicketDetailView, ResellersView, NodesView, SettingsView, LoginView, SetupView)
    - Use dynamic `import()` for all view components (lazy loading)
    - Add catch-all route redirecting to `/dashboard`
    - Set `meta.requiresAuth: true` on all routes under the authenticated shell
    - _Requirements: 1.1, 1.3, 1.4, 1.6, 20.1_

  - [x] 4.2 Implement navigation guards for admin router
    - Add `router.beforeEach` guard implementing `resolveNavigation` algorithm
    - Redirect to `/setup` if `setupRequired` and destination is not setup
    - Redirect to `/login` if route `requiresAuth` and user is not authenticated (preserve `redirect` query param)
    - Redirect authenticated users away from login/setup to overview
    - Check `meta.roles` for role-based access control
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5_

  - [ ]* 4.3 Write property tests for navigation guard
    - **Property 1: Navigation Guard Correctness** — For any route/auth combination, guard returns correct redirect or allows navigation in priority order
    - **Property 2: Catch-All Route Redirect** — Non-matching URLs redirect to `/dashboard`
    - **Validates: Requirements 1.6, 1.7, 2.1, 2.2, 2.5**

  - [x] 4.4 Implement AppShell layout component
    - Create `panel/web/admin/src/layouts/AppShell.vue`
    - Render TheSidebar, TheTopbar, `<router-view>` with `<Transition>` and `<Suspense>`, CommandPalette, ToastProvider, ConfirmProvider
    - Use `v-slot="{ Component, route }"` pattern for route transitions (fade/slide)
    - Show `KPageSkeleton` as Suspense fallback during lazy-load
    - _Requirements: 1.8, 20.3, 24.2_

  - [x] 4.5 Implement TheSidebar component
    - Create `panel/web/admin/src/components/TheSidebar.vue`
    - Render navigation items with icons, labels, and badge counts
    - Support collapsed state toggle
    - Highlight current section based on current route
    - Emit `navigate`, `collapse-toggle`, `logout` events
    - _Requirements: 24.5_

  - [x] 4.6 Implement TheTopbar component
    - Create `panel/web/admin/src/components/TheTopbar.vue`
    - Display breadcrumb navigation reflecting current route hierarchy
    - Display realtime connection status indicator
    - Display notification count badge
    - Emit `open-command-palette`, `open-notifications`, `toggle-theme` events
    - _Requirements: 24.1, 24.4_

  - [x] 4.7 Create admin view stub components
    - Create stub components for all admin views: `DashboardView.vue`, `CustomersView.vue`, `CustomerDetailView.vue`, `PlansView.vue`, `PaymentsView.vue`, `TicketsView.vue`, `TicketDetailView.vue`, `ResellersView.vue`, `NodesView.vue`, `SettingsView.vue`, `LoginView.vue`, `SetupView.vue`
    - Each view stub should import its corresponding store, call load action on mount, and render a skeleton while loading
    - _Requirements: 1.4, 22.1–22.9_

  - [x] 4.8 Update admin App.vue and main.ts to use router and Pinia
    - Refactor `panel/web/admin/src/App.vue` to minimal `<router-view />` with providers
    - Update `panel/web/admin/src/main.ts` to create Pinia instance, create router, and mount with both plugins
    - _Requirements: 1.1, 3.1_

- [x] 5. Checkpoint — Routing and state management operational
  - Ensure all tests pass, ask the user if questions arise.

- [x] 6. Phase 4: Component Library — Shared UI Components
  - [x] 6.1 Implement KButton component
    - Create `panel/web/shared/components/KButton.vue`
    - Support `variant` (`primary`, `ghost`, `danger`, `text`), `size` (`sm`, `md`, `lg`), `loading`, `disabled`, `icon`, `iconPosition`, `fullWidth` props
    - Show spinner overlay when loading; prevent click emission when loading or disabled
    - Apply brand gradient for primary variant using design tokens
    - Ensure `aria-label` or visible text on all instances
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 18.1, 26.3_

  - [ ]* 6.2 Write property tests for KButton
    - **Property 7: Button Click Prevention** — When `loading` or `disabled` is true, zero click events are emitted regardless of click count
    - **Validates: Requirements 4.2, 4.3**

  - [x] 6.3 Implement KDataTable component
    - Create `panel/web/shared/components/KDataTable.vue`
    - Implement sortable column headers with directional indicators
    - Implement per-column filters (text, select, date-range)
    - Implement client-side and server-side pagination via `serverSide` prop
    - Implement row selection with checkboxes and `selection-change` emit
    - Implement `export` event emission for CSV/JSON
    - Show skeleton rows when `loading` is true
    - Add ARIA `role="grid"` or `role="table"` with `role="row"` and `role="cell"` semantics
    - Add keyboard navigation for rows
    - _Requirements: 5.1, 5.2, 5.4, 5.5, 5.6, 5.7, 5.8, 18.6_

  - [ ]* 6.4 Write property tests for KDataTable sorting and filtering
    - **Property 8: Data Table Sort Correctness** — After sorting by any column, rows are correctly ordered
    - **Property 9: Data Table Filter Correctness** — All displayed rows match filter; no matching rows excluded
    - **Property 12: Data Table Selection Consistency** — `selection-change` contains exactly the currently selected rows
    - **Property 13: Data Table Pagination Slice** — Displayed items are the correct slice for page P and size S
    - **Validates: Requirements 5.1, 5.2, 5.5, 5.6**

  - [x] 6.5 Implement virtual scrolling for KDataTable
    - Implement the `calculateVisibleRange` algorithm from the design
    - When `virtualScroll` prop is true, render only `visibleItems + 2 * bufferSize` DOM rows
    - Use fixed `rowHeight` for offset calculations
    - Apply CSS transform for `offsetY` positioning
    - Ensure contiguous, duplicate-free data rendering at any scroll position
    - _Requirements: 5.3, 19.1, 19.2, 19.3_

  - [ ]* 6.6 Write property tests for virtual scrolling
    - **Property 10: Virtual Scroll Bounded Rendering** — Rendered rows never exceed `visibleItems + 2 * bufferSize`
    - **Property 11: Virtual Scroll Data Integrity** — Rendered items form a contiguous, duplicate-free subsequence covering the viewport
    - **Validates: Requirements 5.3, 19.1, 19.2**

  - [x] 6.7 Implement KDrawer component
    - Create `panel/web/shared/components/KDrawer.vue`
    - Slide in from specified side (`right`/`left`) with CSS transition
    - Trap focus within drawer content when open
    - Close on Escape key or overlay click; emit `close` event
    - Apply body scroll lock when open
    - Set `role="dialog"` and `aria-modal="true"` for screen readers
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 18.2_

  - [ ]* 6.8 Write property tests for KDrawer focus trapping
    - **Property 14: Focus Trapping in Dialogs** — Tab/Shift+Tab key presses keep focus within the dialog boundary
    - **Validates: Requirements 6.2, 18.2**

  - [x] 6.9 Implement KConfirmDialog component and useConfirm composable
    - Create `panel/web/shared/components/KConfirmDialog.vue`
    - Create `panel/web/shared/composables/useConfirm.ts`
    - Display modal with title, message, action buttons matching the provided `ConfirmOptions`
    - Return Promise resolving to `true` (confirm) or `false` (cancel/Escape)
    - Auto-focus cancel button for `danger` variant
    - Keyboard: Enter to confirm, Escape to cancel
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_

  - [ ]* 6.10 Write property tests for KConfirmDialog
    - **Property 15: Confirm Dialog Content Display** — Title and message strings are rendered verbatim in visible content
    - **Validates: Requirement 7.1**

  - [x] 6.11 Implement KChart component
    - Create `panel/web/shared/components/KChart.vue`
    - Support `line`, `area`, `bar`, `donut` chart types rendered as SVG
    - Show interactive tooltips on hover when `interactive` is true
    - Animate data transitions when `animate` is true
    - Disable animations when `prefers-reduced-motion` is enabled
    - Support `gradientFill` for area charts using brand colors
    - Emit `point-hover` and `point-click` events with data point payload
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6_

  - [ ]* 6.12 Write property tests for KChart interaction
    - **Property 16: Chart Interaction Data Fidelity** — Hover/click on a data point emits event with exact label, value, and index
    - **Validates: Requirements 8.2, 8.6**

  - [x] 6.13 Implement KFormField component
    - Create `panel/web/shared/components/KFormField.vue`
    - Display label connected to input via `for`/`id` attributes
    - Display validation errors below input with `aria-describedby` linking
    - Show required indicator when `required` prop is true
    - Show `hint` text below input when no errors present
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 18.3_

  - [ ]* 6.14 Write property tests for KFormField accessibility
    - **Property 17: Form Field Accessibility Linking** — Label `for` matches input `id`; when errors exist, `aria-describedby` references the error element
    - **Validates: Requirements 9.1, 9.2, 18.3**

  - [x] 6.15 Implement KInput, KSelect, and KTextarea components
    - Create `panel/web/shared/components/KInput.vue` — text/number/password input with design system styling
    - Create `panel/web/shared/components/KSelect.vue` — dropdown select with design system styling
    - Create `panel/web/shared/components/KTextarea.vue` — multi-line input with design system styling
    - All components integrate with KFormField for validation display
    - _Requirements: 9.1, 9.2, 18.1_

  - [x] 6.16 Implement supplementary components (KStatusPill, KAvatar, KBreadcrumb, KTabs, KPagination)
    - Create `panel/web/shared/components/KStatusPill.vue` — colored badge for entity status
    - Create `panel/web/shared/components/KAvatar.vue` — initials or image in circular container
    - Create `panel/web/shared/components/KBreadcrumb.vue` — navigation trail (first N-1 items clickable, last non-clickable)
    - Create `panel/web/shared/components/KTabs.vue` — tab navigation with `role="tablist"` and arrow-key support
    - Create `panel/web/shared/components/KPagination.vue` — page controls emitting `page-change`
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5, 18.5_

  - [ ]* 6.17 Write property tests for supplementary components
    - **Property 18: Breadcrumb Rendering Correctness** — First N-1 items are links, last is non-clickable text
    - **Property 19: Pagination Event Correctness** — Emitted `page-change` carries the exact clicked page number
    - **Property 34: Status Pill Rendering** — Status value renders inside a colored badge
    - **Validates: Requirements 10.1, 10.3, 10.5**

  - [x] 6.18 Implement feedback and state components (KSkeleton, KEmptyState, KToast, KAlert)
    - Create `panel/web/shared/components/KSkeleton.vue` — animated placeholder shapes matching expected content
    - Create `panel/web/shared/components/KEmptyState.vue` — configurable icon, title, description, optional action button
    - Create `panel/web/shared/components/KToast.vue` — dismissible timed notification
    - Create `panel/web/shared/components/KAlert.vue` — inline banner with `info`, `success`, `warning`, `error` variants
    - Create `panel/web/admin/src/components/ToastProvider.vue` — global toast container
    - _Requirements: 10.6, 10.7, 10.8, 10.9_

  - [ ]* 6.19 Write property tests for toast and empty state
    - **Property 32: Toast Message Display** — Toast renders message string and auto-dismisses after specified duration
    - **Property 33: Empty State Content Rendering** — Empty state renders title and description strings with icon
    - **Validates: Requirements 10.7, 10.8**

- [~] 7. Checkpoint — Component library complete
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 8. Phase 5: Enhancement — Form Validation, Charts, Transitions, Virtual Scrolling
  - [~] 8.1 Implement useFormValidation composable
    - Create `panel/web/shared/composables/useFormValidation.ts`
    - Implement `required`, `minLength`, `maxLength`, `pattern`, and `custom` rule types
    - `validate()` evaluates ALL rules for ALL fields, returns true only if all pass
    - When `validateOnChange` is true, re-evaluate on each value change reactively
    - `reset()` restores initial values and clears all errors
    - Expose `isValid` as reactive computed property
    - Populate field `errors` array with messages from all failing rules
    - Focus first invalid field on submission failure
    - Announce errors via ARIA live region
    - _Requirements: 13.1, 13.2, 13.3, 13.4, 13.5, 13.6, 18.4, 25.3_

  - [ ]* 8.2 Write property tests for useFormValidation
    - **Property 21: Form Validation Rule Evaluation** — `validate()` evaluates every rule for every field; returns true iff all pass; errors contain messages from failing rules
    - **Property 22: Form Validation Reactivity** — When `validateOnChange` is true, errors update immediately on value change
    - **Property 23: Form Reset Round-Trip** — After modification, `reset()` restores initial values and clears errors
    - **Property 31: Validation Focus on Submission Failure** — First invalid field receives focus; all invalid fields show errors
    - **Validates: Requirements 13.1, 13.2, 13.3, 13.4, 13.5, 13.6, 25.3**

  - [~] 8.3 Implement useCommandPalette composable
    - Create `panel/web/admin/src/composables/useCommandPalette.ts`
    - Open/close on Ctrl+K (configurable shortcut)
    - Fuzzy-filter actions against label, description, keywords
    - Arrow-key navigation with wrapping (down: 0→N-1→0, up: 0→N-1)
    - Enter executes selected action; Escape closes and resets query
    - _Requirements: 16.1, 16.2, 16.3, 16.4, 16.5_

  - [ ]* 8.4 Write property tests for useCommandPalette
    - **Property 25: Command Palette Fuzzy Filtering** — All results fuzzy-match at least one of label/description/keywords; no non-matching actions appear
    - **Property 26: Command Palette Arrow Navigation** — Down increments selectedIndex (wrapping); up decrements (wrapping)
    - **Validates: Requirements 16.2, 16.5**

  - [~] 8.5 Implement CommandPalette UI component
    - Create `panel/web/admin/src/components/CommandPalette.vue`
    - Render overlay with search input and filtered action list
    - Integrate with `useCommandPalette` composable
    - Focus search input on open
    - _Requirements: 16.1, 16.2, 16.3, 16.4, 16.5_

  - [~] 8.6 Implement view transitions between routes
    - Update AppShell route transition with CSS fade/slide transitions using `<Transition name="..." mode="out-in">`
    - Respect `prefers-reduced-motion` by disabling transitions
    - Ensure browser URL always reflects current state for back/forward navigation
    - _Requirements: 24.2, 24.3_

  - [~] 8.7 Implement search debouncing and rendering optimizations
    - Add 300ms debounce to search inputs in CustomersView, TicketsView, PaymentsView using `useDebounceFn` from `@vueuse/core`
    - Throttle chart updates from WebSocket to `requestAnimationFrame` cadence
    - Use `shallowRef` for large arrays in realtime store
    - _Requirements: 21.1, 21.2, 21.3_

  - [ ]* 8.8 Write unit tests for rendering optimizations
    - Test debounce delays search processing by 300ms
    - Test shallowRef usage for large arrays
    - _Requirements: 21.1, 21.3_

- [~] 9. Checkpoint — Enhancements and composables complete
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 10. Phase 6: Portal Expansion — Portal Views, Stores, and Router
  - [~] 10.1 Implement portal Pinia stores
    - Create `panel/web/portal/src/stores/auth.ts` — portal authentication with TOTP support
    - Create `panel/web/portal/src/stores/billing.ts` — payments, plans, payment methods
    - Create `panel/web/portal/src/stores/tickets.ts` — ticket list and thread messages
    - Create `panel/web/portal/src/stores/usage.ts` — bandwidth consumption data
    - All stores follow loading state pattern and error preservation
    - _Requirements: 3.2, 3.3, 3.4, 23.1, 23.3, 23.5_

  - [~] 10.2 Implement portal Vue Router configuration
    - Create `panel/web/portal/src/router/index.ts`
    - Configure `createWebHistory('/portal/')` with routes: LoginView, DashboardView, BillingView, UsageView, SupportView, ProfileView, VpnProfilesView
    - Use dynamic `import()` for lazy loading all views
    - Add catch-all route redirecting to `/portal`
    - Add navigation guard: redirect to `/portal/login` if not authenticated
    - _Requirements: 1.2, 1.5, 1.7, 2.1_

  - [~] 10.3 Implement PortalShell layout
    - Create `panel/web/portal/src/layouts/PortalShell.vue`
    - Render PortalNavbar, `<router-view>` with transition, ToastProvider
    - _Requirements: 1.2, 24.2_

  - [~] 10.4 Implement portal view components
    - Create `panel/web/portal/src/views/LoginView.vue` — username/password + TOTP form
    - Create `panel/web/portal/src/views/DashboardView.vue` — current plan, remaining data, connection status
    - Create `panel/web/portal/src/views/BillingView.vue` — payment history, balance, methods, payment form
    - Create `panel/web/portal/src/views/UsageView.vue` — bandwidth charts using KChart
    - Create `panel/web/portal/src/views/SupportView.vue` — ticket creation and thread view
    - Create `panel/web/portal/src/views/ProfileView.vue` — update display name, password, notification prefs
    - Create `panel/web/portal/src/views/VpnProfilesView.vue` — VPN config profiles for download
    - _Requirements: 23.1, 23.2, 23.3, 23.4, 23.5, 23.6, 23.7_

  - [~] 10.5 Update portal App.vue and main.ts
    - Refactor `panel/web/portal/src/App.vue` to minimal `<router-view />`
    - Update `panel/web/portal/src/main.ts` to create Pinia instance, create router, and mount with both plugins
    - _Requirements: 1.2, 3.2_

  - [~] 10.6 Implement portal-specific components
    - Create `panel/web/portal/src/components/PortalNavbar.vue` — navigation bar for portal
    - Create `panel/web/portal/src/components/UsageGauge.vue` — visual data usage indicator
    - Create `panel/web/portal/src/components/PlanCard.vue` — current plan display card
    - Create `panel/web/portal/src/components/TicketThread.vue` — conversation thread for support tickets
    - _Requirements: 23.2, 23.4, 23.5_

  - [~] 10.7 Create portal i18n locale files
    - Create `panel/web/portal/src/i18n/index.ts`, `en.ts`, `fa.ts`, `zh.ts`
    - Include all portal-specific translation keys
    - _Requirements: 15.1_

- [ ] 11. Phase 6 (continued): Admin View Full Implementation
  - [~] 11.1 Implement DashboardView with real-time data
    - Flesh out `panel/web/admin/src/views/DashboardView.vue`
    - Display real-time statistics using StatsGrid component
    - Display traffic charts using KChart (area type with gradient fill)
    - Display active sessions using SessionsTable
    - Wire to realtimeStore for live updates
    - _Requirements: 22.1_

  - [~] 11.2 Implement CustomersView with KDataTable
    - Flesh out `panel/web/admin/src/views/CustomersView.vue`
    - Use KDataTable with sortable/filterable columns, row selection
    - Implement search with 300ms debounce
    - Navigate to CustomerDetailView on row click
    - Use KConfirmDialog for delete actions
    - _Requirements: 22.2, 5.1, 5.2, 21.1_

  - [~] 11.3 Implement CustomerDetailView
    - Flesh out `panel/web/admin/src/views/CustomerDetailView.vue`
    - Display customer profile, usage, subscription history, wallet transactions in tabbed layout using KTabs
    - Use KFormField + useFormValidation for edit forms
    - Load data via customersStore on route enter
    - _Requirements: 22.3, 13.1_

  - [~] 11.4 Implement PlansView, PaymentsView, TicketsView, TicketDetailView
    - Flesh out each view using KDataTable for list display
    - PlansView: table with create/edit/delete using KConfirmDialog
    - PaymentsView: table with status filtering and approve/reject actions
    - TicketsView: table with status and priority filtering
    - TicketDetailView: ticket conversation thread with reply form
    - _Requirements: 22.4, 22.5, 22.6_

  - [~] 11.5 Implement ResellersView and NodesView
    - ResellersView: reseller table with balance/transaction info
    - NodesView: node cards with real-time telemetry (CPU, RAM, disk, bandwidth) and service status using KChart/KSparkline
    - _Requirements: 22.7, 22.8_

  - [~] 11.6 Implement SettingsView with tabbed configuration
    - Flesh out `panel/web/admin/src/views/SettingsView.vue`
    - Use KTabs for tab navigation (general, gateway, notifications, templates)
    - Each tab uses KFormField + useFormValidation for configuration forms
    - Route tab selection to URL parameter (`/settings/:tab`)
    - _Requirements: 22.9_

  - [~] 11.7 Implement LoginView and SetupView
    - LoginView: username/password form with validation, redirect after login
    - SetupView: initial system configuration wizard
    - Both use KFormField, KButton, useFormValidation
    - _Requirements: 2.1, 2.2, 2.3_

  - [~] 11.8 Implement admin i18n locale files
    - Create `panel/web/admin/src/i18n/index.ts`, `en.ts`, `fa.ts`, `zh.ts`
    - Include all admin-specific translation keys
    - _Requirements: 15.1_

  - [~] 11.9 Wire error handling across the application
    - Integrate toast notifications for API errors on user-initiated actions
    - Display offline indicator when WebSocket disconnects (realtimeStore.connected = false)
    - Implement route error state with retry option in AppShell
    - _Requirements: 25.1, 25.2, 25.4_

  - [ ]* 11.10 Write integration tests for admin app flows
    - Test login → dashboard → navigate between sections
    - Test customer CRUD with form validation
    - Test error handling (API failure, session expiry)
    - Test command palette navigation
    - _Requirements: 2.1, 22.2, 25.1, 16.3_

- [~] 12. Final Checkpoint — All features implemented
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation at phase boundaries
- Property tests validate universal correctness properties from the design document
- Unit tests validate specific examples and edge cases
- The design uses TypeScript throughout — all implementations must be type-safe
- Shared components (`@koris/ui`) are imported via Vite aliases, not npm packages
- The existing design system tokens are preserved; no visual redesign is needed
- Virtual scrolling is only needed for tables expected to exceed 100+ items (customers, sessions)
- All ARIA attributes and keyboard navigation must be included in initial component implementation, not retrofitted

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1", "1.2"] },
    { "id": 1, "tasks": ["1.3", "1.4"] },
    { "id": 2, "tasks": ["1.5", "1.7", "1.9", "1.10", "1.12"] },
    { "id": 3, "tasks": ["1.6", "1.8", "1.11"] },
    { "id": 4, "tasks": ["3.1", "3.8"] },
    { "id": 5, "tasks": ["3.2", "3.3", "3.4", "3.5", "3.6", "3.7"] },
    { "id": 6, "tasks": ["3.9"] },
    { "id": 7, "tasks": ["4.1", "4.5", "4.6"] },
    { "id": 8, "tasks": ["4.2", "4.4", "4.7"] },
    { "id": 9, "tasks": ["4.3", "4.8"] },
    { "id": 10, "tasks": ["6.1", "6.7", "6.9", "6.11", "6.13", "6.15", "6.16", "6.18"] },
    { "id": 11, "tasks": ["6.2", "6.3", "6.8", "6.10", "6.14", "6.17", "6.19"] },
    { "id": 12, "tasks": ["6.4", "6.5"] },
    { "id": 13, "tasks": ["6.6", "6.12"] },
    { "id": 14, "tasks": ["8.1", "8.3"] },
    { "id": 15, "tasks": ["8.2", "8.4", "8.5", "8.6", "8.7"] },
    { "id": 16, "tasks": ["8.8"] },
    { "id": 17, "tasks": ["10.1", "10.7"] },
    { "id": 18, "tasks": ["10.2"] },
    { "id": 19, "tasks": ["10.3", "10.4", "10.6"] },
    { "id": 20, "tasks": ["10.5"] },
    { "id": 21, "tasks": ["11.1", "11.2", "11.3", "11.8"] },
    { "id": 22, "tasks": ["11.4", "11.5", "11.6", "11.7"] },
    { "id": 23, "tasks": ["11.9"] },
    { "id": 24, "tasks": ["11.10"] }
  ]
}
```
