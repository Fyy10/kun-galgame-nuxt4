<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import { KUN_MOEMOEPOINT } from '~/constants/moemoepoint'
import { useIdempotencyKey } from '~/composables/useIdempotencyKey'

const { isOpen, target, close } = useUpvoteModal()
const api = useApiClient()
const createKey = useIdempotencyKey()

const description = ref('')
const pending = ref(false)

watch(isOpen, (open) => {
  if (open) description.value = ''
  else close(false)
})

watch(description, (v) => {
  const runes = [...v]
  if (runes.length > 30) description.value = runes.slice(0, 30).join('')
})

const submit = async () => {
  if (!target.value || pending.value) return
  const note = description.value.trim()
  const body = note ? { note } : {}
  pending.value = true
  const result = await settle(
    api.POST('/topics/{topic_id}/upvotes', {
      params: {
        path: { topic_id: target.value.topicId },
        header: {
          'Idempotency-Key': createKey.take(
            `/topics/${target.value.topicId}/upvotes`,
            body
          )
        }
      },
      body
    })
  )
  pending.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  createKey.clear()
  useMessage(10238, 'success')
  close(result.data)
}
</script>

<template>
  <KunModal v-model="isOpen" role="alertdialog" inner-class-name="max-w-md">
    <div class="space-y-4">
      <h3 class="text-lg font-medium">确定推这个话题吗？</h3>
      <p class="text-default-500 text-sm">
        推话题将消耗您
        <span class="text-warning-600 font-medium">{{
          KUN_MOEMOEPOINT.upvoteSender
        }}</span>
        萌萌点，并给被推者增加
        <span class="text-success font-medium">{{
          KUN_MOEMOEPOINT.upvoteOwner
        }}</span>
        萌萌点。
      </p>
      <div>
        <KunInput
          v-model="description"
          placeholder="（可选）一句话，说说为什么推它 ✨"
        />
        <p class="text-default-400 mt-1 text-right text-xs">
          {{ [...description].length }}/30
        </p>
      </div>
      <div class="flex justify-end gap-3">
        <KunButton variant="light" color="default" @click="close(false)">
          取消
        </KunButton>
        <KunButton color="secondary" :loading="pending" @click="submit">
          确定推
        </KunButton>
      </div>
    </div>
  </KunModal>
</template>
