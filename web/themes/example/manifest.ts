/**
 * Example custom Koris theme.
 *
 * This theme inherits every component from @koris/theme and only overrides
 * the CSS (see ./styles/tokens.css and ./styles/polish.css).
 *
 * To override a component, add it to the components map below and place a
 * .vue file next to this manifest under ./components/.
 */

import { manifest as base, type ThemeManifest } from '@koris/theme/manifest'

export const manifest: ThemeManifest = {
  ...base,
  id: 'koris-example',
  name: 'Example (Light)',
  author: 'Koris',
  version: '1.0.0',
  css: [
    // Order matters: our tokens override defaults, then our polish overrides.
    '@koris/theme-example/styles/tokens.css',
    '@koris/theme/styles/components.css',
    '@koris/theme-example/styles/polish.css',
  ],
  // Reuse the entire default component map. Override any individual slot
  // by importing your own and setting it here, e.g.:
  //   import Button from './components/Button.vue'
  //   components: { ...base.components, Button }
  components: base.components,
}

export default manifest
