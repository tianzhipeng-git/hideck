package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yibaiba/hideck/internal/db"
)

func (s *Server) handleGetProfilePhoneNotes(c *gin.Context) {
	notes, err := db.GetProfilePhoneNotes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "读取备注号码失败"})
		return
	}
	c.JSON(http.StatusOK, notes)
}

func (s *Server) handlePutProfilePhoneNote(c *gin.Context) {
	var req struct {
		ManualPhoneNumber *string `json:"manual_phone_number" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请提供 manual_phone_number，空字符串表示清除"})
		return
	}
	row, err := db.SaveProfilePhoneNote(c.Param("iccid"), *req.ManualPhoneNumber)
	if errors.Is(err, db.ErrInvalidPhoneNote) {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "保存备注号码失败"})
		return
	}
	c.JSON(http.StatusOK, row)
}
