package device

import (
	"context"
	"testing"
	"time"

	"github.com/iniwex5/vowifi-go/runtimehost/eventhost"
	"github.com/yibaiba/hideck/internal/db"
	"github.com/yibaiba/hideck/internal/esim"
)

func snapshotTestWorker(t *testing.T) (*Pool, *Worker) {
	t.Helper()
	initDevicePhoneNumberTestDB(t)
	p := NewPool(nil)
	t.Cleanup(p.cancel)
	w := &Worker{ID: "snapshot-device", Pool: p, EsimMgr: &esim.Manager{}}
	w.state.Identity.ICCID = "8944000000000000011"
	w.state.Identity.IMSI = "234150000000011"
	p.workers[w.ID] = w
	setEsimOverviewCacheForTest(t, w.EsimMgr, &esim.EsimOverview{Profiles: []esim.EUICCProfiles{{Profiles: []esim.ProfileItem{
		{ICCID: w.state.Identity.ICCID, Name: "UK travel", State: 1},
		{ICCID: "8944000000000000022", Name: "Other profile", State: 0},
	}}}})
	if err := db.DB.Create(&db.SIMCard{ICCID: w.state.Identity.ICCID, IMSI: w.state.Identity.IMSI}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.RecordVoWiFiPhoneNumber(w.state.Identity.IMSI, w.state.Identity.ICCID, "+447700900011"); err != nil {
		t.Fatal(err)
	}
	return p, w
}

func assertSMSIdentitySnapshot(t *testing.T, want SMSIdentity, count int) {
	t.Helper()
	var rows []db.SMS
	if err := db.DB.Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != count {
		t.Fatalf("rows=%d want %d", len(rows), count)
	}
	for _, sms := range rows {
		if sms.ICCID != want.ICCID || sms.IMSI != want.IMSI || sms.ProfileName != want.ProfileName || sms.LocalPhone != want.LocalPhone {
			t.Fatalf("snapshot mismatch: %+v want %+v", sms, want)
		}
	}
}

func TestSMSSnapshotSurvivesDeviceSwitch(t *testing.T) {
	p, w := snapshotTestWorker(t)
	identity, err := p.ResolveSMSIdentity(w.ID)
	if err != nil {
		t.Fatal(err)
	}
	if identity.ProfileName != "UK travel" || identity.LocalPhone != "+447700900011" {
		t.Fatalf("snapshot=%+v", identity)
	}
	record, err := w.SMSRecordSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	w.state.Identity.ICCID, w.state.Identity.IMSI = "8944000000000000022", "234150000000022"
	for _, status := range []int{2, 3} {
		if err := SaveSentSMS(record, "+10086", "outbound", status, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.persistReceivedSMS(identity, inboundSMSRecord{Sender: "+10086", Content: "inbound", Timestamp: time.Now()}); err != nil {
		t.Fatal(err)
	}
	assertSMSIdentitySnapshot(t, identity, 3)
}

func TestVoWiFiSessionSnapshotSurvivesSwitchAndMultipartCompletion(t *testing.T) {
	p, w := snapshotTestWorker(t)
	identity, err := p.ResolveSMSIdentity(w.ID)
	if err != nil {
		t.Fatal(err)
	}
	dispatch := poolVoWiFiRuntimeDispatcher{pool: p, identity: &identity}
	w.state.Identity.ICCID, w.state.Identity.IMSI = "8944000000000000022", "234150000000022"
	dispatch.Dispatch(context.Background(), eventhost.SMSReceived{DevID: w.ID, Sender: "+10086", Content: "single", Time: time.Now()})
	dispatch.Dispatch(context.Background(), eventhost.SMSSent{DevID: w.ID, TargetURI: "+10086", Content: "sent", Time: time.Now()})
	event := eventhost.SMSReceived{DevID: w.ID, Sender: "+10086", Content: "part", Time: time.Now(), FragmentSessionKey: "session", Incomplete: true}
	dispatch.Dispatch(context.Background(), event)
	// Subsequent metadata changes must not rewrite the first fragment's snapshot.
	if err := db.RecordVoWiFiPhoneNumber(identity.IMSI, identity.ICCID, "+447700900099"); err != nil {
		t.Fatal(err)
	}
	event.Content, event.Incomplete = "complete", false
	dispatch.Dispatch(context.Background(), event)
	assertSMSIdentitySnapshot(t, identity, 3)
}

func TestSMSUnknownMetadataStaysEmpty(t *testing.T) {
	p, w := snapshotTestWorker(t)
	w.EsimMgr = nil
	w.state.Identity.ICCID, w.state.Identity.IMSI = "8944000000000000033", "234150000000033"
	identity, err := p.ResolveSMSIdentity(w.ID)
	if err != nil {
		t.Fatal(err)
	}
	if identity.LocalPhone != "" || identity.ProfileName != "" {
		t.Fatalf("unexpected metadata: %+v", identity)
	}
	if err := w.persistReceivedSMS(identity, inboundSMSRecord{Sender: "+10086", Content: "unknown", Timestamp: time.Now()}); err != nil {
		t.Fatal(err)
	}
	assertSMSIdentitySnapshot(t, identity, 1)
}
