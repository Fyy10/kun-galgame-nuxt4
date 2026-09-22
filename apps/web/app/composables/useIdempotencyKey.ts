export const useIdempotencyKey = () => {
  const key = ref<string | undefined>()
  const fingerprint = ref<string | undefined>()

  const take = (payload: unknown): string => {
    const next = JSON.stringify(payload)
    if (key.value === undefined || fingerprint.value !== next) {
      key.value = crypto.randomUUID()
      fingerprint.value = next
    }
    return key.value
  }

  const clear = () => {
    key.value = undefined
    fingerprint.value = undefined
  }

  return { take, clear }
}
