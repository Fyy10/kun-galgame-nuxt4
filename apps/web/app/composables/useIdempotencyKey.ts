export const useIdempotencyKey = () => {
  const key = ref<string | undefined>()
  const fingerprint = ref<string | undefined>()

  // The server fingerprints method + path + body; this fingerprinted the body
  // alone. <LazyTopicUpvoteModal /> is mounted once in app.vue, so the key
  // outlives the route: a failed upvote on one topic followed by the same note
  // on another reused the key with a different path and answered
  // 409 IDEMPOTENCY_KEY_REUSED for the next 24 hours.
  const take = (target: string, payload: unknown): string => {
    const next = `${target}\n${JSON.stringify(payload)}`
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
