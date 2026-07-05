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
export { default as KButton }        from './components/KButton.vue'
export { default as KInput }         from './components/KInput.vue'
export { default as KTextarea }      from './components/KTextarea.vue'
export { default as KSelect }        from './components/KSelect.vue'
export { default as KFormField }     from './components/KFormField.vue'
export { default as KDataTable }     from './components/KDataTable.vue'
export { default as KPagination }    from './components/KPagination.vue'
export { default as KModal }         from './components/KModal.vue'
export { default as KDrawer }        from './components/KDrawer.vue'
export { default as KSlideOver }     from './components/KSlideOver.vue'
export { default as KConfirmDialog } from './components/KConfirmDialog.vue'
export { default as KToast }         from './components/KToast.vue'
export { default as KAlert }         from './components/KAlert.vue'
export { default as KTabs }          from './components/KTabs.vue'
export { default as KBreadcrumb }    from './components/KBreadcrumb.vue'
export { default as KStatusPill }    from './components/KStatusPill.vue'
export { default as KUsageBar }      from './components/KUsageBar.vue'
export { default as KExpiryChips }   from './components/KExpiryChips.vue'
export { default as KExpandableRow } from './components/KExpandableRow.vue'
export { default as KThreeDotMenu }  from './components/KThreeDotMenu.vue'
export { default as KEmptyState }    from './components/KEmptyState.vue'
export { default as KSkeleton }      from './components/KSkeleton.vue'
export { default as KAvatar }        from './components/KAvatar.vue'
export { default as KChart }         from './components/KChart.vue'
export { default as HealthDot }      from './components/HealthDot.vue'
export { default as PageTransition } from './components/PageTransition.vue'
export { default as SkeletonLoader } from './components/SkeletonLoader.vue'
export { default as SortableList }   from './components/SortableList.vue'
export { default as ThemeEditor }    from './components/ThemeEditor.vue'
export { default as ThemeToggle }    from './components/ThemeToggle.vue'
