import Cookies from 'js-cookie'
import { useMessage } from '../../app/composables/useMessage'
import { usePersistUserStore } from '../../app/store/modules/user'
import { navigateTo } from 'nuxt/app'
import type { FetchResponse } from 'ofetch'

interface KunErrorData {
  data?: {
    code: number
    message: string
  }
  stack: string[]
  statusCode: number
  statusMessage: string
}

interface ResponseMap {
  blob: Blob
  text: string
  arrayBuffer: ArrayBuffer
  stream: ReadableStream<Uint8Array>
}
type ResponseType = keyof ResponseMap | 'json'

type KunOnResponseContext<
  R extends ResponseType,
  JsonType = unknown
> = R extends keyof ResponseMap ? ResponseMap[R] : JsonType

export const onResponse = async <R extends ResponseType>(
  context: KunOnResponseContext<R>
) => {
  const { response } = context as { response: FetchResponse<ResponseType> }
  const errorData = response?._data as KunErrorData | undefined

  if (!errorData) {
    useMessage('网络请求失败，请稍后重试', 'error')
    return
  }
  if (!errorData.data) {
    return
  }

  const { code, message } = errorData.data

  if (code === 205) {
    const navigateCookie = Cookies.get('kungalgame-is-navigate-to-login')
    const userStore = usePersistUserStore()

    if (!navigateCookie && userStore.id) {
      userStore.resetUser()

      useMessage(message || '登录已失效，请重新登录', 'error', 7777)

      Cookies.set('kungalgame-is-navigate-to-login', 'navigated', {
        expires: 1 / 1440
      })

      await navigateTo('/login')
      return
    }
  }

  if (code === 233) {
    useMessage(message, 'error')
  }
}

export const kungalgameResponseHandler = { onResponse }
