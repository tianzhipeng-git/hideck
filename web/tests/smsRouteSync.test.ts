import assert from 'node:assert/strict'
import test from 'node:test'
import { effectScope, nextTick, reactive, ref } from 'vue'
import type { LocationQuery } from 'vue-router'
import { useSmsRouteSync } from '../src/composables/useSmsRouteSync'

test('SMS route updates preserve local selection and handle external navigation', async () => {
  const route = reactive({ name: 'SMS', query: {} as LocationQuery })
  const selectedCard = ref('all')
  const selectedThreadKey = ref('')
  const targetMessageQuery = ref<unknown>(undefined)
  let reloads = 0
  const scope = effectScope()
  scope.run(() => useSmsRouteSync({
    route, selectedCard, selectedThreadKey, targetMessageQuery,
    clearSelectedThread: () => { selectedThreadKey.value = ''; targetMessageQuery.value = undefined },
    fetchMessagesAndThread: async () => { reloads++ }
  }))
  try {
    selectedCard.value = 'A'
    route.query = { card: 'A' }
    await nextTick()
    selectedThreadKey.value = 'A|peer'
    route.query = { card: 'A', contact: 'A|peer' }
    await nextTick()
    assert.equal(reloads, 0, 'local card/contact clicks must not trigger a second load')
    route.query = { card: 'B', contact: 'B|peer' }
    await nextTick()
    assert.equal(selectedCard.value, 'B')
    assert.equal(selectedThreadKey.value, 'B|peer')
    assert.equal(reloads, 1)
    route.query = { message: '42' }
    await nextTick()
    assert.equal(targetMessageQuery.value, '42')
    assert.equal(selectedCard.value, 'all')
    route.query = { message: '42', latest: '1' }
    await nextTick()
    assert.equal(targetMessageQuery.value, undefined)
    assert.equal(reloads, 3)
  } finally { scope.stop() }
})
