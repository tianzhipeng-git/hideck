import type { VoWiFiRuntimeState } from '../types/api'

export const SMS_DOWNLINK_PENDING = 'IMS SMS downlink is awaiting verification'

export function smsDownlinkPending(runtime?: VoWiFiRuntimeState): boolean {
  return runtime?.ims_ready === true && runtime.sms_ready === false
    && runtime.sms_ready_reason === SMS_DOWNLINK_PENDING
}
