package message

import (
	"encoding/json"
	"payment-service/internal/core/domain/model"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog/log"
)

type Producer[T model.Event] struct {
	Producer sarama.SyncProducer
	Topic    string
}

func (p *Producer[T]) GetTopic() *string {
	return &p.Topic
}

func (p *Producer[T]) Send(event T) error {
	value, err := json.Marshal(event)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.message.Producer.Send").
			Msg("Failed to marshal event")
		return err
	}

	message := &sarama.ProducerMessage{
		Topic: p.Topic,
		Key:   sarama.StringEncoder(event.GetId()),
		Value: sarama.ByteEncoder(value),
	}

	partition, offset, err := p.Producer.SendMessage(message)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.message.Producer.Send").
			Msg("Failed to produce message")
		return err
	}

	log.Debug().
		Str("source", "internal.adapter.message.Producer.Send").
		Str("topic", p.Topic).
		Int32("partition", partition).
		Int64("offset", offset).
		Msg("Kafka message published")

	return nil
}
