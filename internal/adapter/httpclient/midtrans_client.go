package httpclient

import (
	"payment-service/config"

	"github.com/gofiber/fiber/v3"
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
	"github.com/rs/zerolog/log"
)

type midtransClient struct {
	cfg *config.Config
}

type MidtransClientInterface interface {
	CreateTransaction(orderID string, amount int64, customerName, customerEmail string) (string, error)
}

func NewMidtransClient(cfg *config.Config) MidtransClientInterface {
	return &midtransClient{
		cfg: cfg,
	}
}

func (m *midtransClient) CreateTransaction(orderID string, amount int64, customerName string, customerEmail string) (string, error) {
	midtrans.ServerKey = m.cfg.Midtrans.ServerKey
	midtrans.Environment = midtrans.EnvironmentType(m.cfg.Midtrans.Environment)

	snapReq := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  orderID,
			GrossAmt: amount,
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName: customerName,
			Email: customerEmail,
		},
	}

	snapRes, err := snap.CreateTransaction(snapReq)
	if err != nil {
		log.Error().
			Err(err).
			Str("order_id", orderID).
			Int64("amount", amount).
			Str("customer_email", customerEmail).
			Msg("failed to create midtrans transaction")

		return "", fiber.NewError(fiber.StatusInternalServerError, "failed create transaction")
	}

	log.Info().
		Str("order_id", orderID).
		Msg("midtrans transaction created")

	return snapRes.Token, nil
}
