export interface CommunityCommentGroup {
  root: GalgameCommunityComment
  replies: GalgameCommunityComment[]
}

const PAGE_LIMIT = 30

export const useCommunityCommentList = async (
  target: CommunityCommentTarget
) => {
  const surface = communityCommentSurface(target)

  const { id: viewerId } = usePersistUserStore()
  const config = useRuntimeConfig()

  const posts = ref<GalgameCommunityComment[]>([])
  const total = ref(0)
  const threadId = ref(0)
  const subscription = ref<CommunityThreadState | null>(null)
  const nextCursor = ref('')
  const seeded = ref(false)
  const loadingMore = ref(false)
  const locked = ref(false)

  const { data, status } = await useKunFetch<GalgameCommunityCommentPage>(
    surface.listUrl,
    {
      lazy: true,
      method: 'GET',
      query: { ...surface.addressQuery, limit: PAGE_LIMIT }
    }
  )

  const seedFrom = (page: GalgameCommunityCommentPage) => {
    posts.value = [...page.posts]
    total.value = page.total
    threadId.value = page.thread_id
    nextCursor.value = page.next_cursor
    locked.value = page.locked
    seeded.value = true
    reportRead()
  }

  // The read receipt is a POST the reader makes, never something inferred from
  // the GET above: the community service refuses to treat a read face as a
  // write, and it answers with the viewer's subscription so the toggle has a
  // state to render. It creates nothing for a wall the viewer never wrote on.
  //
  // Raw $fetch, not kunFetch: nobody asked for this request, so a community
  // service that blinks must not throw 「网络请求失败，请稍后重试」 at a reader
  // who only opened a page.
  const reportRead = async () => {
    if (!viewerId || !threadId.value || import.meta.server) {
      return
    }
    try {
      const resp = await $fetch<{ code: number; data?: CommunityThreadState }>(
        `${config.public.apiBaseUrl}/api/community/thread/${threadId.value}/read`,
        { method: 'POST', credentials: 'include' }
      )
      if (resp?.code === 0 && resp.data) {
        subscription.value = resp.data
      }
    } catch {
      // A missing receipt costs the reader nothing but an unread badge.
    }
  }

  const setLevel = async (level: CommunityNotificationLevel) => {
    if (!viewerId) {
      useAuthModal().open()
      return
    }
    if (!threadId.value) {
      return
    }
    const state = await kunFetch<CommunityThreadState>(
      `/community/thread/${threadId.value}/notification`,
      { method: 'POST', body: { level } }
    )
    if (state) {
      subscription.value = state
    }
  }

  if (data.value && !seeded.value) {
    seedFrom(data.value)
  }
  watch(data, (page) => {
    if (page && !seeded.value) {
      seedFrom(page)
    }
  })

  const hasMore = computed(() => nextCursor.value !== '')

  const loadMore = async () => {
    if (!hasMore.value || loadingMore.value) {
      return
    }
    loadingMore.value = true
    const page = await kunFetch<GalgameCommunityCommentPage>(surface.listUrl, {
      method: 'GET',
      query: {
        ...surface.addressQuery,
        cursor: nextCursor.value,
        limit: PAGE_LIMIT
      }
    })
    loadingMore.value = false
    if (page) {
      const seen = new Set(posts.value.map((p) => p.id))
      posts.value = [
        ...posts.value,
        ...page.posts.filter((p) => !seen.has(p.id))
      ]
      nextCursor.value = page.next_cursor
      total.value = page.total
    }
  }

  const groups = computed<CommunityCommentGroup[]>(() => {
    const list: CommunityCommentGroup[] = []
    const byRootId = new Map<number, CommunityCommentGroup>()
    for (const p of posts.value) {
      if (surface.isFlat || p.root_comment_id == null) {
        const group: CommunityCommentGroup = { root: p, replies: [] }
        byRootId.set(p.id, group)
        list.push(group)
      } else {
        const owner = byRootId.get(p.root_comment_id)
        if (owner) {
          owner.replies.push(p)
        } else {
          list.push({ root: p, replies: [] })
        }
      }
    }
    return list
  })

  const isEmpty = computed(
    () => seeded.value && !locked.value && total.value === 0
  )

  const handleNewComment = (post: GalgameCommunityComment) => {
    if (posts.value.some((p) => p.id === post.id)) {
      return
    }
    posts.value = [...posts.value, post]
    total.value += 1
    // Writing subscribes the author upstream, and on an empty wall this comment
    // is also what created the thread — so the control only becomes real here.
    if (!threadId.value && post.thread_id) {
      threadId.value = post.thread_id
    }
    reportRead()
  }

  const handleUpdated = (updated: GalgameCommunityComment) => {
    posts.value = posts.value.map((p) =>
      p.id === updated.id
        ? { ...updated, target_user: updated.target_user ?? p.target_user }
        : p
    )
  }

  const handleTombstoned = (postId: number) => {
    posts.value = posts.value.map((p) =>
      p.id === postId
        ? { ...p, deleted: true, held: false, content: '', content_html: '' }
        : p
    )
  }

  const scrollToPost = (postId: number) => {
    nextTick(() => {
      setTimeout(() => {
        const el = document.getElementById(`${surface.anchorPrefix}-${postId}`)
        if (!el) {
          return
        }
        el.scrollIntoView({ behavior: 'smooth', block: 'center' })
        const ring = [
          'outline-2',
          'outline-offset-2',
          'outline-primary',
          'rounded-lg'
        ]
        el.classList.add(...ring)
        setTimeout(() => el.classList.remove(...ring), 3000)
      }, 200)
    })
  }

  return {
    surface,
    posts,
    total,
    threadId,
    subscription,
    setLevel,
    status,
    seeded,
    locked,
    hasMore,
    loadingMore,
    groups,
    isEmpty,
    loadMore,
    handleNewComment,
    handleUpdated,
    handleTombstoned,
    scrollToPost
  }
}
