import { trackHistoryPops } from '~/utils/historyPop'

export default defineNuxtPlugin(() => {
  trackHistoryPops(useRouter())
})
