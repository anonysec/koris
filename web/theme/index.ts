/**
 * @koris/theme — default visual theme for Koris apps.
 *
 * Exports:
 *   - The theme manifest (see ./manifest.ts) describing CSS + components
 *   - Every visual component individually, so apps can also import directly
 *     when they don't need the manifest indirection
 *
 * To swap the theme, install a different package that exports the same shape
 * and set your Vite alias `@koris/theme` to point at it.
 */

export { manifest, default } from './manifest'
export type { ThemeManifest } from './manifest'

// Individual component re-exports for direct import.
// (Deep imports `@koris/theme/components/<name>` also work.)
export { default as Button }        from './components/Button.vue'
export { default as Input }         from './components/Input.vue'
export { default as Textarea }      from './components/Textarea.vue'
export { default as Select }        from './components/Select.vue'
export { default as FormField }     from './components/FormField.vue'
export { default as DataTable }     from './components/DataTable.vue'
export { default as Pagination }    from './components/Pagination.vue'
export { default as Modal }         from './components/Modal.vue'
export { default as Drawer }        from './components/Drawer.vue'
export { default as SlideOver }     from './components/SlideOver.vue'
export { default as ConfirmDialog } from './components/ConfirmDialog.vue'
export { default as Toast }         from './components/Toast.vue'
export { default as Alert }         from './components/Alert.vue'
export { default as Tabs }          from './components/Tabs.vue'
export { default as Breadcrumb }    from './components/Breadcrumb.vue'
export { default as StatusPill }    from './components/StatusPill.vue'
export { default as UsageBar }      from './components/UsageBar.vue'
export { default as ExpiryChips }   from './components/ExpiryChips.vue'
export { default as KExpandableRow } from './components/KExpandableRow.vue'
export { default as ThreeDotMenu }  from './components/ThreeDotMenu.vue'
export { default as EmptyState }    from './components/EmptyState.vue'
export { default as Skeleton }      from './components/Skeleton.vue'
export { default as Avatar }        from './components/Avatar.vue'
export { default as Chart }         from './components/Chart.vue'
export { default as HealthDot }      from './components/HealthDot.vue'
export { default as PageTransition } from './components/PageTransition.vue'
export { default as SkeletonLoader } from './components/SkeletonLoader.vue'
export { default as SortableList }   from './components/SortableList.vue'
export { default as ThemeEditor }    from './components/ThemeEditor.vue'
export { default as ThemeToggle }    from './components/ThemeToggle.vue'
