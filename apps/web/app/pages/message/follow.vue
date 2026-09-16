<script setup lang="ts">
definePageMeta({
  middleware: 'auth'
})

useKunDisableSeo('关注的评论区')

const items = ref<CommunityFollowItem[]>([])
const nextCursor = ref('')
const loadingMore = ref(false)

const { data, status } = await useKunFetch<CommunityFollowList>(
  '/community/following',
  { query: { limit: 30 } }
)

watchEffect(() => {
  if (data.value) {
    items.value = [...data.value.items]
    nextCursor.value = data.value.next_cursor
  }
})

const loadMore = async () => {
  if (!nextCursor.value || loadingMore.value) {
    return
  }
  loadingMore.value = true
  const page = await kunFetch<CommunityFollowList>('/community/following', {
    method: 'GET',
    query: { cursor: nextCursor.value, limit: 30 }
  })
  loadingMore.value = false
  if (!page) {
    return
  }
  const seen = new Set(
    items.value.map((item) => `${item.anchor_kind}:${item.anchor_id}`)
  )
  items.value = [
    ...items.value,
    ...page.items.filter(
      (item) => !seen.has(`${item.anchor_kind}:${item.anchor_id}`)
    )
  ]
  nextCursor.value = page.next_cursor
}

const unfollow = async (item: CommunityFollowItem) => {
  const state = await kunFetch<CommunityWallState>('/community/wall/follow', {
    method: 'POST',
    body: {
      anchor_kind: item.anchor_kind,
      anchor_id: item.anchor_id,
      following: false
    }
  })
  if (state) {
    items.value = items.value.filter(
      (row) =>
        !(
          row.anchor_kind === item.anchor_kind &&
          row.anchor_id === item.anchor_id
        )
    )
  }
}
</script>

<template>
  <div class="flex w-full flex-col space-y-3">
    <header class="flex items-center gap-2">
      <KunButton size="lg" :is-icon-only="true" variant="light" href="/message">
        <KunIcon name="lucide:chevron-left" />
      </KunButton>
      <h2 class="text-lg">关注的评论区</h2>
    </header>

    <KunDivider />

    <KunLoading v-if="status === 'pending' && !items.length" />

    <div v-else-if="items.length" class="space-y-2">
      <KunCard
        v-for="item in items"
        :key="`${item.anchor_kind}:${item.anchor_id}`"
        padding="sm"
        :is-hoverable="true"
      >
        <div class="flex w-full items-center gap-2">
          <KunLink
            color="default"
            underline="none"
            :to="item.link"
            class-name="min-w-0 flex-1 truncate font-medium"
          >
            {{ item.title }}
          </KunLink>
          <KunChip size="sm" color="default" class-name="shrink-0">
            {{ item.label }}
          </KunChip>
          <KunButton
            size="sm"
            variant="light"
            color="danger"
            @click="unfollow(item)"
          >
            取消关注
          </KunButton>
        </div>
      </KunCard>
    </div>

    <KunNull v-else description="还没有关注任何评论区, 杂鱼~♡" />

    <KunButton
      v-if="nextCursor"
      variant="light"
      color="primary"
      full-width
      :loading="loadingMore"
      @click="loadMore"
    >
      <KunIcon name="lucide:chevron-down" />
      加载更多
    </KunButton>
  </div>
</template>
