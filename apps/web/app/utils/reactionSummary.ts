import type { ReactionSummary } from '#shared/utils/api/schemas'
import { toKunUser } from './userRef'

export const toKunReactions = (summaries: ReactionSummary[]): KunReaction[] =>
  summaries.map((summary) => ({
    reaction: summary.reaction,
    count: summary.count,
    mine: summary.viewer?.has_reacted ?? false,
    reactors: summary.reactors.map(toKunUser)
  }))
