package app

import (
	"context"
	"os"
	"os/signal"
	"payment-service/config"
	"payment-service/internal/adapter/handler"
	"payment-service/internal/adapter/httpclient"
	"payment-service/internal/adapter/message"
	"payment-service/internal/adapter/repository"
	"payment-service/internal/core/service"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	fiberCors "github.com/gofiber/fiber/v3/middleware/cors"
	fiberRecover "github.com/gofiber/fiber/v3/middleware/recover"

	"github.com/rs/zerolog/log"
)

func RunServer() {
	cfg := config.NewConfig()

	config.NewLogger(cfg.App.AppEnv, cfg.App.LogLevel, cfg.App.AppName)

	db, err := cfg.ConnectionPostgres()
	if err != nil {
		log.Fatal().
			Err(err).
			Str("source", "internal.app.RunServer").
			Msg("failed connect postgres")
	}

	sqlDB, err := db.DB.DB()
	if err != nil {
		log.Fatal().
			Err(err).
			Str("source", "internal.app.RunServer").
			Msg("failed get sql db instance")
	}
	defer sqlDB.Close()

	redis, err := cfg.NewRedisClient()
	if err != nil {
		log.Fatal().
			Err(err).
			Str("source", "internal.app.RunServer").
			Msg("failed connect to redis")
	}
	defer redis.Close()

	producer := cfg.NewKafkaProducer()

	var (
		paymentSuccessProducer *message.PaymentSuccessProducer
	)

	if producer != nil {
		paymentSuccessProducer = message.NewPaymentSuccessProducer(producer, cfg)
	}

	httpClient := httpclient.NewClient(cfg)
	midtransClient := httpclient.NewMidtransClient(cfg)

	paymentRepo := repository.NewPaymentRepository(db.DB)

	jwtService := service.NewJwtService(cfg)
	paymentService := service.NewPaymentService(
		paymentRepo,
		cfg,
		httpClient,
		midtransClient,
		paymentSuccessProducer,
	)

	app := cfg.NewFiber()

	app.Use(fiberRecover.New())
	app.Use(fiberCors.New())

	app.Get("/api/check", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	handler.NewPaymentHandler(app, paymentService, cfg, jwtService, redis)

	go func() {
		if cfg.App.AppPort == "" {
			cfg.App.AppPort = os.Getenv("APP_PORT")
		}

		port := ":" + cfg.App.AppPort

		log.Info().
			Str("port", port).
			Str("source", "internal.app.RunServer").
			Msg("server started")

		err = app.Listen(
			port,
			fiber.ListenConfig{
				EnablePrefork: cfg.App.WebPrefork,
			},
		)

		if err != nil {
			log.Fatal().
				Err(err).
				Str("source", "internal.app.RunServer").
				Msg("failed start server")
		}
	}()

	terminateSignals := make(chan os.Signal, 1)

	signal.Notify(
		terminateSignals,
		os.Interrupt,
		syscall.SIGTERM,
	)

	<-terminateSignals

	log.Info().
		Str("source", "internal.app.RunServer").
		Msg("shutting down server in 5 seconds")

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.app.RunServer").
			Msg("failed shutdown server")
	}

	log.Info().
		Str("source", "internal.app.RunServer").
		Msg("server stopped gracefully")

}
