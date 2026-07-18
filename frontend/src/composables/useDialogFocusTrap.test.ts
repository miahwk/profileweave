import { describe, expect, it } from 'vitest'
import { nextFocusIndex } from './useDialogFocusTrap'

describe('dialog focus trap', () => {
  it('wraps forward focus and enters at the first control', () => {
    expect(nextFocusIndex(-1, 3, false)).toBe(0)
    expect(nextFocusIndex(0, 3, false)).toBe(1)
    expect(nextFocusIndex(2, 3, false)).toBe(0)
  })

  it('wraps backward focus and handles an empty dialog', () => {
    expect(nextFocusIndex(-1, 3, true)).toBe(2)
    expect(nextFocusIndex(0, 3, true)).toBe(2)
    expect(nextFocusIndex(0, 0, false)).toBeNull()
  })
})
