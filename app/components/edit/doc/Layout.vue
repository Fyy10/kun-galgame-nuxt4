<script setup lang="ts">
import { provideDocEditorContext } from './context'
import type { DocEditorMode, DocEditorForm } from './type'
import { computeReadingMinute, formatDatetimeLocalInput } from '~/utils/doc'
import { kungalgameResponseHandler } from '~/utils/responseHandler'

const props = withDefaults(
  defineProps<{
    mode: DocEditorMode
    initialArticle?: DocArticleDetail | null
  }>(),
  {
    initialArticle: null
  }
)

const isRewriteMode = computed(() => props.mode === 'rewrite')

const { data: categoryResponse } = await useFetch<DocCategoryListResponse>(
  '/api/doc/category',
  {
    query: {
      page: 1,
      limit: 200,
      keyword: ''
    },
    ...kungalgameResponseHandler
  }
)

const { data: tagResponse } = await useFetch<DocTagListResponse>(
  '/api/doc/tag',
  {
    query: {
      page: 1,
      limit: 500,
      keyword: ''
    },
    ...kungalgameResponseHandler
  }
)

const categories = computed(
  () => categoryResponse.value?.categories ?? []
)
const tags = computed(() => tagResponse.value?.tags ?? [])

const createDefaultForm = (): DocEditorForm => ({
  articleId: null,
  title: '',
  slug: '',
  path: '',
  description: '',
  banner: '',
  status: 1,
  isPin: false,
  readingMinute: 0,
  contentMarkdown: '',
  categoryId: null,
  tagIds: [],
  publishedTime: formatDatetimeLocalInput(new Date())
})

const form = reactive<DocEditorForm>(createDefaultForm())
const isSubmitting = ref(false)
const isPathCustomized = ref(false)

const applyArticleToForm = (article: DocArticleDetail) => {
  form.articleId = article.id
  form.title = article.title
  form.slug = article.slug
  form.path = article.path
  form.description = article.description
  form.banner = article.banner || ''
  form.status = article.status
  form.isPin = article.isPin
  form.readingMinute = article.readingMinute
  form.contentMarkdown = article.contentMarkdown
  form.categoryId = article.category.id
  form.tagIds = article.tags.map((tag) => tag.id)
  form.publishedTime = formatDatetimeLocalInput(article.publishedTime)
}

const resetForm = () => {
  if (isRewriteMode.value && props.initialArticle) {
    applyArticleToForm(props.initialArticle)
    isPathCustomized.value = true
    return
  }

  Object.assign(form, createDefaultForm())
  isPathCustomized.value = false
}

if (isRewriteMode.value && props.initialArticle) {
  applyArticleToForm(props.initialArticle)
  isPathCustomized.value = true
}

watch(
  () => props.initialArticle,
  (article) => {
    if (isRewriteMode.value && article) {
      applyArticleToForm(article)
      isPathCustomized.value = true
    }
  }
)

watch(
  () => form.slug,
  (slug) => {
    if (isRewriteMode.value || isPathCustomized.value) {
      return
    }
    form.path = slug ? `/doc/${slug}` : ''
  }
)

watch(
  () => form.path,
  (value) => {
    if (isRewriteMode.value) {
      return
    }
    if (!value) {
      isPathCustomized.value = false
    }
  }
)

watch(
  () => form.contentMarkdown,
  (markdown) => {
    form.readingMinute = markdown.trim()
      ? computeReadingMinute(markdown)
      : 0
  },
  { immediate: true }
)

const validateForm = () => {
  if (!form.title.trim()) {
    return '请输入标题'
  }
  if (!form.slug.trim()) {
    return '请输入 slug'
  }
  if (!form.path.trim()) {
    return '请输入访问路径'
  }
  if (!form.description.trim()) {
    return '请输入简介'
  }
  if (!form.contentMarkdown.trim()) {
    return '请输入正文内容'
  }
  if (!form.categoryId) {
    return '请选择文档分类'
  }
  return true
}

const handleSubmit = async () => {
  if (isSubmitting.value) {
    return
  }

  const validation = validateForm()
  if (validation !== true) {
    useMessage(validation, 'warn')
    return
  }

  if (isRewriteMode.value && !form.articleId) {
    useMessage('未找到文档 ID，无法更新', 'error')
    return
  }

  isSubmitting.value = true
  try {
    const publishedTime = form.publishedTime
      ? new Date(form.publishedTime)
      : null

    const body: Record<string, unknown> = {
      title: form.title.trim(),
      slug: form.slug.trim(),
      path: form.path.trim(),
      description: form.description.trim(),
      banner: form.banner.trim(),
      status: form.status,
      isPin: form.isPin,
      readingMinute: form.readingMinute,
      contentMarkdown: form.contentMarkdown,
      categoryId: form.categoryId,
      tagIds: Array.from(new Set(form.tagIds)),
      publishedTime:
        publishedTime && !Number.isNaN(publishedTime.getTime())
          ? publishedTime.toISOString()
          : undefined
    }

    if (isRewriteMode.value) {
      body.articleId = form.articleId
    }

    const result = await $fetch<DocArticleDetail>('/api/doc/article', {
      method: isRewriteMode.value ? 'PUT' : 'POST',
      body,
      ...kungalgameResponseHandler
    })

    if (result) {
      useMessage(
        isRewriteMode.value ? '更新文档成功' : '创建文档成功',
        'success'
      )
      applyArticleToForm(result)
      isPathCustomized.value = true
      await navigateTo(result.path)
    }
  } finally {
    isSubmitting.value = false
  }
}

const markPathCustomized = () => {
  if (!isRewriteMode.value) {
    isPathCustomized.value = true
  }
}

provideDocEditorContext({
  form,
  categories,
  tags,
  mode: props.mode,
  isSubmitting,
  handleSubmit,
  resetForm,
  markPathCustomized
})
</script>

<template>
  <div class="contents">
    <ClientOnly>
      <div class="grid grid-cols-1 gap-3 lg:grid-cols-3">
        <KunCard
          :is-hoverable="false"
          :is-pressable="false"
          :is-transparent="false"
          class-name="lg:col-span-1 order-2 sm:order-1"
          content-class="space-y-6"
        >
          <EditDocMetadataForm />
          <EditDocSubmitActions />
        </KunCard>

        <div class="order-1 space-y-3 sm:order-2 lg:col-span-2">
          <KunCard
            :is-hoverable="false"
            :is-pressable="false"
            :is-transparent="false"
          >
            <EditDocTitle />
          </KunCard>
          <KunCard
            :is-hoverable="false"
            :is-pressable="false"
            :is-transparent="false"
          >
            <EditDocContentEditor />
          </KunCard>
        </div>
      </div>
    </ClientOnly>
  </div>
</template>
