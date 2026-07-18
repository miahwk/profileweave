import { nextTick, onBeforeUnmount, watch, type Ref } from 'vue'

const focusableSelector = [
  'a[href]',
  'button:not([disabled])',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(',')

export function nextFocusIndex(current: number, count: number, backwards: boolean): number | null {
  if (count < 1) return null
  if (current < 0) return backwards ? count - 1 : 0
  return (current + (backwards ? count - 1 : 1)) % count
}

export function useDialogFocusTrap(
  isOpen: () => boolean,
  panel: Ref<HTMLElement | null>,
  initialFocus: Ref<HTMLElement | null>,
  close: () => void,
) {
  let previousFocus: HTMLElement | null = null

  function focusableElements() {
    return panel.value ? [...panel.value.querySelectorAll<HTMLElement>(focusableSelector)] : []
  }

  function handleKeydown(event: KeyboardEvent) {
    if (!isOpen()) return
    if (event.key === 'Escape') {
      event.preventDefault()
      close()
      return
    }
    if (event.key !== 'Tab') return
    const elements = focusableElements()
    const current = elements.indexOf(document.activeElement as HTMLElement)
    const next = nextFocusIndex(current, elements.length, event.shiftKey)
    event.preventDefault()
    if (next === null) panel.value?.focus()
    else elements[next]?.focus()
  }

  function removeListener() {
    if (typeof document !== 'undefined') document.removeEventListener('keydown', handleKeydown, true)
  }

  watch(isOpen, async (open) => {
    if (typeof document === 'undefined') return
    if (open) {
      previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
      document.addEventListener('keydown', handleKeydown, true)
      await nextTick()
      if (isOpen()) (initialFocus.value ?? panel.value)?.focus()
      return
    }
    removeListener()
    await nextTick()
    if (!isOpen() && previousFocus?.isConnected) previousFocus.focus()
    previousFocus = null
  })

  onBeforeUnmount(() => {
    removeListener()
    if (previousFocus?.isConnected) previousFocus.focus()
  })
}
