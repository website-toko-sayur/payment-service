package model

import "time"

type PaymentLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	PaymentID uint      `gorm:"not null;index" json:"payment_id"`
	Status    string    `gorm:"type:varchar(50);not null" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
