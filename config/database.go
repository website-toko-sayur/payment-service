package config

import (
	"fmt"
	"payment-service/internal/core/domain/model"

	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Postgres struct {
	DB *gorm.DB
}

func (cfg Config) ConnectionPostgres() (*Postgres, error) {
	dbConnString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.Psql.User,
		cfg.Psql.Password,
		cfg.Psql.Host,
		cfg.Psql.Port,
		cfg.Psql.DBName)

	db, err := gorm.Open(postgres.Open(dbConnString), &gorm.Config{})
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "config.ConnectionPostgres").
			Str("psql_host", cfg.Psql.Host).
			Msg("Failed to connect to database")
		return nil, err
	}

	db.AutoMigrate(&model.Payment{}, &model.PaymentLog{})
	sqlDB, err := db.DB()
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "config.ConnectionPostgres").
			Msg("Failed to get database connection")
		return nil, err
	}

	sqlDB.SetMaxOpenConns(cfg.Psql.DBMaxOpen)
	sqlDB.SetMaxIdleConns(cfg.Psql.DBMaxIdle)

	return &Postgres{DB: db}, nil
}
