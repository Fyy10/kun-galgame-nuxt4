<script setup lang="ts">
import type { KunTabItem } from '@kungal/ui-vue'
import { useRouteQuery } from '@vueuse/router'

const props = defineProps<{
  userId: number
}>()

const { id } = storeToRefs(usePersistUserStore())
const isOwner = computed(() => !!id.value && id.value === props.userId)

// The tab goes in the URL with the page, or a restored page would land on
// whichever tab happens to be the default. /mine/answered answers for the
// *viewer*, so a hand-typed ?tab=answered on someone else's profile would show
// a reader their own history under that name — hence isOwner here too.
const tabQuery = useRouteQuery<string>('tab', 'publish', { mode: 'replace' })
const tab = computed(() =>
  tabQuery.value === 'answered' && isOwner.value ? 'answered' : 'publish'
)
const tabItems = computed<KunTabItem[]>(() => {
  const items: KunTabItem[] = [
    { value: 'publish', textValue: '出题', icon: 'lucide:pencil-line' }
  ]
  if (isOwner.value) {
    items.push({ value: 'answered', textValue: '答题', icon: 'lucide:history' })
  }
  return items
})

const params = reactive({
  page: usePageQuery(),
  limit: 50,
  user_id: props.userId
})
const requestUrl = computed(() =>
  tab.value === 'answered' ? '/galgame-quiz/mine/answered' : '/galgame-quiz/all'
)
const { data, status } = await useKunFetch<QuizListPage>(requestUrl, {
  method: 'GET',
  query: params
})

const onTab = (v: string) => {
  params.page = 1
  tabQuery.value = v
}
</script>

<template>
  <div class="space-y-3">
    <KunTab
      v-if="tabItems.length > 1"
      :model-value="tab"
      :items="tabItems"
      variant="light"
      color="primary"
      @update:model-value="onTab"
    />

    <template v-if="data && data.quiz_data.length">
      <GalgameQuizList :quizzes="data.quiz_data" />
      <KunPagination
        v-if="data.total > params.limit"
        v-model:current-page="params.page"
        :total-page="Math.ceil(data.total / params.limit)"
        :is-loading="status === 'pending'"
      />
    </template>

    <KunNull
      v-else-if="status !== 'pending'"
      :description="tab === 'answered' ? '还没有作答记录' : '还没有出过题'"
    />
  </div>
</template>
