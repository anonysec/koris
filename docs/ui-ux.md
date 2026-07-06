# 🎨 UI / UX

The Koris frontend is a **Vue 3 + Vite** monorepo (pnpm workspaces) with a token-driven design system shared across every app and tab.

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

---

## Adding or tweaking a theme

1. Add a `[data-ui-theme="yourtheme"]` block to `tokens.css` overriding the tokens you care about.
2. Register it in the theme picker.
3. Optionally add a full skin under `web/themes/<name>/`.

Because the overhaul layer is token-based, new themes inherit all the polish for free. ✨

---

## Building the frontend

```bash
make frontend        # build admin + portal + landing
pnpm --filter admin build
pnpm --filter portal build
```

Requirements: Node 20+, pnpm 9+.
