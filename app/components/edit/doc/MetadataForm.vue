<script setup lang="ts">
import type { KunSelectOption } from '~/components/kun/select/type'
import { KUN_DOC_CATEGORY_MAP, KUN_DOC_STATUS_OPTIONS } from '~/constants/doc'
import { useDocEditorContext } from './context'

const { form, categories, tags, markPathCustomized } = useDocEditorContext()

const categoryOptions = computed<KunSelectOption[]>(() =>
  categories.value.map((category) => ({
    label:
      KUN_DOC_CATEGORY_MAP[category.slug] || `${category.title} (${category.slug})`,
    value: category.id
  }))
)

const statusOptions = computed<KunSelectOption[]>(() =>
  KUN_DOC_STATUS_OPTIONS.map((option) => ({
    label: option.label,
    value: option.value
  }))
)

const tagKeyword = ref('')
const filteredTags = computed(() => {
  if (!tagKeyword.value.trim()) {
    return tags.value
  }
  const keyword = tagKeyword.value.trim().toLowerCase()
  return tags.value.filter(
    (tag) =>
      tag.title.toLowerCase().includes(keyword) ||
      tag.slug.toLowerCase().includes(keyword)
  )
})

const isTagSelected = (tagId: number) => form.tagIds.includes(tagId)

const toggleTag = (tagId: number) => {
  if (isTagSelected(tagId)) {
    form.tagIds = form.tagIds.filter((id) => id !== tagId)
  } else {
    form.tagIds = [...form.tagIds, tagId]
  }
}

const normalizeSlug = () => {
  if (!form.slug.trim()) {
    return
  }
  form.slug = form.slug
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9-]/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '')
}

const handlePathInput = () => {
  markPathCustomized()
}
</script>

<template>
  <div class="space-y-6">
    <div>
      <h3 class="text-lg font-semibold">基础信息</h3>
      <p class="text-default-500 text-sm">
        配置 slug、路径、封面等基础数据
      </p>
    </div>

    <div class="space-y-4">
      <KunInput
        v-model="form.slug"
        label="Slug"
        placeholder="请输入唯一的 slug"
        maxlength="233"
        required
        @blur="normalizeSlug"
      />

      <KunInput
        v-model="form.path"
        label="访问路径"
        placeholder="例如 /doc/awesome-article"
        maxlength="255"
        required
        @update:model-value="handlePathInput"
      />

      <KunInput
        v-model="form.banner"
        label="封面地址"
        placeholder="https://example.com/banner.webp"
        maxlength="777"
      />

      <KunTextarea
        v-model="form.description"
        label="文档简介"
        placeholder="写一段用来展示的简介"
        :rows="4"
        auto-grow
        maxlength="777"
        required
        show-char-count
      />
    </div>

    <div class="space-y-4">
      <KunSelect
        v-model="form.categoryId"
        :options="categoryOptions"
        label="文档分类"
        placeholder="请选择分类"
      />

      <KunSelect
        v-model="form.status"
        :options="statusOptions"
        label="发布状态"
        placeholder="请选择状态"
      />

      <KunInput
        v-model="form.publishedTime"
        label="发布时间"
        type="datetime-local"
      />

      <div class="grid grid-cols-2 gap-3">
        <KunInput
          v-model="form.readingMinute"
          label="阅读时长(分钟)"
          type="number"
          min="0"
          readonly
          helper-text="根据正文自动计算"
        />

        <div class="flex items-center justify-between rounded-lg border border-default-200 px-4 py-2">
          <div>
            <p class="text-sm font-medium">置顶展示</p>
            <p class="text-default-500 text-xs">是否在首页轮播中固定显示</p>
          </div>
          <KunSwitch v-model="form.isPin" color="primary" />
        </div>
      </div>
    </div>

    <div class="space-y-3">
      <div class="flex items-center justify-between gap-3">
        <div>
          <h4 class="font-semibold">文档标签</h4>
          <p class="text-default-500 text-xs">用于在文档详情中展示标签</p>
        </div>
        <KunInput
          v-model="tagKeyword"
          placeholder="搜索标签..."
          size="sm"
          class-name="min-w-44"
        >
          <template #prefix>
            <KunIcon name="lucide:search" class="h-4 w-4 text-default-500" />
          </template>
        </KunInput>
      </div>

      <div class="flex flex-wrap gap-2">
        <KunBadge
          v-for="tagId in form.tagIds"
          :key="tagId"
          color="secondary"
          variant="flat"
        >
          {{
            tags.find((tag) => tag.id === tagId)?.title ??
            `标签 #${tagId}`
          }}
        </KunBadge>
      </div>

      <div
        class="scrollbar-hide max-h-64 space-y-2 overflow-y-auto rounded-lg border border-dashed border-default-200 p-3"
      >
        <KunButton
          v-for="tag in filteredTags"
          :key="tag.id"
          size="sm"
          rounded="full"
          :variant="isTagSelected(tag.id) ? 'solid' : 'light'"
          :color="isTagSelected(tag.id) ? 'primary' : 'default'"
          @click="toggleTag(tag.id)"
        >
          {{ tag.title }}
        </KunButton>
      </div>
    </div>
  </div>
</template>
