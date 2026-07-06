# Example theme

This is a **starter kit** for creating a custom Koris theme.

## What a theme is

A theme is any package that:
1. Exports a default `ThemeManifest` (see `manifest.ts`)
2. Provides Vue components for the visual slots the host app requests
3. Provides CSS files that override the default component styling

Koris apps resolve visual components through the Vite alias `@koris/theme`.
Point that alias at a different directory to swap themes without touching
app code.

## Try this theme

From `web/`:

```bash
# Temporarily point apps at this theme (from web/):
sed -i "s|resolve(__dirname, '../theme')|resolve(__dirname, '../themes/example')|" \
    admin/vite.config.ts portal/vite.config.ts

# Rebuild
pnpm --filter admin build && pnpm --filter portal build
```

Revert:
```bash
sed -i "s|resolve(__dirname, '../themes/example')|resolve(__dirname, '../theme')|" \
    admin/vite.config.ts portal/vite.config.ts
```

## What this theme changes

Compared to `@koris/theme` (the default), this example theme:
- Overrides `styles/tokens.css` with a lighter, more relaxed palette
- Overrides `styles/polish.css` — no aurora backdrop, softer shadows
- Reuses every component from `@koris/theme` (only CSS changes)

That's the minimum-effort theme: replace CSS, inherit components.

## Building a full custom theme

To override a component, copy it from `web/theme/components/YourComponent.vue`
into `web/themes/your-theme/components/YourComponent.vue`, then add it to the
`components: {}` block in `manifest.ts`.

Missing slots fall back to the default theme — you only override what you want.
