package model

import "time"

type Payment struct {
	ID               uint         `gorm:"primaryKey" json:"id"`
	OrderID          uint         `gorm:"not null" json:"order_id"`
	UserID           uint         `gorm:"not null" json:"user_id"`
	PaymentMethod    string       `gorm:"type:varchar(50);not null" json:"payment_method"`
	PaymentStatus    string       `gorm:"type:varchar(50);not null" json:"payment_status"`
	PaymentGatewayID *string      `gorm:"type:varchar(50);null" json:"payment_gateway_id,omitempty"`
	GrossAmount      float64      `gorm:"type:decimal(10,2);not null" json:"gross_amount"`
	PaymentURL       *string      `gorm:"type:text;null" json:"payment_url,omitempty"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
	DeletedAt        *time.Time   `gorm:"index" json:"deleted_at,omitempty"`
	PaymentLogs      []PaymentLog `gorm:"foreignKey:PaymentID;constraint:OnDelete:CASCADE"`
}
