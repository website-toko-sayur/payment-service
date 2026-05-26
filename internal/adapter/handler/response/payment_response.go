package response

type PaymentListResponse struct {
	ID            uint64  `json:"id"`
	OrderCode     string  `json:"order_code"`
	PaymentStatus string  `json:"payment_status"`
	PaymentMethod string  `json:"payment_method"`
	GrossAmount   float64 `json:"gross_amount"`
	ShippingType  string  `json:"shipping_type"`
}

type PaymentDetailResponse struct {
	ID              int64   `json:"id"`
	OrderCode       string  `json:"order_code"`
	PaymentMethod   string  `json:"payment_method"`
	PaymentStatus   string  `json:"payment_status"`
	GrossAmount     float64 `json:"gross_amount"`
	ShippingType    string  `json:"shipping_type"`
	PaymentAt       string  `json:"payment_at"`
	OrderAt         string  `json:"order_at"`
	OrderRemarks    string  `json:"order_remarks"`
	CustomerName    string  `json:"customer_name"`
	CustomerAddress string  `json:"customer_address"`
}
