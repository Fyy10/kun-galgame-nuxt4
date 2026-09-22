import { createTopicSchema } from '~/validations/topic'
import { useTopicEditorStore } from './useTopicEditorStore'
import { coverHashFromToken } from './applyTopicSource'
import { useIdempotencyKey } from '~/composables/useIdempotencyKey'
import {
  TOPIC_SECTION_CONSUME_MOEMOEPOINTS,
  MOEMOEPOINT_COST_FOR_CONSUME_SECTION
} from '~/config/moemoepoint'
import { settle } from '#shared/utils/api/problem'
import type { TopicCreate, TopicPatch } from '#shared/utils/api/schemas'

export const useTopicSubmitter = () => {
  const {
    category,
    section,
    title,
    content,
    isNSFW,
    coverImages,
    accessScope,
    accessRoles,
    accessUserIds
  } = useTopicEditorStore()
  const tempStore = useTempEditStore()
  const persistStore = usePersistEditTopicStore()
  const { moemoepoint } = usePersistUserStore()
  const api = useApiClient()
  const createKey = useIdempotencyKey()

  const rules = reactive({
    isReadRule: false,
    isAgreeCategory: false,
    isValidTitle: false,
    isKnownConsequence: false
  })
  const isSubmitting = ref(false)
  const isRewriteMode = computed(() => tempStore.isTopicRewriting)

  const writePayload = (mode: 'create' | 'rewrite') => {
    const hashes = coverImages.value.map(coverHashFromToken)
    const body: TopicCreate = {
      title: title.value,
      content_markdown: content.value,
      category: category.value as TopicCreate['category'],
      sections: section.value as TopicCreate['sections'],
      is_nsfw: isNSFW.value,
      access_scope: accessScope.value
    }
    if (mode === 'rewrite') {
      body.cover_image_hashes = hashes
    } else if (hashes.length > 0) {
      body.cover_image_hashes = hashes
    }
    if (accessScope.value === 'role') {
      body.access_roles = accessRoles.value
    }
    if (accessScope.value === 'users') {
      body.access_user_ids = accessUserIds.value.map(String)
    }
    return body
  }

  const submit = async () => {
    if (isSubmitting.value) {
      return
    }

    const isReadAllRules = Object.values(rules).every((value) => value)
    if (moemoepoint < 50 && !isReadAllRules) {
      useMessage('请勾选同意所有发布须知后再发布话题', 'warn')
      return
    }

    const mode = isRewriteMode.value ? 'rewrite' : 'create'
    const payload = writePayload(mode)
    const result = createTopicSchema.safeParse(payload)
    if (!result.success) {
      const error = JSON.parse(result.error.message)[0]
      useMessage(formatKunZodIssue(error), 'warn')
      return
    }

    const hasConsumeSection = TOPIC_SECTION_CONSUME_MOEMOEPOINTS.some((item) =>
      payload.sections.includes(item)
    )
    if (
      hasConsumeSection &&
      moemoepoint < MOEMOEPOINT_COST_FOR_CONSUME_SECTION
    ) {
      useMessage(
        `您没有足够的萌萌点来发布求助或者寻求资源的话题, 您可以通过发布 Galgame, 签到, 接受别人的赞赏, 等等来获取萌萌点`,
        'warn'
      )
      return
    }

    isSubmitting.value = true
    try {
      if (mode === 'rewrite') {
        const topicId = String(tempStore.id)
        const patched = await settle(
          api.PATCH('/topics/{topic_id}', {
            params: { path: { topic_id: topicId } },
            body: payload as TopicPatch
          })
        )
        if (!patched.ok) {
          reportProblem(patched.problem)
          return
        }
        useKunLoliInfo('重新编辑成功', 5)
        tempStore.resetRewriteTopicData()
        await navigateTo(`/topic/${patched.data.id}`)
        return
      }

      const created = await settle(
        api.POST('/topics', {
          params: {
            header: { 'Idempotency-Key': createKey.take(payload) }
          },
          body: payload
        })
      )
      if (!created.ok) {
        reportProblem(created.problem)
        return
      }
      createKey.clear()
      useKunLoliInfo('发布成功', 5)
      persistStore.resetTopicData()
      await navigateTo(`/topic/${created.data.id}`)
    } finally {
      isSubmitting.value = false
    }
  }

  return {
    rules,
    submit,
    isSubmitting,
    isRewriteMode
  }
}
