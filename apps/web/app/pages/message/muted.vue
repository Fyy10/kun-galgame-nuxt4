<script setup lang="ts">
import { useRouteQuery } from '@vueuse/router'
import { localNotificationCategories } from '~/constants/notification'

definePageMeta({
  middleware: 'auth'
})

useKunDisableSeo('已静音的消息')

const { data: prefs } = await useKunFetch<NotificationPreference>(
  '/user/notification-preferences'
)
const mutedCategories = computed(() => {
  const muted = new Set(prefs.value?.muted_types ?? [])
  return localNotificationCategories.filter((c) => muted.has(c.key))
})
const tabItems = computed(() => [
  { value: 'all', textValue: '全部' },
  ...mutedCategories.value.map((c) => ({ value: c.key, textValue: c.label }))
])
// Tab and page both in the URL: a notice opens the topic it is about, so coming
// back has to land on the tab and page it was opened from. The tab has to come
// along — a page restored under a different filter is a list the reader never saw.
const activeTab = useRouteQuery<string>('tab', 'all', { mode: 'replace' })
const page = usePageQuery()

const pageData = reactive({
  page,
  limit: 30,
  sort_order: 'desc',
  type: activeTab.value === 'all' ? '' : activeTab.value
})

watch(activeTab, (tab) => {
  pageData.type = tab === 'all' ? '' : tab
  page.value = 1
})

const { data, status, refresh } = await useKunFetch<MessageList>(
  '/message/muted',
  { query: pageData }
)
</script>

<template>
  <div class="flex w-full flex-col space-y-3" v-if="data">
    <header class="flex items-center gap-2">
      <KunButton size="lg" :is-icon-only="true" variant="light" href="/message">
        <KunIcon name="lucide:chevron-left" />
      </KunButton>
      <h2 class="text-lg">已静音的消息</h2>
    </header>

    <KunTab
      v-model="activeTab"
      :items="tabItems"
      variant="underlined"
      color="primary"
      size="sm"
    />

    <KunDivider />

    <KunOverlayScroll v-if="data.messages.length" class="h-full">
      <MessageAsideNotice
        v-for="message in data.messages"
        :key="message.id"
        :message="message"
        :refresh="refresh"
      />
    </KunOverlayScroll>

    <KunNull v-if="!data.total" description="没有已静音的消息" />

    <KunPagination
      v-if="data.total"
      v-model:current-page="pageData.page"
      :total-page="Math.ceil(data.total / pageData.limit)"
      :is-loading="status === 'pending'"
    />
  </div>
</template>
