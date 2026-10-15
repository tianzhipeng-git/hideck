import assert from 'node:assert/strict'
import test from 'node:test'
import type { SmsCard } from '../src/types/sms'
import { createSmsCardChannels, createSmsConversationContext, smsIdentityLabel, normalizeSmsUnreadCount, smsPeerInitial, smsUnreadBadge } from '../src/utils/smsPresentation'

const card = (iccid: string, canSend = false): SmsCard => ({ iccid, local_phone: '+447700900011', profile_name: '旅行卡', device_id: canSend ? 'reader' : '', device_name: canSend ? '读卡器' : '', can_send: canSend, send_reason: canSend ? '当前可发送' : '卡未激活或设备离线，仍可查看历史' })

test('inactive historical cards remain in navigation when another card is activated', () => {
  const rows = createSmsCardChannels([card('8944000011'), card('8944000022', true)])
  assert.deepEqual(rows.map(row => row.id), ['all', '8944000011', '8944000022'])
  assert.equal(rows[1].online, false)
  assert.equal(rows[2].online, true)
  assert.match(rows[1].detail, /000011/)
  assert.match(rows[2].detail, /000022/)
})

test('conversation identity is historical while send eligibility belongs to that exact card', () => {
  const thread = { iccid: '8944000011', localPhone: '+447700900000', profileName: '旧名称' }
  const offline = createSmsConversationContext({ thread, cards: [card('8944000022', true)] })
  assert.equal(offline.canSend, false)
  assert.match(offline.identityLabel, /旧名称/)
  assert.match(offline.identityLabel, /900000/)
  const online = createSmsConversationContext({ thread, cards: [card('8944000011', true)] })
  assert.equal(online.canSend, true)
  assert.equal(online.identityLabel, offline.identityLabel)
})

test('unknown number and profile still expose a distinguishing card tail', () => {
  assert.equal(smsIdentityLabel({ iccid: '8944000011' }), '号码未知 · SIM · 卡尾号 000011')
})

test('unread badges and initials are explicit and bounded', () => {
  assert.equal(smsUnreadBadge(3), 3)
  assert.equal(smsUnreadBadge(-2), 0)
  assert.equal(smsUnreadBadge(3, true), 0)
  assert.equal(smsPeerInitial(' giffgaff '), 'G')
  assert.equal(smsPeerInitial(''), '—')
  assert.equal(normalizeSmsUnreadCount(undefined), 0)
})

test('conversation uses the card number when its historical number is unknown', () => {
  const context = createSmsConversationContext({ thread: { iccid: '8944000011', localPhone: '' }, cards: [card('8944000011')] })
  assert.match(context.identityLabel, /447700900011/)
  const other = createSmsConversationContext({ thread: { iccid: '8944000022' }, cards: [card('8944000011')] })
  assert.match(other.identityLabel, /号码未知/)
})
