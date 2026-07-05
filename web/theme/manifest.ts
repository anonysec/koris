/**
 * Koris theme manifest.
 *
 * A theme package (this one, or a fan-made variant) must export a default
 * manifest matching this shape. The host app reads the manifest to know:
 *   - which CSS files to load, in what order
 *   - which components map to which slots (KButton, KInput, etc.)
 *   - what the theme is called and who made it
 *
 * To create a custom theme:
 *   1. Copy this whole /theme directory to /themes/my-theme
 *   2. Change `id`, `name`, `author` below
 *   3. Override any CSS file or component you want
 *   4. Add `"@koris/theme": "workspace:themes/my-theme"` to your app's package.json
 */

import type { Component } from 'vue'

// ─── Component slot registry ─────────────────────────────────────────
// Every visual component the host app can request. Theme authors override
// any subset; missing slots fall back to @koris/theme (this package).
import KButton         from './components/KButton.vue'
import KInput          from './components/KInput.vue'
import KTextarea       from './components/KTextarea.vue'
import KSelect         from './components/KSelect.vue'
import KFormField      from './components/KFormField.vue'
import KDataTable      from './components/KDataTable.vue'
import KPagination     from './components/KPagination.vue'
import KModal          from './components/KModal.vue'
import KDrawer        from './components/KDrawer.vue'
import KSlideOver      from './components/KSlideOver.vue'
import KConfirmDialog  from './components/KConfirmDialog.vue'
import KToast          from './components/KToast.vue'
import KAlert          from './components/KAlert.vue'
import KTabs           from './components/KTabs.vue'
import KBreadcrumb     from './components/KBreadcrumb.vue'
import KStatusPill     from './components/KStatusPill.vue'
import KUsageBar       from './components/KUsageBar.vue'
import KExpiryChips    from './components/KExpiryChips.vue'
import KExpandableRow  from './components/KExpandableRow.vue'
import KThreeDotMenu   from './components/KThreeDotMenu.vue'
import KEmptyState     from './components/KEmptyState.vue'
import KSkeleton       from './components/KSkeleton.vue'
import KAvatar         from './components/KAvatar.vue'
import KChart          from './components/KChart.vue'
import HealthDot       from './components/HealthDot.vue'
import PageTransition  from './components/PageTransition.vue'
import SkeletonLoader  from './components/SkeletonLoader.vue'
import SortableList    from './components/SortableList.vue'
import ThemeEditor     from './components/ThemeEditor.vue'
import ThemeToggle     from './components/ThemeToggle.vue'

export interface ThemeManifest {
  /** Unique identifier, lowercase-kebab. */
  id: string
  /** Human-readable name shown in the theme picker. */
  name: string
  /** Author / attribution shown in the theme picker. */
  author: string
  /** SemVer of the theme package. */
  version: string
  /**
   * CSS files to load in order.
   * Framework CSS from @koris/core is loaded first automatically by the host app;
   * these files come after it and win by cascade order.
   */
  css: string[]
  /**
   * Component map. Slot name → Vue component.
   * Host app resolves visual components through this map.
   */
  components: Record<string, Component>
}

export const manifest: ThemeManifest = {
  id: 'koris-default',
  name: 'Koris Default',
  author: 'Koris',
  version: '1.0.0',
  css: [
    '@koris/theme/styles/components.css',
    '@koris/theme/styles/polish.css',
  ],
  components: {
    KButton, KInput, KTextarea, KSelect, KFormField,
    KDataTable, KPagination,
    KModal, KDrawer, KSlideOver, KConfirmDialog,
    KToast, KAlert, KTabs, KBreadcrumb,
    KStatusPill, KUsageBar, KExpiryChips,
    KExpandableRow, KThreeDotMenu,
    KEmptyState, KSkeleton, KAvatar,
    KChart, HealthDot,
    PageTransition, SkeletonLoader, SortableList,
    ThemeEditor, ThemeToggle,
  },
}

export default manifest
