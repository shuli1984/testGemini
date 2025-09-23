package models

import (
	"time"
)

// LoginLog records a login attempt.
type LoginLog struct {
	ID        uint      `gorm:"primaryKey"`
	Username  string    `gorm:"size:255"`
	Success   bool
	IPAddress string    `gorm:"size:45"`
	UserAgent string    `gorm:"size:512"`
	CreatedAt time.Time
}
