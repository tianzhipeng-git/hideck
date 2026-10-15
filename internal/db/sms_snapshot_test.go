package db

import (
	"testing"
	"time"
)

func TestSMSPersistsProfileNameWithoutInferringUnknownMetadata(t *testing.T) {
	openTestDB(t)
	for _, name := range []string{"UK travel", ""} {
		err := SaveSMSForIdentity(SMSRecord{
			Identity:    SMSIdentity{IMSI: "234150000000011", ICCID: "8944000000000000011"},
			ProfileName: name, Sender: "+10086", Content: "test", Type: 1, Timestamp: time.Now(),
		})
		if err != nil {
			t.Fatal(err)
		}
		var sms SMS
		if err := DB.Order("id DESC").First(&sms).Error; err != nil {
			t.Fatal(err)
		}
		if sms.ProfileName != name || sms.LocalPhone != "" {
			t.Fatalf("sms=%+v", sms)
		}
	}
}

func TestMultipartKeepsOriginalCardAndProfileSnapshot(t *testing.T) {
	openTestDB(t)
	input := ReceivedMultipartSMS{IMSI: "234150000000011", ICCID: "8944000000000000011", ProfileName: "UK travel", LocalPhone: "+447700900011", Sender: "+10086", Content: "part", Timestamp: time.Now(), FragmentSessionKey: "same-message", Incomplete: true}
	first, err := SaveReceivedMultipartSMS(input)
	if err != nil {
		t.Fatal(err)
	}
	input.ProfileName, input.LocalPhone = "Renamed", "+447700900099"
	input.Content, input.Incomplete = "complete", false
	if _, err := SaveReceivedMultipartSMS(input); err != nil {
		t.Fatal(err)
	}
	var sms SMS
	if err := DB.First(&sms, first.SMSID).Error; err != nil {
		t.Fatal(err)
	}
	if sms.ICCID != "8944000000000000011" || sms.ProfileName != "UK travel" || sms.LocalPhone != "+447700900011" || sms.Content != "complete" {
		t.Fatalf("sms=%+v", sms)
	}
}
