package device

import (
	"strings"
	"time"

	"github.com/iniwex5/vowifi-go/runtimehost/eventhost"
	"github.com/yibaiba/hideck/internal/db"
	"github.com/yibaiba/hideck/internal/smsnotify"
)

const vowifiReceivedSMSDuplicateWindow = 30 * time.Minute

type vowifiSMSHistoryRecorder struct {
	pool     *Pool
	identity *SMSIdentity
}

type vowifiSMSRecordResult struct {
	Stored     bool
	Duplicate  bool
	Suppressed bool
	Reconciled bool
}

type receivedMultipartContext struct {
	SMSIdentity
	Timestamp time.Time
}

func (r vowifiSMSHistoryRecorder) resolveIMSI(devID, fallbackIMSI string) string {
	if r.identity != nil {
		return r.identity.IMSI
	}
	imsi := strings.TrimSpace(fallbackIMSI)
	if imsi != "" {
		return imsi
	}
	if r.pool == nil {
		return ""
	}
	w := r.pool.GetWorker(strings.TrimSpace(devID))
	if w == nil {
		return ""
	}
	if cached := strings.TrimSpace(w.GetCachedIMSI()); cached != "" {
		return cached
	}
	return strings.TrimSpace(w.GetIMSI())
}

func (r vowifiSMSHistoryRecorder) resolveICCID(devID string) string {
	if r.identity != nil {
		return r.identity.ICCID
	}
	if r.pool == nil {
		return ""
	}
	w := r.pool.GetWorker(strings.TrimSpace(devID))
	if w == nil {
		return ""
	}
	if iccid := strings.TrimSpace(w.GetCachedDeviceStatus().ICCID); iccid != "" {
		return iccid
	}
	iccid, err := db.ResolveICCIDForIMSI(r.resolveIMSI(devID, ""))
	if err != nil || strings.HasPrefix(iccid, "imsi:") {
		return ""
	}
	return strings.TrimSpace(iccid)
}

func (r vowifiSMSHistoryRecorder) snapshot(devID string) SMSIdentity {
	if r.identity != nil {
		result := *r.identity
		// IMS registration can learn the number after the session was created.
		var worker *Worker
		if r.pool != nil {
			worker = r.pool.GetWorker(devID)
		}
		return smsIdentitySnapshot(worker, result)
	}
	imsi := r.resolveIMSI(devID, "")
	iccid := r.resolveICCID(devID)
	if iccid == "" {
		iccid = db.GetICCIDForIMSI(imsi)
	}
	var worker *Worker
	if r.pool != nil {
		worker = r.pool.GetWorker(devID)
	}
	return smsIdentitySnapshot(worker, SMSIdentity{IMSI: imsi, ICCID: iccid})
}

func (r vowifiSMSHistoryRecorder) eventTime(at time.Time) time.Time {
	if at.IsZero() {
		return time.Now()
	}
	return at
}

func (r vowifiSMSHistoryRecorder) RecordReceived(e eventhost.SMSReceived) (vowifiSMSRecordResult, error) {
	if smsnotify.ShouldSuppressReceivedSMS(e.Content) {
		return vowifiSMSRecordResult{Suppressed: true}, nil
	}
	identity := r.snapshot(e.DevID)
	imsi := identity.IMSI
	if imsi == "" {
		return vowifiSMSRecordResult{}, nil
	}
	localPhone := identity.LocalPhone
	ts := r.eventTime(e.Time)
	if strings.TrimSpace(e.FragmentSessionKey) != "" {
		return r.recordReceivedMultipart(e, receivedMultipartContext{
			SMSIdentity: identity, Timestamp: ts,
		})
	}

	dup, err := db.HasDuplicateReceivedSMS(imsi, localPhone, e.Sender, localPhone, e.Content, ts, vowifiReceivedSMSDuplicateWindow)
	if err != nil {
		return vowifiSMSRecordResult{}, err
	}
	if dup {
		return vowifiSMSRecordResult{Duplicate: true}, nil
	}

	record := smsRecordForSnapshot(identity)
	record.Sender, record.Recipient, record.Content = strings.TrimSpace(e.Sender), localPhone, e.Content
	record.Type, record.Status, record.Timestamp = 1, 0, ts
	err = db.SaveSMSForIdentity(record)
	if err != nil {
		return vowifiSMSRecordResult{}, err
	}
	return vowifiSMSRecordResult{Stored: true}, nil
}

func (r vowifiSMSHistoryRecorder) recordReceivedMultipart(
	e eventhost.SMSReceived,
	ctx receivedMultipartContext,
) (vowifiSMSRecordResult, error) {
	result, err := db.SaveReceivedMultipartSMS(db.ReceivedMultipartSMS{
		ICCID: ctx.ICCID, ProfileName: ctx.ProfileName,
		IMSI: ctx.IMSI, LocalPhone: ctx.LocalPhone, Sender: strings.TrimSpace(e.Sender),
		Recipient: ctx.LocalPhone, Content: e.Content, Timestamp: ctx.Timestamp,
		FragmentSessionKey: strings.TrimSpace(e.FragmentSessionKey),
		Incomplete:         e.Incomplete,
	})
	if err != nil {
		return vowifiSMSRecordResult{}, err
	}
	return vowifiSMSRecordResult{
		Stored:    result.Created || result.Reconciled,
		Duplicate: result.Duplicate, Reconciled: result.Reconciled,
	}, nil
}

func (r vowifiSMSHistoryRecorder) RecordSent(e eventhost.SMSSent) error {
	identity := r.snapshot(e.DevID)
	imsi := identity.IMSI
	if imsi == "" {
		return nil
	}
	return SaveSentSMS(smsRecordForSnapshot(identity), e.TargetURI, e.Content, 2, r.eventTime(e.Time))
}

func (r vowifiSMSHistoryRecorder) RecordLocalNumberLearned(e eventhost.LocalNumberLearned) error {
	imsi := r.resolveIMSI(e.DevID, e.IMSI)
	number := strings.TrimSpace(e.Number)
	if number == "" {
		return nil
	}
	iccid := r.resolveICCID(e.DevID)
	if imsi == "" && iccid == "" {
		return nil
	}
	return db.RecordVoWiFiPhoneNumber(imsi, iccid, number)
}
