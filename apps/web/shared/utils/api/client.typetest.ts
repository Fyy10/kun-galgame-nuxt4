import type { components } from '../../types/api/v1'
import { createApiClient } from './client'

type Equal<A, B> =
  (<T>() => T extends A ? 1 : 2) extends <T>() => T extends B ? 1 : 2
    ? true
    : false

declare const api: ReturnType<typeof createApiClient>
declare const topic: components['schemas']['TopicSummary']

// @ts-expect-error no such path
api.GET('/topic')

// @ts-expect-error unknown query key
api.GET('/topics', { params: { query: { sortt: 'bumped_desc' } } })

// @ts-expect-error not a sort token
api.GET('/topics', { params: { query: { sort: 'bumped' } } })

// @ts-expect-error misspelled field
const _titel = topic.titel
void _titel

const _assertGetTopics = async () => {
  const client = createApiClient({ origin: 'https://www.kungal.com' })
  const res = await client.GET('/topics', {})
  const _data: Equal<
    NonNullable<typeof res.data>,
    components['schemas']['ListTopicSummary']
  > = true
  const _error: Equal<
    NonNullable<typeof res.error>,
    components['schemas']['Problem']
  > = true
  return [_data, _error] as const
}

void _assertGetTopics
