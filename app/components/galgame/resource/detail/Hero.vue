<script setup lang="ts">
import {
  KUN_GALGAME_RESOURCE_LANGUAGE_MAP,
  KUN_GALGAME_RESOURCE_PLATFORM_MAP,
  KUN_GALGAME_RESOURCE_TYPE_MAP,
  KUN_GALGAME_CONTENT_LIMIT_MAP,
  KUN_GALGAME_AGE_LIMIT_MAP
} from '~/constants/galgame'

const props = defineProps<{
  galgame: GalgameResourceSummary
}>()

const galgameName = computed(() => getPreferredLanguageText(props.galgame.name))

const typeLabels = computed(() => {
  if (!props.galgame.type.length) return ['暂无数据']
  return props.galgame.type.map(
    (type) => KUN_GALGAME_RESOURCE_TYPE_MAP[type] || type
  )
})

const languageLabels = computed(() => {
  if (!props.galgame.language.length) return ['暂无数据']
  return props.galgame.language.map(
    (lang) => KUN_GALGAME_RESOURCE_LANGUAGE_MAP[lang] || lang
  )
})

const platformLabels = computed(() => {
  if (!props.galgame.platform.length) return ['暂无数据']
  return props.galgame.platform.map(
    (platform) => KUN_GALGAME_RESOURCE_PLATFORM_MAP[platform] || platform
  )
})

const contentLimitLabel = computed(
  () => KUN_GALGAME_CONTENT_LIMIT_MAP[props.galgame.contentLimit]
)
const ageLimitLabel = computed(
  () => KUN_GALGAME_AGE_LIMIT_MAP[props.galgame.ageLimit]
)
</script>

<template>
  <KunCard
    :is-hoverable="false"
    :is-transparent="false"
    content-class="space-y-4"
  >
    <div class="flex flex-col gap-4 lg:flex-row">
      <KunImage
        data-kun-lazy-image="true"
        class="max-h-64 object-cover"
        :style="{ aspectRatio: '16/9' }"
        :src="galgame.banner"
        loading="lazy"
        :alt="getPreferredLanguageText(galgame.name)"
      />

      <div class="flex flex-1 flex-col gap-3">
        <div class="flex flex-wrap items-center gap-2">
          <KunBadge
            :color="galgame.contentLimit === 'sfw' ? 'success' : 'danger'"
          >
            {{ contentLimitLabel }}
          </KunBadge>
          <KunBadge color="warning">
            {{ ageLimitLabel }}
          </KunBadge>
          <KunBadge v-for="type in typeLabels" :key="type" variant="flat">
            {{ type }}
          </KunBadge>
        </div>

        <div>
          <h1 class="text-2xl font-bold">
            {{ galgameName }}
          </h1>
          <p class="text-default-500 text-sm">
            {{
              `最近更新 ${formatTimeDifference(galgame.resourceUpdateTime)} · ${galgame.view.toLocaleString()} 次浏览`
            }}
          </p>
        </div>

        <div class="space-y-2">
          <div>
            <p class="text-default-500 text-xs tracking-wide uppercase">
              支持语言
            </p>
            <div class="mt-1 flex flex-wrap gap-1">
              <KunBadge
                v-for="lang in languageLabels"
                :key="lang"
                variant="flat"
              >
                {{ lang }}
              </KunBadge>
            </div>
          </div>

          <div>
            <p class="text-default-500 text-xs tracking-wide uppercase">
              推荐平台
            </p>
            <div class="mt-1 flex flex-wrap gap-1">
              <KunBadge
                v-for="platform in platformLabels"
                :key="platform"
                variant="flat"
              >
                {{ platform }}
              </KunBadge>
            </div>
          </div>
        </div>

        <div class="flex flex-wrap gap-2">
          <KunButton :href="`/galgame/${galgame.id}`">
            访问 Galgame 页面
          </KunButton>
          <KunButton variant="light" href="/galgame-resource">
            浏览更多资源
          </KunButton>
        </div>
      </div>
    </div>
  </KunCard>
</template>
