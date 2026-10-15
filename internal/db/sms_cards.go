package db

import "time"

// SMSCard contains the most recently recorded metadata for a historical card.
type SMSCard struct {
	ICCID       string `gorm:"column:iccid" json:"iccid"`
	LocalPhone  string `json:"local_phone"`
	ProfileName string `json:"profile_name"`
}

func GetSMSCards() ([]SMSCard, error) {
	cards := []SMSCard{}
	if DB == nil {
		return cards, nil
	}
	err := DB.Raw(`SELECT history.iccid,
 COALESCE((SELECT local_phone FROM sms WHERE iccid = history.iccid AND local_phone <> '' ORDER BY timestamp DESC, id DESC LIMIT 1), '') AS local_phone,
 COALESCE((SELECT profile_name FROM sms WHERE iccid = history.iccid AND profile_name <> '' ORDER BY timestamp DESC, id DESC LIMIT 1), '') AS profile_name
 FROM (SELECT DISTINCT iccid FROM sms WHERE iccid <> '') AS history ORDER BY history.iccid`).Scan(&cards).Error
	return cards, err
}

// GetSMSContactPage orders equal timestamps by the complete conversation key.
func GetSMSContactPage(iccid string, limit int, before *time.Time, beforeICCID, beforePeer string) ([]SMSContact, error) {
	rows := []SMSContact{}
	if DB == nil {
		return rows, nil
	}
	if limit <= 0 || limit > 200 {
		limit = 200
	}
	q := DB.Model(&SMSContact{})
	if iccid != "" {
		q = q.Where("iccid = ?", iccid)
	}
	if before != nil {
		q = q.Where("last_timestamp < ? OR (last_timestamp = ? AND (iccid < ? OR (iccid = ? AND peer < ?)))", *before, *before, beforeICCID, beforeICCID, beforePeer)
	}
	err := q.Order("last_timestamp DESC, iccid DESC, peer DESC").Limit(limit).Find(&rows).Error
	return rows, err
}
