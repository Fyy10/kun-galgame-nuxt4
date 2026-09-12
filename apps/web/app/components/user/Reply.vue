<script setup lang="ts">
import {
  kunUserReplyNavItem,
  type KUN_USER_PAGE_REPLY_TYPE
} from '~/constants/user'

const props = defineProps<{
  userId: number
  type: (typeof KUN_USER_PAGE_REPLY_TYPE)[number]
}>()

const activeTab = ref(props.type)
const pageData = reactive({
  page: usePageQuery(),
  limit: 50,
  userId: props.userId,
  type: props.type
})

const { data, status } = await useKunFetch<{
  replies: UserReply[]
  total: number
}>(`/user/${props.userId}/replies`, { query: pageData })
</script>

<template>
  <div class="space-y-6">
    <KunTab
      :items="kunUserReplyNavItem(userId)"
      :model-value="activeTab"
      variant="light"
      size="sm"
      scrollable
    />

    <div v-if="data" class="flex flex-col space-y-3">
      <KunCard
        v-for="(reply, index) in data.replies"
        :key="index"
        :href="replyPermalink(`/topic/${reply.topic_id}`, reply.floor)"
      >
        <div>
          {{ markdownToText(reply.content) }}
        </div>
        <div class="text-default-500 text-sm">
          <KunTime :time="reply.created" type="date" show-year />
        </div>
      </KunCard>

      <KunPagination
        v-if="data.total > pageData.limit"
        v-model:current-page="pageData.page"
        :total-page="Math.ceil(data.total / pageData.limit)"
        :is-loading="status === 'pending'"
      />
    </div>

    <KunNull
      v-if="data && !data.replies.length"
      description="这只笨蛋萝莉没有发布过任何回复"
    />
  </div>
</template>
