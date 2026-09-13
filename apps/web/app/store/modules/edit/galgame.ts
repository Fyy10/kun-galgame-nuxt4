import { defineStore } from 'pinia'
import { reactive, ref } from 'vue'
import { createEmptyLocaleMap, resetReactiveState } from '~/store/index'
import type { GalgameStorePersist } from '~/store/types/edit/galgame'

export const usePersistEditGalgameStore = defineStore(
  'KUNGalgameEditGalgame',
  () => {
    const vndb_id = ref<GalgameStorePersist['vndb_id']>('')
    const name = reactive<GalgameStorePersist['name']>({
      'en-us': '',
      'ja-jp': '',
      'zh-cn': '',
      'zh-tw': ''
    })
    const introduction = reactive<GalgameStorePersist['introduction']>({
      'en-us': '',
      'ja-jp': '',
      'zh-cn': '',
      'zh-tw': ''
    })
    // Unset, never 'sfw': a display verdict nobody looked at put 961 works on
    // catalog's SFW shelf with only explicit cover art, where the election had
    // nothing safe to elect and every viewer got the blurred stand-in instead
    // of the real cover (infra audit-cover-shelf, 2026-09-13). Omitted from
    // persistence below for the same reason — a remembered 'sfw' is no choice.
    const content_limit = ref<GalgameStorePersist['content_limit']>('')
    const age_limit = ref<GalgameStorePersist['age_limit']>('all')
    const original_language =
      ref<GalgameStorePersist['original_language']>('ja-jp')
    const aliases = ref<GalgameStorePersist['aliases']>([])
    const release_date = ref<GalgameStorePersist['release_date']>('')
    const release_date_tba = ref<GalgameStorePersist['release_date_tba']>(false)

    const resetEditGalgameStore = () => {
      vndb_id.value = ''
      resetReactiveState(name, createEmptyLocaleMap())
      resetReactiveState(introduction, createEmptyLocaleMap())
      content_limit.value = ''
      age_limit.value = 'all'
      original_language.value = 'ja-jp'
      aliases.value = []
      release_date.value = ''
      release_date_tba.value = false
    }

    return {
      vndb_id,
      name,
      introduction,
      content_limit,
      age_limit,
      original_language,
      aliases,
      release_date,
      release_date_tba,

      resetEditGalgameStore
    }
  },
  {
    persist: {
      storage: piniaPluginPersistedstate.localStorage(),
      omit: ['content_limit']
    }
  }
)
