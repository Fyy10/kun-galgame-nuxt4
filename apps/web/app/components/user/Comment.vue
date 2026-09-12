<script setup lang="ts">
import {
  kunUserCommentNavItem,
  type KUN_USER_PAGE_COMMENT_TYPE
} from '~/constants/user'

const props = defineProps<{
  userId: number
  type: (typeof KUN_USER_PAGE_COMMENT_TYPE)[number]
}>()

const activeTab = ref(props.type)
const pageData = reactive({
  page: usePageQuery(),
  limit: 50,
  userId: props.userId,
  type: props.type
})

const { data, status } = await useKunFetch<{
  comments: UserComment[]
  total: number
}>(`/user/${props.userId}/comments`, { query: pageData })
</script>

<template>
  <div class="space-y-6">
    <KunTab
      :items="kunUserCommentNavItem(userId)"
      :model-value="activeTab"
      variant="light"
      size="sm"
      scrollable
    />

    <div class="flex flex-col space-y-3" v-if="data && data.comments.length">
      <KunCard
        v-for="(comment, index) in data.comments"
        :key="index"
        :href="commentPermalink(`/topic/${comment.topic_id}`, comment.id)"
      >
        <div>
          {{ comment.content }}
        </div>
        <div class="text-default-500 text-sm">
          <KunTime :time="comment.created" type="date" show-year />
        </div>
      </KunCard>

      <KunPagination
        v-model:current-page="pageData.page"
        :total-page="Math.ceil(data.total / pageData.limit)"
        :is-loading="status === 'pending'"
      />
    </div>

    <KunNull
      v-if="data && !data.comments.length"
      description="这只笨蛋萝莉没有发布过任何评论"
    />
  </div>
</template>
