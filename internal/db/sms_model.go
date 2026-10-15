package db

import "time"

// SMS 短信表 (关联 IMSI)
type SMS struct {
	ProfileName        string    `gorm:"column:profile_name" json:"profile_name"`
	ID                 uint      `gorm:"primaryKey" json:"id"`
	IMSI               string    `gorm:"column:imsi;index:idx_sms_imsi_peer_ts,priority:1;index:idx_sms_imsi_ts,priority:1;index:idx_sms_fragment_session,priority:1" json:"imsi"`
	ICCID              string    `gorm:"column:iccid;index" json:"iccid"`
	Peer               string    `gorm:"column:peer;index:idx_sms_imsi_peer_ts,priority:2" json:"peer"`
	LocalPhone         string    `gorm:"column:local_phone;index" json:"local_phone"`
	Sender             string    `json:"sender"`
	Recipient          string    `json:"recipient"`
	Content            string    `json:"content"`
	Type               int       `json:"type"`   // 1: 接收, 2: 发送
	Status             int       `json:"status"` // 0: 未读, 1: 已读, 2: 发送成功, 3: 发送失败
	Timestamp          time.Time `gorm:"index:idx_sms_imsi_peer_ts,priority:3,sort:desc;index:idx_sms_ts,sort:desc;index:idx_sms_imsi_ts,priority:2,sort:desc" json:"timestamp"`
	CreatedAt          time.Time `json:"created_at"`
	FragmentSessionKey string    `gorm:"column:fragment_session_key;index:idx_sms_fragment_session,priority:2" json:"-"`
	Incomplete         bool      `gorm:"column:incomplete" json:"incomplete,omitempty"`
}
