package entity

type Webhook struct {
	VANumbers         []VANumber      `json:"va_numbers"`
	TransactionTime   string          `json:"transaction_time"` // Waktu transaksi dibuat
	TransactionStatus string          `json:"transaction_status"`
	TransactionID     string          `json:"transaction_id"` // ID transaksi dari payment gateway
	StatusMessage     string          `json:"status_message"`
	StatusCode        string          `json:"status_code"`     // Kode status dari gateway
	SignatureKey      string          `json:"signature_key"`   // Digunakan untuk validasi webhook
	SettlementTime    string          `json:"settlement_time"` // Waktu transaksi sukses (kalau sudah paid)
	PaymentType       string          `json:"payment_type"`    // Jenis pembayaran (bank_transfer, qris, gopay, dll)
	PaymentAmount     []PaymentAmount `json:"payment_amount"`
	OrderID           string          `json:"order_id"`     // ID order internal kamu
	MerchantID        string          `json:"merchant_id"`  // ID merchant di payment gateway
	GrossAmount       string          `json:"gross_amount"` // Total tagihan
	FraudStatus       string          `json:"fraud_status"` // Status fraud check (accept, challenge, dll)
	Currency          string          `json:"currency"`     // Mata uang (IDR)
	Acquirer          *string         `json:"acquirer"`     // Pihak acquiring bank / e-wallet
}

type VANumber struct {
	VaNumber string `json:"va_number"` // Daftar Virtual Account (khusus bank transfer)
	Bank     string `json:"bank"`
}

type PaymentAmount struct {
	PaidAt *string `json:"paid_at"`
	Amount *string `json:"amount"`
}
