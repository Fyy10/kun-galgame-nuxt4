<script setup lang="ts">
import { ref, computed } from 'vue'
import type { Poll } from '#shared/utils/api/schemas'
import { usePoll } from '~/composables/topic/usePoll'
import { TOPIC_POLL_VISIBILITY_MAP } from '~/constants/topic'
import { toKunUser } from '~/utils/userRef'

const props = defineProps<{
  poll: Poll
}>()

const emits = defineEmits<{
  edit: [poll: Poll]
  refresh: []
}>()

const poll = ref<Poll>(props.poll)
watch(
  () => props.poll,
  (value) => {
    poll.value = value
  }
)

const { deletePoll, setVote, clearVote } = usePoll(poll.value.topic_id)
const selectedOptions = ref<string[]>([
  ...(poll.value.viewer?.chosen_option_ids ?? [])
])
const isLoading = ref(false)
const isLogModalOpen = ref(false)

watch(
  () => poll.value.viewer?.chosen_option_ids,
  (chosen) => {
    selectedOptions.value = [...(chosen ?? [])]
  }
)

const nowMs = useState(`kun-poll-now-${useId()}`, () => Date.now())
onMounted(() => {
  nowMs.value = Date.now()
})

const isPollEnded = computed(() => {
  if (!poll.value.closes_at) {
    return false
  }
  return new Date(poll.value.closes_at).getTime() < nowMs.value
})

const results = computed(() => poll.value.results)
const totalVoteCount = computed(() => results.value?.total_vote_count ?? 0)
const countByOption = computed(
  () =>
    new Map(
      (results.value?.options ?? []).map((item) => [
        item.option_id,
        item.vote_count
      ])
    )
)

const voteCountOf = (optionId: string) => countByOption.value.get(optionId) ?? 0
const percentOf = (optionId: string) =>
  (voteCountOf(optionId) / (totalVoteCount.value || 1)) * 100

const isLocked = computed(
  () => isPollEnded.value || poll.value.viewer?.can_vote === false
)
const canSubmitVote = computed(
  () => !isPollEnded.value && (poll.value.viewer?.can_vote ?? true)
)
const canRetractVote = computed(
  () =>
    !isPollEnded.value &&
    !!poll.value.viewer?.has_voted &&
    poll.value.viewer.can_change_vote
)
const canViewVotes = computed(() => !!results.value && !poll.value.is_anonymous)

const hiddenResultsHint = computed(() => {
  if (poll.value.result_visibility === 'after_vote') {
    return '投票后可以看到结果'
  }
  if (poll.value.result_visibility === 'after_deadline') {
    return '结束后才会公开结果'
  }
  return '你暂时看不到这个投票的结果'
})

const metaLine = computed(() => [
  poll.value.min_choice === poll.value.max_choice
    ? `必选 ${poll.value.max_choice} 项`
    : `可选 ${poll.value.min_choice}-${poll.value.max_choice} 项`,
  TOPIC_POLL_VISIBILITY_MAP[poll.value.result_visibility],
  poll.value.can_change_vote ? '可修改投票' : '投出后不可修改',
  poll.value.is_anonymous ? '匿名投票' : '实名投票',
  results.value ? `共 ${results.value.total_vote_count} 票` : ''
])

const handleOptionClick = (optionId: string) => {
  if (isLocked.value) {
    return
  }
  if (poll.value.choice_type === 'single') {
    selectedOptions.value = [optionId]
    return
  }
  const index = selectedOptions.value.indexOf(optionId)
  if (index > -1) {
    selectedOptions.value.splice(index, 1)
    return
  }
  if (selectedOptions.value.length >= poll.value.max_choice) {
    selectedOptions.value.shift()
  }
  selectedOptions.value.push(optionId)
}

const handleVote = async () => {
  if (!requireLogin()) {
    return
  }
  if (selectedOptions.value.length < poll.value.min_choice) {
    useMessage(`请至少选择 ${poll.value.min_choice} 项`, 'warn')
    return
  }
  isLoading.value = true
  const result = await setVote(poll.value.id, selectedOptions.value)
  isLoading.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  poll.value = result.data
  useMessage('投票成功', 'success')
}

const handleRetract = async () => {
  if (!requireLogin()) {
    return
  }
  isLoading.value = true
  const result = await clearVote(poll.value.id)
  isLoading.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  poll.value = result.data
  useMessage('已撤回投票', 'success')
}

const handleDelete = async () => {
  const result = await deletePoll(poll.value.id)
  if (!result) {
    return
  }
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage('删除投票成功', 'success')
  emits('refresh')
}
</script>

<template>
  <KunCard
    :is-hoverable="false"
    :is-transparent="false"
    content-class="space-y-3"
  >
    <TopicMiniappHeader
      app-key="poll"
      :title="poll.title"
      :meta="metaLine"
      :status="isPollEnded ? '已结束' : '进行中'"
      :status-color="isPollEnded ? 'default' : 'success'"
    />

    <p v-if="poll.description" class="text-default-500 text-sm">
      {{ poll.description }}
    </p>

    <div class="flex flex-col gap-2">
      <div
        v-for="option in poll.options"
        :key="option.id"
        :class="
          cn(
            'border-default-200 relative overflow-hidden rounded-lg border transition-colors',
            !isLocked && 'hover:border-primary/60 cursor-pointer',
            selectedOptions.includes(option.id) && 'border-primary bg-primary/5'
          )
        "
        @click="handleOptionClick(option.id)"
      >
        <div
          v-if="results"
          class="bg-primary/15 absolute inset-y-0 left-0 transition-all duration-500"
          :style="{ width: `${percentOf(option.id)}%` }"
        />

        <div
          class="relative flex items-center justify-between gap-3 px-3 py-2.5"
        >
          <div class="flex min-w-0 items-center gap-3">
            <KunCheckBox
              color="primary"
              :type="poll.choice_type"
              :model-value="selectedOptions.includes(option.id)"
              :disabled="isLocked"
              @click.stop
              @change="handleOptionClick(option.id)"
            />

            <span class="truncate text-sm">{{ option.text }}</span>
            <KunIcon
              v-if="poll.viewer?.chosen_option_ids.includes(option.id)"
              name="lucide:check-circle-2"
              class="text-primary shrink-0"
            />
          </div>

          <div
            v-if="results"
            class="flex shrink-0 items-baseline gap-2 tabular-nums"
          >
            <span class="text-default-500 text-xs">
              {{ voteCountOf(option.id) }} 票
            </span>
            <span class="w-14 text-right text-sm font-semibold">
              {{ percentOf(option.id).toFixed(1) }}%
            </span>
          </div>
        </div>
      </div>
    </div>

    <div
      class="border-default-200 flex flex-wrap items-center justify-between gap-3 border-t pt-3"
    >
      <span v-if="!results" class="text-default-500 text-sm">
        {{ hiddenResultsHint }}
      </span>
      <KunAvatarGroup
        v-else-if="results.sample_voters.length"
        :users="results.sample_voters.map(toKunUser)"
        :total="results.voter_count"
      />
      <span v-else class="text-default-500 text-sm">还没有人投票</span>

      <div class="ml-auto flex items-center gap-2">
        <KunButton
          v-if="canRetractVote"
          variant="light"
          color="default"
          size="sm"
          :loading="isLoading"
          @click="handleRetract"
        >
          撤回投票
        </KunButton>

        <KunButton
          v-if="canSubmitVote"
          color="primary"
          size="sm"
          :loading="isLoading"
          :disabled="selectedOptions.length === 0"
          @click="handleVote"
        >
          {{ poll.viewer?.has_voted ? '修改投票' : '投票' }}
        </KunButton>

        <KunTooltip v-if="canViewVotes" text="查看投票记录">
          <KunButton
            variant="light"
            color="default"
            size="sm"
            :is-icon-only="true"
            @click="isLogModalOpen = true"
          >
            <KunIcon name="lucide:history" />
          </KunButton>
        </KunTooltip>

        <KunTooltip v-if="poll.viewer?.can_edit" text="编辑投票">
          <KunButton
            variant="light"
            color="default"
            size="sm"
            :is-icon-only="true"
            @click="emits('edit', poll)"
          >
            <KunIcon name="lucide:pencil" />
          </KunButton>
        </KunTooltip>

        <KunTooltip v-if="poll.viewer?.can_delete" text="删除投票">
          <KunButton
            variant="light"
            color="danger"
            size="sm"
            :is-icon-only="true"
            @click="handleDelete"
          >
            <KunIcon name="lucide:trash-2" />
          </KunButton>
        </KunTooltip>
      </div>
    </div>

    <TopicPollLog
      v-if="isLogModalOpen"
      v-model="isLogModalOpen"
      :poll-id="poll.id"
      :options="poll.options"
    />
  </KunCard>
</template>
