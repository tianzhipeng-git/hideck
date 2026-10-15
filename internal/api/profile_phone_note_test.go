package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yibaiba/hideck/internal/db"
)

func TestProfilePhoneNoteAPIAndSMSFallback(t *testing.T) {
	previous := db.DB
	if err := db.Init(filepath.Join(t.TempDir(), "notes.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.DB = previous })
	gin.SetMode(gin.TestMode)
	server := &Server{}
	router := gin.New()
	router.PUT("/cards/:iccid/phone-note", server.handlePutProfilePhoneNote)
	router.GET("/cards/phone-notes", server.handleGetProfilePhoneNotes)
	for _, tc := range []struct {
		body   string
		status int
	}{
		{`{}`, 400}, {`{"manual_phone_number":null}`, 400},
		{`{"manual_phone_number":"abc"}`, 400},
		{`{"manual_phone_number":"+447700900123"}`, 200},
	} {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/cards/8944000011/phone-note", strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, req)
		if recorder.Code != tc.status {
			t.Fatalf("body=%s status=%d response=%s", tc.body, recorder.Code, recorder.Body.String())
		}
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/cards/phone-notes", nil))
	var notes map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &notes); err != nil || notes["8944000011"] != "+447700900123" {
		t.Fatalf("notes=%v err=%v", notes, err)
	}
	if err := db.SaveSMSForIdentity(db.SMSRecord{Identity: db.SMSIdentity{ICCID: "8944000011", IMSI: "imsi-note"}, Sender: "10086", Content: "hello", Type: 1, Timestamp: time.Now()}); err != nil {
		t.Fatal(err)
	}
	cards, err := server.smsCards()
	if err != nil || len(cards) != 1 || cards[0].LocalPhone != "+447700900123" || cards[0].CanSend {
		t.Fatalf("cards=%+v err=%v", cards, err)
	}
	if err := db.DB.Model(&db.SMS{}).Where("iccid = ?", "8944000011").Update("local_phone", "+447700900456").Error; err != nil {
		t.Fatal(err)
	}
	cards, err = server.smsCards()
	if err != nil || cards[0].LocalPhone != "+447700900456" {
		t.Fatalf("automatic number lost: %+v err=%v", cards, err)
	}
}
