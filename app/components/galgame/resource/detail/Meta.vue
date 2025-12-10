<script setup lang="ts">
import {
  KUN_GALGAME_RESOURCE_LANGUAGE_MAP,
  KUN_GALGAME_RESOURCE_PLATFORM_MAP,
  KUN_GALGAME_RESOURCE_TYPE_MAP,
  KUN_GALGAME_AGE_LIMIT_MAP
} from '~/constants/galgame'

const props = defineProps<{
  galgame: GalgameResourceSummary
}>()

const stats = computed(() => {
  const originalLanguage =
    KUN_GALGAME_RESOURCE_LANGUAGE_MAP[props.galgame.originalLanguage] ||
    props.galgame.originalLanguage

  return [
    {
      label: '最近刷新',
      value: formatTimeDifference(props.galgame.resourceUpdateTime)
    },
    {
      label: '累计浏览',
      value: props.galgame.view.toLocaleString()
    },
    {
      label: '原始语言',
      value: originalLanguage
    },
    {
      label: '年龄分级',
      value: KUN_GALGAME_AGE_LIMIT_MAP[props.galgame.ageLimit]
    }
  ]
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

const typeLabels = computed(() => {
  if (!props.galgame.type.length) return ['暂无数据']
  return props.galgame.type.map(
    (type) => KUN_GALGAME_RESOURCE_TYPE_MAP[type] || type
  )
})
</script>

<template>
  <KunCard
    :is-hoverable="false"
    :is-transparent="false"
    content-class="space-y-4"
  >
    <KunHeader
      name="Galgame 速览"
      description="掌握该 Galgame 的基础信息，便于快速判断是否符合需求。"
      scale="h2"
    />

    <div class="space-y-2 text-sm">
      <div
        v-for="stat in stats"
        :key="stat.label"
        class="bg-content2 flex items-center justify-between rounded-xl px-3 py-2"
      >
        <span class="text-default-500">{{ stat.label }}</span>
        <span class="font-medium">{{ stat.value }}</span>
      </div>
    </div>

    <KunDivider />

    <div class="space-y-3 text-sm">
      <div>
        <p class="text-default-500 text-xs tracking-wide uppercase">语言覆盖</p>
        <div class="mt-1 flex flex-wrap gap-1">
          <KunBadge
            v-for="lang in languageLabels"
            :key="`stat-lang-${lang}`"
            variant="flat"
          >
            {{ lang }}
          </KunBadge>
        </div>
      </div>

      <div>
        <p class="text-default-500 text-xs tracking-wide uppercase">平台支持</p>
        <div class="mt-1 flex flex-wrap gap-1">
          <KunBadge
            v-for="platform in platformLabels"
            :key="`stat-platform-${platform}`"
            variant="flat"
          >
            {{ platform }}
          </KunBadge>
        </div>
      </div>

      <div>
        <p class="text-default-500 text-xs tracking-wide uppercase">资源类型</p>
        <div class="mt-1 flex flex-wrap gap-1">
          <KunBadge
            v-for="type in typeLabels"
            :key="`stat-type-${type}`"
            variant="flat"
          >
            {{ type }}
          </KunBadge>
        </div>
      </div>
    </div>
  </KunCard>
</template>
