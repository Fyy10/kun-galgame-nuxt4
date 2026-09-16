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
}

/** 0=muted 1=normal 2=tracking 3=watching, as the community service numbers them. */
export type CommunityNotificationLevel = 0 | 1 | 2 | 3

export interface CommunityThreadState {
  thread_id: number
  subscribed: boolean
  notification_level: CommunityNotificationLevel
  unread_count: number
}

export interface CommunityUnreadItem {
  thread_id: number
  link: string
  title: string
  label: string
  galgame_id?: number
  unread_count: number
  notification_level: CommunityNotificationLevel
  last_posted_at: string
}

export interface CommunityUnreadResult {
  items: CommunityUnreadItem[]
  next_cursor: string
  total: number
}
