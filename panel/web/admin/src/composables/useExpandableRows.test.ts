import { describe, it, expect } from 'vitest'
import { useExpandableRows } from './useExpandableRows'

/**
 * Unit tests for useExpandableRows composable.
 *
 * **Validates: Requirements 6.2, 6.4, 6.6**
 */

describe('useExpandableRows', () => {
  it('starts with no expanded rows', () => {
    const { expandedIds, isExpanded } = useExpandableRows()

    expect(expandedIds.value.size).toBe(0)
    expect(isExpanded(1)).toBe(false)
    expect(isExpanded(99)).toBe(false)
  })

  it('toggle expands a row', () => {
    const { toggle, isExpanded } = useExpandableRows()

    toggle(5)

    expect(isExpanded(5)).toBe(true)
  })

  it('toggle collapses an already expanded row', () => {
    const { toggle, isExpanded } = useExpandableRows()

    toggle(3)
    expect(isExpanded(3)).toBe(true)

    toggle(3)
    expect(isExpanded(3)).toBe(false)
  })

  it('allows multiple rows expanded simultaneously', () => {
    const { toggle, isExpanded } = useExpandableRows()

    toggle(1)
    toggle(2)
    toggle(3)

    expect(isExpanded(1)).toBe(true)
    expect(isExpanded(2)).toBe(true)
    expect(isExpanded(3)).toBe(true)
  })

  it('toggling one row does not affect others', () => {
    const { toggle, isExpanded } = useExpandableRows()

    toggle(10)
    toggle(20)

    // Collapse row 10
    toggle(10)

    expect(isExpanded(10)).toBe(false)
    expect(isExpanded(20)).toBe(true)
  })

  it('collapseAll collapses all expanded rows', () => {
    const { toggle, isExpanded, collapseAll, expandedIds } = useExpandableRows()

    toggle(1)
    toggle(2)
    toggle(3)
    expect(expandedIds.value.size).toBe(3)

    collapseAll()

    expect(expandedIds.value.size).toBe(0)
    expect(isExpanded(1)).toBe(false)
    expect(isExpanded(2)).toBe(false)
    expect(isExpanded(3)).toBe(false)
  })

  it('collapseAll on empty state is a no-op', () => {
    const { collapseAll, expandedIds } = useExpandableRows()

    collapseAll()

    expect(expandedIds.value.size).toBe(0)
  })

  it('expandedIds ref is reactive when toggling', () => {
    const { toggle, expandedIds } = useExpandableRows()

    const initialSize = expandedIds.value.size
    toggle(42)

    expect(expandedIds.value.size).toBe(initialSize + 1)
    expect(expandedIds.value.has(42)).toBe(true)
  })
})
