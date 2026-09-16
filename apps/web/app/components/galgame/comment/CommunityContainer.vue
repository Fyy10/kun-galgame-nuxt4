<script setup lang="ts">
const emit = defineEmits<{
  'update:loading': [boolean]
}>()

const route = useRoute()
const config = useRuntimeConfig()
const gid = parseInt((route.params as { gid: string }).gid)

const target: CommunityCommentTarget = { kind: 'galgame', galgameId: gid }

const galgame = inject<GalgameDetail>('galgame')

const {
  posts,
  status,
  following,
  setFollowing,
  seeded,
  groups,
  isEmpty,
  hasMore,
  loadingMore,
  loadMore,
  handleNewComment,
  handleUpdated,
  handleTombstoned,
  scrollToPost
} = await useCommunityCommentList(target)

watchEffect(() => emit('update:loading', status.value === 'pending'))

const showEmpty = computed(
  () => isEmpty.value && galgame?.is_on_forum !== false
)

const locateLegacy = async (legacyId: number): Promise<number | null> => {
  try {
    const resp = await $fetch<{ code: number; data?: { post_id: number } }>(
      `${config.public.apiBaseUrl}/api/galgame/${gid}/comments/locate`,
      { query: { legacy_id: legacyId }, credentials: 'include' }
    )
    return resp && resp.code === 0 && resp.data ? resp.data.post_id : null
  } catch {
    return null
  }
}

const ensureLoadedAndScroll = async (postId: number) => {
  let guard = 0
  while (
    !posts.value.some((p) => p.id === postId) &&
    hasMore.value &&
    guard < 50
  ) {
    await loadMore()
    guard += 1
  }
  if (posts.value.some((p) => p.id === postId)) {
    scrollToPost(postId)
  }
}

const resolveDeepLink = async () => {
  const commentParam = Number(route.query.comment) || 0
  const hasThreadParam = route.query.thread != null
  const hashMatch = (route.hash || '').match(/^#galgame-comment-(\d+)$/)

  if (commentParam) {
    if (hasThreadParam) {
      const postId = await locateLegacy(commentParam)
      if (postId) {
        await ensureLoadedAndScroll(postId)
      }
      return
    }
    await ensureLoadedAndScroll(commentParam)
    return
  }

  if (hashMatch) {
    const raw = Number(hashMatch[1])
    await ensureLoadedAndScroll(raw)
    if (posts.value.some((p) => p.id === raw)) {
      return
    }
    const mapped = await locateLegacy(raw)
    if (mapped) {
      await ensureLoadedAndScroll(mapped)
    }
  }
}

onMounted(() => {
  if (seeded.value) {
    resolveDeepLink()
    return
  }
  const stop = watch(seeded, (ready) => {
    if (ready) {
      stop()
      resolveDeepLink()
    }
  })
})
</script>

<template>
  <div class="space-y-5">
    <KunHeader name="游戏评论" scale="h2">
      <template #endContent>
        <div class="flex flex-wrap items-center gap-3">
          <KunLink size="sm" to="/topic/1482">
            Galgame 评论注意事项, 资源失效, 解压密码错误等问题反馈
          </KunLink>
          <CommentCommunitySubscribe
            :following="following"
            :submit="setFollowing"
          />
        </div>
      </template>
    </KunHeader>

    <CommentCommunityComposer :target="target" @submitted="handleNewComment" />

    <KunLoading v-if="status === 'pending'" />

    <KunNull
      v-else-if="showEmpty"
      description="没人评论, 是没人要这个 Galgame 的小只可爱软萌女孩子了吗, 呜呜呜呜呜呜！！"
    />

    <div v-else-if="groups.length" class="space-y-8">
      <CommentCommunityRow
        v-for="group in groups"
        :key="group.root.id"
        :comment="group.root"
        :replies="group.replies"
        :target="target"
        :depth="0"
        @reply-added="handleNewComment"
        @updated="handleUpdated"
        @tombstoned="handleTombstoned"
      />
    </div>

    <KunButton
      v-if="hasMore"
      variant="light"
      color="primary"
      full-width
      :loading="loadingMore"
      @click="loadMore"
    >
      <KunIcon name="lucide:chevron-down" />
      加载更多评论
    </KunButton>

    <CommentCommunityFlagModal />
  </div>
</template>
