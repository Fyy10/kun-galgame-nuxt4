<script setup lang="ts">
const { content_limit } = storeToRefs(usePersistEditGalgameStore())

const contentLimitOptions = [
  {
    value: 'sfw',
    label: 'SFW (可以在公共场所打开)',
    description: '游戏名、介绍、封面都没有不适宜在公共场所显示的内容'
  },
  {
    value: 'nsfw',
    label: 'NSFW (不适合在公共场所打开)',
    description: '游戏名、介绍、封面任意一项露骨, 或者封面需要打码才能放上来'
  }
] as const
</script>

<template>
  <div class="space-y-2">
    <h2 class="space-x-2 text-xl">
      <span>内容限制</span>
      <span class="font-base text-danger text-sm">必选</span>
    </h2>
    <p class="text-default-500 text-sm">
      如果您觉得您的游戏介绍以及封面不适合在公共场所打开 (例如:
      用淫乱喷雾强制贞淑人妻们发情), 请选择 NSFW (Not safe for work), 若没有
      任何不适宜打开的内容 (例如: 永不枯萎的世界与终结之花), 请选择 SFW (Safe
      for work)。这将防止社死并有助于网站索引。
    </p>

    <KunInfo
      color="danger"
      title="再次请大家注意 NSFW 问题"
      description="网站目前的 NSFW 认定标准可能比较苛刻, 看起来不能在公司报告大会上放在 PPT 里展示的游戏都是 NSFW ,封面需要打码才能放上去的一律算 NSFW, 总之就是越严越好，可以错杀不可以放过，因为会导致网站违反 Google 或 Bing 的条款"
    >
      <p class="text-default-500 text-sm">
        例如下面两张图就算作 NSFW 的游戏, 有 NSFW 的游戏名或介绍等等也算作 NSFW
      </p>
      <div class="flex gap-2">
        <KunImage alt="nsfw-image2" src="/edit/1.avif" :width="200" />
        <KunImage alt="nsfw-image2" src="/edit/2.avif" :width="200" />
      </div>
    </KunInfo>

    <p>请注意这个选项, 越严越好, 只要有一点不对立即选择 NSFW</p>
    <KunRadioGroup
      v-model="content_limit"
      :options="contentLimitOptions"
      variant="card"
      aria-label="内容限制"
    />
  </div>
</template>
