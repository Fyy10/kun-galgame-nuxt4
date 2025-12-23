<script setup lang="ts">
const { showKUNGalgameContentLimit } = storeToRefs(usePersistSettingsStore())

const isNSFWEnable = computed(
  () =>
    showKUNGalgameContentLimit.value === 'nsfw' ||
    showKUNGalgameContentLimit.value === 'all'
)

const open = ref(false)
const { isOpen, onOpen, onClose } = useKunDisclosure({
  isOpen: open.value,
  onChange: (v) => (open.value = v)
})

const description = isNSFWEnable.value
  ? '网站已启用 NSFW, 杂鱼~♡ 臭杂鱼♡, 请注意您周围没有人'
  : '网站有部分 R18 Galgame 不可见, 要打开 R18 模式, 请开启下面的 NSFW 按钮'

const title = isNSFWEnable.value
  ? '网站已显示全部 R18 游戏'
  : '部分 Galgame 不可见, 点击查看'
</script>

<template>
  <span
    :class="
      cn(
        'cursor-pointer font-bold underline underline-offset-3',
        isNSFWEnable ? 'text-success-600' : 'text-danger-600'
      )
    "
    @click="onOpen"
  >
    {{ title }}

    <KunModal inner-class-name="overflow-visible" v-model:modal-value="open">
      <KunInfo
        :title="title"
        :color="isNSFWEnable ? 'danger' : 'warning'"
        :description="description"
        :icon="isNSFWEnable ? 'lucide:ban' : 'lucide:shield-check'"
      >
        <div class="mt-3 text-base">
          <KunSettingPanelComponentsNSFW />
        </div>
      </KunInfo>
    </KunModal>
  </span>
</template>
