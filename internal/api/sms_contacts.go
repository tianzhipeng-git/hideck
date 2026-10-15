package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/yibaiba/hideck/internal/db"
	"net/http"
	"strings"
	"time"
)

type SMSContactWithDevice struct {
	db.SMSContact
	DeviceID    string `json:"device_id"`
	DeviceName  string `json:"device_name"`
	LocalPhone  string `json:"local_phone"`
	ProfileName string `json:"profile_name"`
}

func (s *Server) handleGetSMSContacts(c *gin.Context) {
	deviceID := c.Query("device_id")
	imsi := c.Query("imsi")

	limitStr := c.DefaultQuery("limit", "50")
	var limit int
	fmt.Sscanf(limitStr, "%d", &limit)

	var beforeTs *time.Time
	if v := strings.TrimSpace(c.Query("before_ts")); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			beforeTs = &t
		}
	}
	beforePeer := strings.TrimSpace(c.Query("before_peer"))

	iccid := db.CanonicalICCID(c.Query("iccid"))
	if iccid == "" && strings.TrimSpace(deviceID) != "" {
		resolved, status, msg := s.resolveSMSICCID(deviceID, imsi)
		if status != 0 {
			c.JSON(status, gin.H{"status": "error", "message": msg})
			return
		}
		iccid = resolved
	}

	contacts, err := db.GetSMSContactPage(iccid, limit, beforeTs, db.CanonicalICCID(c.Query("before_iccid")), beforePeer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "查询数据库失败: " + err.Error()})
		return
	}

	iccidDevice := s.smsDeviceInfoByICCID()

	var latest []db.SMS
	ids := make([]uint, 0, len(contacts))
	for _, contact := range contacts {
		ids = append(ids, contact.LastSMSID)
	}
	if len(ids) > 0 {
		if err := db.DB.Where("id IN ?", ids).Find(&latest).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "读取会话身份失败"})
			return
		}
	}
	snapshots := make(map[uint]db.SMS, len(latest))
	for _, sms := range latest {
		snapshots[sms.ID] = sms
	}

	enriched := make([]SMSContactWithDevice, 0, len(contacts))
	for _, ct := range contacts {
		info := iccidDevice[ct.ICCID]
		enriched = append(enriched, SMSContactWithDevice{
			SMSContact:  ct,
			DeviceID:    info.id,
			DeviceName:  info.name,
			LocalPhone:  snapshots[ct.LastSMSID].LocalPhone,
			ProfileName: snapshots[ct.LastSMSID].ProfileName,
		})
	}

	c.JSON(http.StatusOK, enriched)
}
