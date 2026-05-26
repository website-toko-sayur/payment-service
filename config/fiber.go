package config

import (
	"payment-service/internal/adapter/handler/response"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
)

type structValidator struct {
	validate *validator.Validate
}

func (v *structValidator) Validate(out any) error {
	return v.validate.Struct(out)
}

func (cfg Config) NewFiber() *fiber.App {
	validate := validator.New()

	app := fiber.New(fiber.Config{
		AppName: cfg.App.AppName,

		CaseSensitive: true,
		StrictRouting: false,

		BodyLimit: 10 * 1024 * 1024,

		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,

		StructValidator: &structValidator{
			validate: validate,
		},

		ErrorHandler: NewErrorHandler(),
	})

	return app
}

func NewErrorHandler() fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		message := "internal server error"

		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
			message = e.Message
		} else {
			message = err.Error()
		}

		log.Error().
			Err(err).
			Int("status_code", code).
			Str("method", c.Method()).
			Str("path", c.Path()).
			Str("ip", c.IP()).
			Str("source", "internal.config.NewErrorHandler").
			Msg("http request failed")

		return c.Status(code).JSON(response.DefaultResponse{
			Message: message,
			Data:    nil,
		})
	}
}
