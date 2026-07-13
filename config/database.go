package config

import (
	"fmt"
	"net/url"
	"payment-service/internal/core/domain/model"

	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Postgres struct {
	DB *gorm.DB
}

func (cfg Config) ConnectionPostgres() (*Postgres, error) {
	dbConfig := cfg.Psql
	encodedPassword := url.QueryEscape(dbConfig.Password)

	// 1. Connect TANPA database (default ke postgres)
	baseURI := fmt.Sprintf("postgresql://%s:%s@%s:%s/postgres?sslmode=disable",
		dbConfig.User,
		encodedPassword,
		dbConfig.Host,
		dbConfig.Port,
	)

	baseDB, err := gorm.Open(postgres.Open(baseURI), &gorm.Config{})
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "config.ConnectionPostgres").
			Str("psql_host", cfg.Psql.Host).
			Msg("Failed to connect to database")
		return nil, err
	}

	// 2. Create database jika belum ada
	createDBQuery := fmt.Sprintf("CREATE DATABASE %s", dbConfig.DBName)
	err = baseDB.Exec(createDBQuery).Error
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "config.ConnectionPostgres").
			Str("psql_host", cfg.Psql.Host).
			Msg("Failed to create database")
	} else {
		log.Info().
			Str("source", "config.ConnectionPostgres").
			Str("psql_host", cfg.Psql.Host).
			Str("database ready", dbConfig.DBName)
	}

	// 3. Connect ke database target
	targetURI := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		dbConfig.User,
		encodedPassword,
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.DBName,
	)

	db, err := gorm.Open(postgres.Open(targetURI), &gorm.Config{})
	if err != nil {
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
