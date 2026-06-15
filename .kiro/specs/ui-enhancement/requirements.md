# Requirements Document

## Introduction

KorisPanel is a VPN management platform with two Vue 3 + TypeScript frontend applications: an Admin Panel and a Customer Portal. This document defines the requirements for a comprehensive UI overhaul that refactors the monolithic architecture into a modular codebase with Vue Router, Pinia state management, a shared component library, composable-based logic extraction, interactive data visualization, accessibility compliance (WCAG 2.1 AA), and performance optimizations.

## Glossary

- **Admin_Panel**: The administrative frontend application for system management, served at `/dashboard/`
- **Customer_Portal**: The end-user self-service frontend application, served at `/portal/`
- **Shared_Component_Library**: A package (`@koris/ui`) of reusable Vue components shared between Admin_Panel and Customer_Portal
- **Vue_Router**: Client-side routing library providing URL-based navigation and route guards
- **Pinia_Store**: Reactive state management stores that hold application data and expose actions
- **Composable**: A reusable function following the Vue Composition API pattern that encapsulates reactive logic
- **Navigation_Guard**: A function executed before route transitions to enforce authentication and access control
- **Virtual_Scroll_Engine**: An algorithm that renders only visible rows plus a buffer for large datasets
- **Form_Validation_Engine**: A composable-based system that evaluates validation rules against form field values
- **Command_Palette**: A keyboard-triggered overlay (Ctrl+K) for searching and executing application actions
- **Design_System_Tokens**: CSS custom properties defining colors, spacing, and typography (`#070a12` background, `#2563eb` primary, `#22d3ee` accent)
- **KDataTable**: A shared data table component with sorting, filtering, pagination, and virtual scrolling
- **KDrawer**: A slide-in panel component for detail views with focus trapping
- **KConfirmDialog**: A branded modal dialog that replaces native `confirm()` and returns a Promise
- **KChart**: An interactive SVG-based chart component with tooltips and animations
- **KFormField**: A form field wrapper component with integrated validation error display
- **AppShell**: The application layout component containing sidebar, topbar, router-view, and global providers
- **WebSocket_Reconnection**: Exponential backoff algorithm for automatic reconnection after disconnection
- **Skeleton_State**: A placeholder UI shown during data loading to prevent layout shift

## Requirements

### Requirement 1: Architecture and Routing

**User Story:** As a developer, I want the monolithic App.vue decomposed into modular views with client-side routing, so that the codebase is maintainable and navigable by URL.

#### Acceptance Criteria

1. THE Admin_Panel SHALL use Vue_Router with `createWebHistory('/dashboard/')` to provide URL-based navigation for all views
2. THE Customer_Portal SHALL use Vue_Router with `createWebHistory('/portal/')` to provide URL-based navigation for all views
3. WHEN a user navigates to a route, THE Vue_Router SHALL lazy-load the target view component using dynamic `import()` statements
4. THE Admin_Panel SHALL provide routes for DashboardView, CustomersView, CustomerDetailView, PlansView, PaymentsView, TicketsView, TicketDetailView, ResellersView, NodesView, and SettingsView
5. THE Customer_Portal SHALL provide routes for LoginView, DashboardView, BillingView, UsageView, SupportView, ProfileView, and VpnProfilesView
6. WHEN a user navigates to a non-existent route in Admin_Panel, THE Vue_Router SHALL redirect to `/dashboard`
7. WHEN a user navigates to a non-existent route in Customer_Portal, THE Vue_Router SHALL redirect to `/portal`
8. THE AppShell SHALL render TheSidebar, TheTopbar, router-view, CommandPalette, ToastProvider, and ConfirmProvider as its layout structure

### Requirement 2: Authentication and Navigation Guards

**User Story:** As an administrator, I want routes to be protected by authentication guards, so that unauthenticated users cannot access protected content.

#### Acceptance Criteria

1. WHEN a user navigates to a route with `meta.requiresAuth = true` and the user is not authenticated, THE Navigation_Guard SHALL redirect to the login page
2. WHEN the system requires initial setup and the user navigates to any route other than `/setup`, THE Navigation_Guard SHALL redirect to the setup page
3. WHEN an authenticated user navigates to the login page, THE Navigation_Guard SHALL redirect to the dashboard
4. WHEN an API call returns HTTP 401 during an active session, THE Admin_Panel SHALL clear authentication state, redirect to login with a `redirect` query parameter preserving the current location, and display a "Session expired" toast notification
5. WHEN a route specifies required roles in `meta.roles` and the authenticated user lacks the specified role, THE Navigation_Guard SHALL redirect to `/dashboard`

### Requirement 3: Pinia State Management

**User Story:** As a developer, I want reactive state managed through Pinia stores with clear loading states, so that components can reliably observe data changes.

#### Acceptance Criteria

1. THE Admin_Panel SHALL use Pinia_Store instances for auth, customers, plans, payments, tickets, nodes, resellers, and realtime data
2. THE Customer_Portal SHALL use Pinia_Store instances for auth, billing, tickets, and usage data
3. WHEN a Pinia_Store action initiates an API call, THE Pinia_Store SHALL set `loading = true` before the request and `loading = false` after completion regardless of success or failure
4. WHEN a Pinia_Store action encounters an API error, THE Pinia_Store SHALL preserve existing data and surface the error message through a toast notification for user-initiated actions

### Requirement 4: Shared Component Library — Buttons and Actions

**User Story:** As a developer, I want unified button components with consistent variants, so that all interactive controls follow the design system.

#### Acceptance Criteria

1. THE KButton component SHALL support `variant` prop with values `primary`, `ghost`, `danger`, and `text` matching Design_System_Tokens
2. WHEN the `loading` prop is `true`, THE KButton SHALL display a spinner overlay and prevent click event emission
3. WHEN the `disabled` prop is `true`, THE KButton SHALL prevent click event emission and apply disabled visual styling
4. THE KButton SHALL support `size` prop with values `sm`, `md`, and `lg`
5. THE KButton SHALL support `icon` and `iconPosition` props for icon-enhanced buttons

### Requirement 5: Shared Component Library — Data Table

**User Story:** As an administrator, I want a feature-rich data table with sorting, filtering, and pagination, so that I can efficiently browse and manage large datasets.

#### Acceptance Criteria

1. WHEN a user clicks a sortable column header, THE KDataTable SHALL sort the data by that column and display a directional indicator
2. THE KDataTable SHALL support per-column filtering with `text`, `select`, and `date-range` filter types
3. WHEN `virtualScroll` prop is `true`, THE KDataTable SHALL render only the visible rows plus a configurable buffer to maintain performance for datasets exceeding 1000 rows
4. WHEN `loading` prop is `true`, THE KDataTable SHALL display skeleton placeholder rows
5. THE KDataTable SHALL support row selection with checkbox and emit `selection-change` events
6. THE KDataTable SHALL support both client-side and server-side pagination via the `serverSide` prop
7. WHEN a user triggers an export action, THE KDataTable SHALL emit an `export` event with format `csv` or `json`
8. THE KDataTable SHALL provide keyboard navigation for rows with proper ARIA `role` and `aria-` attributes

### Requirement 6: Shared Component Library — Drawer and Modal

**User Story:** As an administrator, I want slide-in panels for detailed views and modals for focused interactions, so that complex content can be viewed without full page navigation.

#### Acceptance Criteria

1. WHEN the `open` prop is set to `true`, THE KDrawer SHALL slide in from the specified side with a CSS transition
2. WHILE the KDrawer is open, THE KDrawer SHALL trap keyboard focus within its content area
3. WHEN a user presses the Escape key while the KDrawer is open, THE KDrawer SHALL close and emit a `close` event
4. WHILE the KDrawer is open, THE KDrawer SHALL apply a body scroll lock to prevent background scrolling
5. THE KDrawer SHALL announce its presence to screen readers via `role="dialog"` and `aria-modal="true"` attributes

### Requirement 7: Shared Component Library — Confirm Dialog

**User Story:** As a user, I want branded confirmation dialogs instead of native browser confirms, so that destructive actions require explicit acknowledgment with accessible UI.

#### Acceptance Criteria

1. WHEN `confirm()` is called, THE KConfirmDialog SHALL display a modal with the specified title, message, and action buttons
2. WHEN a user clicks the confirm button, THE KConfirmDialog SHALL resolve its Promise with `true` and close
3. WHEN a user clicks the cancel button or presses Escape, THE KConfirmDialog SHALL resolve its Promise with `false` and close
4. WHEN the `variant` is `danger`, THE KConfirmDialog SHALL auto-focus the cancel button to prevent accidental destructive actions
5. THE KConfirmDialog SHALL be keyboard accessible with Enter to confirm and Escape to cancel

### Requirement 8: Shared Component Library — Interactive Charts

**User Story:** As an administrator, I want interactive charts with tooltips and animations replacing static SVG polylines, so that I can explore data visually.

#### Acceptance Criteria

1. THE KChart SHALL support `line`, `area`, `bar`, and `donut` chart types rendered as SVG
2. WHEN `interactive` prop is `true` and a user hovers over a data point, THE KChart SHALL display a tooltip with the point's label and value
3. WHEN `animate` prop is `true`, THE KChart SHALL animate data transitions with smooth interpolation
4. WHEN the user's system has `prefers-reduced-motion` enabled, THE KChart SHALL disable animations
5. THE KChart SHALL support `gradientFill` for area charts using Design_System_Tokens brand colors
6. THE KChart SHALL emit `point-hover` and `point-click` events with the data point payload

### Requirement 9: Shared Component Library — Form Field and Validation Display

**User Story:** As a user, I want form fields that display validation errors inline with accessible annotations, so that I can quickly identify and fix input mistakes.

#### Acceptance Criteria

1. THE KFormField SHALL display a label connected to its input via `for`/`id` attributes
2. WHEN validation errors exist for a field, THE KFormField SHALL display error messages below the input with `aria-describedby` linking
3. WHEN the `required` prop is `true`, THE KFormField SHALL display a required indicator on the label
4. THE KFormField SHALL support display of a `hint` text below the input when no errors are present

### Requirement 10: Shared Component Library — Supplementary Components

**User Story:** As a developer, I want a comprehensive set of UI primitives (status pills, avatars, breadcrumbs, tabs, pagination, skeletons, empty states, toasts, alerts), so that all views use consistent, reusable patterns.

#### Acceptance Criteria

1. THE KStatusPill SHALL render a colored badge with text representing entity status values
2. THE KAvatar SHALL display user initials or an image within a circular container
3. THE KBreadcrumb SHALL render a navigation trail with clickable ancestor links and a non-clickable current location
4. THE KTabs SHALL provide tab navigation with keyboard arrow-key support and `role="tablist"` semantics
5. THE KPagination SHALL render page controls and emit `page-change` events when the user navigates between pages
6. WHEN data is loading, THE KSkeleton SHALL render animated placeholder shapes matching the expected content layout
7. WHEN a dataset is empty, THE KEmptyState SHALL display a configurable icon, title, and description with an optional action button
8. WHEN a toast notification is triggered, THE KToast SHALL display a dismissible message that auto-hides after a configurable duration
9. THE KAlert SHALL render an inline banner with `info`, `success`, `warning`, and `error` variants

### Requirement 11: Composable — useApi

**User Story:** As a developer, I want a unified API composable that handles loading state, error handling, and authentication failures, so that all API interactions follow consistent patterns.

#### Acceptance Criteria

1. THE useApi composable SHALL provide `get`, `post`, `put`, `patch`, and `del` methods that return typed Promise responses
2. WHEN an API request begins, THE useApi composable SHALL set `loading.value = true` and set it to `false` after completion
3. WHEN an API response has status 401, THE useApi composable SHALL invoke the `onUnauthorized` callback
4. WHEN an API response has a non-2xx status code, THE useApi composable SHALL set `error.value` with a human-readable message
5. THE useApi composable SHALL automatically set `Content-Type: application/json` for POST, PUT, and PATCH requests

### Requirement 12: Composable — useWebSocket

**User Story:** As an administrator, I want real-time data updates via WebSocket with automatic reconnection, so that the dashboard reflects live system state.

#### Acceptance Criteria

1. WHEN `autoConnect` is `true`, THE useWebSocket composable SHALL establish a WebSocket connection on composable initialization
2. WHEN the WebSocket connection drops and `reconnect` is `true`, THE useWebSocket composable SHALL attempt reconnection with exponential backoff starting at 1 second and capped at 30 seconds
3. WHEN `disconnect()` is called, THE useWebSocket composable SHALL close the socket, cancel all pending reconnection timers, and set `connected.value = false`
4. WHEN the component unmounts, THE useWebSocket composable SHALL clean up the connection and all timers to prevent memory leaks
5. WHEN `maxReconnectAttempts` is exceeded, THE useWebSocket composable SHALL stop reconnection attempts and leave `connected.value = false`

### Requirement 13: Composable — useFormValidation

**User Story:** As a developer, I want a composable-based form validation engine with per-field rules, so that forms provide immediate feedback without submitting invalid data.

#### Acceptance Criteria

1. THE useFormValidation composable SHALL evaluate `required`, `minLength`, `maxLength`, `pattern`, and `custom` rule types against field values
2. WHEN `validate()` is called, THE Form_Validation_Engine SHALL evaluate ALL rules for ALL registered fields and return `true` only if every field passes every rule
3. WHEN `validateOnChange` is `true`, THE Form_Validation_Engine SHALL re-evaluate rules on each value change and update `errors` reactively
4. WHEN `reset()` is called, THE Form_Validation_Engine SHALL restore all values to their initial state and clear all error arrays
5. THE useFormValidation composable SHALL expose `isValid` as a reactive computed property that reflects overall form validity
6. WHEN a field fails validation, THE Form_Validation_Engine SHALL populate that field's `errors` array with the messages from all failing rules

### Requirement 14: Composable — useTheme

**User Story:** As a user, I want to toggle between dark and light themes with my preference persisted, so that the interface respects my visual preference across sessions.

#### Acceptance Criteria

1. WHEN `toggle()` is called, THE useTheme composable SHALL switch the active theme and persist the choice to `localStorage`
2. WHEN the application loads, THE useTheme composable SHALL apply the persisted theme before first paint to prevent a flash of incorrect theme
3. THE useTheme composable SHALL apply theme changes by updating CSS custom properties on the document root

### Requirement 15: Composable — useI18n

**User Story:** As a user, I want the interface available in multiple languages, so that I can use the application in my preferred language.

#### Acceptance Criteria

1. THE useI18n composable SHALL support English (en), Persian (fa), and Chinese (zh) locales
2. WHEN a translation key is requested via `t()`, THE useI18n composable SHALL return the translated string for the active locale
3. IF a translation key is missing in the active locale, THEN THE useI18n composable SHALL fall back to the English translation rather than displaying a raw key string

### Requirement 16: Composable — useCommandPalette

**User Story:** As an administrator, I want a keyboard-triggered command palette for quick navigation and action execution, so that I can work efficiently without mouse interaction.

#### Acceptance Criteria

1. WHEN the user presses Ctrl+K (or a configured shortcut), THE Command_Palette SHALL open with an empty search input focused
2. WHEN the user types in the Command_Palette, THE Command_Palette SHALL display fuzzy-matched actions filtered against label, description, and keywords
3. WHEN the user presses Enter with an action selected, THE Command_Palette SHALL execute that action and close
4. WHEN the user presses Escape, THE Command_Palette SHALL close and reset the search query
5. WHEN the Command_Palette is open, THE Command_Palette SHALL support arrow-key navigation between filtered results

### Requirement 17: Composable — useClipboard

**User Story:** As a user, I want to copy values to the clipboard with a single click, so that I can quickly share or reuse displayed data.

#### Acceptance Criteria

1. WHEN `copy(text)` is called, THE useClipboard composable SHALL write the provided text to the system clipboard
2. WHEN a clipboard copy succeeds, THE useClipboard composable SHALL set `copied.value = true` for a brief duration to enable UI feedback

### Requirement 18: Accessibility — WCAG 2.1 AA Compliance

**User Story:** As a user with assistive technology, I want all interactive elements to be properly labeled and keyboard-navigable, so that I can use the application without a mouse.

#### Acceptance Criteria

1. THE Shared_Component_Library SHALL ensure every interactive element has one of: visible text label, `aria-label`, or `aria-labelledby`
2. WHILE a modal or KDrawer is open, THE component SHALL trap focus so that Tab and Shift+Tab cycle only through elements within the dialog
3. THE KFormField SHALL associate validation errors with inputs via `aria-describedby` so screen readers announce errors
4. WHEN validation errors occur, THE Form_Validation_Engine SHALL announce errors via an ARIA live region
5. THE KTabs SHALL implement `role="tablist"`, `role="tab"`, and `role="tabpanel"` with keyboard arrow-key navigation between tabs
6. THE KDataTable SHALL use `role="grid"` or `role="table"` with appropriate `role="row"` and `role="cell"` semantics

### Requirement 19: Performance — Virtual Scrolling

**User Story:** As an administrator, I want tables with thousands of rows to remain responsive, so that large datasets do not degrade the user interface.

#### Acceptance Criteria

1. WHEN `virtualScroll` is enabled on KDataTable, THE Virtual_Scroll_Engine SHALL render at most `visibleItems + 2 * bufferSize` DOM rows
2. WHEN the user scrolls to any position, THE Virtual_Scroll_Engine SHALL render the correct data slice without gaps or duplicate rows
3. THE Virtual_Scroll_Engine SHALL use a fixed row height for offset calculations to ensure accurate positioning

### Requirement 20: Performance — Code Splitting and Lazy Loading

**User Story:** As a user, I want the application to load quickly, so that I can start working without waiting for unnecessary code to download.

#### Acceptance Criteria

1. THE Admin_Panel SHALL lazy-load all view components at the route level using dynamic `import()` so that initial bundle size remains under 100KB gzipped
2. THE Admin_Panel SHALL split heavy components (charts, data tables) into separate chunks loaded on demand
3. WHEN a lazy-loaded view is being fetched, THE AppShell SHALL display a Skeleton_State fallback via `<Suspense>`

### Requirement 21: Performance — Rendering Optimizations

**User Story:** As a user, I want the interface to remain fluid during real-time data updates, so that frequent WebSocket messages do not cause jank.

#### Acceptance Criteria

1. THE Admin_Panel SHALL debounce search input processing by 300 milliseconds to reduce unnecessary filter recalculations
2. THE Admin_Panel SHALL throttle chart updates from WebSocket data to `requestAnimationFrame` cadence
3. THE Admin_Panel SHALL use `shallowRef` for large arrays (customers, sessions) to avoid deep reactivity overhead

### Requirement 22: Admin Panel Views

**User Story:** As an administrator, I want each management section (dashboard, customers, plans, payments, tickets, resellers, nodes, settings) as a dedicated view, so that I can navigate directly to any section by URL.

#### Acceptance Criteria

1. THE DashboardView SHALL display real-time statistics, traffic charts, and active session information
2. THE CustomersView SHALL display a paginated, sortable, and filterable table of all customers with search functionality
3. THE CustomerDetailView SHALL display comprehensive customer information including profile, usage, subscription history, and wallet transactions accessible via tabs
4. THE PlansView SHALL display all subscription plans in a manageable table with create, edit, and delete capabilities
5. THE PaymentsView SHALL display all payment records with status filtering and approval/rejection actions
6. THE TicketsView SHALL display all support tickets with status and priority filtering
7. THE ResellersView SHALL display reseller accounts with balance and transaction information
8. THE NodesView SHALL display all VPN nodes with real-time telemetry (CPU, RAM, disk, bandwidth) and service status
9. THE SettingsView SHALL provide tabbed configuration for general settings, gateway configuration, notification settings, and template management

### Requirement 23: Customer Portal Views

**User Story:** As a customer, I want self-service views for managing my account, viewing usage, handling billing, and getting support, so that I can manage my VPN subscription independently.

#### Acceptance Criteria

1. THE Portal LoginView SHALL provide username and password authentication with TOTP support where configured
2. THE Portal DashboardView SHALL display the customer's current plan, remaining data usage, and connection status
3. THE BillingView SHALL display payment history, current balance, available payment methods, and a payment submission form
4. THE UsageView SHALL display bandwidth consumption over time with interactive charts
5. THE SupportView SHALL provide ticket creation and thread-based conversation for existing tickets
6. THE ProfileView SHALL allow the customer to update display name, password, and notification preferences
7. THE VpnProfilesView SHALL display available VPN configuration profiles for download

### Requirement 24: Navigation and UX

**User Story:** As a user, I want contextual breadcrumbs, smooth view transitions, and consistent navigation patterns, so that I always know where I am and can move efficiently.

#### Acceptance Criteria

1. THE TheTopbar SHALL display breadcrumb navigation reflecting the current route hierarchy
2. WHEN navigating between views, THE AppShell SHALL apply a CSS transition (fade or slide) between outgoing and incoming view components
3. THE Admin_Panel browser URL SHALL always reflect the current application state so that browser back/forward navigation restores the previous view
4. WHEN a WebSocket connection is active, THE TheTopbar SHALL display a realtime connection status indicator
5. THE TheSidebar SHALL support a collapsed state toggled by user interaction, preserving the selection state

### Requirement 25: Error Handling

**User Story:** As a user, I want clear error feedback and graceful degradation when things go wrong, so that I can understand problems and continue working.

#### Acceptance Criteria

1. WHEN an API request fails, THE useApi composable SHALL display a toast notification with a human-readable error message for user-initiated actions
2. WHEN a WebSocket disconnection occurs, THE Admin_Panel SHALL display an offline status indicator and continue functioning with last-known data
3. WHEN form validation fails on submission, THE Form_Validation_Engine SHALL focus the first invalid field and display all field-level errors simultaneously
4. IF a route navigation fails due to a loading error, THEN THE AppShell SHALL display an error state with a retry option

### Requirement 26: Design System Consistency

**User Story:** As a designer, I want all components to use shared design tokens and follow the dark command-center aesthetic, so that the visual language is consistent across both applications.

#### Acceptance Criteria

1. THE Shared_Component_Library SHALL use CSS custom properties from Design_System_Tokens for all colors, spacing, and typography
2. THE useTheme composable SHALL apply theme changes exclusively through CSS custom property updates on the document root element
3. THE KButton `primary` variant SHALL render with the brand gradient using Design_System_Tokens primary (`#2563eb`) and accent (`#22d3ee`) colors
