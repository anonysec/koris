# Shared Library Usage Status

This document tracks the usage status of all shared components (`@koris/ui`) and composables (`@koris/composables`) across the admin and portal applications.

---

## Components

### Actively Used

| Component | Where Used |
|-----------|-----------|
| **KAvatar** | admin/CustomerDetailView.vue |
| **KButton** | Widely used across admin and portal views |
| **KChart** | admin/DashboardView.vue, portal/UsageView.vue |
| **KConfirmDialog** | admin/CustomersView.vue (via `useConfirm`) |
| **KDataTable** | admin: TemplatesView, PaymentsView, ResellersView; portal: BillingView, SupportView, UsageView |
| **KEmptyState** | admin: NodesView, TemplatesView, PlansView, TicketsView; portal: UsageView |
| **KFormField** | Widely used across admin and portal views |
| **KInput** | Widely used across admin and portal views |
| **KSelect** | Widely used across admin and portal views |
| **KSkeleton** | admin: CustomerDetailView, DashboardView, NodesView, TicketDetailView; portal: BillingView, UsageView |
| **KStatusPill** | Widely used across admin and portal views |
| **KTabs** | admin: CustomerDetailView, NodesView, SettingsView |
| **KTextarea** | admin: CustomerDetailView, TemplatesView, TicketDetailView, TicketsView; portal: SupportView |
| **KToast** | admin/ToastProvider.vue |

### Available for Future Integration

These components are fully implemented and ready to use but have not yet been imported into any view. They are intentionally kept in the shared library as they align with the [DESIGN_SYSTEM.md](../DESIGN_SYSTEM.md) roadmap for planned features.

| Component | Purpose | Planned Integration |
|-----------|---------|---------------------|
| **HealthDot** | Colored dot (green/yellow/red) indicating node health score (0-1) | admin/NodesView.vue next to node name in card header |
| **KAlert** | Inline info/warning/error/success messages with optional dismissal | General use across admin and portal for contextual alerts |
| **KBreadcrumb** | Breadcrumb navigation component | Navigation enhancement for nested views |
| **KDrawer** | Slide-out side panel (right or left) | Customer detail drawer (see DESIGN_SYSTEM.md "Next UI components to add") |
| **KPagination** | Pagination control for tables/lists | Alternative UX for large datasets in KDataTable (see note below) |

---

## Composables

### Actively Used

| Composable | Where Used |
|-----------|-----------|
| **useApi** | All admin stores (auth, customers, nodes, payments, plans, resellers, settings, templates, tickets); portal/VpnProfilesView |
| **useClipboard** | portal/VpnProfilesView |
| **useConfirm** | admin/CustomersView (provides confirmation dialog via KConfirmDialog) |
| **useFormatDate** | portal: BillingView, SupportView, TicketThread (consolidated from duplicate inline functions) |
| **useFormValidation** | Shared internally; has property-based tests |
| **useFreshData** | portal: BillingView, UsageView (portal-only; admin uses WebSocket realtime updates via useWebSocket instead of stale-data guards) |
| **useI18n** | admin: TheSidebar, AppShell, main.ts |
| **useTheme** | admin: TheSidebar, AppShell; portal: PortalNavbar, PortalShell |
| **useToast** | admin: CustomerDetailView, NodesView, SettingsView (via ToastProvider + KToast) |
| **useWebSocket** | admin/stores/realtime.ts |

### Available for Future Integration

| Composable | Purpose | Notes |
|-----------|---------|-------|
| **useVirtualScroll** | Windowed rendering for large datasets (1000+ rows) | Specifically designed to work with KDataTable for large dataset optimization. Implements virtual scrolling with configurable buffer size. |

---

## Notes

- **useVirtualScroll + KDataTable**: For tables with 1000+ rows, `useVirtualScroll` provides efficient windowed rendering by only mounting visible rows plus a configurable buffer. As an alternative UX approach for large datasets, `KPagination` can be used to split results into pages instead of virtual scrolling.

- **Unused does not mean dead code**: The components and composables listed under "Available for Future Integration" are intentionally maintained. They align with the DESIGN_SYSTEM.md roadmap under "Next UI components to add" (customer detail drawer, plan CRUD, ticket conversation panel, etc.) and are ready for integration as those features are built out.
