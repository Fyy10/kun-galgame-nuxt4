export interface GalgameCommunityComment {
  id: number
  thread_id: number
  content: string
  content_html: string
  galgame_id: number
  user: KunUser
  parent_comment_id: number | null
  root_comment_id: number | null
  target_user?: KunUser | null
  like_count: number
  is_liked: boolean
  created: string
  edited: string | null
  edited_by_moderator: boolean
  status: number
  deleted: boolean
  held: boolean
}

export interface GalgameCommunityCommentPage {
  thread_id: number
  posts: GalgameCommunityComment[]
  next_cursor: string
  total: number
  locked: boolean
  anchor_kind: number
  anchor_id: string
}

export interface CommunityWallState {
  anchor_kind: number
  anchor_id: string
  thread_id: number
  following: boolean
  notification_level: number
}

export interface CommunityFollowItem {
  anchor_kind: number
  anchor_id: string
  link: string
  title: string
  label: string
  galgame_id?: number
}

export interface CommunityFollowList {
  items: CommunityFollowItem[]
  next_cursor: string
}
