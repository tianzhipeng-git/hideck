package api

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/yibaiba/hideck/internal/config"
	"github.com/yibaiba/hideck/internal/db"
	"github.com/yibaiba/hideck/internal/device"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestSMSCardHistorySurvivesCurrentProfileChange(t *testing.T) {
	previous := db.DB
	if err := db.Init(filepath.Join(t.TempDir(), "sms-cards.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.DB = previous })
	for _, id := range []string{"CARD-A", "CARD-B"} {
		if err := db.SaveSMSForIdentity(db.SMSRecord{Identity: db.SMSIdentity{ICCID: id, IMSI: "IMSI-" + id}, LocalPhone: "+447700900001", ProfileName: id, Sender: "+10086", Content: "history", Type: 1, Timestamp: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}
	cardA := "CARD-A"
	if err := db.DB.Create(&db.Device{IMEI: "reader", Alias: "reader", CurrentICCID: &cardA}).Error; err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"CARD-A", "CARD-B"} {
		if err := db.DB.Create(&db.SIMCard{ICCID: id, IMSI: "IMSI-" + id}).Error; err != nil {
			t.Fatal(err)
		}
	}
	pool := device.NewPool(&config.Config{})
	pool.AttachWorkerForTest(&device.Worker{ID: "reader", Pool: pool})
	server := &Server{pool: pool}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/sms/cards", server.handleGetSMSCards)
	router.GET("/sms/contacts", server.handleGetSMSContacts)
	router.GET("/sms/thread", server.handleGetSMSThread)
	for _, active := range []string{"CARD-A", "CARD-B"} {
		if err := db.DB.Model(&db.Device{}).Where("imei = ?", "reader").Update("iccid", active).Error; err != nil {
			t.Fatal(err)
		}
		result := httptest.NewRecorder()
		router.ServeHTTP(result, httptest.NewRequest(http.MethodGet, "/sms/cards", nil))
		var cards []smsCardView
		if result.Code != 200 {
			t.Fatal(result.Body.String())
		}
		if err := json.Unmarshal(result.Body.Bytes(), &cards); err != nil {
			t.Fatal(err)
		}
		if len(cards) != 2 || cards[0].ICCID != "CARD-A" || cards[1].ICCID != "CARD-B" {
			t.Fatalf("cards=%+v", cards)
		}
		for _, card := range cards {
			if card.ICCID != active && (card.DeviceID != "" || card.CanSend) {
				t.Fatalf("inactive card=%+v", card)
			}
		}
		result = httptest.NewRecorder()
		router.ServeHTTP(result, httptest.NewRequest(http.MethodGet, "/sms/contacts?iccid=CARD-A", nil))
		var contacts []SMSContactWithDevice
		if err := json.Unmarshal(result.Body.Bytes(), &contacts); err != nil {
			t.Fatal(err)
		}
		if result.Code != 200 || len(contacts) != 1 || contacts[0].ICCID != "CARD-A" || contacts[0].ProfileName != "CARD-A" {
			t.Fatalf("contacts=%+v code=%d", contacts, result.Code)
		}
		result = httptest.NewRecorder()
		router.ServeHTTP(result, httptest.NewRequest(http.MethodGet, "/sms/thread?iccid=CARD-A&peer=%2B10086", nil))
		if result.Code != 200 {
			t.Fatal(result.Body.String())
		}
	}
}
