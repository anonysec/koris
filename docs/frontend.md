# 🎨 Frontend Design System & Components

> Consolidated from the former `web/DESIGN_SYSTEM.md` and `web/theme/COMPONENT_USAGE.md`.
> For the cross-cutting polish layer, see [UI / UX](ui-ux.md).

The Koris frontend is a **Vue 3 + Vite** pnpm monorepo (`admin`, `portal`, `landing` + shared `core`/`theme`). It uses a compact, premium SaaS/dashboard style optimized for VPN operations.

---

## 🧭 Product mood

- Professional network operations, not consumer entertainment.
- Dense but readable admin workflows.
- Dark command-center surface with cyan/blue signal accents.
- Clear operational status language: online, active, limited, expired, disabled.

## 📐 Design rules

- Keep primary actions obvious and close to the user's current task.
- Prefer text, status pills, and simple glyphs over decorative icons for critical controls.
- Tables must stay compact, scannable, and responsive.
- Every form control needs a visible focus state and clear error state.
- Empty states should explain the next action.
- Respect reduced motion (`prefers-reduced-motion`).

## 🎛️ Tokens

Defined in `web/core/styles/tokens.css`; each theme overrides them. Defaults:

| Token | Value |
|-------|-------|
| Background | `#070a12`, `#0b1120` |
| Surface | translucent slate panels with subtle glass depth |
| Border | low-contrast slate line, stronger on hover/focus |
| Primary | `#2563eb` |
| Accent | `#22d3ee` |
| Success | `#22c55e` |
| Warning | `#f59e0b` |
| Danger | `#ef4444` |

The [`overhaul.css`](ui-ux.md) layer builds on these tokens to restyle every tab consistently across all 6 themes and light/dark modes.

---

## 🧩 Shared components (`@koris/theme`)

### Actively used

| Component | Where |
|-----------|-------|
| **Button, Input, Select, FormField, Textarea** | Widely used across admin & portal |
| **DataTable** | admin: Templates, Payments, Resellers; portal: Billing, Support, Usage |
| **StatusPill** | Widely used |
| **EmptyState** | admin: Nodes, Templates, Plans, Tickets; portal: Usage |
| **Skeleton / SkeletonLoader** | admin: CustomerDetail, Dashboard, Nodes, TicketDetail; portal: Billing, Usage |
| **Tabs** | admin: CustomerDetail, Nodes, Settings |
| **Chart** | admin: Dashboard; portal: Usage |
| **Avatar** | admin: CustomerDetail |
| **ConfirmDialog** | admin: Customers (via `useConfirm`) |
| **Toast** | admin: ToastProvider |

### Available for future integration

| Component | Purpose |
|-----------|---------|
| **HealthDot** | Node health score dot (green/yellow/red) |
| **Alert** | Inline info/warning/error/success messages |
| **Breadcrumb** | Nested-view navigation |
| **SlideOver / Drawer** | Customer detail drawer |
| **Pagination** | Alternative to virtual scroll for large tables |

---

## 🪝 Composables (`@koris/core`)

### Actively used

| Composable | Where |
|-----------|-------|
| **useApi** | All admin stores; portal VPN profiles |
| **useClipboard** | portal: VPN profiles |
| **useConfirm** | admin: Customers |
| **useFormatDate** | portal: Billing, Support, TicketThread |
| **useFormValidation** | shared; property-based tests |
| **useFreshData** | portal: Billing, Usage |
| **useI18n** | admin: Sidebar, AppShell, main.ts |
| **useTheme** | admin & portal shells/nav |
| **useToast** | admin: CustomerDetail, Nodes, Settings |
| **useWebSocket** | admin: realtime store |

### Available for future integration

| Composable | Purpose |
|-----------|---------|
| **useVirtualScroll** | Windowed rendering for 1000+ row tables (pairs with DataTable) |

---

## 🛣️ Roadmap UI components

- Customer detail drawer/page
- Plan cards + plan CRUD table
- Payment review queue
- Ticket conversation panel
- Node telemetry cards
- Global command/search palette (partially shipped as CommandPalette)
