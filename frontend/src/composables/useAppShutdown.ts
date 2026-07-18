import { ref } from 'vue'
import { ApiError, api as defaultApi, type HealthResponse, type ShutdownResponse } from '@/api/client'

interface ShutdownService {
  shutdown(): Promise<ShutdownResponse>
  getHealth(): Promise<HealthResponse>
}

interface ShutdownOptions {
  pollIntervalMs?: number
  timeoutMs?: number
  now?: () => number
  sleep?: (milliseconds: number) => Promise<void>
}

const shutdownTimeoutMessage = '本地服务在等待时间内仍可访问。退出可能被运行中的会话阻止，请稍后重试。'

function messageFrom(error: unknown): string {
  if (error instanceof ApiError) return error.message
  return error instanceof Error ? error.message : '退出请求失败，请稍后重试'
}

export function useAppShutdown(service: ShutdownService = defaultApi, options: ShutdownOptions = {}) {
  const confirmOpen = ref(false)
  const busy = ref(false)
  const error = ref('')
  const stopped = ref(false)
  const pollIntervalMs = options.pollIntervalMs ?? 250
  const timeoutMs = options.timeoutMs ?? 15_000
  const now = options.now ?? Date.now
  const sleep = options.sleep ?? ((milliseconds) => new Promise<void>((resolve) => {
    window.setTimeout(resolve, milliseconds)
  }))

  function request() {
    error.value = ''
    confirmOpen.value = true
  }

  function cancel() {
    if (busy.value) return
    confirmOpen.value = false
    error.value = ''
  }

  async function waitUntilOffline(): Promise<boolean> {
    const startedAt = now()
    while (now() - startedAt <= timeoutMs) {
      try {
        await service.getHealth()
      } catch (cause) {
        // An HTTP error proves a server is still answering; a transport failure
        // confirms the loopback service is no longer reachable.
        if (!(cause instanceof ApiError)) return true
      }
      if (now() - startedAt >= timeoutMs) break
      await sleep(pollIntervalMs)
    }
    return false
  }

  async function confirm(): Promise<boolean> {
    if (busy.value) return false
    busy.value = true
    error.value = ''
    try {
      await service.shutdown()
      if (await waitUntilOffline()) {
        confirmOpen.value = false
        stopped.value = true
        return true
      }
      error.value = shutdownTimeoutMessage
      return false
    } catch (cause) {
      error.value = messageFrom(cause)
      return false
    } finally {
      busy.value = false
    }
  }

  return { confirmOpen, busy, error, stopped, request, cancel, confirm }
}
