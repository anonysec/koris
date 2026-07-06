/**
 * Koris theme manifest.
 *
 * A theme package (this one, or a fan-made variant) must export a default
 * manifest matching this shape. The host app reads the manifest to know:
 *   - which CSS files to load, in what order
 *   - which components map to which slots (Button, Input, etc.)
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
import Button         from './components/Button.vue'
import Input          from './components/Input.vue'
import Textarea       from './components/Textarea.vue'
import Select         from './components/Select.vue'
import FormField      from './components/FormField.vue'
import DataTable      from './components/DataTable.vue'
import Pagination     from './components/Pagination.vue'
import Modal          from './components/Modal.vue'
import Drawer        from './components/Drawer.vue'
import SlideOver      from './components/SlideOver.vue'
import ConfirmDialog  from './components/ConfirmDialog.vue'
import Toast          from './components/Toast.vue'
import Alert          from './components/Alert.vue'
import Tabs           from './components/Tabs.vue'
import Breadcrumb     from './components/Breadcrumb.vue'
import StatusPill     from './components/StatusPill.vue'
import UsageBar       from './components/UsageBar.vue'
import ExpiryChips    from './components/ExpiryChips.vue'
import KExpandableRow  from './components/KExpandableRow.vue'
import ThreeDotMenu   from './components/ThreeDotMenu.vue'
import EmptyState     from './components/EmptyState.vue'
import Skeleton       from './components/Skeleton.vue'
import Avatar         from './components/Avatar.vue'
import Chart          from './components/Chart.vue'
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
    Button, Input, Textarea, Select, FormField,
    DataTable, Pagination,
    Modal, Drawer, SlideOver, ConfirmDialog,
    Toast, Alert, Tabs, Breadcrumb,
    StatusPill, UsageBar, ExpiryChips,
    KExpandableRow, ThreeDotMenu,
    EmptyState, Skeleton, Avatar,
    Chart, HealthDot,
    PageTransition, SkeletonLoader, SortableList,
    ThemeEditor, ThemeToggle,
  },
}

export default manifest
