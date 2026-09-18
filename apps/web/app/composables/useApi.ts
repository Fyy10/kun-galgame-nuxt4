import {
  createApiClient,
  sessionCookie,
  type ApiClient
} from '#shared/utils/api/client'
import { settle, type ClientProblem } from '#shared/utils/api/problem'

let browserClient: ApiClient | undefined

export const useApiClient = (): ApiClient => {
  if (import.meta.server) {
    return createApiClient({
      origin: useRuntimeConfig().apiBaseUrl,
      cookie: sessionCookie(useRequestHeaders(['cookie']).cookie),
      timeoutMs: 10000
    })
  }
  browserClient ??= createApiClient({
    origin: useRuntimeConfig().public.apiBaseUrl
  })
  return browserClient
}

type UseApiOptions = {
  lazy?: boolean
  server?: boolean
  immediate?: boolean
}

type UseApiReturn<T> = {
  data: ComputedRef<T | undefined>
  problem: ComputedRef<ClientProblem | null>
  status: ReturnType<typeof useAsyncData>['status']
  refresh: ReturnType<typeof useAsyncData>['refresh']
}

export const useApi = <T>(
  key: MaybeRefOrGetter<string>,
  call: (
    api: ApiClient,
    ctx: { signal: AbortSignal }
  ) => Promise<{ data?: T; error?: unknown; response: Response }>,
  options?: UseApiOptions
): UseApiReturn<T> & Promise<UseApiReturn<T>> => {
  const api = useApiClient()
  const asyncData = useAsyncData(
    () => `api:${toValue(key)}`,
    (_app, { signal }) => settle(call(api, { signal })),
    options
  )

  const result: UseApiReturn<T> = {
    data: computed(() =>
      asyncData.data.value?.ok ? asyncData.data.value.data : undefined
    ),
    problem: computed(() =>
      asyncData.data.value && !asyncData.data.value.ok
        ? asyncData.data.value.problem
        : null
    ),
    status: asyncData.status,
    refresh: asyncData.refresh
  }

  return Object.assign(
    Promise.resolve(asyncData).then(() => result),
    result
  )
}
