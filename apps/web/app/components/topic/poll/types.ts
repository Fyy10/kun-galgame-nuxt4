import type { Poll } from '#shared/utils/api/schemas'

export interface PollFormOption {
  id?: string
  text: string
  _status?: 'new' | 'existing' | 'deleted'
}

export interface PollFormData {
  title: string
  description: string
  options: PollFormOption[]
  choice_type: Poll['choice_type']
  min_choice: number
  max_choice: number
  closes_at?: string
  result_visibility: Poll['result_visibility']
  is_anonymous: boolean
  can_change_vote: boolean
}
