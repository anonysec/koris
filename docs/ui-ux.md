# 🎨 UI / UX & Frontend Design System

The Koris frontend is a **Vue 3 + Vite** pnpm monorepo (`admin`, `portal`, `landing` + shared `core`/`theme`). It uses a compact, premium SaaS/dashboard style optimized for VPN operations.

---

## Product mood

- Professional network operations, not consumer entertainment.
- Dense but readable admin workflows.
- Dark command-center surface with cyan/blue signal accents.
- Clear operational status language: online, active, limited, expired, disabled.

## Design rules

- Keep primary actions obvious and close to the user's current task.
- Prefer text, status pills, and simple glyphs over decorative icons for critical controls.
- Tables must stay compact, scannable, and responsive.
- Every form control needs a visible focus state and clear error state.
- Empty states should explain the next action.
- Respect reduced motion (`prefers-reduced-motion`).

---

## Design-system layering

Styles load in a deliberate cascade order (see `web/admin/src/main.ts`):

```
1. @koris/styles/reset.css          reset
2. @koris/styles/tokens.css         🎛️ design tokens (colors, spacing, type, radii, shadows)
3. @koris/styles/transitions.css    motion primitives
4. @koris/styles/utilities.css      utility classes
5. @koris/styles/rtl.css            RTL support
6. ./style.css                      app-local base
7. @koris/theme/styles/components.css   shared component styles
8. ./styles/micro-interactions.css
9. @koris/theme/styles/polish.css       fine polish
10. @koris/theme/styles/overhaul.css    ⭐ cross-cutting UI/UX overhaul (loads LAST)
```

Because everything is driven by CSS custom properties, **restyling a token restyles every tab** — no per-view edits required.

---

## Tokens

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

---

## The overhaul layer ⭐

`web/theme/styles/overhaul.css` is a single cross-cutting polish pass applied on top of the tokens. It loads last so it wins the cascade without touching any of the ~50 individual views. It covers:

- 🃏 **Elevation & depth** — softer, layered card/panel/modal shadows with hairline borders
- 📋 **Data tables** — sticky, blurred headers; denser rows; hover highlight; uppercase column labels
- 🔘 **Buttons** — hover lift, press feedback, gradient primary with brand glow
- ⌨️ **Inputs** — consistent radius, hover/focus states, 3px focus ring
- 🧭 **Sidebar/nav** — clear active state with a gradient accent bar
- 📌 **Page headers** — sticky, blurred, layered
- 🎯 **Focus rings** — accessible `:focus-visible` outlines for keyboard users
- 🖱️ **Scrollbars** — slim, theme-matched (WebKit + Firefox)
- ♿ **Reduced motion** — respects `prefers-reduced-motion`

It automatically adapts to all **UI themes** and **light/dark** modes because it references the same tokens each theme overrides.

---

## Themes

Selectable per-user in the admin UI:

| Theme | Vibe |
|-------|------|
| **Default** | Dark command-center, cyan/blue accents |
| **Kiro** | Teal/cyan + purple, medium radius |
| **GitHub** | GitHub palette, large radius, roomy |
| **Soft-Dark** | Warm, low-contrast, soft glow |
| **Corporate** | Clean, light, tight |
| **Midnight** | Sharp edges, strong dark shadows, dense |

Each theme overrides not just color but radius, shadow, typography scale, and spacing (`web/core/styles/tokens.css`). Light/dark/system modes toggle via `data-theme`.

### Adding or tweaking a theme

1. Add a `[data-ui-theme="yourtheme"]` block to `tokens.css` overriding the tokens you care about.
2. Register it in the theme picker.
3. Optionally add a full skin under `web/themes/<name>/`.

Because the overhaul layer is token-based, new themes inherit all the polish for free. ✨

---

## Shared components (`@koris/theme`)

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
| **ConfirmDialog** | admin: Customers (`useConfirm`) |
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

## Composables (`@koris/core`)

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

## Building the frontend

```bash
make frontend        # build admin + portal + landing
pnpm --filter admin build
pnpm --filter portal build
```

Requirements: Node 20+, pnpm 9+.
