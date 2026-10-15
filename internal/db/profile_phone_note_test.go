package db

import (
	"errors"
	"testing"
)

func TestProfilePhoneNoteIsIndependentOfAutomaticNumbers(t *testing.T) {
	openTestDB(t)
	const iccid = "8944000012345678901"
	row, err := SaveProfilePhoneNote(" "+iccid+"FF ", "+44 7700 900123")
	if err != nil || row.ICCID != iccid || row.ManualPhoneNumber != "+447700900123" {
		t.Fatalf("row=%+v err=%v", row, err)
	}
	if err := RecordModemPhoneNumber("", iccid, "+447700900456"); err != nil {
		t.Fatal(err)
	}
	notes, err := GetProfilePhoneNotes()
	if err != nil || notes[iccid] != "+447700900123" {
		t.Fatalf("notes=%v err=%v", notes, err)
	}
	auto, err := GetPhoneNumberByIMSIOrICCID("", iccid)
	if err != nil || auto != "+447700900456" {
		t.Fatalf("auto=%q err=%v", auto, err)
	}
	if _, err := SaveProfilePhoneNote(iccid, "invalid"); !errors.Is(err, ErrInvalidPhoneNote) {
		t.Fatalf("err=%v", err)
	}
	notes, _ = GetProfilePhoneNotes()
	if notes[iccid] != "+447700900123" {
		t.Fatal("invalid input overwrote note")
	}
	if _, err := SaveProfilePhoneNote(iccid, ""); err != nil {
		t.Fatal(err)
	}
	notes, _ = GetProfilePhoneNotes()
	if notes[iccid] != "" {
		t.Fatal("note was not cleared")
	}
	auto, _ = GetPhoneNumberByIMSIOrICCID("", iccid)
	if auto != "+447700900456" {
		t.Fatal("clearing note changed automatic number")
	}
}
