package model

type PaymentSuccessEvent struct {
	OrderID       string `json:"orderID"`
	PaymentMethod string `json:"paymentMethod"`
}

func (u *PaymentSuccessEvent) GetId() string {
	return u.OrderID
}
