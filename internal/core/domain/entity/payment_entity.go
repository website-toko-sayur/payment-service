package entity

type PaymentEntity struct {
	ID                uint
	OrderID           uint
	UserID            uint
	PaymentMethod     string
	PaymentStatus     string
	PaymentGatewayID  string
	GrossAmount       float64
	PaymentURL        string
	PaymentLogs       []PaymentLogEntity
	PaymentAt         string
	Remarks           string
	OrderCode         string
	OrderShippingType string
	CustomerName      string
	CustomerEmail     string
	CustomerAddress   string
	OrderAt           string
	OrderRemarks      string
	OrderStatus       string
}

type PaymentQueryStringRequest struct {
	Limit     int64
	Page      int64
	UserID    int64
	Status    string
	OrderType string
	OrderBy   string
	Search    string
}
