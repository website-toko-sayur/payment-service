package config

import (
	"github.com/IBM/sarama"
	"github.com/rs/zerolog/log"
)

func (cfg Config) NewKafkaConsumerGroup() sarama.ConsumerGroup {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Consumer.Return.Errors = true

	offsetReset := cfg.Kafka.AutoOffsetReset
	if offsetReset == "earliest" {
		saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	} else {
		saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	}

	brokers := cfg.Kafka.BootstrapServers
	groupID := cfg.Kafka.GroupID

	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, saramaConfig)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "config.NewKafkaConsumerGroup").
			Strs("brokers", brokers).
			Str("group_id", groupID).
			Msg("Failed to create kafka consumer group")
	}

	log.Info().
		Str("source", "config.NewKafkaConsumerGroup").
		Strs("brokers", brokers).
		Str("group_id", groupID).
		Msg("Kafka consumer group connected")

	return consumerGroup
}

func (cfg Config) NewKafkaProducer() sarama.SyncProducer {
	if !cfg.Kafka.ProducerEnabled {
		log.Info().
			Msg("Kafka producer is disabled")
		return nil
	}

	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
	saramaConfig.Producer.Retry.Max = 3

	brokers := cfg.Kafka.BootstrapServers

	producer, err := sarama.NewSyncProducer(brokers, saramaConfig)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "config.NewKafkaProducer").
			Strs("brokers", brokers).
			Msg("Failed to connect to create consumer group")

	}
	return producer
}
