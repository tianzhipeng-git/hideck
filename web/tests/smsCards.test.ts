import assert from 'node:assert/strict'
import test from 'node:test'
import { api } from '../src/stores/auth'
import { smsService, normalizeThread } from '../src/services/sms'
import type { SMSContact } from '../src/types/api'

const contact = (iccid: string, peer: string): SMSContact => ({ iccid, imsi: 'imsi', peer, last_sms_id: 1, last_timestamp: '2026-10-15T10:00:00Z', last_content: '收到', last_type: 1, unread_count: 0 })

test('A to B and B to A are two own-card conversations regardless of direction', () => {
  const a = normalizeThread(contact('card-A', '+447700900002'))
  const b = normalizeThread(contact('card-B', '+447700900001'))
  assert.notEqual(a.key, b.key)
  assert.equal(normalizeThread({ ...contact('card-A', '+447700900002'), last_type: 2 }).key, a.key)
})

test('card-scoped history loads beyond 200 conversations using a complete cursor, never a device', async () => {
  const original = api.get
  const calls: Record<string, string>[] = []
  api.get = (async (_url, config) => {
    const params = config?.params as Record<string, string>
    calls.push(params)
    return { data: calls.length === 1 ? Array.from({ length: 200 }, (_, i) => contact('card-A', String(300 - i))) : [contact('card-A', '100')] }
  }) as typeof api.get
  try {
    const result = await smsService.listContacts('card-A')
    assert.equal(result.ok && result.data.length, 201)
    assert.deepEqual(calls[0], { limit: '200', iccid: 'card-A' })
    assert.deepEqual(calls[1], { limit: '200', iccid: 'card-A', before_ts: '2026-10-15T10:00:00Z', before_iccid: 'card-A', before_peer: '101' })
  } finally { api.get = original }
})
