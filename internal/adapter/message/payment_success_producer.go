package message

import (
	"payment-service/config"
	"payment-service/internal/core/domain/model"

	"github.com/IBM/sarama"
)

type PaymentSuccessProducer struct {
	Producer[*model.PaymentSuccessEvent]
}

func NewPaymentSuccessProducer(producer sarama.SyncProducer, cfg *config.Config) *PaymentSuccessProducer {
	return &PaymentSuccessProducer{
		Producer: Producer[*model.PaymentSuccessEvent]{
			Producer: producer,
			Topic:    cfg.Topic.PaymentSuccess,
		},
	}
}
