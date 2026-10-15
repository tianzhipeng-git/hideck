package api

import (
	"github.com/gin-gonic/gin"
	"github.com/yibaiba/hideck/internal/db"
	"net/http"
	"sort"
)

type smsCardView struct {
	db.SMSCard
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	CanSend    bool   `json:"can_send"`
	SendReason string `json:"send_reason"`
}

func (s *Server) smsCards() ([]smsCardView, error) {
	history, err := db.GetSMSCards()
	if err != nil {
		return nil, err
	}
	byICCID := make(map[string]db.SMSCard, len(history))
	for _, card := range history {
		byICCID[card.ICCID] = card
	}
	if s.pool != nil {
		for _, worker := range s.pool.GetAllWorkers() {
			identity, err := s.pool.ResolveSMSIdentity(worker.ID)
			if err != nil {
				continue
			}
			card := byICCID[identity.ICCID]
			card.ICCID = identity.ICCID
			if identity.LocalPhone != "" {
				card.LocalPhone = identity.LocalPhone
			}
			if identity.ProfileName != "" {
				card.ProfileName = identity.ProfileName
			}
			byICCID[identity.ICCID] = card
		}
	}
	cards := make([]smsCardView, 0, len(byICCID))
	for _, card := range byICCID {
		row := smsCardView{SMSCard: card, SendReason: "卡未激活或设备离线，仍可查看历史"}
		if s.pool != nil {
			worker, _, _ := s.resolveSMSWorkerByICCID(card.ICCID)
			if worker != nil {
				row.DeviceID, row.DeviceName = worker.ID, worker.Config.Name
				if row.DeviceName == "" {
					row.DeviceName = worker.ID
				}
				if _, err := s.pool.ResolveSMSIdentity(worker.ID); err == nil {
					if s.pool.ShouldRouteSMSViaVoWiFi(worker.ID) {
						state, ok := s.pool.GetVoWiFiRuntimeState(worker.ID)
						row.CanSend = ok && (state.SMSMOReady || state.SMSReady)
						if !row.CanSend {
							row.SendReason = "此卡已激活，VoWiFi 短信发送尚未就绪"
						}
					} else {
						row.CanSend = worker.GetCachedHealthy()
						if !row.CanSend {
							row.SendReason = "此卡设备尚未就绪"
						}
					}
				}
			}
		}
		if row.CanSend {
			row.SendReason = "当前可发送"
		}
		cards = append(cards, row)
	}
	sort.Slice(cards, func(i, j int) bool { return cards[i].ICCID < cards[j].ICCID })
	return cards, nil
}

func (s *Server) handleGetSMSCards(c *gin.Context) {
	cards, err := s.smsCards()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "读取短信卡列表失败"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, cards)
}
