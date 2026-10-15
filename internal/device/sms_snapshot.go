package device

import (
	"strings"
	"time"

	"github.com/yibaiba/hideck/internal/db"
)

// smsIdentitySnapshot captures metadata for the supplied card, never the active profile by name.
func smsIdentitySnapshot(worker *Worker, identity SMSIdentity) SMSIdentity {
	identity = normalizeSMSIdentity(identity)
	identity.LocalPhone, _ = db.GetPhoneNumberByIMSIOrICCID(identity.IMSI, identity.ICCID)
	if worker != nil && worker.EsimMgr != nil {
		if name, found := worker.EsimMgr.CachedProfileNameForICCID(identity.ICCID); found {
			identity.ProfileName = name
		}
	}
	return identity
}

// SMSRecordSnapshot must be taken before sending, while the intended card is selected.
func (w *Worker) SMSRecordSnapshot() (db.SMSRecord, error) {
	identity, err := w.resolveSMSIdentity()
	if err != nil {
		return db.SMSRecord{}, err
	}
	return smsRecordForSnapshot(identity), nil
}

func smsRecordForSnapshot(identity SMSIdentity) db.SMSRecord {
	return db.SMSRecord{
		Identity:   db.SMSIdentity{ICCID: identity.ICCID, IMSI: identity.IMSI},
		LocalPhone: identity.LocalPhone, ProfileName: identity.ProfileName,
	}
}

// SaveSentSMS persists a send result using the identity captured before transmission.
func SaveSentSMS(snapshot db.SMSRecord, target, content string, status int, at time.Time) error {
	snapshot.Sender = snapshot.LocalPhone
	snapshot.Recipient = strings.TrimSpace(target)
	snapshot.Content, snapshot.Type, snapshot.Status, snapshot.Timestamp = content, 2, status, at
	return db.SaveSMSForIdentity(snapshot)
}
