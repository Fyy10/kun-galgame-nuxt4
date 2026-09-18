import type { components } from '#shared/types/api/v1'
import { useApi } from './useApi'

type Equal<A, B> =
  (<T>() => T extends A ? 1 : 2) extends <T>() => T extends B ? 1 : 2
    ? true
    : false

const _unused = () => {
  const result = useApi('k', (api) => api.GET('/topics', {}))
  const _data: Equal<
    typeof result.data.value,
    components['schemas']['ListTopicSummary'] | undefined
  > = true
  return _data
}

void _unused
