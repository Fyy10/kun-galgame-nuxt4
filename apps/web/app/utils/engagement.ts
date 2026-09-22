import type { Topic, TopicEngagement } from '#shared/utils/api/schemas'

// A write answers with the whole engagement snapshot. Applying only the fields
// the button itself changed left the rest of the page stale: favoriting kept
// the old like counts, a reaction kept the old favorite count, and nothing
// carried upvoted_at back.
export const mergeTopicEngagement = (
  topic: Topic,
  engagement: TopicEngagement
): Topic => ({
  ...topic,
  like_count: engagement.like_count,
  dislike_count: engagement.dislike_count,
  favorite_count: engagement.favorite_count,
  upvote_count: engagement.upvote_count,
  upvoted_at: engagement.upvoted_at,
  reactions: engagement.reactions,
  viewer: engagement.viewer
})
