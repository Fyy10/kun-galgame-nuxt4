import type { Comment } from '#shared/utils/api/schemas'

export const threadComments = (list: Comment[]) => {
  const byId = new Map(list.map((comment) => [comment.id, comment]))
  const rootOf = (comment: Comment): Comment => {
    let cur = comment
    const seen = new Set<string>()
    while (
      cur.parent_comment_id != null &&
      byId.has(cur.parent_comment_id) &&
      !seen.has(cur.id)
    ) {
      seen.add(cur.id)
      cur = byId.get(cur.parent_comment_id)!
    }
    return cur
  }
  const byTime = (left: Comment, right: Comment) =>
    new Date(left.created_at).getTime() - new Date(right.created_at).getTime()

  const roots: Comment[] = []
  const childrenOf = new Map<string, Comment[]>()
  for (const comment of list) {
    const root = rootOf(comment)
    if (root.id === comment.id) {
      roots.push(comment)
    } else {
      const arr = childrenOf.get(root.id) ?? []
      arr.push(comment)
      childrenOf.set(root.id, arr)
    }
  }
  roots.sort(byTime)

  const out: { comment: Comment; depth: number }[] = []
  for (const root of roots) {
    out.push({ comment: root, depth: 0 })
    for (const kid of (childrenOf.get(root.id) ?? []).slice().sort(byTime)) {
      out.push({ comment: kid, depth: 1 })
    }
  }
  return out
}
