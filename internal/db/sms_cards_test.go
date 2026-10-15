package db

import (
	"fmt"
	"testing"
	"time"
)

func TestSMSCardsAndThreadsKeepBothSidesOfConversation(t *testing.T) {
	openTestDB(t)
	at := time.Now()
	for _, side := range []struct{ card, imsi, own, peer string }{
		{"CARD-A", "IMSI-A", "+447700900001", "+447700900002"},
		{"CARD-B", "IMSI-B", "+447700900002", "+447700900001"},
	} {
		for _, direction := range []int{1, 2} {
			sender, recipient := side.peer, side.own
			if direction == 2 {
				sender, recipient = recipient, sender
			}
			if err := SaveSMSForIdentity(SMSRecord{Identity: SMSIdentity{ICCID: side.card, IMSI: side.imsi}, LocalPhone: side.own, ProfileName: side.card, Sender: sender, Recipient: recipient, Content: "hello", Type: direction, Timestamp: at}); err != nil {
				t.Fatal(err)
			}
		}
	}
	cards, err := GetSMSCards()
	if err != nil || len(cards) != 2 {
		t.Fatalf("cards=%+v err=%v", cards, err)
	}
	if cards[0].ICCID != "CARD-A" || cards[1].ICCID != "CARD-B" {
		t.Fatalf("historical card identities were lost: %+v", cards)
	}
	if cards[0].LocalPhone != "+447700900001" || cards[1].LocalPhone != "+447700900002" {
		t.Fatalf("historical card metadata mixed: %+v", cards)
	}
	threads, err := GetSMSContactPage("", 200, nil, "", "")
	if err != nil || len(threads) != 2 {
		t.Fatalf("threads=%+v err=%v", threads, err)
	}
	own, err := GetSMSContactPage("CARD-A", 200, nil, "", "")
	if err != nil || len(own) != 1 || own[0].Peer != "+447700900002" {
		t.Fatalf("own=%+v err=%v", own, err)
	}
	messages, err := GetSMSByICCIDAndPeer("CARD-A", "+447700900002", 20, nil, 0)
	if err != nil || len(messages) != 2 || messages[0].Type == messages[1].Type {
		t.Fatalf("messages=%+v err=%v", messages, err)
	}
}

func TestSMSCardsAndCursorIncludeAllHistoryWithEqualTimestamps(t *testing.T) {
	openTestDB(t)
	at := time.Now().Truncate(time.Second)
	for i := 0; i < 205; i++ {
		identity := SMSIdentity{ICCID: fmt.Sprintf("CARD-%03d", i), IMSI: fmt.Sprintf("IMSI-%03d", i)}
		if err := SaveSMSForIdentity(SMSRecord{Identity: identity, Sender: "+10086", Content: "history", Type: 1, Timestamp: at}); err != nil {
			t.Fatal(err)
		}
	}
	cards, err := GetSMSCards()
	if err != nil || len(cards) != 205 {
		t.Fatalf("cards=%d err=%v", len(cards), err)
	}
	first, err := GetSMSContactPage("", 200, nil, "", "")
	if err != nil || len(first) != 200 {
		t.Fatalf("first=%d err=%v", len(first), err)
	}
	last := first[len(first)-1]
	second, err := GetSMSContactPage("", 200, &last.LastTimestamp, last.ICCID, last.Peer)
	if err != nil || len(second) != 5 {
		t.Fatalf("second=%d err=%v", len(second), err)
	}
	seen := map[string]bool{}
	for _, thread := range append(first, second...) {
		if seen[thread.ICCID] {
			t.Fatalf("duplicate card %s", thread.ICCID)
		}
		seen[thread.ICCID] = true
	}
}
