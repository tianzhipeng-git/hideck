package db

import (
	"errors"
	"strings"

	"gorm.io/gorm/clause"
)

// ProfilePhoneNote belongs to an ICCID even before the profile has an IMSI.
// Automatically learned phone numbers never write this table.
type ProfilePhoneNote struct {
	ICCID             string `gorm:"column:iccid;primaryKey" json:"iccid"`
	ManualPhoneNumber string `json:"manual_phone_number"`
}

var ErrInvalidPhoneNote = errors.New("请输入 6–15 位电话号码，可带 + 国际区号")

func SaveProfilePhoneNote(iccid, phone string) (ProfilePhoneNote, error) {
	row := ProfilePhoneNote{ICCID: CanonicalICCID(iccid), ManualPhoneNumber: canonicalLocalPhone(strings.TrimSpace(phone))}
	if row.ICCID == "" {
		return row, errors.New("ICCID 不能为空")
	}
	if row.ManualPhoneNumber != "" && !looksLikePhoneNumber(row.ManualPhoneNumber) {
		return row, ErrInvalidPhoneNote
	}
	err := DB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "iccid"}}, DoUpdates: clause.AssignmentColumns([]string{"manual_phone_number"})}).Create(&row).Error
	return row, err
}

func GetProfilePhoneNotes() (map[string]string, error) {
	notes := map[string]string{}
	if DB == nil {
		return notes, nil
	}
	var rows []ProfilePhoneNote
	if err := DB.Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		notes[row.ICCID] = row.ManualPhoneNumber
	}
	return notes, nil
}
