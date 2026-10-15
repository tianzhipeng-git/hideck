import { watch, type Ref } from 'vue'
import type { RouteLocationNormalizedLoaded } from 'vue-router'

export function useSmsRouteSync(options: {
  route: Pick<RouteLocationNormalizedLoaded, 'query' | 'name'>
  selectedCard: Ref<string>
  selectedThreadKey: Ref<string>
  targetMessageQuery: Ref<unknown>
  clearSelectedThread: (syncRoute: boolean) => void
  fetchMessagesAndThread: () => Promise<void>
}) {
  const { route } = options
  watch(() => route.query, async (query, previous) => {
    if (route.name !== 'SMS') return
    const card = typeof query.card === 'string' ? query.card : 'all'
    const contact = typeof query.contact === 'string' ? query.contact : ''
    const target = query.latest === '1' ? undefined : query.message
    const cardChanged = card !== options.selectedCard.value
    const contactChanged = query.contact !== previous.contact && contact !== options.selectedThreadKey.value
    const targetChanged = target !== options.targetMessageQuery.value
    if (!cardChanged && !contactChanged && !targetChanged) return
    options.clearSelectedThread(false)
    options.selectedCard.value = card
    options.selectedThreadKey.value = contact
    options.targetMessageQuery.value = target
    await options.fetchMessagesAndThread()
  })
}
