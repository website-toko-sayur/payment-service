package entity

type OrderHttpClientResponse struct {
	Message string                  `json:"message"`
	Data    OrderDetailHttpResponse `json:"data"`
}

type GetOrderIDByCodeResponse struct {
	Message string `json:"message"`
	Data    struct {
		OrderID int `json:"orderID"`
	} `json:"data"`
}

type OrderDetailHttpResponse struct {
	ID            int64         `json:"id"`
	OrderCode     string        `json:"order_code"`
	ProductImage  string        `json:"product_image"`
	OrderDatetime string        `json:"order_datetime"`
	Status        string        `json:"order_status"`
	PaymentMethod string        `json:"payment_method"`
	ShippingFee   int64         `json:"shipping_fee"`
	Remarks       string        `json:"remarks"`
	ShippingType  string        `json:"shipping_type"`
	TotalAmount   int64         `json:"total_amount"`
	Customer      CustomerOrder `json:"customer"`
	OrderDetail   []OrderDetail `json:"order_detail"`
}

type CustomerOrder struct {
	CustomerName    string `json:"customer_name"`
	CustomerPhone   string `json:"customer_phone"`
	CustomerAddress string `json:"customer_address"`
	CustomerEmail   string `json:"customer_email"`
	CustomerID      int64  `json:"customer_id"`
}

type OrderDetail struct {
	ProductName  string `json:"product_name"`
	ProductImage string `json:"product_image"`
	ProductPrice int64  `json:"product_price"`
	Quantity     int64  `json:"quantity"`
}
