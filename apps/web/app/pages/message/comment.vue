<script setup lang="ts">
definePageMeta({
  middleware: 'auth'
})

useKunDisableSeo('未读评论')

const items = ref<CommunityUnreadItem[]>([])
const total = ref(0)
const nextCursor = ref('')
const loadingMore = ref(false)

const { data, status } = await useKunFetch<CommunityUnreadResult>(
  '/community/unread',
  { query: { limit: 30 } }
)

watchEffect(() => {
  if (data.value) {
    items.value = [...data.value.items]
    total.value = data.value.total
    nextCursor.value = data.value.next_cursor
  }
})

const loadMore = async () => {
  if (!nextCursor.value || loadingMore.value) {
    return
  }
  loadingMore.value = true
  const page = await kunFetch<CommunityUnreadResult>('/community/unread', {
    method: 'GET',
    query: { cursor: nextCursor.value, limit: 30 }
  })
  loadingMore.value = false
  if (!page) {
    return
  }
  const seen = new Set(items.value.map((item) => item.thread_id))
  items.value = [
    ...items.value,
    ...page.items.filter((item) => !seen.has(item.thread_id))
  ]
  nextCursor.value = page.next_cursor
}
</script>

<template>
  <div class="flex w-full flex-col space-y-3">
    <header class="flex items-center gap-2">
      <KunButton size="lg" :is-icon-only="true" variant="light" href="/message">
        <KunIcon name="lucide:chevron-left" />
      </KunButton>
      <h2 class="text-lg">未读评论</h2>
      <KunChip v-if="total" color="primary" class-name="ml-auto">
        {{ total }}
      </KunChip>
    </header>

    <p class="text-default-500 text-sm">
      你评论过或关注过的评论区, 在你上次看过之后又有了新评论。
    </p>

    <KunDivider />

    <KunLoading v-if="status === 'pending' && !items.length" />

    <div v-else-if="items.length" class="space-y-2">
      <KunCard
        v-for="item in items"
        :key="item.thread_id"
        padding="sm"
        :is-hoverable="true"
      >
        <KunLink
          color="default"
          underline="none"
          :to="item.link"
          class-name="flex-col items-start w-full gap-1.5"
        >
          <div class="flex w-full items-center gap-2">
            <span class="min-w-0 flex-1 truncate font-medium">
              {{ item.title }}
            </span>
            <KunChip size="sm" color="default" class-name="shrink-0">
              {{ item.label }}
            </KunChip>
            <KunChip size="sm" color="primary" class-name="shrink-0">
              {{ item.unread_count }}
            </KunChip>
          </div>
          <div class="text-default-500 w-full text-xs">
            最新评论 <KunTime :time="item.last_posted_at" />
          </div>
        </KunLink>
      </KunCard>
    </div>

    <KunNull v-else description="没有未读的评论, 杂鱼~♡" />

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
